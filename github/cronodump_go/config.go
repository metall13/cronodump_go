package main

// Config содержит конфигурацию конвертера
type Config struct {
	InputDir   string // Путь к папке с базой данных Cronos
	OutputDir  string // Папка для сохранения SQL файлов
	ClickHouse string // URL подключения к ClickHouse
	BatchSize  int    // Размер батча для вставки данных
	Cleanup    bool   // Удалить временные файлы после импорта
}