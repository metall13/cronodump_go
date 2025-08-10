package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseClient обертка для работы с ClickHouse
type ClickHouseClient struct {
	conn driver.Conn
	cfg  Config
}

// NewClickHouseClient создает новое подключение к ClickHouse
func NewClickHouseClient(cfg Config) (*ClickHouseClient, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.ClickHouseHost, cfg.ClickHousePort)},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouseDatabase,
			Username: cfg.ClickHouseUsername,
			Password: cfg.ClickHousePassword,
		},
		// TLS настройки
		Settings: map[string]interface{}{
			"secure": cfg.ClickHouseSSL,
		},
		Debugf: func(format string, v ...interface{}) {
			fmt.Printf("[ClickHouse Debug] "+format+"\n", v...)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к ClickHouse: %w", err)
	}

	client := &ClickHouseClient{
		conn: conn,
		cfg:  cfg,
	}

	// Проверяем подключение
	if err := client.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("не удалось проверить подключение к ClickHouse: %w", err)
	}

	return client, nil
}

// Ping проверяет подключение к ClickHouse
func (c *ClickHouseClient) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// Close закрывает подключение к ClickHouse
func (c *ClickHouseClient) Close() error {
	return c.conn.Close()
}

// CreateTable создает таблицу в ClickHouse для данных из Kronos
func (c *ClickHouseClient) CreateTable(ctx context.Context, tableName string, columns []Column) error {
	// Формируем SQL для создания таблицы
	var columnDefinitions []string
	
	for _, col := range columns {
		columnName := TransliterateColumnName(col.Name)
		// Все данные из Kronos сохраняем как строки
		columnDefinitions = append(columnDefinitions, fmt.Sprintf("`%s` String", columnName))
	}
	
	// Добавляем служебные колонки
	columnDefinitions = append(columnDefinitions, "`import_timestamp` DateTime DEFAULT now()")
	columnDefinitions = append(columnDefinitions, "`source_table` String")
	
	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			%s
		) ENGINE = MergeTree()
		ORDER BY import_timestamp
	`, tableName, strings.Join(columnDefinitions, ",\n\t\t\t"))
	
	fmt.Printf("Создаем таблицу: %s\n", tableName)
	fmt.Printf("SQL: %s\n", sql)
	
	return c.conn.Exec(ctx, sql)
}

// InsertData вставляет данные в таблицу ClickHouse
func (c *ClickHouseClient) InsertData(ctx context.Context, tableName string, columns []Column, rows [][]string, sourceTable string) error {
	if len(rows) == 0 {
		return nil
	}
	
	// Подготавливаем имена колонок
	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, fmt.Sprintf("`%s`", TransliterateColumnName(col.Name)))
	}
	
	// Добавляем служебные колонки
	columnNames = append(columnNames, "`import_timestamp`", "`source_table`")
	
	// Создаем placeholders для VALUES
	placeholders := make([]string, len(columnNames))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	
	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columnNames, ", "),
		strings.Join(placeholders, ", "),
	)
	
	fmt.Printf("Вставляем %d строк в таблицу %s\n", len(rows), tableName)
	
	// Подготавливаем batch для вставки
	batch, err := c.conn.PrepareBatch(ctx, sql)
	if err != nil {
		return fmt.Errorf("не удалось подготовить batch: %w", err)
	}
	
	// Добавляем строки в batch
	for _, row := range rows {
		// Убеждаемся, что количество значений соответствует количеству колонок
		if len(row) != len(columns) {
			// Дополняем или обрезаем строку до нужного размера
			adjustedRow := make([]string, len(columns))
			for i := range adjustedRow {
				if i < len(row) {
					adjustedRow[i] = row[i]
				} else {
					adjustedRow[i] = ""
				}
			}
			row = adjustedRow
		}
		
		// Подготавливаем значения для вставки (все как строки + служебные колонки)
		values := make([]interface{}, len(columnNames))
		for i, value := range row {
			values[i] = value
		}
		// Добавляем текущее время и источник
		values[len(row)] = "now()" // import_timestamp
		values[len(row)+1] = sourceTable // source_table
		
		if err := batch.Append(values...); err != nil {
			return fmt.Errorf("не удалось добавить строку в batch: %w", err)
		}
	}
	
	// Выполняем batch
	if err := batch.Send(); err != nil {
		return fmt.Errorf("не удалось выполнить batch insert: %w", err)
	}
	
	fmt.Printf("Успешно вставлено %d строк\n", len(rows))
	return nil
}

// GetTableInfo получает информацию о существующей таблице
func (c *ClickHouseClient) GetTableInfo(ctx context.Context, tableName string) ([]Column, error) {
	sql := `
		SELECT name, type 
		FROM system.columns 
		WHERE database = ? AND table = ?
	`
	
	rows, err := c.conn.Query(ctx, sql, c.cfg.ClickHouseDatabase, tableName)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить информацию о таблице: %w", err)
	}
	defer rows.Close()
	
	var columns []Column
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return nil, fmt.Errorf("не удалось прочитать информацию о колонке: %w", err)
		}
		
		columns = append(columns, Column{
			Name: name,
			Type: dataType,
		})
	}
	
	return columns, rows.Err()
}

// TableExists проверяет существование таблицы
func (c *ClickHouseClient) TableExists(ctx context.Context, tableName string) (bool, error) {
	sql := `
		SELECT count() 
		FROM system.tables 
		WHERE database = ? AND name = ?
	`
	
	var count uint64
	err := c.conn.QueryRow(ctx, sql, c.cfg.ClickHouseDatabase, tableName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("не удалось проверить существование таблицы: %w", err)
	}
	
	return count > 0, nil
}