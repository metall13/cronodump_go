.PHONY: build clean test run install

# Переменные
BINARY_NAME=cronodump_go
BUILD_DIR=build
VERSION=1.0.0

# Сборка
build:
	@echo "Собираем $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Сборка завершена: $(BUILD_DIR)/$(BINARY_NAME)"

# Очистка
clean:
	@echo "Очищаем файлы сборки..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Очистка завершена"

# Тестирование
test:
	@echo "Запускаем тесты..."
	@go test -v ./...

# Установка зависимостей
deps:
	@echo "Устанавливаем зависимости..."
	@go mod download
	@go mod tidy

# Запуск с примером
run:
	@echo "Запускаем $(BINARY_NAME)..."
	@go run . -input ./test_data -verbose

# Установка в систему
install: build
	@echo "Устанавливаем $(BINARY_NAME) в систему..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Установка завершена"

# Создание релиза
release: clean build
	@echo "Создаем релиз $(VERSION)..."
	@mkdir -p release
	@cp $(BUILD_DIR)/$(BINARY_NAME) release/$(BINARY_NAME)-$(VERSION)
	@echo "Релиз создан: release/$(BINARY_NAME)-$(VERSION)"

# Показать помощь
help:
	@echo "Доступные команды:"
	@echo "  build    - Собрать проект"
	@echo "  clean    - Очистить файлы сборки"
	@echo "  test     - Запустить тесты"
	@echo "  deps     - Установить зависимости"
	@echo "  run      - Запустить с примером"
	@echo "  install  - Установить в систему"
	@echo "  release  - Создать релиз"
	@echo "  help     - Показать эту справку"