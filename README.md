# Cronodump Go

Переписанная на Go версия библиотеки cronodump с веб-интерфейсом на JavaScript для обработки и объединения баз данных с транслитерацией для ClickHouse.

## Возможности

- 🔄 **Обработка множественных баз данных** - PostgreSQL, MySQL, ClickHouse
- 🔤 **Автоматическая транслитерация** - русские названия полей и таблиц переводятся в транслит
- 📊 **Объединение таблиц** - все таблицы объединяются в одну по ключевым полям
- 📅 **Форматирование дат** - даты рождения в формате ДД.ММ.ГГГГ
- 📝 **Конвертация данных** - все поля конвертируются в строки
- 🌐 **Веб-интерфейс** - современный интерфейс с real-time обновлениями
- 📈 **Мониторинг прогресса** - отображение статистики обработки
- 📋 **Логирование** - подробные логи процесса
- 💾 **Временные файлы** - результаты сохраняются во временные файлы

## Установка

### Требования

- Go 1.21 или выше
- Поддерживаемые базы данных: PostgreSQL, MySQL, ClickHouse

### Сборка

```bash
# Клонирование репозитория
git clone <repository-url>
cd cronodump-go

# Установка зависимостей
make deps

# Сборка приложения
make build

# Запуск
make run
```

### Установка как сервис

```bash
# Установка приложения
make install

# Создание systemd сервиса
make install-service

# Запуск сервиса
make start-service
```

## Использование

### Веб-интерфейс

1. Откройте браузер и перейдите по адресу `http://localhost:9999`
2. Добавьте базы данных для обработки
3. Нажмите "Начать обработку"
4. Следите за прогрессом в реальном времени
5. Скачайте результат после завершения

### API

#### Тестирование подключения к базе данных

```bash
curl -X POST http://localhost:9999/api/databases/test \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test_db",
    "type": "postgres",
    "host": "localhost",
    "port": 5432,
    "database": "mydb",
    "username": "user",
    "password": "password"
  }'
```

#### Получение списка таблиц

```bash
curl -X POST http://localhost:9999/api/databases/tables \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test_db",
    "type": "postgres",
    "host": "localhost",
    "port": 5432,
    "database": "mydb",
    "username": "user",
    "password": "password"
  }'
```

#### Запуск обработки

```bash
curl -X POST http://localhost:9999/api/process \
  -H "Content-Type: application/json" \
  -d '{
    "databases": [
      {
        "name": "db1",
        "type": "postgres",
        "host": "localhost",
        "port": 5432,
        "database": "mydb1",
        "username": "user",
        "password": "password"
      }
    ],
    "temp_dir": "/tmp/cronodump"
  }'
```

#### Получение статуса задачи

```bash
curl http://localhost:9999/api/jobs/{jobId}
```

#### Скачивание результата

```bash
curl http://localhost:9999/api/jobs/{jobId}/download -o result.csv
```

## Конфигурация

Настройки можно изменить через переменные окружения:

```bash
# Уровень логирования
export LOG_LEVEL=info

# Временная директория
export TEMP_DIR=/tmp/cronodump

# Максимальный размер файла (в байтах)
export MAX_FILE_SIZE=104857600

# Настройки ClickHouse
export CLICKHOUSE_HOST=localhost
export CLICKHOUSE_PORT=9000
export CLICKHOUSE_DATABASE=cronodump
export CLICKHOUSE_USERNAME=default
export CLICKHOUSE_PASSWORD=

# Настройки базы данных по умолчанию
export DB_HOST=localhost
export DB_PORT=5432
export DB_DATABASE=cronodump
export DB_USERNAME=postgres
export DB_PASSWORD=
export DB_DRIVER=postgres
```

## Архитектура

### Backend (Go)

- **main.go** - точка входа приложения
- **internal/config** - конфигурация
- **internal/logger** - логирование
- **internal/database** - работа с базами данных
- **internal/transliteration** - транслитерация русских названий
- **internal/processor** - обработка данных
- **internal/api** - REST API и WebSocket

### Frontend (JavaScript)

- **web/index.html** - главная страница
- **web/styles.css** - стили
- **web/app.js** - логика приложения

## Особенности реализации

### Транслитерация

Все русские названия полей и таблиц автоматически переводятся в транслит с соблюдением правил для ClickHouse:

- Кириллические символы заменяются на латинские
- Пробелы заменяются на подчеркивания
- Удаляются специальные символы
- Если название начинается с цифры, добавляется префикс

### Форматирование данных

- Все поля конвертируются в строки
- Даты форматируются в ДД.ММ.ГГГГ
- NULL значения заменяются на пустые строки

### Объединение таблиц

- Все таблицы объединяются в одну CSV таблицу
- Добавляются колонки: source_db, source_table, record_id
- Колонки с одинаковыми транслитерированными именами объединяются

## Разработка

### Структура проекта

```
cronodump-go/
├── main.go
├── go.mod
├── Makefile
├── README.md
├── internal/
│   ├── api/
│   ├── config/
│   ├── database/
│   ├── logger/
│   ├── processor/
│   └── transliteration/
└── web/
    ├── index.html
    ├── styles.css
    └── app.js
```

### Запуск в режиме разработки

```bash
# Установка зависимостей
make deps

# Запуск приложения
make run
```

### Тестирование

```bash
make test
```

## Лицензия

MIT License

## Вклад в проект

1. Форкните репозиторий
2. Создайте ветку для новой функции
3. Внесите изменения
4. Добавьте тесты
5. Создайте Pull Request

## Поддержка

Если у вас возникли вопросы или проблемы, создайте Issue в репозитории.