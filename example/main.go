package main

import (
	"fmt"
	"log"

	"cronodamp"
)

func main() {
	// Создаем конфигурацию
	config := cronodamp.Config{
		KronosDBPath:       "/home/usersamba/smb/",
		ClickHouseDSN:      "localhost:9000",
		ClickHouseDatabase: "default",
		ClickHouseUser:     "default",
		ClickHousePassword: "default",
		BatchSize:          1000,
		Debug:              true,
	}

	// Создаем экспортер
	exporter, err := cronodamp.NewExporter(config)
	if err != nil {
		log.Fatalf("Ошибка создания экспортера: %v", err)
	}
	defer exporter.Close()

	// Проверяем подключение
	if err := exporter.ValidateConnection(); err != nil {
		log.Fatalf("Ошибка подключения к ClickHouse: %v", err)
	}
	fmt.Println("✓ Подключение к ClickHouse успешно")

	// Получаем список таблиц
	tables, err := exporter.ListTables()
	if err != nil {
		log.Fatalf("Ошибка получения списка таблиц: %v", err)
	}

	fmt.Printf("Найдено %d таблиц:\n", len(tables))
	for i, table := range tables {
		clickHouseTableName := exporter.GetClickHouseTableName(table)
		fmt.Printf("%d. %s/%s -> %s (%d колонок)\n", 
			i+1, table.FolderName, table.TableName, 
			clickHouseTableName, len(table.Columns))
	}

	// Экспортируем все таблицы
	fmt.Println("\nНачинаем экспорт...")
	err = exporter.ExportAll()
	if err != nil {
		log.Fatalf("Ошибка экспорта: %v", err)
	}

	// Выводим статистику
	exporter.PrintStats()
	fmt.Println("✓ Экспорт завершен!")
}