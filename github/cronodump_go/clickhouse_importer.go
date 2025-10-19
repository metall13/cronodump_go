package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// ClickHouseImporter импортирует данные в ClickHouse
type ClickHouseImporter struct {
	config  *Config
	verbose bool
	conn    clickhouse.Conn
}

// NewClickHouseImporter создает новый импортер
func NewClickHouseImporter(config *Config, verbose bool) *ClickHouseImporter {
	return &ClickHouseImporter{
		config:  config,
		verbose: verbose,
	}
}

// ImportTables импортирует таблицы в ClickHouse
func (i *ClickHouseImporter) ImportTables(tables []CronosTable) error {
	if i.config.ClickHouse == "" {
		return fmt.Errorf("URL ClickHouse не указан")
	}

	if i.verbose {
		fmt.Printf("Подключаемся к ClickHouse: %s\n", i.config.ClickHouse)
	}

	// Подключаемся к ClickHouse
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{i.config.ClickHouse},
	})
	if err != nil {
		return fmt.Errorf("ошибка подключения к ClickHouse: %v", err)
	}
	defer conn.Close()

	i.conn = conn

	// Проверяем соединение
	ctx := context.Background()
	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("ошибка проверки соединения с ClickHouse: %v", err)
	}

	if i.verbose {
		fmt.Println("Успешно подключились к ClickHouse")
	}

	// Импортируем каждую таблицу
	for _, table := range tables {
		err := i.importTable(ctx, table)
		if err != nil {
			return fmt.Errorf("ошибка импорта таблицы %s: %v", table.Name, err)
		}
	}

	if i.verbose {
		fmt.Println("Импорт в ClickHouse завершен успешно")
	}

	return nil
}

// importTable импортирует одну таблицу
func (i *ClickHouseImporter) importTable(ctx context.Context, table CronosTable) error {
	if i.verbose {
		fmt.Printf("Импортируем таблицу: %s (%d записей)\n", table.Name, len(table.Records))
	}

	// Создаем таблицу
	err := i.createTable(ctx, table)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы: %v", err)
	}

	// Вставляем данные батчами
	err = i.insertData(ctx, table)
	if err != nil {
		return fmt.Errorf("ошибка вставки данных: %v", err)
	}

	return nil
}

// createTable создает таблицу в ClickHouse
func (i *ClickHouseImporter) createTable(ctx context.Context, table CronosTable) error {
	// Генерируем SQL для создания таблицы
	exporter := NewClickHouseExporter(i.config, i.verbose)
	createSQL := exporter.generateCreateTableSQL(table)

	if i.verbose {
		fmt.Printf("Создаем таблицу: %s\n", table.Name)
	}

	// Выполняем SQL
	return i.conn.Exec(ctx, createSQL)
}

// insertData вставляет данные в таблицу
func (i *ClickHouseImporter) insertData(ctx context.Context, table CronosTable) error {
	if len(table.Records) == 0 {
		return nil
	}

	// Подготавливаем данные для вставки
	exporter := NewClickHouseExporter(i.config, i.verbose)
	
	// Вставляем данные батчами
	batchSize := i.config.BatchSize
	totalRecords := len(table.Records)

	for i := 0; i < totalRecords; i += batchSize {
		end := i + batchSize
		if end > totalRecords {
			end = totalRecords
		}

		batch := table.Records[i:end]
		insertSQL := exporter.generateInsertSQL(table, batch)

		if i.verbose && len(batch) > 0 {
			fmt.Printf("  Вставляем записи: %d-%d из %d\n", i+1, end, totalRecords)
		}

		// Выполняем вставку
		err := i.conn.Exec(ctx, insertSQL)
		if err != nil {
			return fmt.Errorf("ошибка вставки батча %d-%d: %v", i+1, end, err)
		}
	}

	return nil
}

// Close закрывает соединение с ClickHouse
func (i *ClickHouseImporter) Close() error {
	if i.conn != nil {
		return i.conn.Close()
	}
	return nil
}

// TestConnection тестирует соединение с ClickHouse
func (i *ClickHouseImporter) TestConnection() error {
	if i.config.ClickHouse == "" {
		return fmt.Errorf("URL ClickHouse не указан")
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{i.config.ClickHouse},
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return conn.Ping(ctx)
}