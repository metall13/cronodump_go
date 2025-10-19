# Cronodump Go

Конвертер баз данных Cronos в ClickHouse SQL с поддержкой объединения связанных таблиц и транслитерации.

## 🚀 Возможности

- ✅ **Парсинг баз данных Cronos** - полная поддержка файлов CroStru.dat, CroBank.dat
- ✅ **Транслитерация** - автоматическая транслитерация русских имен таблиц и полей в латиницу
- ✅ **Анализ связей** - автоматическое обнаружение связей между таблицами (типы 7, 8, 9, 17)
- ✅ **Объединение таблиц** - связанные таблицы объединяются в одну с префиксами полей
- ✅ **Экспорт в SQL** - генерация SQL файлов для ClickHouse
- ✅ **Прямой импорт** - возможность прямого импорта в ClickHouse
- ✅ **Очистка** - автоматическое удаление временных файлов
- ✅ **Логирование** - подробный вывод процесса конвертации

## Установка

```bash
# Клонируем репозиторий
git clone <repository-url>
cd cronodump_go

# Устанавливаем зависимости
go mod tidy

# Собираем проект
go build -o cronodump_go
```

## Использование

### Базовое использование

```bash
./cronodump_go -input /path/to/cronos/database
```

### Полный пример с импортом в ClickHouse

```bash
./cronodump_go \
  -input /path/to/cronos/database \
  -output ./output \
  -clickhouse "tcp://localhost:9000" \
  -verbose \
  -cleanup
```

### Параметры

- `-input` - Путь к папке с базой данных Cronos (обязательный)
- `-output` - Папка для сохранения SQL файлов (по умолчанию: ./output)
- `-clickhouse` - URL подключения к ClickHouse (например: tcp://localhost:9000)
- `-verbose` - Подробный вывод процесса
- `-cleanup` - Удалить временные файлы после импорта (по умолчанию: true)
- `-batch` - Размер батча для вставки данных (по умолчанию: 1000)

## Структура проекта

```
cronodump_go/
├── main.go                 # Главный файл с точкой входа
├── config.go              # Конфигурация
├── models.go              # Модели данных
├── cronos_parser.go       # Парсер базы данных Cronos
├── transliteration.go     # Транслитерация русских символов
├── converter.go           # Основной конвертер
├── clickhouse_exporter.go # Экспорт в SQL файлы
├── clickhouse_importer.go # Импорт в ClickHouse
├── go.mod                 # Зависимости Go
└── README.md              # Документация
```

## Типы полей Cronos

| Тип | Название | ClickHouse тип |
|-----|----------|----------------|
| 0   | Системный номер | UInt32 |
| 1   | Числовое | Int32 |
| 2   | Текстовое | String |
| 3   | Словарное | String |
| 4   | Дата | Nullable(Date) |
| 5   | Время | Nullable(DateTime) |
| 6   | Файл (внутренний) | String |
| 7   | Прямая ссылка | String |
| 8   | Обратная ссылка | String |
| 9   | Прямо-обратная ссылка | String |
| 17  | Связь по полю | String |
| 29  | Внешний файл | String |

## Связи между таблицами

Конвертер автоматически анализирует связи между таблицами по типам полей 7, 8, 9, 17 и объединяет связанные таблицы в одну, добавляя поля из связанных таблиц с префиксом имени таблицы.

## Транслитерация

Все русские имена таблиц и полей автоматически транслитерируются в латиницу:
- `Пользователи` → `polzovateli`
- `Дата создания` → `data_sozdaniya`
- `Системный номер` → `system_number`

## Примеры

### Конвертация без импорта в ClickHouse

```bash
./cronodump_go -input /data/cronos_db -output ./sql_files -verbose
```

### Конвертация с импортом в ClickHouse

```bash
./cronodump_go \
  -input /data/cronos_db \
  -clickhouse "tcp://localhost:9000" \
  -verbose
```

### Только экспорт SQL файлов

```bash
./cronodump_go -input /data/cronos_db -output ./export -cleanup=false
```

## Требования

- Go 1.21+
- ClickHouse (опционально, для прямого импорта)

## Лицензия

MIT License