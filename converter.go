package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Converter основной класс для конвертации баз Cronos
type Converter struct {
	config  Config
	verbose bool
}

// NewConverter создает новый экземпляр конвертера
func NewConverter(config Config, verbose bool) *Converter {
	return &Converter{
		config:  config,
		verbose: verbose,
	}
}

// FindCronosDatabases ищет все базы данных Cronos в указанной директории
func (c *Converter) FindCronosDatabases() ([]string, error) {
	var databases []string

	err := filepath.Walk(c.config.DatabasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ищем директории содержащие файлы CroStru.dat
		if info.IsDir() {
			struFile := filepath.Join(path, "CroStru.dat")
			if _, err := os.Stat(struFile); err == nil {
				databases = append(databases, path)
				if c.verbose {
					fmt.Printf("Найдена база данных: %s\n", path)
				}
			}
		}
		return nil
	})

	return databases, err
}

// ConvertDatabase конвертирует одну базу данных Cronos
func (c *Converter) ConvertDatabase(dbPath string) error {
	if c.verbose {
		fmt.Printf("Начинаем конвертацию: %s\n", dbPath)
	}

	// Создаем парсер базы данных
	parser, err := NewCronosParser(dbPath, c.verbose)
	if err != nil {
		return fmt.Errorf("ошибка создания парсера: %v", err)
	}
	defer parser.Close()

	// Получаем список таблиц
	tables, err := parser.GetTables()
	if err != nil {
		return fmt.Errorf("ошибка получения таблиц: %v", err)
	}

	if c.verbose {
		fmt.Printf("Найдено таблиц: %d\n", len(tables))
	}

	// Создаем экспортер в ClickHouse
	exporter, err := NewClickHouseExporter(c.config.ClickHouseConfig, c.verbose)
	if err != nil {
		return fmt.Errorf("ошибка создания экспортера ClickHouse: %v", err)
	}
	defer exporter.Close()

	// Конвертируем каждую таблицу
	for _, table := range tables {
		tableName := c.generateTableName(dbPath, table.Name)
		
		if c.verbose {
			fmt.Printf("Конвертируем таблицу: %s -> %s\n", table.Name, tableName)
		}

		if err := c.convertTable(parser, exporter, table, tableName); err != nil {
			return fmt.Errorf("ошибка конвертации таблицы %s: %v", table.Name, err)
		}
	}

	return nil
}

// generateTableName генерирует имя таблицы из пути к базе и имени таблицы
func (c *Converter) generateTableName(dbPath, tableName string) string {
	// Получаем имя папки базы данных
	dbName := filepath.Base(dbPath)
	
	// Объединяем имя базы и таблицы
	fullName := fmt.Sprintf("%s_%s", dbName, tableName)
	
	// Транслитерируем в латиницу
	transliterated := c.transliterate(fullName)
	
	// Приводим к правилам ClickHouse
	return c.sanitizeClickHouseName(transliterated)
}

// convertTable конвертирует одну таблицу
func (c *Converter) convertTable(parser *CronosParser, exporter *ClickHouseExporter, table *Table, tableName string) error {
	// Создаем таблицу в ClickHouse
	if err := exporter.CreateTable(tableName, table.Fields); err != nil {
		return fmt.Errorf("ошибка создания таблицы: %v", err)
	}

	// Читаем данные батчами по 200 записей
	batchSize := 200
	offset := 0

	for {
		records, err := parser.GetRecords(table, offset, batchSize)
		if err != nil {
			return fmt.Errorf("ошибка чтения записей: %v", err)
		}

		if len(records) == 0 {
			break // Больше записей нет
		}

		// Вставляем батч в ClickHouse
		if err := exporter.InsertBatch(tableName, table.Fields, records); err != nil {
			return fmt.Errorf("ошибка вставки батча: %v", err)
		}

		if c.verbose {
			fmt.Printf("Обработано записей: %d\n", offset+len(records))
		}

		offset += len(records)

		// Если получили меньше записей чем запрашивали, значит это последний батч
		if len(records) < batchSize {
			break
		}
	}

	return nil
}

// transliterate транслитерирует русский текст в латиницу
func (c *Converter) transliterate(text string) string {
	// Карта транслитерации
	translitMap := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
		'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
		'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
		'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
		'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
		'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "Yo",
		'Ж': "Zh", 'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M",
		'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U",
		'Ф': "F", 'Х': "H", 'Ц': "Ts", 'Ч': "Ch", 'Ш': "Sh", 'Щ': "Sch",
		'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "Yu", 'Я': "Ya",
	}

	var result strings.Builder
	for _, r := range text {
		if replacement, exists := translitMap[r]; exists {
			result.WriteString(replacement)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// sanitizeClickHouseName приводит имя к правилам ClickHouse
func (c *Converter) sanitizeClickHouseName(name string) string {
	// Заменяем недопустимые символы на подчеркивания
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, name)

	// Убираем множественные подчеркивания
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	// Убираем подчеркивания в начале и конце
	result = strings.Trim(result, "_")

	// Если имя начинается с цифры, добавляем префикс
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "table_" + result
	}

	// Если имя пустое, используем дефолтное
	if result == "" {
		result = "unknown_table"
	}

	return result
}