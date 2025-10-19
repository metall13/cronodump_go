package main

import (
	"fmt"
	"os"
	"strings"
)

// ClickHouseExporter экспортирует данные в формат ClickHouse
type ClickHouseExporter struct {
	config  *Config
	verbose bool
}

// NewClickHouseExporter создает новый экспортер
func NewClickHouseExporter(config *Config, verbose bool) *ClickHouseExporter {
	return &ClickHouseExporter{
		config:  config,
		verbose: verbose,
	}
}

// ExportToFile экспортирует базу данных в SQL файл
func (e *ClickHouseExporter) ExportToFile(database *CronosDatabase, outputPath string) error {
	if e.verbose {
		fmt.Printf("Экспорт в файл: %s\n", outputPath)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Записываем заголовок
	_, err = file.WriteString("-- Экспорт базы данных Cronos в ClickHouse\n")
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("-- Исходная база: %s\n", database.Path))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("-- Создано: %s\n\n", "2025"))
	if err != nil {
		return err
	}

	// Создаем и экспортируем каждую таблицу
	for _, table := range database.Tables {
		if e.verbose {
			fmt.Printf("Экспорт таблицы: %s (%d записей)\n", table.Name, len(table.Records))
		}

		err = e.exportTable(file, table)
		if err != nil {
			return fmt.Errorf("ошибка экспорта таблицы %s: %v", table.Name, err)
		}
	}

	if e.verbose {
		fmt.Printf("Экспорт завершен успешно\n")
	}

	return nil
}

// exportTable экспортирует одну таблицу
func (e *ClickHouseExporter) exportTable(file *os.File, table CronosTable) error {
	// Создаем SQL для создания таблицы
	createSQL := e.generateCreateTableSQL(table)
	_, err := file.WriteString(createSQL + "\n\n")
	if err != nil {
		return err
	}

	// Экспортируем данные батчами
	batchSize := e.config.BatchSize
	totalRecords := len(table.Records)
	
	for i := 0; i < totalRecords; i += batchSize {
		end := i + batchSize
		if end > totalRecords {
			end = totalRecords
		}

		batch := table.Records[i:end]
		insertSQL := e.generateInsertSQL(table, batch)
		
		_, err := file.WriteString(insertSQL + "\n\n")
		if err != nil {
			return err
		}

		if e.verbose && len(batch) > 0 {
			fmt.Printf("  Экспортировано записей: %d-%d из %d\n", i+1, end, totalRecords)
		}
	}

	return nil
}

// generateCreateTableSQL генерирует SQL для создания таблицы
func (e *ClickHouseExporter) generateCreateTableSQL(table CronosTable) string {
	var sql strings.Builder
	
	sql.WriteString(fmt.Sprintf("-- Таблица: %s (из папки: %s)\n", table.Name, table.FolderName))
	sql.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table.Name))

	for i, field := range table.Fields {
		if i > 0 {
			sql.WriteString(",\n")
		}
		
		clickhouseType := e.mapFieldType(field)
		sql.WriteString(fmt.Sprintf("    %s %s", field.Name, clickhouseType))
	}

	sql.WriteString("\n) ENGINE = MergeTree()")
	
	// Используем первое поле как ключ сортировки (обычно это system_number)
	if len(table.Fields) > 0 {
		sql.WriteString(fmt.Sprintf(" ORDER BY %s", table.Fields[0].Name))
	}
	
	sql.WriteString(";")

	return sql.String()
}

// mapFieldType преобразует тип поля Cronos в тип ClickHouse
func (e *ClickHouseExporter) mapFieldType(field CronosField) string {
	// Все поля в ClickHouse будут строками
	return "String"
}

// generateInsertSQL генерирует SQL для вставки данных
func (e *ClickHouseExporter) generateInsertSQL(table CronosTable, records []CronosRecord) string {
	if len(records) == 0 {
		return ""
	}

	var sql strings.Builder
	
	sql.WriteString(fmt.Sprintf("INSERT INTO %s (", table.Name))
	
	// Добавляем имена полей
	for i, field := range table.Fields {
		if i > 0 {
			sql.WriteString(", ")
		}
		sql.WriteString(field.Name)
	}
	
	sql.WriteString(") VALUES\n")

	// Добавляем записи
	for recordIndex, record := range records {
		if recordIndex > 0 {
			sql.WriteString(",\n")
		}
		
		sql.WriteString("(")
		
		for fieldIndex, fieldValue := range record.Fields {
			if fieldIndex > 0 {
				sql.WriteString(", ")
			}
			
			sql.WriteString(e.formatValue(fieldValue.Value, fieldValue.Field.Type))
		}
		
		sql.WriteString(")")
	}
	
	sql.WriteString(";")

	return sql.String()
}

// formatValue форматирует значение для SQL
func (e *ClickHouseExporter) formatValue(value interface{}, fieldType int) string {
	if value == nil {
		return "NULL"
	}

	// Все значения конвертируем в строки
	str := fmt.Sprintf("%v", value)
	
	// Специальная обработка для дат (тип 4) - конвертируем в формат дд.мм.гггг
	if fieldType == 4 { // DATE
		if str != "" && str != "NULL" {
			// Пытаемся распарсить дату и переформатировать
			formattedDate := e.formatDateString(str)
			if formattedDate != "" {
				str = formattedDate
			}
		}
	}
	
	// Экранируем одинарные кавычки
	str = strings.ReplaceAll(str, "'", "''")
	// Убираем управляющие символы
	str = strings.ReplaceAll(str, "\n", "\\n")
	str = strings.ReplaceAll(str, "\r", "\\r")
	str = strings.ReplaceAll(str, "\t", "\\t")
	str = strings.ReplaceAll(str, "\\", "\\\\")
	
	return fmt.Sprintf("'%s'", str)
}

// ExportTableToFile экспортирует одну таблицу в уже открытый файл
func (e *ClickHouseExporter) ExportTableToFile(file *os.File, table CronosTable) error {
	// Создаем SQL для создания таблицы
	createSQL := e.generateCreateTableSQL(table)
	_, err := file.WriteString(createSQL + "\n\n")
	if err != nil {
		return err
	}

	// Экспортируем данные батчами
	batchSize := e.config.BatchSize
	totalRecords := len(table.Records)
	
	for i := 0; i < totalRecords; i += batchSize {
		end := i + batchSize
		if end > totalRecords {
			end = totalRecords
		}

		batch := table.Records[i:end]
		insertSQL := e.generateInsertSQL(table, batch)
		
		_, err := file.WriteString(insertSQL + "\n\n")
		if err != nil {
			return err
		}

		if e.verbose && len(batch) > 0 {
			fmt.Printf("    Экспортировано записей: %d-%d из %d\n", i+1, end, totalRecords)
		}
	}

	return nil
}