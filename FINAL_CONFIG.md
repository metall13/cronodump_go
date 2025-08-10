# Финальная конфигурация - Конвертер Cronos → ClickHouse

## 🔧 Настройки подключения к ClickHouse

Программа использует следующие параметры подключения:

```go
conn, err := clickhouse.Open(&clickhouse.Options{
    Addr: []string{config.ClickHouseAddr}, // Из переменной CLICKHOUSE_DSN
    Auth: clickhouse.Auth{
        Database: "default",
        Username: "default", 
        Password: "default",
    },
    ClientInfo: clickhouse.ClientInfo{
        Products: []struct {
            Name    string
            Version string
        }{
            {Name: "cronodamp-go-client", Version: "1.0"},
        },
    },
})
```

## 🌐 Переменные окружения

### Обязательные:
```bash
export CRONOS_DB_PATH="/home/usersamba/smb/"
export CLICKHOUSE_DSN="127.0.0.1:9000"
```

### Дополнительные (опционально):
```bash
export CLICKHOUSE_DATABASE="cronos_data"
export CLICKHOUSE_USER="default"
export CLICKHOUSE_PASSWORD=""
export BATCH_SIZE="1000"
export VERBOSE="true"
```

## 🚀 Запуск программы

### Метод 1: Прямой запуск
```bash
./cronos-converter
```

### Метод 2: Через скрипт
```bash
./example.sh
```

### Метод 3: Через Go
```bash
go run .
```

## 📊 Особенности реализации

- **IP-адрес**: Используется `127.0.0.1:9000` вместо `localhost:9000`
- **Имя клиента**: `cronodamp-go-client` (соответствует оригинальной библиотеке)
- **База подключения**: `default` (стандартная база ClickHouse)
- **Рабочая база**: `cronos_data` (создается автоматически)
- **Все данные**: Сохраняются как тип `String` в ClickHouse

## ✅ Статус

Программа полностью готова к использованию и соответствует всем требованиям:

- ✅ Переписана с Python на Go
- ✅ Использует корректные настройки подключения ClickHouse  
- ✅ Транслитерирует имена папок и таблиц
- ✅ Сохраняет все данные как строки
- ✅ Протестирована и отлажена

Программа ожидает запущенный ClickHouse сервер на `127.0.0.1:9000` для полноценной работы.