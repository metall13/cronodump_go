package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config структура конфигурации приложения
type Config struct {
	// Основные настройки
	DatabasePath string `json:"database_path"`
	OutputPath   string `json:"output_path"`
	
	// Настройки ClickHouse
	ClickHouseHost     string `json:"clickhouse_host"`
	ClickHousePort     int    `json:"clickhouse_port"`
	ClickHouseDatabase string `json:"clickhouse_database"`
	ClickHouseUsername string `json:"clickhouse_username"`
	ClickHousePassword string `json:"clickhouse_password"`
	ClickHouseSSL      bool   `json:"clickhouse_ssl"`
	
	// Настройки обработки
	BatchSize       int  `json:"batch_size"`
	MaxConnections  int  `json:"max_connections"`
	EnableLogging   bool `json:"enable_logging"`
	LogLevel        string `json:"log_level"`
	
	// Настройки фильтрации
	IncludeTables   []string `json:"include_tables"`
	ExcludeTables   []string `json:"exclude_tables"`
	IncludeViews    bool     `json:"include_views"`
	IncludeIndexes  bool     `json:"include_indexes"`
	
	// Настройки преобразования
	TransliterateNames bool `json:"transliterate_names"`
	LowerCaseNames     bool `json:"lowercase_names"`
	
	// Настройки безопасности
	MaxMemoryUsage  int64 `json:"max_memory_usage"`
	ConnectionTimeout int `json:"connection_timeout"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		DatabasePath:       "",
		OutputPath:         "./output",
		
		ClickHouseHost:     "localhost",
		ClickHousePort:     9000,
		ClickHouseDatabase: "default",
		ClickHouseUsername: "default",
		ClickHousePassword: "",
		ClickHouseSSL:      false,
		
		BatchSize:          1000,
		MaxConnections:     5,
		EnableLogging:      true,
		LogLevel:          "INFO",
		
		IncludeTables:      []string{},
		ExcludeTables:      []string{},
		IncludeViews:       true,
		IncludeIndexes:     false,
		
		TransliterateNames: true,
		LowerCaseNames:     true,
		
		MaxMemoryUsage:     1024 * 1024 * 1024, // 1GB
		ConnectionTimeout:  30,
	}
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(path string) (Config, error) {
	config := DefaultConfig()
	
	// Если файл не существует, возвращаем конфигурацию по умолчанию
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config, nil
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	
	return config, nil
}

// SaveConfig сохраняет конфигурацию в файл
func (c Config) SaveConfig(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}

// Validate проверяет корректность конфигурации
func (c Config) Validate() error {
	if c.DatabasePath == "" {
		return fmt.Errorf("database_path не может быть пустым")
	}
	
	if c.OutputPath == "" {
		return fmt.Errorf("output_path не может быть пустым")
	}
	
	if c.ClickHouseHost == "" {
		return fmt.Errorf("clickhouse_host не может быть пустым")
	}
	
	if c.ClickHousePort <= 0 || c.ClickHousePort > 65535 {
		return fmt.Errorf("clickhouse_port должен быть от 1 до 65535")
	}
	
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch_size должен быть больше 0")
	}
	
	if c.MaxConnections <= 0 {
		return fmt.Errorf("max_connections должен быть больше 0")
	}
	
	return nil
}