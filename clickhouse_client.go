package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Column описание колонки для ClickHouse
type Column struct {
	Name string
	Type string
}

// ClickHouseClient клиент для работы с ClickHouse
type ClickHouseClient struct {
	conn   driver.Conn
	config *Config
	logger func(format string, args ...interface{})
}

// NewClickHouseClient создает новое подключение к ClickHouse
func NewClickHouseClient(config *Config) (*ClickHouseClient, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: config.GetClickHouseAddr(),
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "default",
		},
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: "cronos-converter-go", Version: "1.0"},
			},
		},
		Debugf: func(format string, v ...interface{}) {
			if config.Verbose {
				fmt.Printf("[ClickHouse Debug] "+format+"\n", v...)
			}
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:      time.Second * 30,
		MaxOpenConns:     5,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к ClickHouse: %w", err)
	}

	client := &ClickHouseClient{
		conn:   conn,
		config: config,
		logger: func(format string, args ...interface{}) {
			fmt.Printf("[ClickHouse] "+format+"\n", args...)
		},
	}

	// Проверяем подключение
	if err := client.Ping(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("не удалось проверить подключение к ClickHouse: %w", err)
	}

	// Создаем базу данных если не существует
	if err := client.initDatabase(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("не удалось инициализировать базу данных: %w", err)
	}

	return client, nil
}

// SetLogger устанавливает функцию логирования
func (c *ClickHouseClient) SetLogger(logger func(format string, args ...interface{})) {
	c.logger = logger
}

// Ping проверяет подключение к ClickHouse
func (c *ClickHouseClient) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// Close закрывает подключение к ClickHouse
func (c *ClickHouseClient) Close() error {
	return c.conn.Close()
}

// initDatabase создает базу данных если не существует
func (c *ClickHouseClient) initDatabase(ctx context.Context) error {
	// Подключаемся к default базе, но создаем нашу рабочую базу
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", c.config.ClickHouseDatabase)
	c.logger("Создание базы данных: %s", c.config.ClickHouseDatabase)
	return c.conn.Exec(ctx, sql)
}

// CreateTableFromCronos создает таблицу в ClickHouse на основе структуры Cronos
func (c *ClickHouseClient) CreateTableFromCronos(ctx context.Context, folderName string, cronosTable *CronosTable) error {
	// Генерируем имя таблицы
	tableName := TransliterateTableName(folderName, cronosTable.Name)
	
	// Формируем SQL для создания таблицы
	var columnDefinitions []string
	
	for _, field := range cronosTable.Fields {
		columnName := TransliterateColumnName(field.Name)
		// Все данные из Cronos сохраняем как строки для максимальной совместимости
		columnDefinitions = append(columnDefinitions, fmt.Sprintf("`%s` String", columnName))
	}
	
	// Добавляем служебные колонки
	columnDefinitions = append(columnDefinitions,
		"`cronos_import_timestamp` DateTime DEFAULT now()",
		"`cronos_source_folder` String",
		"`cronos_source_table` String",
		"`cronos_record_count` UInt32",
	)
	
	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS `+"`%s`.`%s`"+` (
			%s
		) ENGINE = MergeTree()
		ORDER BY cronos_import_timestamp
		PARTITION BY toYYYYMM(cronos_import_timestamp)
		SETTINGS index_granularity = 8192
	`, c.config.ClickHouseDatabase, tableName, strings.Join(columnDefinitions, ",\n\t\t\t"))
	
	c.logger("Создание таблицы: %s", tableName)
	if err := c.conn.Exec(ctx, sql); err != nil {
		return fmt.Errorf("не удалось создать таблицу %s: %w", tableName, err)
	}

	c.logger("Таблица %s успешно создана", tableName)
	return nil
}

// InsertCronosData вставляет данные из Cronos таблицы в ClickHouse
func (c *ClickHouseClient) InsertCronosData(ctx context.Context, folderName string, cronosTable *CronosTable) error {
	if len(cronosTable.Records) == 0 {
		c.logger("Таблица %s пуста, пропускаем вставку", cronosTable.Name)
		return nil
	}

	tableName := TransliterateTableName(folderName, cronosTable.Name)
	
	// Подготавливаем имена колонок
	var columnNames []string
	for _, field := range cronosTable.Fields {
		columnNames = append(columnNames, fmt.Sprintf("`%s`", TransliterateColumnName(field.Name)))
	}
	
	// Добавляем служебные колонки
	columnNames = append(columnNames, 
		"`cronos_import_timestamp`", 
		"`cronos_source_folder`", 
		"`cronos_source_table`", 
		"`cronos_record_count`")
	
	// Создаем placeholders для VALUES
	placeholders := make([]string, len(columnNames))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	
	sql := fmt.Sprintf(
		"INSERT INTO `%s`.`%s` (%s) VALUES (%s)",
		c.config.ClickHouseDatabase,
		tableName,
		strings.Join(columnNames, ", "),
		strings.Join(placeholders, ", "),
	)
	
	c.logger("Вставка %d записей в таблицу %s", len(cronosTable.Records), tableName)
	
	// Обрабатываем данные батчами
	batchSize := c.config.BatchSize
	for i := 0; i < len(cronosTable.Records); i += batchSize {
		end := i + batchSize
		if end > len(cronosTable.Records) {
			end = len(cronosTable.Records)
		}
		
		batch := cronosTable.Records[i:end]
		if err := c.insertBatch(ctx, sql, batch, cronosTable.Fields, folderName, cronosTable.Name, len(cronosTable.Records)); err != nil {
			return fmt.Errorf("ошибка вставки батча %d-%d: %w", i, end, err)
		}
		
		c.logger("Обработано записей: %d/%d", end, len(cronosTable.Records))
	}
	
	c.logger("Успешно вставлено %d записей в таблицу %s", len(cronosTable.Records), tableName)
	return nil
}

// insertBatch вставляет батч записей
func (c *ClickHouseClient) insertBatch(ctx context.Context, sql string, batch [][]string, fields []CronosField, folderName, tableName string, totalRecords int) error {
	// Подготавливаем batch для вставки
	clickhouseBatch, err := c.conn.PrepareBatch(ctx, sql)
	if err != nil {
		return fmt.Errorf("не удалось подготовить batch: %w", err)
	}
	
	currentTime := time.Now()
	
	// Добавляем строки в batch
	for _, row := range batch {
		// Убеждаемся, что количество значений соответствует количеству полей
		adjustedRow := make([]string, len(fields))
		for i := range adjustedRow {
			if i < len(row) {
				adjustedRow[i] = c.sanitizeValue(row[i])
			} else {
				adjustedRow[i] = ""
			}
		}
		
		// Подготавливаем значения для вставки
		values := make([]interface{}, len(fields)+4) // +4 для служебных колонок
		
		// Копируем данные полей
		for i, value := range adjustedRow {
			values[i] = value
		}
		
		// Добавляем служебные поля
		values[len(fields)] = currentTime    // cronos_import_timestamp
		values[len(fields)+1] = folderName   // cronos_source_folder
		values[len(fields)+2] = tableName    // cronos_source_table
		values[len(fields)+3] = totalRecords // cronos_record_count
		
		if err := clickhouseBatch.Append(values...); err != nil {
			return fmt.Errorf("не удалось добавить строку в batch: %w", err)
		}
	}
	
	// Выполняем batch
	if err := clickhouseBatch.Send(); err != nil {
		return fmt.Errorf("не удалось выполнить batch insert: %w", err)
	}
	
	return nil
}

// sanitizeValue очищает значение для безопасной вставки
func (c *ClickHouseClient) sanitizeValue(value string) string {
	// Удаляем или заменяем проблематичные символы
	value = strings.ReplaceAll(value, "\x00", "") // Удаляем null bytes
	value = strings.TrimSpace(value)
	
	// Ограничиваем длину строки (ClickHouse может иметь ограничения)
	maxLength := 65536
	if len(value) > maxLength {
		value = value[:maxLength]
	}
	
	return value
}

// TableExists проверяет существование таблицы
func (c *ClickHouseClient) TableExists(ctx context.Context, tableName string) (bool, error) {
	sql := `
		SELECT count() 
		FROM system.tables 
		WHERE database = ? AND name = ?
	`
	
	var count uint64
	err := c.conn.QueryRow(ctx, sql, c.config.ClickHouseDatabase, tableName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("не удалось проверить существование таблицы: %w", err)
	}
	
	return count > 0, nil
}

// GetTableInfo получает информацию о существующей таблице
func (c *ClickHouseClient) GetTableInfo(ctx context.Context, tableName string) ([]Column, error) {
	sql := `
		SELECT name, type 
		FROM system.columns 
		WHERE database = ? AND table = ?
		ORDER BY position
	`
	
	rows, err := c.conn.Query(ctx, sql, c.config.ClickHouseDatabase, tableName)
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

// DropTable удаляет таблицу
func (c *ClickHouseClient) DropTable(ctx context.Context, tableName string) error {
	sql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`", c.config.ClickHouseDatabase, tableName)
	c.logger("Удаление таблицы: %s", tableName)
	return c.conn.Exec(ctx, sql)
}

// GetTablesCount возвращает количество таблиц в базе данных
func (c *ClickHouseClient) GetTablesCount(ctx context.Context) (int, error) {
	sql := "SELECT count() FROM system.tables WHERE database = ?"
	
	var count int
	err := c.conn.QueryRow(ctx, sql, c.config.ClickHouseDatabase).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("не удалось получить количество таблиц: %w", err)
	}
	
	return count, nil
}

// OptimizeTable оптимизирует таблицу
func (c *ClickHouseClient) OptimizeTable(ctx context.Context, tableName string) error {
	sql := fmt.Sprintf("OPTIMIZE TABLE `%s`.`%s`", c.config.ClickHouseDatabase, tableName)
	c.logger("Оптимизация таблицы: %s", tableName)
	return c.conn.Exec(ctx, sql)
}