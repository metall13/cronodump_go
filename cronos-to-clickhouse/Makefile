.PHONY: build clean install test

# Сборка приложения
build:
	go mod tidy
	go build -o cronos-converter

# Очистка
clean:
	rm -f cronos-converter
	rm -f cronos_export_*.sql

# Установка зависимостей
install:
	go mod download

# Тест на примере данных
test: build
	./cronos-converter -v -db ../cronodump/test_data/all_field_types

# Запуск с базами по умолчанию
run: build
	./cronos-converter -v

# Создание дистрибутива
dist: build
	tar -czf cronos-to-clickhouse.tar.gz cronos-converter *.go go.mod go.sum README.md config.json Makefile