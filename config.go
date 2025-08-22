package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config содержит конфигурацию приложения
type Config struct {
	DatabasePath     string `json:"database_path"`
	OutputPath       string `json:"output_path"`
	ClickHouseConfig struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Database string `json:"database"`
		Username string `json:"username"`
		Password string `json:"password"`
		ConnStr  string `json:"connection_string"`
	} `json:"clickhouse"`
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(path string) (Config, error) {
	var config Config

	// Устанавливаем значения по умолчанию
	config.OutputPath = "cronodump-output"
	config.ClickHouseConfig.Host = "localhost"
	config.ClickHouseConfig.Port = 9000
	config.ClickHouseConfig.Database = "default"
	config.ClickHouseConfig.Username = "default"
	config.ClickHouseConfig.Password = ""

	// Если файл конфигурации существует, загружаем его
	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			return config, err
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&config); err != nil {
			return config, err
		}
	}

	// Если строка подключения не указана, формируем её
	if config.ClickHouseConfig.ConnStr == "" {
		config.ClickHouseConfig.ConnStr = fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s",
			config.ClickHouseConfig.Username,
			config.ClickHouseConfig.Password,
			config.ClickHouseConfig.Host,
			config.ClickHouseConfig.Port,
			config.ClickHouseConfig.Database)
	}

	return config, nil
}