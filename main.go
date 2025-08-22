package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	var (
		configPath = flag.String("config", "config.json", "Путь к файлу конфигурации")
		dbPath     = flag.String("db", "/home/bd_cronos", "Путь к директории с базами Cronos")
		verbose    = flag.Bool("v", false, "Подробный вывод")
	)
	flag.Parse()

	// Загружаем конфигурацию
	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Printf("Ошибка загрузки конфигурации: %v", err)
		log.Println("Используем значения по умолчанию")
	}

	// Переопределяем путь к базам, если указан в командной строке
	if *dbPath != "/home/bd_cronos" {
		config.DatabasePath = *dbPath
	} else if config.DatabasePath == "" {
		config.DatabasePath = *dbPath
	}

	fmt.Printf("Конвертер баз данных Cronos в ClickHouse\n")
	fmt.Printf("Путь к базам: %s\n", config.DatabasePath)
	fmt.Printf("Путь вывода: %s\n", config.OutputPath)

	// Проверяем существование директории с базами
	if _, err := os.Stat(config.DatabasePath); os.IsNotExist(err) {
		log.Fatalf("Директория с базами не существует: %s", config.DatabasePath)
	}

	// Создаем конвертер
	converter := NewConverter(config, *verbose)

	// Ищем базы данных Cronos
	databases, err := converter.FindCronosDatabases()
	if err != nil {
		log.Fatalf("Ошибка поиска баз данных: %v", err)
	}

	fmt.Printf("Найдено баз данных: %d\n", len(databases))

	// Конвертируем каждую базу
	for _, dbPath := range databases {
		fmt.Printf("Обрабатываем базу: %s\n", dbPath)
		if err := converter.ConvertDatabase(dbPath); err != nil {
			log.Printf("Ошибка конвертации базы %s: %v", dbPath, err)
			continue
		}
	}

	fmt.Println("Конвертация завершена!")
}