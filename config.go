package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config структура конфигурации приложения
type Config struct {
	// Пути
	PathToDBDir string `env:"PATH_TO_DB"`

	// ClickHouse настройки
	ClickHouseHost     string `env:"CLICKHOUSE_HOST"`
	ClickHousePort     int    `env:"CLICKHOUSE_PORT"`
	ClickHouseDatabase string `env:"CLICKHOUSE_DATABASE"`
	ClickHouseUser     string `env:"CLICKHOUSE_USER"`
	ClickHousePassword string `env:"CLICKHOUSE_PASSWORD"`

	// Дополнительные настройки
	LogLevel   string `env:"LOG_LEVEL"`
	BatchSize  int    `env:"BATCH_SIZE"`
	MaxWorkers int    `env:"MAX_WORKERS"`
}

// NewConfig создает новую конфигурацию из переменных окружения
func NewConfig() (*Config, error) {
	// Пытаемся загрузить .env файл
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Предупреждение: не удалось загрузить .env файл: %v\n", err)
	}

	cfg := &Config{
		// Значения по умолчанию
		ClickHouseHost:     "localhost",
		ClickHousePort:     9000,
		ClickHouseDatabase: "cronos_data",
		ClickHouseUser:     "default",
		ClickHousePassword: "",
		LogLevel:           "info",
		BatchSize:          1000,
		MaxWorkers:         4,
	}

	// Загружаем значения из переменных окружения
	if pathToDb := os.Getenv("PATH_TO_DB"); pathToDb != "" {
		cfg.PathToDBDir = strings.TrimSuffix(pathToDb, "/") + "/"
	}

	if host := os.Getenv("CLICKHOUSE_HOST"); host != "" {
		cfg.ClickHouseHost = host
	}

	if portStr := os.Getenv("CLICKHOUSE_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.ClickHousePort = port
		}
	}

	if db := os.Getenv("CLICKHOUSE_DATABASE"); db != "" {
		cfg.ClickHouseDatabase = db
	}

	if user := os.Getenv("CLICKHOUSE_USER"); user != "" {
		cfg.ClickHouseUser = user
	}

	if password := os.Getenv("CLICKHOUSE_PASSWORD"); password != "" {
		cfg.ClickHousePassword = password
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	if batchSizeStr := os.Getenv("BATCH_SIZE"); batchSizeStr != "" {
		if batchSize, err := strconv.Atoi(batchSizeStr); err == nil {
			cfg.BatchSize = batchSize
		}
	}

	if maxWorkersStr := os.Getenv("MAX_WORKERS"); maxWorkersStr != "" {
		if maxWorkers, err := strconv.Atoi(maxWorkersStr); err == nil {
			cfg.MaxWorkers = maxWorkers
		}
	}

	// Валидация
	if cfg.PathToDBDir == "" {
		return nil, fmt.Errorf("PATH_TO_DB должен быть установлен")
	}

	return cfg, nil
}

// GetClickHouseAddr возвращает адрес ClickHouse в формате host:port
func (c *Config) GetClickHouseAddr() []string {
	return []string{fmt.Sprintf("%s:%d", c.ClickHouseHost, c.ClickHousePort)}
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.PathToDBDir == "" {
		return fmt.Errorf("путь к базам данных не может быть пустым")
	}

	if _, err := os.Stat(c.PathToDBDir); os.IsNotExist(err) {
		return fmt.Errorf("директория с базами данных не существует: %s", c.PathToDBDir)
	}

	if c.ClickHouseHost == "" {
		return fmt.Errorf("хост ClickHouse не может быть пустым")
	}

	if c.ClickHousePort <= 0 || c.ClickHousePort > 65535 {
		return fmt.Errorf("некорректный порт ClickHouse: %d", c.ClickHousePort)
	}

	if c.BatchSize <= 0 {
		return fmt.Errorf("размер батча должен быть больше 0")
	}

	if c.MaxWorkers <= 0 {
		return fmt.Errorf("количество воркеров должно быть больше 0")
	}

	return nil
}