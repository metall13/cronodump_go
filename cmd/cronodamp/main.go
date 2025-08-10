package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"cronodamp"
)

func main() {
	// Параметры командной строки
	var (
		listOnly     = flag.Bool("list", false, "Только показать список таблиц без экспорта")
		specificPath = flag.String("file", "", "Экспортировать конкретный файл")
		folderName   = flag.String("folder", "", "Экспортировать все таблицы из указанной папки")
		debug        = flag.Bool("debug", false, "Режим отладки")
		validate     = flag.Bool("validate", false, "Только проверить подключение к ClickHouse")
	)
	flag.Parse()

	// Создаем конфигурацию из переменных окружения
	config := cronodamp.NewConfigFromEnv()
	
	// Переопределяем отладку из параметров командной строки
	if *debug {
		config.Debug = true
	}

	fmt.Println("=== Cronodamp - Экспортер данных из Kronos в ClickHouse ===")
	fmt.Printf("Путь к базам Kronos: %s\n", config.KronosDBPath)
	fmt.Printf("ClickHouse DSN: %s\n", config.ClickHouseDSN)
	fmt.Printf("ClickHouse DB: %s\n", config.ClickHouseDatabase)
	
	// Создаем экспортер
	exporter, err := cronodamp.NewExporter(config)
	if err != nil {
		log.Fatalf("Ошибка создания экспортера: %v", err)
	}
	defer exporter.Close()

	// Обработка сигналов для корректного завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nПолучен сигнал завершения, останавливаем экспорт...")
		exporter.Cancel()
	}()

	// Проверяем подключение
	if err := exporter.ValidateConnection(); err != nil {
		log.Fatalf("Ошибка подключения к ClickHouse: %v", err)
	}
	fmt.Println("✓ Подключение к ClickHouse успешно")

	if *validate {
		fmt.Println("✓ Проверка подключения завершена успешно")
		return
	}

	// Режим "только список таблиц"
	if *listOnly {
		tables, err := exporter.ListTables()
		if err != nil {
			log.Fatalf("Ошибка получения списка таблиц: %v", err)
		}

		fmt.Printf("\nНайдено %d таблиц:\n", len(tables))
		for i, table := range tables {
			clickHouseTableName := exporter.GetClickHouseTableName(table)
			fmt.Printf("%d. %s/%s -> %s (%d колонок)\n", 
				i+1, table.FolderName, table.TableName, 
				clickHouseTableName, len(table.Columns))
		}
		return
	}

	// Экспорт конкретного файла
	if *specificPath != "" {
		fmt.Printf("Экспортируем файл: %s\n", *specificPath)
		err := exporter.ExportSpecific([]string{*specificPath})
		if err != nil {
			log.Fatalf("Ошибка экспорта файла: %v", err)
		}
		exporter.PrintStats()
		return
	}

	// Экспорт конкретной папки
	if *folderName != "" {
		fmt.Printf("Экспортируем папку: %s\n", *folderName)
		err := exporter.ExportFolder(*folderName)
		if err != nil {
			log.Fatalf("Ошибка экспорта папки: %v", err)
		}
		exporter.PrintStats()
		return
	}

	// Экспорт всех таблиц
	fmt.Println("Начинаем экспорт всех таблиц...")
	err = exporter.ExportAll()
	if err != nil {
		log.Fatalf("Ошибка экспорта: %v", err)
	}

	// Выводим статистику
	exporter.PrintStats()
	fmt.Println("✓ Экспорт завершен успешно!")
}