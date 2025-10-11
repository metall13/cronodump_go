package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config содержит настройки для конвертера Cronos -> ClickHouse
type Config struct {
	// Путь к папке с базами данных Cronos
	CronosBasePath string
	
	// Настройки ClickHouse
	ClickHouseAddr     string
	ClickHouseDatabase string
	ClickHouseUser     string
	ClickHousePassword string
	
	// Настройки обработки
	BatchSize int
	
	// Настройки логирования
	Verbose bool
}

// LoadConfigFromEnv загружает конфигурацию из переменных окружения
func LoadConfigFromEnv() (*Config, error) {
	config := &Config{
		// Значения по умолчанию
		ClickHouseDatabase: "cronos_data",
		ClickHouseUser:     "default",
		ClickHousePassword: "",
		BatchSize:          1000,
		Verbose:            false,
	}
	
	// Путь к базам Cronos (обязательный параметр)
	cronosPath := os.Getenv("CRONOS_DB_PATH")
	if cronosPath == "" {
		cronosPath = os.Getenv("pach_to_db") // поддержка старого названия
	}
	if cronosPath == "" {
		return nil, fmt.Errorf("переменная окружения CRONOS_DB_PATH не установлена")
	}
	
	// Проверяем существование пути
	if _, err := os.Stat(cronosPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("путь к базам Cronos не существует: %s", cronosPath)
	}
	config.CronosBasePath = filepath.Clean(cronosPath)
	
	// Адрес ClickHouse (обязательный параметр)
	clickhouseAddr := os.Getenv("CLICKHOUSE_DSN")
	if clickhouseAddr == "" {
		clickhouseAddr = os.Getenv("clichause_dsn") // поддержка старого названия
	}
	if clickhouseAddr == "" {
		return nil, fmt.Errorf("переменная окружения CLICKHOUSE_DSN не установлена")
	}
	config.ClickHouseAddr = clickhouseAddr
	
	// Дополнительные настройки ClickHouse
	if db := os.Getenv("CLICKHOUSE_DATABASE"); db != "" {
		config.ClickHouseDatabase = db
	}
	if user := os.Getenv("CLICKHOUSE_USER"); user != "" {
		config.ClickHouseUser = user
	}
	if password := os.Getenv("CLICKHOUSE_PASSWORD"); password != "" {
		config.ClickHousePassword = password
	}
	
	// Размер батча
	if batchSizeStr := os.Getenv("BATCH_SIZE"); batchSizeStr != "" {
		if batchSize, err := strconv.Atoi(batchSizeStr); err == nil && batchSize > 0 {
			config.BatchSize = batchSize
		}
	}
	
	// Режим отладки
	if verbose := os.Getenv("VERBOSE"); verbose == "true" || verbose == "1" {
		config.Verbose = true
	}
	
	return config, nil
}

// GetClickHouseAddr возвращает адрес ClickHouse в формате для драйвера
// Примечание: В текущей версии используется прямое обращение к config.ClickHouseAddr
func (c *Config) GetClickHouseAddr() []string {
	// Поддерживаем как один адрес, так и список через запятую
	addresses := strings.Split(c.ClickHouseAddr, ",")
	for i, addr := range addresses {
		addresses[i] = strings.TrimSpace(addr)
	}
	return addresses
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.CronosBasePath == "" {
		return fmt.Errorf("путь к базам Cronos не может быть пустым")
	}
	
	if c.ClickHouseAddr == "" {
		return fmt.Errorf("адрес ClickHouse не может быть пустым")
	}
	
	if c.ClickHouseDatabase == "" {
		return fmt.Errorf("имя базы данных ClickHouse не может быть пустым")
	}
	
	if c.BatchSize <= 0 {
		return fmt.Errorf("размер батча должен быть больше 0")
	}
	
	return nil
}

// String возвращает строковое представление конфигурации (без пароля)
func (c *Config) String() string {
	return fmt.Sprintf("Config{CronosBasePath: %s, ClickHouseAddr: %s, Database: %s, User: %s, BatchSize: %d, Verbose: %t}",
		c.CronosBasePath, c.ClickHouseAddr, c.ClickHouseDatabase, c.ClickHouseUser, c.BatchSize, c.Verbose)
}