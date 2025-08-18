# Cronodump Go

Утилита для конвертации баз данных CronosPro в ClickHouse или CSV формат, написанная на Go.

## Особенности

- Автоматическое сканирование папок с базами Kronos
- Транслитерация имен таблиц и колонок согласно правилам ClickHouse
- Именование таблиц по схеме: `{папка}_{имя_таблицы}`
- Поддержка экспорта в ClickHouse и CSV
- Обработка зашифрованных баз данных Kronos
- Поддержка различных версий формата Kronos (v3, v4)

## Установка

```bash
cd github/cronodump_go
go mod tidy
go build -o cronodump_go
```

## Конфигурация

Создайте файл `config.json` со следующими параметрами:

```json
{
  "kronos_databases_path": "/home/usersamba/smb/bd_cronos",
  "clickhouse_url": "",
  "output_path": "./output",
  "clickhouse": {
    "host": "localhost",
    "port": 9000,
    "database": "kronos_data",
    "username": "default",
    "password": "",
    "ssl": false
  }
}
```

## Использование

### Список доступных папок с базами:
```bash
./cronodump_go -list
```

### Экспорт всех баз в ClickHouse:
```bash
./cronodump_go -clickhouse
```

### Экспорт конкретной папки:
```bash
./cronodump_go -folder "название_папки" -clickhouse
```

### Экспорт конкретной таблицы:
```bash
./cronodump_go -table "название_таблицы" -clickhouse
```

### Экспорт в CSV:
```bash
./cronodump_go -csv
```

### Подробный вывод:
```bash
./cronodump_go -v -clickhouse
```

## Структура проекта

- `main.go` - основная логика приложения
- `config.go` - работа с конфигурацией
- `kronos_parser.go` - парсер баз данных Kronos
- `kronos_model.go` - модель данных и типы полей
- `clickhouse_client.go` - клиент для работы с ClickHouse
- `transliteration.go` - транслитерация имен

## Поддерживаемые типы полей Kronos

- Системный номер (PRIMARY KEY) → UInt32
- Целое число → Int32  
- Строка → String
- Текст → String
- Дата → Date
- Время → String
- Ссылка на файл → String

## Правила именования таблиц

Имена таблиц формируются по схеме: `{имя_папки}_{имя_таблицы}`

Все имена транслитерируются и приводятся к нижнему регистру согласно правилам ClickHouse:
- Русские символы транслитерируются в латиницу
- Спецсимволы заменяются на подчеркивания
- Зарезервированные слова получают префикс `col_`
- Имена должны начинаться с буквы или подчеркивания