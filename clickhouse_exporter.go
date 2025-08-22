package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseExporter экспортер данных в ClickHouse
type ClickHouseExporter struct {
	conn    driver.Conn
	config  struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Database string `json:"database"`
		Username string `json:"username"`
		Password string `json:"password"`
		ConnStr  string `json:"connection_string"`
	}
	verbose   bool
	sqlFile   *os.File
	outputSQL bool
}

// NewClickHouseExporter создает новый экспортер
func NewClickHouseExporter(config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	ConnStr  string `json:"connection_string"`
}, verbose bool) (*ClickHouseExporter, error) {
	exporter := &ClickHouseExporter{
		config:  config,
		verbose: verbose,
	}

	// Пытаемся подключиться к ClickHouse
	ctx := context.Background()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
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
		if verbose {
			fmt.Printf("Не удалось подключиться к ClickHouse: %v\n", err)
			fmt.Println("Переключаемся на режим SQL файла")
		}
		exporter.outputSQL = true
		
		// Создаем SQL файл для вывода
		sqlFileName := fmt.Sprintf("cronos_export_%s.sql", time.Now().Format("20060102_150405"))
		exporter.sqlFile, err = os.Create(sqlFileName)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания SQL файла: %v", err)
		}
		
		if verbose {
			fmt.Printf("Создан SQL файл: %s\n", sqlFileName)
		}
	} else {
		exporter.conn = conn
		
		// Проверяем подключение
		if err := conn.Ping(ctx); err != nil {
			return nil, fmt.Errorf("ошибка проверки подключения к ClickHouse: %v", err)
		}
		
		if verbose {
			fmt.Println("Успешно подключились к ClickHouse")
		}
	}

	return exporter, nil
}

// CreateTable создает таблицу в ClickHouse
func (e *ClickHouseExporter) CreateTable(tableName string, fields []Field) error {
	var columns []string
	
	for _, field := range fields {
		columnType := e.mapCronosTypeToClickHouse(field.Type, field.MaxLen)
		columns = append(columns, fmt.Sprintf("`%s` %s", field.Name, columnType))
	}

	createSQL := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS `%s` (\n    %s\n) ENGINE = MergeTree() ORDER BY tuple();",
		tableName,
		strings.Join(columns, ",\n    "),
	)

	if e.verbose {
		fmt.Printf("Создаем таблицу: %s\n", tableName)
	}

	if e.outputSQL {
		// Записываем в SQL файл
		_, err := e.sqlFile.WriteString(createSQL + "\n\n")
		return err
	} else {
		// Выполняем в ClickHouse
		ctx := context.Background()
		return e.conn.Exec(ctx, createSQL)
	}
}

// InsertBatch вставляет батч записей в таблицу
func (e *ClickHouseExporter) InsertBatch(tableName string, fields []Field, records []*Record) error {
	if len(records) == 0 {
		return nil
	}

	// Формируем список колонок
	var columnNames []string
	for _, field := range fields {
		columnNames = append(columnNames, fmt.Sprintf("`%s`", field.Name))
	}

	if e.outputSQL {
		// Записываем INSERT statements в SQL файл
		for _, record := range records {
			var values []string
			for i, fieldValue := range record.Fields {
				value := e.formatValueForSQL(fields[i].Type, fieldValue.Content)
				values = append(values, value)
			}
			
			insertSQL := fmt.Sprintf(
				"INSERT INTO `%s` (%s) VALUES (%s);",
				tableName,
				strings.Join(columnNames, ", "),
				strings.Join(values, ", "),
			)
			
			if _, err := e.sqlFile.WriteString(insertSQL + "\n"); err != nil {
				return err
			}
		}
		
		// Добавляем пустую строку после батча
		_, err := e.sqlFile.WriteString("\n")
		return err
	} else {
		// Вставляем в ClickHouse через batch
		ctx := context.Background()
		batch, err := e.conn.PrepareBatch(ctx, fmt.Sprintf(
			"INSERT INTO `%s` (%s)",
			tableName,
			strings.Join(columnNames, ", "),
		))
		if err != nil {
			return err
		}

		for _, record := range records {
			var values []interface{}
			for i, fieldValue := range record.Fields {
				value := e.convertValueForClickHouse(fields[i].Type, fieldValue.Content)
				values = append(values, value)
			}
			
			if err := batch.Append(values...); err != nil {
				return err
			}
		}

		return batch.Send()
	}
}

// mapCronosTypeToClickHouse преобразует типы Cronos в типы ClickHouse
func (e *ClickHouseExporter) mapCronosTypeToClickHouse(cronosType int, maxLen uint32) string {
	switch cronosType {
	case 0: // INTEGER PRIMARY KEY
		return "UInt32"
	case 1: // INTEGER
		return "Int32"
	case 2: // VARCHAR
		if maxLen > 0 {
			return fmt.Sprintf("String") // ClickHouse не ограничивает длину String
		}
		return "String"
	case 3: // TEXT (dictionary)
		return "String"
	case 4: // DATE
		return "Date"
	case 5: // TIMESTAMP
		return "DateTime"
	case 6: // TEXT (file reference)
		return "String"
	default:
		return "String"
	}
}

// formatValueForSQL форматирует значение для SQL
func (e *ClickHouseExporter) formatValueForSQL(fieldType int, content string) string {
	switch fieldType {
	case 0, 1: // INTEGER
		if content == "" {
			return "0"
		}
		return content
	case 4: // DATE
		if content == "" {
			return "'1900-01-01'"
		}
		return fmt.Sprintf("'%s'", strings.ReplaceAll(content, "'", "''"))
	case 5: // TIMESTAMP
		if content == "" {
			return "'1900-01-01 00:00:00'"
		}
		return fmt.Sprintf("'%s'", strings.ReplaceAll(content, "'", "''"))
	default: // STRING
		return fmt.Sprintf("'%s'", strings.ReplaceAll(content, "'", "''"))
	}
}

// convertValueForClickHouse конвертирует значение для ClickHouse
func (e *ClickHouseExporter) convertValueForClickHouse(fieldType int, content string) interface{} {
	switch fieldType {
	case 0, 1: // INTEGER
		if content == "" {
			return 0
		}
		// Здесь нужно будет добавить парсинг строки в число
		return content
	case 4: // DATE
		if content == "" {
			return time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		// Здесь нужно будет добавить парсинг даты
		return content
	case 5: // TIMESTAMP
		if content == "" {
			return time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		// Здесь нужно будет добавить парсинг времени
		return content
	default: // STRING
		return content
	}
}

// Close закрывает экспортер
func (e *ClickHouseExporter) Close() error {
	var errors []string

	if e.conn != nil {
		if err := e.conn.Close(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if e.sqlFile != nil {
		if err := e.sqlFile.Close(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("ошибки закрытия экспортера: %s", strings.Join(errors, "; "))
	}

	return nil
}