#!/bin/bash

# Пример скрипта для запуска конвертера Cronos -> ClickHouse
# Автор: AI Assistant
# Версия: 1.0

echo "=== Настройка переменных окружения ==="

# Обязательные переменные
export CRONOS_DB_PATH="/home/usersamba/smb/"
export CLICKHOUSE_DSN="localhost:9000"

# Дополнительные настройки (опционально)
export CLICKHOUSE_DATABASE="cronos_data"
export CLICKHOUSE_USER="default" 
export CLICKHOUSE_PASSWORD=""
export BATCH_SIZE="1000"
export VERBOSE="true"

echo "✅ Переменные окружения установлены:"
echo "   CRONOS_DB_PATH: $CRONOS_DB_PATH"
echo "   CLICKHOUSE_DSN: $CLICKHOUSE_DSN"
echo "   CLICKHOUSE_DATABASE: $CLICKHOUSE_DATABASE"
echo "   VERBOSE: $VERBOSE"
echo ""

echo "=== Проверка файлов ==="

# Проверяем существование исполняемого файла
if [ ! -f "./cronos-converter" ]; then
    echo "❌ Исполняемый файл cronos-converter не найден"
    echo "   Выполните: go build -o cronos-converter"
    exit 1
fi

# Проверяем путь к базам Cronos
if [ ! -d "$CRONOS_DB_PATH" ]; then
    echo "⚠️  Предупреждение: Путь к базам Cronos не найден: $CRONOS_DB_PATH"
    echo "   Проверьте правильность пути в переменной CRONOS_DB_PATH"
fi

echo "✅ Предварительные проверки пройдены"
echo ""

echo "=== Запуск конвертера ==="
echo "Нажмите Enter для продолжения или Ctrl+C для отмены..."
read

# Запускаем программу
./cronos-converter

echo ""
echo "=== Завершение работы ==="
echo "Для повторного запуска используйте: $0"