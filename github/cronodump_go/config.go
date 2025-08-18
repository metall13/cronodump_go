package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config содержит конфигурацию приложения
type Config struct {
	// Путь до папки с базами Kronos
	KronosDatabasesPath string `json:"kronos_databases_path"`
	
	// Строка подключения к ClickHouse
	ClickHouseURL string `json:"clickhouse_url"`
	
	// Путь для сохранения CSV файлов (если не используется ClickHouse)
	OutputPath string `json:"output_path"`
	
	// Настройки ClickHouse
	ClickHouse ClickHouseConfig `json:"clickhouse"`
}

// ClickHouseConfig содержит настройки подключения к ClickHouse
type ClickHouseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSL      bool   `json:"ssl"`
}

// LoadConfig загружает конфигурацию из JSON файла
func LoadConfig(path string) (Config, error) {
	config := Config{
		// Значения по умолчанию
		KronosDatabasesPath: "/home/usersamba/smb/bd_cronos",
		OutputPath:          "./output",
		ClickHouse: ClickHouseConfig{
			Host:     "localhost",
			Port:     9000,
			Database: "kronos_data",
			Username: "default",
			Password: "",
			SSL:      false,
		},
	}

	// Если файл конфигурации не существует, создаем с дефолтными значениями
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config, SaveConfig(path, config)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	// Формируем ClickHouseURL если он не задан
	if config.ClickHouseURL == "" {
		protocol := "tcp"
		if config.ClickHouse.SSL {
			protocol = "https"
		}
		config.ClickHouseURL = fmt.Sprintf("%s://%s:%s@%s:%d/%s",
			protocol,
			config.ClickHouse.Username,
			config.ClickHouse.Password,
			config.ClickHouse.Host,
			config.ClickHouse.Port,
			config.ClickHouse.Database)
	}

	return config, nil
}

// SaveConfig сохраняет конфигурацию в JSON файл
func SaveConfig(path string, config Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}