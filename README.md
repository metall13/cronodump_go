# Cronodamp - Экспортер данных из Kronos в ClickHouse

Cronodamp - это Go библиотека для экспорта данных из баз данных Kronos в ClickHouse с автоматической транслитерацией названий таблиц и колонок.

## Основные возможности

- ✅ Автоматическое обнаружение таблиц в папках Kronos
- ✅ Транслитерация русских названий в латиницу по правилам ClickHouse
- ✅ Именование таблиц по схеме: `папка_таблица`
- ✅ Все данные экспортируются как строки (String)
- ✅ Поддержка различных форматов файлов (.txt, .csv, .dat)
- ✅ Батчевая загрузка данных
- ✅ Детальная статистика экспорта
- ✅ Обработка ошибок и логирование

## Установка

```bash
go mod init your-project
go get github.com/ClickHouse/clickhouse-go/v2
```

## Конфигурация

Библиотека использует переменные окружения:

```bash
export KRONOS_DB_PATH="/home/usersamba/smb/"
export CLICKHOUSE_DSN="localhost:9000"
export CLICKHOUSE_DB="default"
export CLICKHOUSE_USER="default"
export CLICKHOUSE_PASSWORD="default"
export DEBUG="true"
```

## Использование

### Простой пример

```go
package main

import (
    "log"
    "cronodamp"
)

func main() {
    // Создаем конфигурацию из переменных окружения
    config := cronodamp.NewConfigFromEnv()
    
    // Создаем экспортер
    exporter, err := cronodamp.NewExporter(config)
    if err != nil {
        log.Fatal(err)
    }
    defer exporter.Close()
    
    // Экспортируем все таблицы
    err = exporter.ExportAll()
    if err != nil {
        log.Fatal(err)
    }
    
    // Выводим статистику
    exporter.PrintStats()
}
```

### Использование CLI

Собираем и запускаем:

```bash
go build -o cronodamp ./cmd/cronodamp
./cronodamp --help
```

Примеры команд:

```bash
# Показать список таблиц
./cronodamp --list

# Проверить подключение
./cronodamp --validate

# Экспортировать все таблицы
./cronodamp

# Экспортировать конкретный файл
./cronodamp --file "/path/to/file.txt"

# Экспортировать все таблицы из папки
./cronodamp --folder "folder_name"

# Включить отладку
./cronodamp --debug
```

## Структура таблиц в ClickHouse

Каждая таблица создается со следующими колонками:

- Все колонки из исходного файла (как String)
- `import_timestamp` - время импорта (DateTime)
- `source_table` - имя исходной таблицы (String)
- `source_folder` - имя папки (String)
- `source_file_path` - полный путь к файлу (String)

## Правила транслитерации

- Русские символы транслитерируются в латиницу
- Небуквенные символы заменяются на подчеркивание
- Имена таблиц формируются как `папка_таблица`
- Все имена приводятся к нижнему регистру
- Максимальная длина имени - 63 символа

Примеры:

- `Отчеты/Продажи.txt` → `otchety_prodazhi`
- `База данных/Клиенты 2024.csv` → `baza_dannykh_klienty_2024`

## API библиотеки

### Основные типы

```go
type Config struct {
    KronosDBPath       string
    ClickHouseDSN      string
    ClickHouseDatabase string
    ClickHouseUser     string
    ClickHousePassword string
    BatchSize          int
    Debug              bool
}

type KronosTable struct {
    FolderName string
    TableName  string
    FilePath   string
    Columns    []Column
}

type Exporter struct {
    // ...
}
```

### Основные методы

```go
// Создание экспортера
exporter, err := cronodamp.NewExporter(config)

// Экспорт всех таблиц
err = exporter.ExportAll()

// Экспорт конкретных файлов
err = exporter.ExportSpecific([]string{"file1.txt", "file2.csv"})

// Экспорт папки
err = exporter.ExportFolder("folder_name")

// Получение списка таблиц
tables, err := exporter.ListTables()

// Получение статистики
stats := exporter.GetStats()
```

## Поддерживаемые форматы файлов

- `.txt` - текстовые файлы с разделителями
- `.csv` - CSV файлы
- `.dat` - файлы данных

Поддерживаемые разделители:
- Табуляция (`\t`)
- Точка с запятой (`;`)
- Запятая (`,`)

## Обработка ошибок

Библиотека продолжает работу при ошибках в отдельных таблицах и ведет статистику ошибок. Все ошибки логируются в режиме отладки.

## Требования

- Go 1.21+
- ClickHouse сервер
- Доступ к папке с базами Kronos

## Лицензия

MIT License