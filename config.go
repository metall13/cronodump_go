package cronodamp

import (
	"os"
	"strings"
)

// Config содержит конфигурацию для библиотеки cronodamp
type Config struct {
	// Путь к директории где хранятся базы данных Kronos
	DatabasePath string
	// Строка подключения к ClickHouse (адрес:порт)
	ClickHouseDSN string
	// Пользователь ClickHouse
	ClickHouseUser string
	// Пароль ClickHouse
	ClickHousePassword string
	// База данных ClickHouse
	ClickHouseDatabase string
}

// NewConfigFromEnv создает конфигурацию из переменных окружения
func NewConfigFromEnv() Config {
	return Config{
		DatabasePath:       getEnvDefault("PATH_TO_DB", "/home/usersamba/smb/"),
		ClickHouseDSN:      getEnvDefault("CLICKHOUSE_DSN", "localhost:9000"),
		ClickHouseUser:     getEnvDefault("CLICKHOUSE_USER", "default"),
		ClickHousePassword: getEnvDefault("CLICKHOUSE_PASSWORD", "default"),
		ClickHouseDatabase: getEnvDefault("CLICKHOUSE_DATABASE", "default"),
	}
}

// getEnvDefault возвращает значение переменной окружения или значение по умолчанию
func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetClickHouseAddr возвращает массив адресов для подключения к ClickHouse
func (c Config) GetClickHouseAddr() []string {
	return strings.Split(c.ClickHouseDSN, ",")
}