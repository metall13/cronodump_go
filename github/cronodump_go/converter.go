package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CronosConverter основной конвертер
type CronosConverter struct {
	config        *Config
	verbose       bool
	parser        *CronosParser
	transliterator *Transliterator
	exporter      *ClickHouseExporter
	clickhouse    *ClickHouseImporter
}

// NewCronosConverter создает новый конвертер
func NewCronosConverter(config *Config, verbose bool) *CronosConverter {
	return &CronosConverter{
		config:        config,
		verbose:       verbose,
		parser:        NewCronosParser(verbose),
		transliterator: NewTransliterator(),
		exporter:      NewClickHouseExporter(config, verbose),
		clickhouse:    NewClickHouseImporter(config, verbose),
	}
}

// Convert выполняет полную конвертацию
func (c *CronosConverter) Convert() error {
	startTime := time.Now()
	
	if c.verbose {
		fmt.Printf("Начинаем конвертацию базы данных Cronos...\n")
		fmt.Printf("Входная папка: %s\n", c.config.InputDir)
		fmt.Printf("Выходная папка: %s\n", c.config.OutputDir)
	}

	// 1. Находим все базы данных Cronos в папке и подпапках
	databases, err := c.parser.FindCronosDatabases(c.config.InputDir)
	if err != nil {
		return fmt.Errorf("ошибка поиска баз данных: %v", err)
	}

	if len(databases) == 0 {
		return fmt.Errorf("не найдено ни одной базы данных Cronos в папке %s", c.config.InputDir)
	}

	if c.verbose {
		fmt.Printf("Найдено баз данных: %d\n", len(databases))
	}

	// 2. Создаем выходную папку
	err = os.MkdirAll(c.config.OutputDir, 0755)
	if err != nil {
		return fmt.Errorf("ошибка создания выходной папки: %v", err)
	}

	// 3. Обрабатываем все найденные базы данных
	err = c.processAllDatabases(databases)
	if err != nil {
		return err
	}

	// 4. Удаляем временные файлы (если включено)
	if c.config.Cleanup {
		err = c.cleanupTempFiles()
		if err != nil && c.verbose {
			fmt.Printf("Предупреждение: ошибка удаления временных файлов: %v\n", err)
		}
	}

	duration := time.Since(startTime)
	if c.verbose {
		fmt.Printf("\n=== Конвертация завершена за %v ===\n", duration)
		fmt.Printf("Обработано баз данных: %d\n", len(databases))
	}

	return nil
}

// applyTransliteration применяет транслитерацию к именам таблиц и полей
func (c *CronosConverter) applyTransliteration(database *CronosDatabase) {
	if c.verbose {
		fmt.Println("Применяем транслитерацию...")
	}

	for i := range database.Tables {
		table := &database.Tables[i]
		
		// Транслитерируем имя таблицы
		table.Name = c.transliterator.TransliterateTableName(table.FolderName)
		
		// Транслитерируем имена полей
		for j := range table.Fields {
			field := &table.Fields[j]
			field.Name = c.transliterator.TransliterateFieldName(field.OriginalName)
		}
	}
}

// analyzeRelations анализирует связи между таблицами
func (c *CronosConverter) analyzeRelations(database *CronosDatabase) {
	if c.verbose {
		fmt.Println("Анализируем связи между таблицами...")
	}

	for i := range database.Tables {
		table := &database.Tables[i]
		
		// Ищем поля-ссылки
		for _, field := range table.Fields {
			if field.Type == FieldTypeDirectLink || 
			   field.Type == FieldTypeBackLink || 
			   field.Type == FieldTypeDirectBack ||
			   field.Type == FieldTypeFieldLink {
				
				// Определяем целевую таблицу по типу связи
				targetTable := c.findTargetTable(database, field, table.Name)
				if targetTable != "" {
					relation := TableRelation{
						SourceTable:  table.Name,
						TargetTable:  targetTable,
						SourceField:  field.Name,
						TargetField:  "system_number", // По умолчанию ссылаемся на системный номер
						RelationType: field.Type,
						IsRequired:   (field.Flags & FlagObligatory) != 0,
					}
					table.Relations = append(table.Relations, relation)
				}
			}
		}
	}
}

// findTargetTable находит целевую таблицу для связи
func (c *CronosConverter) findTargetTable(database *CronosDatabase, field CronosField, sourceTable string) string {
	// Упрощенная логика - в реальной реализации нужно анализировать метаданные
	// Пока возвращаем первую найденную таблицу, отличную от исходной
	for _, table := range database.Tables {
		if table.Name != sourceTable {
			return table.Name
		}
	}
	return ""
}

// mergeRelatedTables объединяет связанные таблицы
func (c *CronosConverter) mergeRelatedTables(database *CronosDatabase) ([]CronosTable, error) {
	if c.verbose {
		fmt.Println("Объединяем связанные таблицы...")
	}

	var mergedTables []CronosTable
	
	// Создаем карту таблиц для быстрого поиска
	tableMap := make(map[string]*CronosTable)
	for i := range database.Tables {
		tableMap[database.Tables[i].Name] = &database.Tables[i]
	}

	// Обрабатываем каждую таблицу
	for _, table := range database.Tables {
		mergedTable := table
		
		// Добавляем поля из связанных таблиц
		for _, relation := range table.Relations {
			if targetTable, exists := tableMap[relation.TargetTable]; exists {
				// Добавляем поля из целевой таблицы с префиксом
				for _, field := range targetTable.Fields {
					if field.Name != "system_number" { // Пропускаем системный номер
						prefixedField := field
						prefixedField.Name = relation.TargetTable + "_" + field.Name
						prefixedField.OriginalName = relation.TargetTable + "_" + field.OriginalName
						mergedTable.Fields = append(mergedTable.Fields, prefixedField)
					}
				}
			}
		}

		// Обогащаем записи данными из связанных таблиц
		c.enrichRecordsWithRelatedData(&mergedTable, tableMap)
		
		mergedTables = append(mergedTables, mergedTable)
	}

	return mergedTables, nil
}

// enrichRecordsWithRelatedData обогащает записи данными из связанных таблиц
func (c *CronosConverter) enrichRecordsWithRelatedData(table *CronosTable, tableMap map[string]*CronosTable) {
	// Создаем индексы связанных таблиц для быстрого поиска
	relatedData := make(map[string]map[uint32]map[string]interface{})
	
	for _, relation := range table.Relations {
		if targetTable, exists := tableMap[relation.TargetTable]; exists {
			relatedData[relation.TargetTable] = make(map[uint32]map[string]interface{})
			
			// Индексируем данные целевой таблицы по системному номеру
			for _, record := range targetTable.Records {
				recordData := make(map[string]interface{})
				for _, fieldValue := range record.Fields {
					recordData[fieldValue.Field.Name] = fieldValue.Value
				}
				relatedData[relation.TargetTable][record.SystemNumber] = recordData
			}
		}
	}

	// Обогащаем записи
	for i := range table.Records {
		record := &table.Records[i]
		
		// Добавляем данные из связанных таблиц
		for _, relation := range table.Relations {
			if targetData, exists := relatedData[relation.TargetTable]; exists {
				// Ищем связанную запись (упрощенная логика)
				for systemNumber, data := range targetData {
					// В реальной реализации нужно использовать правильную логику связывания
					if systemNumber == record.SystemNumber {
						for fieldName, value := range data {
							prefixedFieldName := relation.TargetTable + "_" + fieldName
							fieldValue := CronosFieldValue{
								Field: CronosField{Name: prefixedFieldName},
								Value: value,
							}
							record.Fields = append(record.Fields, fieldValue)
						}
						break
					}
				}
			}
		}
	}
}

// importToClickHouse импортирует данные в ClickHouse
func (c *CronosConverter) importToClickHouse(tables []CronosTable) error {
	if c.verbose {
		fmt.Println("Импортируем данные в ClickHouse...")
	}

	return c.clickhouse.ImportTables(tables)
}

// cleanupTempFiles удаляет временные файлы
func (c *CronosConverter) cleanupTempFiles() error {
	if c.verbose {
		fmt.Println("Удаляем временные файлы...")
	}

	// Удаляем SQL файлы
	return os.RemoveAll(c.config.OutputDir)
}

// processAllDatabases обрабатывает все найденные базы данных
func (c *CronosConverter) processAllDatabases(databases []string) error {
	var allMergedTables []CronosTable
	totalTables := 0

	// Обрабатываем каждую найденную базу данных
	for i, dbPath := range databases {
		if c.verbose {
			fmt.Printf("\n=== Обработка базы данных %d из %d: %s ===\n", i+1, len(databases), dbPath)
		}

		// Парсим базу данных
		database, err := c.parser.ParseDatabase(dbPath)
		if err != nil {
			if c.verbose {
				fmt.Printf("Предупреждение: ошибка парсинга базы %s: %v\n", dbPath, err)
			}
			continue
		}

		// Применяем транслитерацию к именам таблиц и полей
		c.applyTransliteration(database)

		// Анализируем связи между таблицами
		c.analyzeRelations(database)

		// Объединяем связанные таблицы
		mergedTables, err := c.mergeRelatedTables(database)
		if err != nil {
			if c.verbose {
				fmt.Printf("Предупреждение: ошибка объединения таблиц в базе %s: %v\n", dbPath, err)
			}
			continue
		}

		// Добавляем префикс имени базы к именам таблиц для избежания конфликтов
		baseName := filepath.Base(dbPath)
		for j := range mergedTables {
			mergedTables[j].Name = fmt.Sprintf("%s_%s", baseName, mergedTables[j].Name)
		}

		// Экспортируем в SQL файлы
		for _, table := range mergedTables {
			outputPath := filepath.Join(c.config.OutputDir, table.Name+".sql")
			err = c.exporter.ExportToFile(&CronosDatabase{Tables: []CronosTable{table}}, outputPath)
			if err != nil {
				if c.verbose {
					fmt.Printf("Предупреждение: ошибка экспорта таблицы %s: %v\n", table.Name, err)
				}
				continue
			}
		}

		allMergedTables = append(allMergedTables, mergedTables...)
		totalTables += len(mergedTables)

		if c.verbose {
			fmt.Printf("Обработано таблиц в базе %s: %d\n", baseName, len(mergedTables))
		}
	}

	// Импортируем в ClickHouse (если указан URL)
	if c.config.ClickHouse != "" {
		err := c.importToClickHouse(allMergedTables)
		if err != nil {
			return fmt.Errorf("ошибка импорта в ClickHouse: %v", err)
		}
	}

	if c.verbose {
		fmt.Printf("\n=== Итого обработано таблиц: %d ===\n", totalTables)
	}

	return nil
}