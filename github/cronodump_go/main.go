package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	var (
		inputDir    = flag.String("input", "", "Путь к папке с базой данных Cronos")
		outputDir   = flag.String("output", "", "Папка для сохранения SQL файлов (по умолчанию: ./output)")
		clickhouse  = flag.String("clickhouse", "", "URL подключения к ClickHouse (например: tcp://localhost:9000)")
		verbose     = flag.Bool("verbose", false, "Подробный вывод")
		cleanup     = flag.Bool("cleanup", true, "Удалить временные файлы после импорта")
		batchSize   = flag.Int("batch", 1000, "Размер батча для вставки данных")
	)
	flag.Parse()

	if *inputDir == "" {
		fmt.Println("Использование: cronodump_go -input <путь_к_базе_cronos> [опции]")
		fmt.Println("\nОпции:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *outputDir == "" {
		*outputDir = "./output"
	}

	// Создаем конфигурацию
	config := &Config{
		InputDir:    *inputDir,
		OutputDir:   *outputDir,
		ClickHouse:  *clickhouse,
		BatchSize:   *batchSize,
		Cleanup:     *cleanup,
	}

	// Создаем конвертер
	converter := NewCronosConverter(config, *verbose)

	// Запускаем конвертацию
	err := converter.Convert()
	if err != nil {
		log.Fatalf("Ошибка конвертации: %v", err)
	}

	fmt.Println("Конвертация завершена успешно!")
}