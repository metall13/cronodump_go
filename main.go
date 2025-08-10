package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Println("=== Cronodump Go Converter ===")
	fmt.Println("Конвертер баз данных Cronos в ClickHouse")
	fmt.Println()

	// Загружаем конфигурацию
	config, err := NewConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Валидируем конфигурацию
	if err := config.Validate(); err != nil {
		log.Fatalf("Ошибка конфигурации: %v", err)
	}

	fmt.Printf("Директория с базами данных: %s\n", config.PathToDBDir)
	fmt.Printf("ClickHouse: %s:%d (база: %s)\n", config.ClickHouseHost, config.ClickHousePort, config.ClickHouseDatabase)
	fmt.Printf("Размер батча: %d, Воркеров: %d\n", config.BatchSize, config.MaxWorkers)
	fmt.Println()

	// Проверяем аргументы командной строки
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "help", "-h", "--help":
			fmt.Println("Использование:")
			fmt.Printf("  %s                  - конвертировать все базы данных\n", os.Args[0])
			fmt.Printf("  %s single <путь>    - конвертировать одну базу данных\n", os.Args[0])
			fmt.Printf("  %s stats            - показать статистику\n", os.Args[0])
			fmt.Printf("  %s help             - показать справку\n", os.Args[0])
			return
		case "single":
			fmt.Println("Демо режим: конвертация одной базы данных")
			if len(os.Args) > 2 {
				fmt.Printf("Указанный путь: %s\n", os.Args[2])
			}
			return
		case "stats":
			fmt.Println("Демо режим: показ статистики")
			return
		}
	}

	// Демо: ищем DBF файлы в указанной директории
	fmt.Printf("Поиск баз данных Cronos в: %s\n", config.PathToDBDir)
	
	dbfCount := 0
	err = filepath.Walk(config.PathToDBDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Игнорируем ошибки доступа
		}

		if !info.IsDir() && filepath.Ext(strings.ToLower(info.Name())) == ".dbf" {
			dbfCount++
			relPath, _ := filepath.Rel(config.PathToDBDir, path)
			
			folderName := filepath.Dir(relPath)
			if folderName == "." {
				folderName = "root"
			}
			
			tableName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			finalTableName := TransliterateTableName(folderName, tableName)
			
			fmt.Printf("  Найден: %s -> %s (таблица: %s)\n", relPath, finalTableName, tableName)
			
			// Демо парсинга одной таблицы
			if dbfCount == 1 {
				fmt.Printf("\nДемо парсинг файла: %s\n", path)
				parser := NewCronosParser(filepath.Dir(path))
				
				table := &CronosTable{
					Name:     tableName,
					FilePath: path,
				}
				
				if err := parser.parseTableData(table); err != nil {
					fmt.Printf("  Ошибка: %v\n", err)
				} else {
					fmt.Printf("  Поля: %d, Записей: %d\n", len(table.Fields), len(table.Records))
					
					// Показываем первые поля
					fmt.Println("  Структура полей:")
					for i, field := range table.Fields {
						if i >= 5 { // Показываем только первые 5 полей
							fmt.Printf("    ... и еще %d полей\n", len(table.Fields)-5)
							break
						}
						translatedName := TransliterateColumnName(field.Name)
						fmt.Printf("    %s (%s) -> %s\n", field.Name, getFieldTypeString(field.Type), translatedName)
					}
					
					// Показываем первые записи
					if len(table.Records) > 0 {
						fmt.Println("  Примеры данных:")
						maxRecords := len(table.Records)
						if maxRecords > 3 {
							maxRecords = 3
						}
						for i := 0; i < maxRecords; i++ {
							fmt.Printf("    Запись %d: %v\n", i+1, table.Records[i])
						}
						if len(table.Records) > 3 {
							fmt.Printf("    ... и еще %d записей\n", len(table.Records)-3)
						}
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Ошибка при сканировании директории: %v\n", err)
	}

	if dbfCount == 0 {
		fmt.Println("DBF файлы не найдены!")
		fmt.Println("\nДля тестирования поместите DBF файлы в указанную директорию.")
	} else {
		fmt.Printf("\nНайдено DBF файлов: %d\n", dbfCount)
		fmt.Println("\nДемо завершено! В реальном режиме данные будут загружены в ClickHouse.")
	}

	fmt.Println("\n--- Тестирование функций транслитерации ---")
	testExamples := []struct {
		original string
		expected string
	}{
		{"Сотрудники", "sotrudniki"},
		{"Отдел кадров", "otdel_kadrov"},
		{"База-данных 123", "baza_dannyh_123"},
		{"SELECT", "col_select"},
	}

	for _, example := range testExamples {
		result := SanitizeForClickHouse(example.original)
		fmt.Printf("'%s' -> '%s'\n", example.original, result)
	}
}

func getFieldTypeString(fieldType CronosFieldType) string {
	switch fieldType {
	case FieldTypeString:
		return "String"
	case FieldTypeInteger:
		return "Integer"
	case FieldTypeFloat:
		return "Float"
	case FieldTypeDate:
		return "Date"
	case FieldTypeBoolean:
		return "Boolean"
	case FieldTypeMemo:
		return "Memo"
	case FieldTypeBinary:
		return "Binary"
	default:
		return "Unknown"
	}
}