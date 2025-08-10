package cronodamp

import (
	"fmt"
	"os"
	"strings"
)

// Config конфигурация для cronodamp
type Config struct {
	// Путь к папке с базами данных Kronos
	KronosDBPath string
	
	// Строка подключения к ClickHouse (хост:порт)
	ClickHouseDSN      string
	ClickHouseDatabase string
	ClickHouseUser     string
	ClickHousePassword string
	
	// Дополнительные настройки
	BatchSize int // Размер батча для вставки данных
	Debug     bool
}

// NewConfigFromEnv создает конфигурацию из переменных окружения
func NewConfigFromEnv() Config {
	return Config{
		KronosDBPath:       getEnvOrDefault("KRONOS_DB_PATH", "/home/usersamba/smb/"),
		ClickHouseDSN:      getEnvOrDefault("CLICKHOUSE_DSN", "localhost:9000"),
		ClickHouseDatabase: getEnvOrDefault("CLICKHOUSE_DB", "default"),
		ClickHouseUser:     getEnvOrDefault("CLICKHOUSE_USER", "default"),
		ClickHousePassword: getEnvOrDefault("CLICKHOUSE_PASSWORD", "default"),
		BatchSize:          1000,
		Debug:              getEnvOrDefault("DEBUG", "false") == "true",
	}
}

// GetClickHouseAddr возвращает массив адресов ClickHouse
func (c Config) GetClickHouseAddr() []string {
	// Поддерживаем множественные адреса через запятую
	return strings.Split(c.ClickHouseDSN, ",")
}

// Validate проверяет корректность конфигурации
func (c Config) Validate() error {
	if c.KronosDBPath == "" {
		return fmt.Errorf("не указан путь к базам данных Kronos")
	}
	
	if c.ClickHouseDSN == "" {
		return fmt.Errorf("не указана строка подключения к ClickHouse")
	}
	
	// Проверяем существование папки с базами
	if _, err := os.Stat(c.KronosDBPath); os.IsNotExist(err) {
		return fmt.Errorf("папка с базами Kronos не существует: %s", c.KronosDBPath)
	}
	
	return nil
}

// getEnvOrDefault получает значение переменной окружения или возвращает значение по умолчанию
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}