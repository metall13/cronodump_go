package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	
	"cronodamp"
)

func main() {
	// Парсим аргументы командной строки
	var (
		dbPath     = flag.String("db", "", "Путь к базе данных Cronos")
		chDSN      = flag.String("clickhouse", "localhost:9000", "Адрес ClickHouse")
		chUser     = flag.String("user", "default", "Пользователь ClickHouse")
		chPassword = flag.String("password", "default", "Пароль ClickHouse")
		chDatabase = flag.String("database", "default", "База данных ClickHouse")
		listOnly   = flag.Bool("list", false, "Только показать список таблиц")
		validate   = flag.Bool("validate", false, "Только проверить конфигурацию")
		help       = flag.Bool("help", false, "Показать справку")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Создаем конфигурацию
	cfg := createConfig(*dbPath, *chDSN, *chUser, *chPassword, *chDatabase)

	// Создаем экспортер
	exporter, err := cronodamp.NewCronodampExporter(cfg)
	if err != nil {
		log.Fatalf("❌ Ошибка при создании экспортера: %v", err)
	}
	defer exporter.Close()

	ctx := context.Background()

	// Проверяем конфигурацию
	if *validate {
		if err := exporter.ValidateConfig(); err != nil {
			log.Fatalf("❌ Ошибка конфигурации: %v", err)
		}
		fmt.Println("✅ Конфигурация корректна!")
		return
	}

	// Показываем список таблиц
	if *listOnly {
		tables, err := exporter.ListTables(ctx)
		if err != nil {
			log.Fatalf("❌ Ошибка при получении списка таблиц: %v", err)
		}
		
		fmt.Printf("📋 Найдено %d таблиц:\n", len(tables))
		for i, table := range tables {
			fmt.Printf("   %d. %s\n", i+1, table)
		}
		return
	}

	// Экспортируем базу данных
	fmt.Println("🚀 Cronodamp Go - экспорт данных из Cronos в ClickHouse")
	fmt.Println(strings.Repeat("=", 60))

	results, err := exporter.ExportWithProgress(ctx)
	if err != nil {
		log.Fatalf("❌ Ошибка при экспорте: %v", err)
	}

	// Выводим финальную статистику
	printResults(results)
}

// createConfig создает конфигурацию из параметров командной строки или переменных окружения
func createConfig(dbPath, chDSN, chUser, chPassword, chDatabase string) cronodamp.Config {
	cfg := cronodamp.NewConfigFromEnv()

	// Переопределяем значения из командной строки, если они указаны
	if dbPath != "" {
		cfg.DatabasePath = dbPath
	}
	if chDSN != "localhost:9000" {
		cfg.ClickHouseDSN = chDSN
	}
	if chUser != "default" {
		cfg.ClickHouseUser = chUser
	}
	if chPassword != "default" {
		cfg.ClickHousePassword = chPassword
	}
	if chDatabase != "default" {
		cfg.ClickHouseDatabase = chDatabase
	}

	// Проверяем обязательные параметры
	if cfg.DatabasePath == "" {
		fmt.Println("❌ Путь к базе данных не указан!")
		fmt.Println("Используйте: --db /path/to/cronos/database")
		fmt.Println("Или установите переменную окружения: PATH_TO_DB")
		os.Exit(1)
	}

	return cfg
}

// showHelp показывает справку по использованию
func showHelp() {
	fmt.Println("🚀 Cronodamp Go - экспорт данных из Cronos в ClickHouse")
	fmt.Println("")
	fmt.Println("ИСПОЛЬЗОВАНИЕ:")
	fmt.Println("  cronodamp [опции]")
	fmt.Println("")
	fmt.Println("ОПЦИИ:")
	fmt.Println("  --db PATH              Путь к базе данных Cronos (обязательно)")
	fmt.Println("  --clickhouse ADDR      Адрес ClickHouse (по умолчанию: localhost:9000)")
	fmt.Println("  --user USER            Пользователь ClickHouse (по умолчанию: default)")
	fmt.Println("  --password PASS        Пароль ClickHouse (по умолчанию: default)")
	fmt.Println("  --database DB          База данных ClickHouse (по умолчанию: default)")
	fmt.Println("  --list                 Только показать список таблиц")
	fmt.Println("  --validate             Только проверить конфигурацию")
	fmt.Println("  --help                 Показать эту справку")
	fmt.Println("")
	fmt.Println("ПЕРЕМЕННЫЕ ОКРУЖЕНИЯ:")
	fmt.Println("  PATH_TO_DB             Путь к базе данных Cronos")
	fmt.Println("  CLICKHOUSE_DSN         Адрес ClickHouse")
	fmt.Println("  CLICKHOUSE_USER        Пользователь ClickHouse")
	fmt.Println("  CLICKHOUSE_PASSWORD    Пароль ClickHouse")
	fmt.Println("  CLICKHOUSE_DATABASE    База данных ClickHouse")
	fmt.Println("")
	fmt.Println("ПРИМЕРЫ:")
	fmt.Println("  # Экспорт с указанием пути к базе")
	fmt.Println("  cronodamp --db /home/user/cronos_data")
	fmt.Println("")
	fmt.Println("  # Проверка конфигурации")
	fmt.Println("  cronodamp --db /home/user/cronos_data --validate")
	fmt.Println("")
	fmt.Println("  # Список таблиц")
	fmt.Println("  cronodamp --db /home/user/cronos_data --list")
	fmt.Println("")
	fmt.Println("  # Экспорт с настройками ClickHouse")
	fmt.Println("  cronodamp --db /home/user/cronos_data --clickhouse remote:9000 --user myuser")
}

// printResults выводит статистику результатов экспорта
func printResults(results []*cronodamp.ExportResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 ИТОГОВАЯ СТАТИСТИКА")
	fmt.Println(strings.Repeat("=", 60))

	successful := 0
	failed := 0
	totalRows := 0

	for _, result := range results {
		if result.Success {
			successful++
			totalRows += result.RowsExported
		} else {
			failed++
		}
	}

	fmt.Printf("✅ Успешно экспортировано: %d таблиц\n", successful)
	fmt.Printf("❌ Ошибки при экспорте:    %d таблиц\n", failed)
	fmt.Printf("📋 Общее количество строк: %d\n", totalRows)
	fmt.Printf("🕒 Общее количество таблиц: %d\n", len(results))

	if failed > 0 {
		fmt.Println("\n❌ ТАБЛИЦЫ С ОШИБКАМИ:")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("   • %s: %v\n", result.TableName, result.Error)
			}
		}
	}

	if successful > 0 {
		fmt.Println("\n✅ УСПЕШНО ЭКСПОРТИРОВАННЫЕ ТАБЛИЦЫ:")
		for _, result := range results {
			if result.Success {
				fmt.Printf("   • %s (%d строк)\n", result.TableName, result.RowsExported)
			}
		}
	}

	fmt.Println("\n🎉 Экспорт завершен!")
}