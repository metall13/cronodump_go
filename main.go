package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CronosConverter основной конвертер
type CronosConverter struct {
	config          *Config
	parser          *CronosParser
	clickHouseClient *ClickHouseClient
	logger          func(format string, args ...interface{})
}

// NewCronosConverter создает новый конвертер
func NewCronosConverter(config *Config) (*CronosConverter, error) {
	// Создаем парсер Cronos
	parser := NewCronosParser()
	
	// Создаем клиент ClickHouse
	clickHouseClient, err := NewClickHouseClient(config)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать клиент ClickHouse: %w", err)
	}
	
	converter := &CronosConverter{
		config:          config,
		parser:          parser,
		clickHouseClient: clickHouseClient,
		logger: func(format string, args ...interface{}) {
			log.Printf("[CronosConverter] "+format, args...)
		},
	}
	
	// Настраиваем логирование
	if config.Verbose {
		parser.SetLogger(converter.logger)
		clickHouseClient.SetLogger(converter.logger)
	}
	
	return converter, nil
}

// Close закрывает соединения
func (c *CronosConverter) Close() error {
	if c.clickHouseClient != nil {
		return c.clickHouseClient.Close()
	}
	return nil
}

// ConvertAll конвертирует все базы данных в указанной папке
func (c *CronosConverter) ConvertAll(ctx context.Context) error {
	c.logger("Начинаем конвертацию всех баз данных из: %s", c.config.CronosBasePath)
	
	// Находим все папки с базами данных
	databases, err := c.findDatabases()
	if err != nil {
		return fmt.Errorf("ошибка поиска баз данных: %w", err)
	}
	
	c.logger("Найдено баз данных: %d", len(databases))
	
	totalTables := 0
	totalRecords := 0
	
	// Конвертируем каждую базу данных
	for _, dbPath := range databases {
		tables, records, err := c.ConvertDatabase(ctx, dbPath)
		if err != nil {
			c.logger("Ошибка конвертации базы данных %s: %v", dbPath, err)
			continue
		}
		
		totalTables += tables
		totalRecords += records
		c.logger("База данных %s: таблиц=%d, записей=%d", filepath.Base(dbPath), tables, records)
	}
	
	c.logger("Конвертация завершена: всего таблиц=%d, записей=%d", totalTables, totalRecords)
	return nil
}

// ConvertDatabase конвертирует одну базу данных
func (c *CronosConverter) ConvertDatabase(ctx context.Context, dbPath string) (int, int, error) {
	c.logger("Конвертация базы данных: %s", dbPath)
	
	// Парсим базу данных Cronos
	database, err := c.parser.ParseDatabase(dbPath)
	if err != nil {
		return 0, 0, fmt.Errorf("ошибка парсинга базы данных: %w", err)
	}
	
	if len(database.Tables) == 0 {
		c.logger("База данных %s не содержит таблиц", database.Name)
		return 0, 0, nil
	}
	
	folderName := database.Name
	totalRecords := 0
	
	// Конвертируем каждую таблицу
	for _, table := range database.Tables {
		if err := c.ConvertTable(ctx, folderName, &table); err != nil {
			c.logger("Ошибка конвертации таблицы %s: %v", table.Name, err)
			continue
		}
		
		totalRecords += len(table.Records)
	}
	
	return len(database.Tables), totalRecords, nil
}

// ConvertTable конвертирует одну таблицу
func (c *CronosConverter) ConvertTable(ctx context.Context, folderName string, table *CronosTable) error {
	c.logger("Конвертация таблицы: %s/%s", folderName, table.Name)
	
	// Создаем таблицу в ClickHouse
	if err := c.clickHouseClient.CreateTableFromCronos(ctx, folderName, table); err != nil {
		return fmt.Errorf("ошибка создания таблицы: %w", err)
	}
	
	// Вставляем данные
	if err := c.clickHouseClient.InsertCronosData(ctx, folderName, table); err != nil {
		return fmt.Errorf("ошибка вставки данных: %w", err)
	}
	
	return nil
}

// findDatabases находит все папки с базами данных Cronos
func (c *CronosConverter) findDatabases() ([]string, error) {
	var databases []string
	
	// Читаем содержимое корневой папки
	entries, err := os.ReadDir(c.config.CronosBasePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать папку %s: %w", c.config.CronosBasePath, err)
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			dbPath := filepath.Join(c.config.CronosBasePath, entry.Name())
			
			// Проверяем, есть ли в папке файлы .dat (признак базы данных Cronos)
			if c.hasCronosFiles(dbPath) {
				databases = append(databases, dbPath)
			}
		}
	}
	
	return databases, nil
}

// hasCronosFiles проверяет, содержит ли папка файлы Cronos
func (c *CronosConverter) hasCronosFiles(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".dat" {
			return true
		}
	}
	
	return false
}

// main функция программы
func main() {
	fmt.Println("=== Конвертер Cronos -> ClickHouse ===")
	fmt.Println("Версия: 1.0")
	fmt.Println()
	
	// Загружаем конфигурацию
	config, err := LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}
	
	// Валидируем конфигурацию
	if err := config.Validate(); err != nil {
		log.Fatalf("Ошибка валидации конфигурации: %v", err)
	}
	
	fmt.Printf("Конфигурация: %s\n", config.String())
	fmt.Println()
	
	// Создаем конвертер
	converter, err := NewCronosConverter(config)
	if err != nil {
		log.Fatalf("Ошибка создания конвертера: %v", err)
	}
	defer converter.Close()
	
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	// Запускаем конвертацию
	startTime := time.Now()
	if err := converter.ConvertAll(ctx); err != nil {
		log.Fatalf("Ошибка конвертации: %v", err)
	}
	
	duration := time.Since(startTime)
	fmt.Printf("\nКонвертация завершена успешно за %v\n", duration)
}