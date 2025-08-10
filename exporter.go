package cronodamp

import (
	"context"
	"fmt"
	"time"
)

// CronodampExporter основной класс для экспорта данных из Cronos в ClickHouse
type CronodampExporter struct {
	config        Config
	cronosReader  *CronosFileReader
	clickHouseClient *ClickHouseClient
}

// NewCronodampExporter создает новый экспортер
func NewCronodampExporter(cfg Config) (*CronodampExporter, error) {
	// Создаем ридер для Cronos
	cronosReader := NewCronosReader(cfg)
	
	// Создаем клиент ClickHouse (может быть nil для операций только чтения)
	clickHouseClient, err := NewClickHouseClient(cfg)
	if err != nil {
		fmt.Printf("⚠️ Предупреждение: не удалось подключиться к ClickHouse: %v\n", err)
		fmt.Printf("Доступны только операции просмотра данных\n")
		clickHouseClient = nil
	}
	
	return &CronodampExporter{
		config:           cfg,
		cronosReader:     cronosReader,
		clickHouseClient: clickHouseClient,
	}, nil
}

// NewCronodampReader создает новый ридер только для чтения (без ClickHouse)
func NewCronodampReader(cfg Config) (*CronodampExporter, error) {
	// Создаем только ридер для Cronos
	cronosReader := NewCronosReader(cfg)
	
	return &CronodampExporter{
		config:           cfg,
		cronosReader:     cronosReader,
		clickHouseClient: nil,
	}, nil
}

// Close закрывает все подключения
func (e *CronodampExporter) Close() error {
	if e.clickHouseClient != nil {
		return e.clickHouseClient.Close()
	}
	return nil
}

// ExportDatabase экспортирует всю базу данных из Cronos в ClickHouse
func (e *CronodampExporter) ExportDatabase(ctx context.Context) ([]*ExportResult, error) {
	if e.clickHouseClient == nil {
		return nil, fmt.Errorf("ClickHouse клиент не подключен - экспорт невозможен")
	}
	
	fmt.Printf("Начинаем экспорт базы данных из: %s\n", e.config.DatabasePath)
	
	// Читаем базу данных Cronos
	database, err := e.cronosReader.ReadDatabase(e.config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать базу данных Cronos: %w", err)
	}
	
	var results []*ExportResult
	
	// Экспортируем каждую таблицу
	for _, table := range database.Tables {
		result := e.exportSingleTable(ctx, &table)
		results = append(results, result)
		
		if result.Success {
			fmt.Printf("✅ Таблица %s экспортирована успешно (%d строк)\n", result.TableName, result.RowsExported)
		} else {
			fmt.Printf("❌ Ошибка при экспорте таблицы %s: %v\n", result.TableName, result.Error)
		}
	}
	
	// Выводим итоговую статистику
	successful := 0
	totalRows := 0
	for _, result := range results {
		if result.Success {
			successful++
			totalRows += result.RowsExported
		}
	}
	
	fmt.Printf("\n📊 Итоги экспорта:\n")
	fmt.Printf("   Успешно экспортировано: %d/%d таблиц\n", successful, len(results))
	fmt.Printf("   Общее количество строк: %d\n", totalRows)
	
	return results, nil
}

// ExportTable экспортирует отдельную таблицу
func (e *CronodampExporter) ExportTable(ctx context.Context, tablePath string) (*ExportResult, error) {
	// Читаем таблицу
	table, err := e.cronosReader.ReadTable(tablePath)
	if err != nil {
		return &ExportResult{
			TableName: tablePath,
			Success:   false,
			Error:     fmt.Errorf("не удалось прочитать таблицу: %w", err),
			CreatedAt: time.Now(),
		}, err
	}
	
	// Экспортируем таблицу
	result := e.exportSingleTable(ctx, table)
	return result, nil
}

// exportSingleTable экспортирует одну таблицу в ClickHouse
func (e *CronodampExporter) exportSingleTable(ctx context.Context, table *Table) *ExportResult {
	startTime := time.Now()
	
	if e.clickHouseClient == nil {
		return &ExportResult{
			TableName: table.Name,
			CreatedAt: startTime,
			Success:   false,
			Error:     fmt.Errorf("ClickHouse клиент не подключен"),
		}
	}
	
	result := &ExportResult{
		TableName: table.Name,
		CreatedAt: startTime,
	}
	
	// Проверяем, есть ли данные для экспорта
	if len(table.Data) == 0 {
		fmt.Printf("⚠️  Таблица %s пуста, пропускаем\n", table.Name)
		result.Success = true
		result.RowsExported = 0
		return result
	}
	
	// Создаем таблицу в ClickHouse
	err := e.clickHouseClient.CreateTable(ctx, table.Name, table.Columns)
	if err != nil {
		result.Error = fmt.Errorf("не удалось создать таблицу в ClickHouse: %w", err)
		result.Success = false
		return result
	}
	
	// Вставляем данные
	err = e.clickHouseClient.InsertData(ctx, table.Name, table.Columns, table.Data, table.SourcePath)
	if err != nil {
		result.Error = fmt.Errorf("не удалось вставить данные в ClickHouse: %w", err)
		result.Success = false
		return result
	}
	
	result.Success = true
	result.RowsExported = len(table.Data)
	
	return result
}

// GetDatabaseInfo возвращает информацию о базе данных
func (e *CronodampExporter) GetDatabaseInfo(ctx context.Context) (*DatabaseInfo, error) {
	tableNames, err := e.cronosReader.GetTableNames(e.config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список таблиц: %w", err)
	}
	
	return &DatabaseInfo{
		Path:       e.config.DatabasePath,
		TableCount: len(tableNames),
		CreatedAt:  time.Now(),
	}, nil
}

// ListTables возвращает список всех таблиц в базе данных
func (e *CronodampExporter) ListTables(ctx context.Context) ([]string, error) {
	return e.cronosReader.GetTableNames(e.config.DatabasePath)
}

// ValidateConfig проверяет корректность конфигурации
func (e *CronodampExporter) ValidateConfig() error {
	// Проверяем путь к базе данных
	if e.config.DatabasePath == "" {
		return fmt.Errorf("путь к базе данных не указан")
	}
	
	// Проверяем подключение к ClickHouse только если клиент доступен
	if e.clickHouseClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		err := e.clickHouseClient.Ping(ctx)
		if err != nil {
			fmt.Printf("⚠️ Предупреждение: не удалось подключиться к ClickHouse: %v\n", err)
			fmt.Printf("✅ Конфигурация валидна (только чтение)\n")
			fmt.Printf("   База данных: %s\n", e.config.DatabasePath)
			fmt.Printf("   ClickHouse: не подключен\n")
		} else {
			fmt.Printf("✅ Конфигурация валидна\n")
			fmt.Printf("   База данных: %s\n", e.config.DatabasePath)
			fmt.Printf("   ClickHouse: %s ✅\n", e.config.ClickHouseDSN)
		}
	} else {
		fmt.Printf("✅ Конфигурация валидна (только чтение)\n")
		fmt.Printf("   База данных: %s\n", e.config.DatabasePath)
		fmt.Printf("   ClickHouse: не подключен\n")
	}
	
	return nil
}

// ExportWithProgress экспортирует базу данных с отображением прогресса
func (e *CronodampExporter) ExportWithProgress(ctx context.Context) ([]*ExportResult, error) {
	// Сначала получаем список таблиц для отображения прогресса
	tableNames, err := e.cronosReader.GetTableNames(e.config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список таблиц: %w", err)
	}
	
	fmt.Printf("🚀 Начинаем экспорт %d таблиц...\n", len(tableNames))
	
	// Читаем базу данных
	database, err := e.cronosReader.ReadDatabase(e.config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать базу данных: %w", err)
	}
	
	var results []*ExportResult
	
	for i, table := range database.Tables {
		fmt.Printf("📋 [%d/%d] Экспортируем таблицу: %s\n", i+1, len(database.Tables), table.Name)
		
		result := e.exportSingleTable(ctx, &table)
		results = append(results, result)
		
		if result.Success {
			fmt.Printf("   ✅ Успешно (%d строк)\n", result.RowsExported)
		} else {
			fmt.Printf("   ❌ Ошибка: %v\n", result.Error)
		}
	}
	
	return results, nil
}