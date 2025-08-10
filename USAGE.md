# Руководство по использованию Cronodamp Go

## 🚀 Быстрый старт

### 1. Установка зависимостей

```bash
go mod tidy
```

### 2. Компиляция

```bash
go build -o cronodamp ./cmd/cronodamp
```

### 3. Настройка переменных окружения

```bash
export PATH_TO_DB="/home/usersamba/smb/"
export CLICKHOUSE_DSN="localhost:9000"
```

### 4. Базовое использование

```bash
# Просмотр списка таблиц (работает без ClickHouse)
./cronodamp --db /path/to/cronos/data --list

# Проверка конфигурации
./cronodamp --db /path/to/cronos/data --validate

# Экспорт в ClickHouse (требует запущенный ClickHouse)
./cronodamp --db /path/to/cronos/data
```

## 📁 Структура проекта

```
cronodamp/
├── cmd/cronodamp/           # CLI приложение
│   └── main.go             # Точка входа
├── example/                # Примеры использования
│   └── main.go            # Программный пример
├── test_data/             # Тестовые данные
│   ├── documents/         # Папка с документами
│   │   └── employees.csv  # CSV файл сотрудников
│   └── reports/          # Папка с отчетами
│       └── sales.txt     # Текстовый отчет
├── config.go             # Конфигурация
├── types.go              # Типы данных
├── transliteration.go    # Транслитерация
├── cronos_reader.go      # Чтение Cronos
├── clickhouse.go         # Работа с ClickHouse
├── exporter.go           # Основная логика
├── go.mod               # Go модули
├── README.md            # Основная документация
└── USAGE.md             # Это руководство
```

## 🔧 Конфигурация

### Переменные окружения

| Переменная | Описание | Значение по умолчанию | Обязательная |
|------------|----------|-----------------------|--------------|
| `PATH_TO_DB` | Путь к базе данных Cronos | `/home/usersamba/smb/` | ✅ |
| `CLICKHOUSE_DSN` | Адрес ClickHouse | `localhost:9000` | ❌ |
| `CLICKHOUSE_USER` | Пользователь ClickHouse | `default` | ❌ |
| `CLICKHOUSE_PASSWORD` | Пароль ClickHouse | `default` | ❌ |
| `CLICKHOUSE_DATABASE` | База данных ClickHouse | `default` | ❌ |

### Пример .env файла

```env
PATH_TO_DB=/home/usersamba/smb/
CLICKHOUSE_DSN=localhost:9000
CLICKHOUSE_USER=default
CLICKHOUSE_PASSWORD=default
CLICKHOUSE_DATABASE=default
```

## 📊 Поддерживаемые форматы данных

### 1. CSV файлы (.csv)

- **Структура**: Первая строка содержит заголовки, последующие - данные
- **Разделитель**: Запятая (`,`)
- **Кодировка**: UTF-8
- **Пример**:
  ```csv
  Имя,Фамилия,Возраст
  Иван,Петров,25
  Мария,Сидорова,30
  ```

### 2. Текстовые файлы (.txt)

- **Структура**: Каждая строка становится отдельной записью
- **Колонка**: `content` (содержимое строки)
- **Пример**:
  ```txt
  Отчет по продажам за январь
  Общая сумма: 1,500,000 рублей
  Количество сделок: 45
  ```

### 3. Бинарные файлы Cronos (.dat, .idx, и др.)

- **Обработка**: Hex dump для анализа
- **Колонки**: `offset`, `hex_data`, `ascii_data`
- **Применение**: Отладка и исследование форматов

## 🏗️ Архитектура системы

### Компоненты

1. **CronosReader** - чтение и парсинг данных Cronos
2. **Transliteration** - конвертация имен в валидный формат
3. **ClickHouseClient** - подключение и работа с ClickHouse
4. **Exporter** - координация процесса экспорта

### Поток данных

```
Cronos DB → CronosReader → Transliteration → ClickHouse
    ↓              ↓              ↓             ↓
 Файлы         Таблицы      Валидные       Таблицы
данных         в памяти     имена          в БД
```

## 🔄 Процесс транслитерации

### Правила

1. **Объединение**: `{папка}_{файл}` → `documents_employees`
2. **Транслитерация**: `документы_сотрудники` → `dokumenty_sotrudniki`
3. **Нормализация**: `DokumentY_SotrudnikI` → `dokumenty_sotrudniki`
4. **Валидация**: `dokument@y-sotru#dniki` → `dokumenty_sotrudniki`
5. **Ограничения**: Максимум 127 символов

### Примеры

| Вход | Выход |
|------|-------|
| `Документы/Сотрудники.csv` | `dokumenty_sotrudniki` |
| `Reports-2023/Sales Data.txt` | `reports_2023_sales_data` |
| `База/Клиенты (активные).db` | `baza_klienty_aktivnye_db` |

## 📋 Командная строка

### Основные команды

```bash
# Справка
./cronodamp --help

# Список таблиц
./cronodamp --db /path/to/data --list

# Валидация конфигурации
./cronodamp --db /path/to/data --validate

# Экспорт данных
./cronodamp --db /path/to/data

# Экспорт с настройками ClickHouse
./cronodamp --db /path/to/data \
  --clickhouse remote:9000 \
  --user myuser \
  --password mypass \
  --database mydatabase
```

### Флаги

| Флаг | Короткий | Описание |
|------|----------|----------|
| `--db PATH` | | Путь к базе данных Cronos |
| `--clickhouse ADDR` | | Адрес ClickHouse |
| `--user USER` | | Пользователь ClickHouse |
| `--password PASS` | | Пароль ClickHouse |
| `--database DB` | | База данных ClickHouse |
| `--list` | | Показать список таблиц |
| `--validate` | | Проверить конфигурацию |
| `--help` | | Показать справку |

## 🖥️ Программное использование

### Базовый пример

```go
package main

import (
    "context"
    "cronodamp"
)

func main() {
    // Конфигурация
    cfg := cronodamp.Config{
        DatabasePath:       "/path/to/cronos/data",
        ClickHouseDSN:      "localhost:9000",
        ClickHouseUser:     "default",
        ClickHousePassword: "default",
        ClickHouseDatabase: "default",
    }
    
    // Создание экспортера
    exporter, err := cronodamp.NewCronodampExporter(cfg)
    if err != nil {
        panic(err)
    }
    defer exporter.Close()
    
    // Экспорт
    ctx := context.Background()
    results, err := exporter.ExportDatabase(ctx)
    if err != nil {
        panic(err)
    }
    
    // Обработка результатов
    for _, result := range results {
        if result.Success {
            fmt.Printf("✅ %s: %d строк\n", 
                result.TableName, result.RowsExported)
        }
    }
}
```

### Работа только с чтением

```go
// Создание ридера без ClickHouse
exporter, err := cronodamp.NewCronodampReader(cfg)
if err != nil {
    panic(err)
}

// Получение списка таблиц
tables, err := exporter.ListTables(ctx)
if err != nil {
    panic(err)
}

// Получение информации о БД
info, err := exporter.GetDatabaseInfo(ctx)
if err != nil {
    panic(err)
}
```

## 🗄️ ClickHouse

### Структура таблиц

Все таблицы создаются с единой структурой:

```sql
CREATE TABLE table_name (
    `column1` String,
    `column2` String,
    `column3` String,
    -- ... остальные колонки ...
    `import_timestamp` DateTime DEFAULT now(),
    `source_table` String
) ENGINE = MergeTree()
ORDER BY import_timestamp;
```

### Служебные колонки

- **`import_timestamp`** - время импорта записи
- **`source_table`** - путь к исходному файлу

### Запросы для анализа

```sql
-- Список всех импортированных таблиц
SELECT DISTINCT source_table, count() as rows
FROM table_name 
GROUP BY source_table;

-- Данные за последний час
SELECT * FROM table_name 
WHERE import_timestamp >= now() - INTERVAL 1 HOUR;

-- Статистика по источникам
SELECT 
    source_table,
    count() as total_rows,
    min(import_timestamp) as first_import,
    max(import_timestamp) as last_import
FROM table_name 
GROUP BY source_table;
```

## 🐛 Отладка и диагностика

### Уровни логирования

- ✅ **Успешные операции** - зеленые сообщения
- ❌ **Ошибки** - красные сообщения с описанием
- ⚠️ **Предупреждения** - желтые сообщения
- 📋 **Информация** - синие информационные сообщения

### Частые проблемы

#### 1. Ошибка подключения к ClickHouse

```
❌ Ошибка: dial tcp [::1]:9000: connect: connection refused
```

**Решение:**
- Убедитесь, что ClickHouse запущен
- Проверьте правильность адреса и порта
- Используйте `--list` для работы без ClickHouse

#### 2. Пустой список таблиц

```
📋 Найдено 0 таблиц
```

**Решение:**
- Проверьте правильность пути к данным
- Убедитесь, что в папке есть поддерживаемые файлы
- Проверьте права доступа к файлам

#### 3. Ошибки транслитерации

**Решение:**
- Все символы автоматически конвертируются
- Проверьте финальные имена таблиц в ClickHouse
- При необходимости переименуйте исходные файлы

### Debug режим

ClickHouse клиент выводит отладочную информацию:

```
[ClickHouse Debug] Connected to ClickHouse
[ClickHouse Debug] Creating table: documents_employees
[ClickHouse Debug] Inserting 100 rows
```

## 📈 Производительность

### Рекомендации

1. **Для больших файлов:**
   - Используйте SSD диски
   - Увеличьте RAM для ClickHouse
   - Рассмотрите разбиение на части

2. **Для множества файлов:**
   - Обрабатывайте по частям
   - Используйте мониторинг памяти
   - Настройте batch размеры в ClickHouse

3. **Оптимизация ClickHouse:**
   ```sql
   -- Оптимизация таблицы после импорта
   OPTIMIZE TABLE table_name;
   
   -- Сжатие данных
   ALTER TABLE table_name MODIFY SETTING 
   storage_policy = 'hot_and_cold';
   ```

### Мониторинг

```bash
# Просмотр активности
watch -n 1 './cronodamp --db /path/to/data --list | wc -l'

# Мониторинг ClickHouse
clickhouse-client --query "SHOW PROCESSLIST"
```

## 🔐 Безопасность

### Рекомендации

1. **Доступ к файлам:**
   - Минимальные права доступа
   - Безопасные пути к файлам
   - Проверка существования файлов

2. **ClickHouse:**
   - Используйте отдельного пользователя
   - Ограничьте права доступа
   - Настройте SSL при необходимости

3. **Данные:**
   - Не логируйте конфиденциальную информацию
   - Используйте маскирование в prod
   - Регулярные бэкапы

## 🤝 Расширение функциональности

### Добавление новых форматов

1. **Создайте метод в `CronosFileReader`:**
   ```go
   func (r *CronosFileReader) readCustomFormat(path string) (*Table, error) {
       // Ваша логика чтения
   }
   ```

2. **Добавьте определение в `isCronosFile`:**
   ```go
   if strings.HasSuffix(ext, ".custom") {
       return true
   }
   ```

3. **Обновите `ReadTable`:**
   ```go
   if strings.HasSuffix(path, ".custom") {
       return r.readCustomFormat(path)
   }
   ```

### Создание собственных экспортеров

```go
type CustomExporter struct {
    reader *CronosFileReader
}

func (e *CustomExporter) ExportToCSV(table *Table) error {
    // Ваша логика экспорта в CSV
}

func (e *CustomExporter) ExportToJSON(table *Table) error {
    // Ваша логика экспорта в JSON
}
```

## 📞 Поддержка

При возникновении проблем:

1. **Проверьте документацию** - README.md и USAGE.md
2. **Запустите диагностику** - `--validate` и `--list`
3. **Изучите логи** - обратите внимание на сообщения об ошибках
4. **Создайте Issue** - приложите логи и примеры данных

---

**Cronodamp Go** - эффективный инструмент для миграции данных из Cronos в современные системы! 🚀