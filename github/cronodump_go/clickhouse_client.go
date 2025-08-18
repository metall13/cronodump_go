package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseClient клиент для работы с ClickHouse
type ClickHouseClient struct {
	conn driver.Conn
}

// NewClickHouseClient создает новый клиент ClickHouse
func NewClickHouseClient(config Config) (*ClickHouseClient, error) {
	options := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.ClickHouse.Host, config.ClickHouse.Port)},
		Auth: clickhouse.Auth{
			Database: config.ClickHouse.Database,
			Username: config.ClickHouse.Username,
			Password: config.ClickHouse.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 30,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к ClickHouse: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("не удалось проверить соединение с ClickHouse: %w", err)
	}

	return &ClickHouseClient{conn: conn}, nil
}

// Close закрывает соединение с ClickHouse
func (c *ClickHouseClient) Close() error {
	return c.conn.Close()
}

// CreateTable создает таблицу в ClickHouse
func (c *ClickHouseClient) CreateTable(ctx context.Context, tableName string, fields []FieldDefinition) error {
	var columns []string
	
	for _, field := range fields {
		columnName := TransliterateColumnName(field.Name)
		columnType := field.GetClickHouseType()
		columns = append(columns, fmt.Sprintf("`%s` %s", columnName, columnType))
	}

	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			%s
		) ENGINE = MergeTree()
		ORDER BY %s
	`, tableName, strings.Join(columns, ",\n\t\t\t"), TransliterateColumnName(fields[0].Name))

	fmt.Printf("Создаем таблицу: %s\n", tableName)
	if err := c.conn.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("не удалось создать таблицу %s: %w", tableName, err)
	}

	return nil
}

// InsertData вставляет данные в таблицу ClickHouse
func (c *ClickHouseClient) InsertData(ctx context.Context, tableName string, fields []FieldDefinition, rows [][]string, originalTableName string) error {
	if len(rows) == 0 {
		return nil
	}

	// Формируем список колонок
	var columns []string
	for _, field := range fields {
		columns = append(columns, TransliterateColumnName(field.Name))
	}

	// Подготавливаем batch
	batch, err := c.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s (%s)", tableName, strings.Join(columns, ", ")))
	if err != nil {
		return fmt.Errorf("не удалось подготовить batch: %w", err)
	}

	// Добавляем строки в batch
	for _, row := range rows {
		// Убеждаемся что количество значений соответствует количеству полей
		values := make([]interface{}, len(fields))
		for i, field := range fields {
			if i < len(row) {
				values[i] = convertValueForClickHouse(field, row[i])
			} else {
				values[i] = getDefaultValue(field.Type)
			}
		}

		if err := batch.Append(values...); err != nil {
			return fmt.Errorf("не удалось добавить строку в batch: %w", err)
		}
	}

	// Выполняем batch
	if err := batch.Send(); err != nil {
		return fmt.Errorf("не удалось выполнить batch: %w", err)
	}

	return nil
}

// convertValueForClickHouse конвертирует значение для ClickHouse
func convertValueForClickHouse(field FieldDefinition, value string) interface{} {
	if value == "" {
		return getDefaultValue(field.Type)
	}

	switch field.Type {
	case FieldTypeSystemNumber, FieldTypeInteger:
		if val, err := strconv.ParseInt(value, 10, 32); err == nil {
			return int32(val)
		}
		return int32(0)
		
	case FieldTypeDate:
		if t, err := time.Parse("2006-01-02", value); err == nil {
			return t
		}
		return time.Time{}
		
	default:
		return value
	}
}

// getDefaultValue возвращает значение по умолчанию для типа
func getDefaultValue(fieldType FieldType) interface{} {
	switch fieldType {
	case FieldTypeSystemNumber, FieldTypeInteger:
		return int32(0)
	case FieldTypeDate:
		return time.Time{}
	default:
		return ""
	}
}