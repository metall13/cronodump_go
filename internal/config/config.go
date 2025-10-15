package config

import (
	"os"
	"strconv"
)

type Config struct {
	LogLevel     string
	TempDir      string
	MaxFileSize  int64
	ClickHouse   ClickHouseConfig
	Database     DatabaseConfig
}

type ClickHouseConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Driver   string
}

func Load() *Config {
	return &Config{
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		TempDir:     getEnv("TEMP_DIR", "/tmp/cronodump"),
		MaxFileSize: getEnvAsInt64("MAX_FILE_SIZE", 100*1024*1024), // 100MB
		ClickHouse: ClickHouseConfig{
			Host:     getEnv("CLICKHOUSE_HOST", "localhost"),
			Port:     getEnvAsInt("CLICKHOUSE_PORT", 9000),
			Database: getEnv("CLICKHOUSE_DATABASE", "cronodump"),
			Username: getEnv("CLICKHOUSE_USERNAME", "default"),
			Password: getEnv("CLICKHOUSE_PASSWORD", ""),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			Database: getEnv("DB_DATABASE", "cronodump"),
			Username: getEnv("DB_USERNAME", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Driver:   getEnv("DB_DRIVER", "postgres"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}