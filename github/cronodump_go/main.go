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
		configPath     = flag.String("config", "config.json", "Путь к файлу конфигурации")
		folderName     = flag.String("folder", "", "Имя конкретной папки с базой для обработки")
		tableName      = flag.String("table", "", "Имя конкретной таблицы для экспорта")
		clickhouseMode = flag.Bool("clickhouse", true, "Экспорт в ClickHouse (по умолчанию)")
		csvMode        = flag.Bool("csv", false, "Экспорт в CSV")
		verbose        = flag.Bool("v", false, "Подробный вывод")
		listFolders    = flag.Bool("list", false, "Показать список доступных папок с базами")
	)

	flag.Parse()

	// Загружаем конфигурацию
	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Не удалось загрузить конфигурацию: %v", err)
	}

	fmt.Printf("Путь к базам Kronos: %s\n", config.KronosDatabasesPath)

	// Проверяем существование папки с базами
	if _, err := os.Stat(config.KronosDatabasesPath); os.IsNotExist(err) {
		log.Fatalf("Папка с базами Kronos не найдена: %s", config.KronosDatabasesPath)
	}

	// Если запрошен список папок
	if *listFolders {
		err := listDatabaseFolders(config.KronosDatabasesPath)
		if err != nil {
			log.Fatalf("Ошибка получения списка папок: %v", err)
		}
		return
	}

	// Получаем список папок с базами данных
	dbFolders, err := findDatabaseFolders(config.KronosDatabasesPath)
	if err != nil {
		log.Fatalf("Не удалось найти папки с базами: %v", err)
	}

	fmt.Printf("Найдено папок с базами: %d\n", len(dbFolders))

	// Фильтруем папки если указано конкретное имя
	if *folderName != "" {
		filtered := []string{}
		for _, folder := range dbFolders {
			if strings.Contains(strings.ToLower(filepath.Base(folder)), strings.ToLower(*folderName)) {
				filtered = append(filtered, folder)
			}
		}
		if len(filtered) == 0 {
			log.Fatalf("Папка с именем '%s' не найдена", *folderName)
		}
		dbFolders = filtered
		fmt.Printf("Отфильтровано папок: %d\n", len(dbFolders))
	}

	// Определяем режим экспорта
	useClickHouse := *clickhouseMode && !*csvMode

	if useClickHouse {
		err = exportAllToClickHouse(dbFolders, config, *tableName, *verbose)
	} else {
		err = exportAllToCSV(dbFolders, config, *tableName, *verbose)
	}

	if err != nil {
		log.Fatalf("Ошибка экспорта: %v", err)
	}

	fmt.Println("Экспорт завершен успешно!")
}

// findDatabaseFolders находит все папки содержащие базы данных Kronos
func findDatabaseFolders(basePath string) ([]string, error) {
	var dbFolders []string

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Пропускаем ошибки доступа
		}

		if !info.IsDir() {
			return nil
		}

		// Проверяем наличие файлов Kronos в папке
		if hasKronosFiles(path) {
			dbFolders = append(dbFolders, path)
		}

		return nil
	})

	return dbFolders, err
}

// hasKronosFiles проверяет наличие файлов базы Kronos в папке
func hasKronosFiles(folderPath string) bool {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return false
	}

	hasStru := false
	hasBank := false

	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if strings.HasPrefix(name, "crostru.") {
			hasStru = true
		}
		if strings.HasPrefix(name, "crobank.") {
			hasBank = true
		}
	}

	return hasStru && hasBank
}

// listDatabaseFolders выводит список доступных папок с базами
func listDatabaseFolders(basePath string) error {
	folders, err := findDatabaseFolders(basePath)
	if err != nil {
		return err
	}

	fmt.Printf("Доступные папки с базами Kronos:\n")
	for i, folder := range folders {
		fmt.Printf("%d. %s\n", i+1, folder)
	}

	return nil
}

// exportAllToClickHouse экспортирует все базы в ClickHouse
func exportAllToClickHouse(dbFolders []string, config Config, tableName string, verbose bool) error {
	// Создаем клиент ClickHouse
	chClient, err := NewClickHouseClient(config)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к ClickHouse: %w", err)
	}
	defer chClient.Close()

	ctx := context.Background()

	for _, dbFolder := range dbFolders {
		fmt.Printf("\n=== Обрабатываем папку: %s ===\n", dbFolder)
		
		// Извлекаем имя папки для префикса таблиц
		folderName := ExtractFolderName(dbFolder)
		
		// Создаем парсер для этой базы
		parser, err := NewKronosParser(dbFolder)
		if err != nil {
			fmt.Printf("Ошибка создания парсера для %s: %v\n", dbFolder, err)
			continue
		}

		// Получаем список таблиц
		tables, err := parser.GetTables()
		if err != nil {
			fmt.Printf("Ошибка получения таблиц для %s: %v\n", dbFolder, err)
			parser.Close()
			continue
		}

		fmt.Printf("Найдено таблиц: %d\n", len(tables))

		// Фильтруем таблицы если указано конкретное имя
		if tableName != "" {
			filtered := []string{}
			for _, table := range tables {
				if strings.Contains(strings.ToLower(table), strings.ToLower(tableName)) {
					filtered = append(filtered, table)
				}
			}
			tables = filtered
		}

		// Экспортируем каждую таблицу
		for _, table := range tables {
			err = exportTableToClickHouse(parser, chClient, ctx, folderName, table, verbose)
			if err != nil {
				fmt.Printf("Ошибка экспорта таблицы %s: %v\n", table, err)
			}
		}

		parser.Close()
	}

	return nil
}

// exportTableToClickHouse экспортирует одну таблицу в ClickHouse
func exportTableToClickHouse(parser *KronosParser, chClient *ClickHouseClient, ctx context.Context, folderName, tableName string, verbose bool) error {
	fmt.Printf("Экспортируем таблицу: %s\n", tableName)

	// Получаем структуру таблицы
	fields, err := parser.GetTableStructure(tableName)
	if err != nil {
		return fmt.Errorf("не удалось получить структуру таблицы: %w", err)
	}

	// Формируем имя таблицы в ClickHouse
	chTableName := TransliterateTableName(folderName, tableName)

	// Создаем таблицу в ClickHouse
	err = chClient.CreateTable(ctx, chTableName, fields)
	if err != nil {
		return fmt.Errorf("не удалось создать таблицу в ClickHouse: %w", err)
	}

	// Получаем данные
	rows, err := parser.GetTableData(tableName)
	if err != nil {
		return fmt.Errorf("не удалось получить данные таблицы: %w", err)
	}

	if len(rows) == 0 {
		fmt.Printf("  Таблица %s пустая\n", tableName)
		return nil
	}

	// Вставляем данные в ClickHouse батчами
	batchSize := 1000
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}

		batch := rows[i:end]
		err = chClient.InsertData(ctx, chTableName, fields, batch, tableName)
		if err != nil {
			return fmt.Errorf("не удалось вставить данные: %w", err)
		}

		if verbose {
			fmt.Printf("  Обработано %d/%d строк\n", end, len(rows))
		}
	}

	fmt.Printf("  Успешно экспортировано %d строк в таблицу %s\n", len(rows), chTableName)
	return nil
}

// exportAllToCSV экспортирует все базы в CSV
func exportAllToCSV(dbFolders []string, config Config, tableName string, verbose bool) error {
	// Создаем выходную директорию
	if err := os.MkdirAll(config.OutputPath, 0755); err != nil {
		return fmt.Errorf("не удалось создать выходную директорию: %w", err)
	}

	for _, dbFolder := range dbFolders {
		fmt.Printf("\n=== Обрабатываем папку: %s ===\n", dbFolder)
		
		// Извлекаем имя папки для префикса файлов
		folderName := ExtractFolderName(dbFolder)
		
		// Создаем парсер для этой базы
		parser, err := NewKronosParser(dbFolder)
		if err != nil {
			fmt.Printf("Ошибка создания парсера для %s: %v\n", dbFolder, err)
			continue
		}

		// Получаем список таблиц
		tables, err := parser.GetTables()
		if err != nil {
			fmt.Printf("Ошибка получения таблиц для %s: %v\n", dbFolder, err)
			parser.Close()
			continue
		}

		// Фильтруем таблицы если указано конкретное имя
		if tableName != "" {
			filtered := []string{}
			for _, table := range tables {
				if strings.Contains(strings.ToLower(table), strings.ToLower(tableName)) {
					filtered = append(filtered, table)
				}
			}
			tables = filtered
		}

		// Экспортируем каждую таблицу
		for _, table := range tables {
			err = exportTableToCSV(parser, config.OutputPath, folderName, table, verbose)
			if err != nil {
				fmt.Printf("Ошибка экспорта таблицы %s: %v\n", table, err)
			}
		}

		parser.Close()
	}

	return nil
}

// exportTableToCSV экспортирует одну таблицу в CSV
func exportTableToCSV(parser *KronosParser, outputPath, folderName, tableName string, verbose bool) error {
	fmt.Printf("Экспортируем таблицу: %s\n", tableName)

	// Получаем структуру таблицы
	fields, err := parser.GetTableStructure(tableName)
	if err != nil {
		return fmt.Errorf("не удалось получить структуру таблицы: %w", err)
	}

	// Получаем данные
	rows, err := parser.GetTableData(tableName)
	if err != nil {
		return fmt.Errorf("не удалось получить данные таблицы: %w", err)
	}

	// Создаем CSV файл
	fileName := fmt.Sprintf("%s_%s.csv", TransliterateTableName(folderName, ""), TransliterateTableName("", tableName))
	fileName = strings.Trim(fileName, "_")
	filePath := filepath.Join(outputPath, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("не удалось создать файл %s: %w", filePath, err)
	}
	defer file.Close()

	// Записываем заголовки
	var headers []string
	for _, field := range fields {
		headers = append(headers, TransliterateColumnName(field.Name))
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

	if verbose {
		fmt.Printf("  Сохранено в %s (%d строк)\n", filePath, len(rows))
	} else {
		fmt.Printf("  Сохранено %d строк\n", len(rows))
	}

	return nil
}