# Cronodamp Go

🚀 **Cronodamp Go** - это мощная библиотека для экспорта данных из баз данных CronosPro в ClickHouse, написанная на Go.

Эта библиотека является переписанной на Go версией оригинальной [cronodump](https://github.com/alephdata/cronodump) Python-библиотеки и предназначена для работы с базами данных CronosPro, популярными среди российских государственных учреждений и компаний.

## ✨ Особенности

- 📊 **Полная поддержка форматов Cronos**: CSV, текстовые файлы, бинарные файлы
- 🔄 **Автоматическая транслитерация**: Конвертация названий таблиц и колонок по правилам ClickHouse
- 📁 **Умное именование**: Имя таблицы формируется из имени папки и файла
- 🗃️ **Строковый формат данных**: Все данные экспортируются как строки для максимальной совместимости
- ⚡ **Высокая производительность**: Оптимизированные batch-операции для больших объемов данных
- 🛡️ **Надежность**: Обработка ошибок и восстановление после сбоев
- 📈 **Прогресс выполнения**: Детальная информация о ходе экспорта

## 🏗️ Архитектура

Библиотека состоит из нескольких ключевых модулей:

- **CronosReader** - чтение данных из различных форматов Cronos
- **ClickHouseClient** - работа с базой данных ClickHouse
- **Transliteration** - транслитерация и валидация имен
- **Exporter** - основная логика экспорта

## 📦 Установка

```bash
# Клонирование репозитория
git clone <repository-url>
cd cronodamp

# Установка зависимостей
go mod tidy

# Сборка приложения
go build -o cronodamp main.go
```

## ⚙️ Конфигурация

### Переменные окружения

```bash
# Обязательные
export PATH_TO_DB="/home/usersamba/smb/"

# ClickHouse (опциональные, есть значения по умолчанию)
export CLICKHOUSE_DSN="localhost:9000"
export CLICKHOUSE_USER="default"
export CLICKHOUSE_PASSWORD="default"
export CLICKHOUSE_DATABASE="default"
```

### Пример .env файла

```env
PATH_TO_DB=/home/usersamba/smb/
CLICKHOUSE_DSN=localhost:9000
CLICKHOUSE_USER=default
CLICKHOUSE_PASSWORD=default
CLICKHOUSE_DATABASE=default
```

## 🚀 Использование

### Командная строка

```bash
# Простой экспорт
./cronodamp --db /path/to/cronos/data

# Экспорт с настройками ClickHouse
./cronodamp --db /path/to/cronos/data --clickhouse remote:9000 --user myuser --password mypass

# Проверка конфигурации
./cronodamp --db /path/to/cronos/data --validate

# Список таблиц
./cronodamp --db /path/to/cronos/data --list

# Справка
./cronodamp --help
```

### Программное использование

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "your-module/cronodamp"
)

func main() {
    // Создание конфигурации
    cfg := cronodamp.Config{
        DatabasePath:       "/home/usersamba/smb/",
        ClickHouseDSN:      "localhost:9000",
        ClickHouseUser:     "default",
        ClickHousePassword: "default",
        ClickHouseDatabase: "default",
    }
    
    // Создание экспортера
    exporter, err := cronodamp.NewCronodampExporter(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer exporter.Close()
    
    // Экспорт базы данных
    ctx := context.Background()
    results, err := exporter.ExportWithProgress(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Обработка результатов
    for _, result := range results {
        if result.Success {
            fmt.Printf("✅ %s: %d строк\n", result.TableName, result.RowsExported)
        } else {
            fmt.Printf("❌ %s: %v\n", result.TableName, result.Error)
        }
    }
}
```

## 📋 Правила транслитерации

Библиотека автоматически конвертирует названия таблиц и колонок по следующим правилам:

1. **Объединение имен**: `{папка}_{таблица}`
2. **Транслитерация**: Кириллица → латиница
3. **Нормализация**: Приведение к нижнему регистру
4. **Валидация**: Замена недопустимых символов на подчеркивания
5. **Ограничения ClickHouse**: Максимум 127 символов

### Примеры транслитерации

```
Входные данные:
Папка: "Документы", Файл: "Сотрудники.csv"
Результат: "dokumenty_sotrudniki"

Папка: "Reports-2023", Файл: "Sales Data.txt"
Результат: "reports_2023_sales_data"
```

## 📊 Структура таблиц ClickHouse

Все таблицы создаются со следующей структурой:

```sql
CREATE TABLE example_table (
    `column1` String,
    `column2` String,
    `column3` String,
    `import_timestamp` DateTime DEFAULT now(),
    `source_table` String
) ENGINE = MergeTree()
ORDER BY import_timestamp
```

### Служебные колонки

- **import_timestamp** - время импорта записи
- **source_table** - путь к исходному файлу

## 🗂️ Поддерживаемые форматы

### CSV файлы
- Автоматическое определение заголовков
- Разделитель: запятая
- Все значения конвертируются в строки

### Текстовые файлы (.txt)
- Каждая строка становится записью
- Одна колонка: `content`

### Бинарные файлы Cronos
- Hex dump для отладки
- Колонки: `offset`, `hex_data`, `ascii_data`

### SQLite файлы
- Автоматическое определение
- Поддержка запланирована в будущих версиях

## 🔧 Расширенные возможности

### Пакетная обработка

```go
// Экспорт отдельной таблицы
result, err := exporter.ExportTable(ctx, "/path/to/table.csv")

// Получение информации о базе данных
info, err := exporter.GetDatabaseInfo(ctx)
fmt.Printf("Таблиц: %d\n", info.TableCount)

// Список таблиц
tables, err := exporter.ListTables(ctx)
```

### Обработка ошибок

```go
results, err := exporter.ExportDatabase(ctx)
for _, result := range results {
    if !result.Success {
        log.Printf("Ошибка в таблице %s: %v", result.TableName, result.Error)
        // Продолжить обработку других таблиц
    }
}
```

## 📈 Производительность

- **Batch-операции**: Оптимизированная вставка данных
- **Потоковая обработка**: Минимальное потребление памяти
- **Параллельная обработка**: Готовность к многопоточности

### Рекомендации

- Для больших баз данных используйте SSD диски
- Увеличьте batch_size для больших таблиц
- Мониторьте использование памяти при работе с большими файлами

## 🛠️ Разработка

### Структура проекта

```
cronodamp/
├── config.go           # Конфигурация
├── types.go            # Типы данных
├── transliteration.go  # Транслитерация
├── cronos_reader.go    # Чтение Cronos
├── clickhouse.go       # Работа с ClickHouse
├── exporter.go         # Основная логика
├── main.go             # CLI приложение
├── go.mod              # Go модули
└── README.md           # Документация
```

### Добавление новых форматов

1. Реализуйте метод в `CronosFileReader`
2. Добавьте определение формата в `isCronosFile`
3. Создайте тесты для нового формата

## 🐛 Отладка

### Включение debug-режима

ClickHouse клиент автоматически выводит debug информацию:

```
[ClickHouse Debug] Connected to ClickHouse
[ClickHouse Debug] Creating table: example_table
[ClickHouse Debug] Inserting 1000 rows
```

### Логирование

Все операции логируются с префиксами:
- ✅ Успешные операции
- ❌ Ошибки
- ⚠️ Предупреждения
- 📋 Информационные сообщения

## 🤝 Вклад в проект

1. Fork репозитория
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменения (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📄 Лицензия

Этот проект лицензирован под MIT License - смотрите файл [LICENSE](LICENSE) для деталей.

## 🙏 Благодарности

- Оригинальный проект [cronodump](https://github.com/alephdata/cronodump) от alephdata
- Сообщество Go за отличные библиотеки
- Команда ClickHouse за производительную СУБД

## 📞 Поддержка

Если у вас есть вопросы или проблемы:

1. Проверьте [Issues](../../issues) на GitHub
2. Создайте новый Issue с подробным описанием
3. Приложите логи и примеры данных (без конфиденциальной информации)

---

**Cronodamp Go** - делаем миграцию данных из Cronos простой и надежной! 🚀