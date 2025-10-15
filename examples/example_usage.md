# Примеры использования Cronodump Go

## Быстрый старт

### 1. Запуск приложения

```bash
# Сборка и запуск
make deps
make build
make run
```

### 2. Открытие веб-интерфейса

Откройте браузер и перейдите по адресу: http://localhost:9999

## Примеры конфигурации баз данных

### PostgreSQL

```json
{
  "name": "postgres_db",
  "type": "postgres",
  "host": "localhost",
  "port": 5432,
  "database": "mydb",
  "username": "postgres",
  "password": "password"
}
```

### MySQL

```json
{
  "name": "mysql_db",
  "type": "mysql",
  "host": "localhost",
  "port": 3306,
  "database": "mydb",
  "username": "root",
  "password": "password"
}
```

### ClickHouse

```json
{
  "name": "clickhouse_db",
  "type": "clickhouse",
  "host": "localhost",
  "port": 9000,
  "database": "mydb",
  "username": "default",
  "password": ""
}
```

## Примеры API запросов

### Тестирование подключения

```bash
curl -X POST http://localhost:8080/api/databases/test \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test_db",
    "type": "postgres",
    "host": "localhost",
    "port": 5432,
    "database": "testdb",
    "username": "testuser",
    "password": "testpass"
  }'
```

### Получение списка таблиц

```bash
curl -X POST http://localhost:8080/api/databases/tables \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test_db",
    "type": "postgres",
    "host": "localhost",
    "port": 5432,
    "database": "testdb",
    "username": "testuser",
    "password": "testpass"
  }'
```

### Запуск обработки нескольких баз данных

```bash
curl -X POST http://localhost:8080/api/process \
  -H "Content-Type: application/json" \
  -d '{
    "databases": [
      {
        "name": "db1",
        "type": "postgres",
        "host": "localhost",
        "port": 5432,
        "database": "mydb1",
        "username": "user1",
        "password": "pass1"
      },
      {
        "name": "db2",
        "type": "mysql",
        "host": "localhost",
        "port": 3306,
        "database": "mydb2",
        "username": "user2",
        "password": "pass2"
      }
    ],
    "temp_dir": "/tmp/cronodump"
  }'
```

### Получение статуса задачи

```bash
curl http://localhost:8080/api/jobs/job_1234567890
```

### Скачивание результата

```bash
curl http://localhost:8080/api/jobs/job_1234567890/download -o result.csv
```

## Примеры данных

### Исходные данные в PostgreSQL

```sql
-- Таблица пользователей
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    имя VARCHAR(100),
    фамилия VARCHAR(100),
    дата_рождения DATE,
    email VARCHAR(100)
);

-- Таблица заказов
CREATE TABLE заказы (
    id SERIAL PRIMARY KEY,
    пользователь_id INTEGER,
    дата_заказа TIMESTAMP,
    сумма DECIMAL(10,2)
);

-- Вставка тестовых данных
INSERT INTO users (имя, фамилия, дата_рождения, email) VALUES
('Иван', 'Иванов', '1990-05-15', 'ivan@example.com'),
('Петр', 'Петров', '1985-12-03', 'petr@example.com');

INSERT INTO заказы (пользователь_id, дата_заказа, сумма) VALUES
(1, '2023-01-15 10:30:00', 1500.00),
(2, '2023-01-16 14:20:00', 2300.50);
```

### Результат обработки

После обработки будет создан CSV файл со следующими колонками:

```csv
source_db,source_table,record_id,id,imya,familiya,data_rozhdeniya,email,polzovatel_id,data_zakaza,summa
postgres_db,users,1,1,Иван,Иванов,15.05.1990,ivan@example.com,,,
postgres_db,users,2,2,Петр,Петров,03.12.1985,petr@example.com,,,
postgres_db,заказы,3,1,,,,"",1,15.01.2023 10:30:00,1500.00
postgres_db,заказы,4,2,,,,"",2,16.01.2023 14:20:00,2300.50
```

## Особенности транслитерации

### Названия полей

- `имя` → `imya`
- `фамилия` → `familiya`
- `дата_рождения` → `data_rozhdeniya`
- `пользователь_id` → `polzovatel_id`
- `дата_заказа` → `data_zakaza`

### Названия таблиц

- `users` → `users` (остается без изменений)
- `заказы` → `zakazy`

## Docker Compose пример

```yaml
version: '3.8'

services:
  cronodump-go:
    build: .
    ports:
      - "8080:8080"
    environment:
      - LOG_LEVEL=info
      - TEMP_DIR=/tmp/cronodump
    volumes:
      - ./temp:/tmp/cronodump
    depends_on:
      - postgres
      - mysql

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=testdb
      - POSTGRES_USER=testuser
      - POSTGRES_PASSWORD=testpass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=rootpass
      - MYSQL_DATABASE=testdb
      - MYSQL_USER=testuser
      - MYSQL_PASSWORD=testpass
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql

volumes:
  postgres_data:
  mysql_data:
```

## Мониторинг и логи

### Просмотр логов

```bash
# Логи приложения
make logs-service

# Логи Docker контейнера
docker logs cronodump-go

# Логи с фильтрацией
docker logs cronodump-go 2>&1 | grep ERROR
```

### Мониторинг через API

```bash
# Список всех задач
curl http://localhost:8080/api/jobs

# Статус конкретной задачи
curl http://localhost:8080/api/jobs/job_1234567890
```

## Производительность

### Рекомендации

1. **Размер файлов**: Установите `MAX_FILE_SIZE` для ограничения размера выходных файлов
2. **Память**: Для больших баз данных увеличьте лимиты памяти
3. **Сеть**: Используйте локальные подключения для лучшей производительности
4. **Диск**: Используйте SSD для временных файлов

### Настройка производительности

```bash
# Увеличение лимита файлов
export MAX_FILE_SIZE=1073741824  # 1GB

# Увеличение лимита памяти (для Docker)
docker run -m 2g cronodump-go

# Использование tmpfs для временных файлов
docker run -v /tmp/cronodump:/tmp/cronodump:tmpfs cronodump-go
```

## Устранение неполадок

### Частые проблемы

1. **Ошибка подключения к базе данных**
   - Проверьте правильность параметров подключения
   - Убедитесь, что база данных доступна
   - Проверьте права пользователя

2. **Ошибка транслитерации**
   - Проверьте кодировку данных в базе
   - Убедитесь, что используются корректные символы

3. **Ошибка записи файла**
   - Проверьте права на запись в временную директорию
   - Убедитесь, что достаточно места на диске

### Отладка

```bash
# Включение отладочных логов
export LOG_LEVEL=debug

# Проверка статуса сервиса
make status-service

# Перезапуск сервиса
make stop-service
make start-service
```