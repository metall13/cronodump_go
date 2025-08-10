package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cronodamp"
)

func main() {
	// Пример 1: Простое использование с переменными окружения
	fmt.Println("🚀 Пример использования библиотеки Cronodamp Go")
	fmt.Println("=" + strings.Repeat("=", 50))

	// Создание конфигурации из переменных окружения
	cfg := cronodamp.NewConfigFromEnv()
	
	// Переопределяем путь к базе данных для примера
	cfg.DatabasePath = "./test_data"

	// Создание экспортера
	exporter, err := cronodamp.NewCronodampExporter(cfg)
	if err != nil {
		log.Fatal("Ошибка при создании экспортера:", err)
	}
	defer exporter.Close()

	ctx := context.Background()

	// Пример 2: Получение списка таблиц
	fmt.Println("\n📋 Получаем список таблиц...")
	tables, err := exporter.ListTables(ctx)
	if err != nil {
		log.Fatal("Ошибка при получении списка таблиц:", err)
	}

	fmt.Printf("Найдено %d таблиц:\n", len(tables))
	for i, table := range tables {
		fmt.Printf("  %d. %s\n", i+1, table)
	}

	// Пример 3: Получение информации о базе данных
	fmt.Println("\n📊 Получаем информацию о базе данных...")
	dbInfo, err := exporter.GetDatabaseInfo(ctx)
	if err != nil {
		log.Fatal("Ошибка при получении информации о БД:", err)
	}

	fmt.Printf("База данных: %s\n", dbInfo.Path)
	fmt.Printf("Количество таблиц: %d\n", dbInfo.TableCount)
	fmt.Printf("Дата создания: %s\n", dbInfo.CreatedAt.Format(time.RFC3339))

	// Пример 4: Валидация конфигурации
	fmt.Println("\n✅ Проверяем конфигурацию...")
	if err := exporter.ValidateConfig(); err != nil {
		log.Fatal("Ошибка конфигурации:", err)
	}

	// Пример 5: Экспорт в ClickHouse (если доступен)
	fmt.Println("\n🔄 Пытаемся экспортировать данные...")
	results, err := exporter.ExportDatabase(ctx)
	if err != nil {
		fmt.Printf("❌ Экспорт невозможен: %v\n", err)
		fmt.Println("Для экспорта необходим запущенный ClickHouse")
	} else {
		fmt.Println("✅ Экспорт завершен успешно!")
		
		// Статистика экспорта
		successful := 0
		totalRows := 0
		for _, result := range results {
			if result.Success {
				successful++
				totalRows += result.RowsExported
			}
		}
		
		fmt.Printf("Экспортировано таблиц: %d/%d\n", successful, len(results))
		fmt.Printf("Общее количество строк: %d\n", totalRows)
	}

	fmt.Println("\n🎉 Пример завершен!")
}