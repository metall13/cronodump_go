package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var (
		dbPath       = flag.String("db", "", "Путь к файлу базы данных Kronos (.fdb)")
		outputPath   = flag.String("output", "", "Путь для сохранения CSV файлов")
		configPath   = flag.String("config", "config.json", "Путь к файлу конфигурации")
		tableName    = flag.String("table", "", "Имя конкретной таблицы для экспорта")
		clickhouseMode = flag.Bool("clickhouse", false, "Экспорт в ClickHouse")
		verbose      = flag.Bool("v", false, "Подробный вывод")
	)

	flag.Parse()

	// Загружаем конфигурацию
	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Не удалось загрузить конфигурацию: %v", err)
	}

	// Если указан путь к базе через флаг, используем его
	if *dbPath != "" {
		config.DatabasePath = *dbPath
	}

	if *outputPath != "" {
		config.OutputPath = *outputPath
	}

	if config.DatabasePath == "" {
		log.Fatal("Не указан путь к базе данных Kronos")
	}

	// Проверяем существование файла базы данных
	if _, err := os.Stat(config.DatabasePath); os.IsNotExist(err) {
		log.Fatalf("Файл базы данных не найден: %s", config.DatabasePath)
	}

	fmt.Printf("Обрабатываем базу данных: %s\n", config.DatabasePath)

	// Создаем парсер Kronos
	parser, err := NewKronosParser(config.DatabasePath)
	if err != nil {
		log.Fatalf("Не удалось создать парсер Kronos: %v", err)
	}
	defer parser.Close()

	// Получаем список таблиц
	tables, err := parser.GetTables()
	if err != nil {
		log.Fatalf("Не удалось получить список таблиц: %v", err)
	}

	fmt.Printf("Найдено таблиц: %d\n", len(tables))

	if *verbose {
		for _, table := range tables {
			fmt.Printf("  - %s\n", table)
		}
	}

	// Фильтруем таблицы если указано конкретное имя
	if *tableName != "" {
		found := false
		for _, table := range tables {
			if table == *tableName {
				tables = []string{*tableName}
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("Таблица '%s' не найдена в базе данных", *tableName)
		}
	}

	if *clickhouseMode {
		// Экспорт в ClickHouse
		err = exportToClickHouse(parser, tables, config, *verbose)
	} else {
		// Экспорт в CSV
		err = exportToCSV(parser, tables, config, *verbose)
	}

	if err != nil {
		log.Fatalf("Ошибка экспорта: %v", err)
	}

	fmt.Println("Экспорт завершен успешно!")
}

func exportToCSV(parser *KronosParser, tables []string, config Config, verbose bool) error {
	// Создаем выходную директорию
	if err := os.MkdirAll(config.OutputPath, 0755); err != nil {
		return fmt.Errorf("не удалось создать выходную директорию: %w", err)
	}

	for _, tableName := range tables {
		fmt.Printf("Экспортируем таблицу: %s\n", tableName)

		// Получаем структуру таблицы
		columns, err := parser.GetTableStructure(tableName)
		if err != nil {
			fmt.Printf("Ошибка получения структуры таблицы %s: %v\n", tableName, err)
			continue
		}

		// Получаем данные
		rows, err := parser.GetTableData(tableName)
		if err != nil {
			fmt.Printf("Ошибка получения данных таблицы %s: %v\n", tableName, err)
			continue
		}

		// Создаем CSV файл
		fileName := fmt.Sprintf("%s.csv", TransliterateTableName(tableName))
		filePath := filepath.Join(config.OutputPath, fileName)

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("Ошибка создания файла %s: %v\n", filePath, err)
			continue
		}

		// Записываем заголовки
		var headers []string
		for _, col := range columns {
			headers = append(headers, TransliterateColumnName(col.Name))
		}
		fmt.Fprintf(file, "%s\n", strings.Join(headers, ","))

		// Записываем данные
		for _, row := range rows {
			// Экранируем значения для CSV
			var escapedRow []string
			for _, value := range row {
				escaped := strings.ReplaceAll(value, `"`, `""`)
				if strings.Contains(value, ",") || strings.Contains(value, "\n") || strings.Contains(value, `"`) {
					escaped = fmt.Sprintf(`"%s"`, escaped)
				}
				escapedRow = append(escapedRow, escaped)
			}
			fmt.Fprintf(file, "%s\n", strings.Join(escapedRow, ","))
		}

		file.Close()

		if verbose {
			fmt.Printf("  Сохранено в %s (%d строк)\n", filePath, len(rows))
		} else {
			fmt.Printf("  Сохранено %d строк\n", len(rows))
		}
	}

	return nil
}

func exportToClickHouse(parser *KronosParser, tables []string, config Config, verbose bool) error {
	// Создаем клиент ClickHouse
	chClient, err := NewClickHouseClient(config)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к ClickHouse: %w", err)
	}
	defer chClient.Close()

	ctx := context.Background()

	for _, tableName := range tables {
		fmt.Printf("Экспортируем таблицу в ClickHouse: %s\n", tableName)

		// Получаем структуру таблицы
		columns, err := parser.GetTableStructure(tableName)
		if err != nil {
			fmt.Printf("Ошибка получения структуры таблицы %s: %v\n", tableName, err)
			continue
		}

		// Формируем имя таблицы в ClickHouse
		chTableName := TransliterateTableName(tableName)

		// Создаем таблицу в ClickHouse
		err = chClient.CreateTable(ctx, chTableName, columns)
		if err != nil {
			fmt.Printf("Ошибка создания таблицы %s в ClickHouse: %v\n", chTableName, err)
			continue
		}

		// Получаем данные
		rows, err := parser.GetTableData(tableName)
		if err != nil {
			fmt.Printf("Ошибка получения данных таблицы %s: %v\n", tableName, err)
			continue
		}

		// Вставляем данные в ClickHouse батчами
		batchSize := 1000
		for i := 0; i < len(rows); i += batchSize {
			end := i + batchSize
			if end > len(rows) {
				end = len(rows)
			}

			batch := rows[i:end]
			err = chClient.InsertData(ctx, chTableName, columns, batch, tableName)
			if err != nil {
				fmt.Printf("Ошибка вставки данных в таблицу %s: %v\n", chTableName, err)
				break
			}

			if verbose {
				fmt.Printf("  Обработано %d/%d строк\n", end, len(rows))
			}
		}

		if err == nil {
			fmt.Printf("  Успешно экспортировано %d строк в таблицу %s\n", len(rows), chTableName)
		}
	}

	return nil
}