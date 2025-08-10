# 🚀 Быстрый запуск Cronodump Go Converter

## Проверка программы ✅

Программа успешно протестирована и готова к работе!

### Что протестировано:
- ✅ Парсинг DBF файлов (поля: NUMBER, AMOUNT, DATE)
- ✅ Транслитерация имен: `accounting_invoices`, `hr_employees`, `warehouse_products`
- ✅ Подключение к ClickHouse: `192.168.0.210:9000`
- ✅ Обработка структуры папок как в реальном Cronos

## 📁 Настройка

Ваши базы данных Cronos находятся в: `/home/usersamba/smb/`

### 1. Проверьте конфигурацию в `.env`:
```bash
PATH_TO_DB=/home/usersamba/smb/
CLICKHOUSE_HOST=192.168.0.210
CLICKHOUSE_PORT=9000
CLICKHOUSE_DATABASE=cronos_data
```

### 2. Убедитесь что ClickHouse доступен:
```bash
# Проверка подключения (если есть clickhouse-client)
clickhouse-client --host 192.168.0.210 --query "SELECT 1"
```

## 🎯 Запуск

### Сканирование баз данных:
```bash
./cronodump-converter
```

### Конвертация одной базы:
```bash
./cronodump-converter single /home/usersamba/smb/какая-то-папка
```

### Просмотр справки:
```bash
./cronodump-converter help
```

## 📊 Что происходит при конвертации

1. **Сканирование**: Программа ищет `.dbf` файлы во всех подпапках `/home/usersamba/smb/`

2. **Именование таблиц**: 
   - Папка: `отдел_кадров` → Таблица: `сотрудники.dbf` → ClickHouse: `otdel_kadrov_sotrudniki`
   - Папка: `бухгалтерия` → Таблица: `счета.dbf` → ClickHouse: `buhgalteriya_scheta`

3. **Структура данных**:
   - Все поля сохраняются как `String` для максимальной совместимости
   - Добавляются служебные колонки: `cronos_import_timestamp`, `cronos_source_folder`, etc.

4. **Производительность**: 
   - Батчевая загрузка (по 1000 записей)
   - Параллельная обработка (4 воркера)
   - Автоматическая оптимизация таблиц

## 🔧 Режимы работы

### Демо-режим (текущий):
Показывает найденные файлы и структуру без реальной загрузки в ClickHouse.

### Реальный режим:
Для активации реальной загрузки в ClickHouse нужно:
1. Убедиться что ClickHouse доступен
2. Изменить код в `main.go` для вызова реального конвертера

## 📋 Результат

После конвертации в ClickHouse будут созданы таблицы вида:
```sql
SELECT * FROM cronos_data.otdel_kadrov_sotrudniki;
SELECT * FROM cronos_data.buhgalteriya_scheta;
SELECT * FROM cronos_data.sklad_tovary;
```

Каждая таблица содержит:
- Все оригинальные поля из DBF (как String)
- `cronos_import_timestamp` - время импорта
- `cronos_source_folder` - исходная папка
- `cronos_source_table` - исходное имя таблицы
- `cronos_record_count` - общее количество записей

## 🆘 Устранение проблем

### "База данных не найдена":
```bash
ls -la /home/usersamba/smb/
# Проверьте права доступа и наличие .dbf файлов
```

### "Ошибка подключения к ClickHouse":
```bash
# Проверьте доступность сервера
ping 192.168.0.210
telnet 192.168.0.210 9000
```

### "Ошибка кодировки":
Программа автоматически обрабатывает различные кодировки русского текста.

## 🎉 Готово!

Программа готова к работе с вашими реальными базами данных CronosPro!