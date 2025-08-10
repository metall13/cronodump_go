# Cronodamp Makefile

# Переменные
BINARY_NAME=cronodamp
MAIN_PATH=./cmd/cronodamp
BUILD_DIR=./build
EXAMPLE_PATH=./example

# Go параметры
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Основные цели
.PHONY: all build clean test deps example help

all: build

# Сборка основного бинарника
build:
	@echo "Собираем $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Сборка завершена: $(BUILD_DIR)/$(BINARY_NAME)"

# Сборка с оптимизацией для релиза
build-release:
	@echo "Собираем релизную версию $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Релизная сборка завершена: $(BUILD_DIR)/$(BINARY_NAME)"

# Сборка примера
example:
	@echo "Собираем пример..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/example $(EXAMPLE_PATH)
	@echo "✓ Пример собран: $(BUILD_DIR)/example"

# Установка зависимостей
deps:
	@echo "Устанавливаем зависимости..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✓ Зависимости установлены"

# Запуск тестов
test:
	@echo "Запускаем тесты..."
	$(GOTEST) -v ./...

# Очистка
clean:
	@echo "Очищаем файлы сборки..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	@echo "✓ Очистка завершена"

# Проверка кода
lint:
	@echo "Проверяем код..."
	@which golangci-lint > /dev/null || (echo "golangci-lint не установлен. Установите: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

# Форматирование кода
fmt:
	@echo "Форматируем код..."
	$(GOCMD) fmt ./...
	@echo "✓ Код отформатирован"

# Запуск с параметрами по умолчанию
run: build
	@echo "Запускаем $(BINARY_NAME) с параметрами по умолчанию..."
	$(BUILD_DIR)/$(BINARY_NAME) --list

# Запуск в режиме отладки
run-debug: build
	@echo "Запускаем $(BINARY_NAME) в режиме отладки..."
	$(BUILD_DIR)/$(BINARY_NAME) --debug --list

# Запуск примера
run-example: example
	@echo "Запускаем пример..."
	$(BUILD_DIR)/example

# Проверка подключения к ClickHouse
check-connection: build
	@echo "Проверяем подключение к ClickHouse..."
	$(BUILD_DIR)/$(BINARY_NAME) --validate

# Установка инструментов разработки
install-tools:
	@echo "Устанавливаем инструменты разработки..."
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✓ Инструменты установлены"

# Сборка для разных платформ
build-all:
	@echo "Собираем для всех платформ..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	
	# Windows amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	
	# macOS amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	
	# macOS arm64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	
	@echo "✓ Сборка для всех платформ завершена"

# Создание архивов для релиза
package: build-all
	@echo "Создаем архивы для релиза..."
	@cd $(BUILD_DIR) && \
	tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
	zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe && \
	tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
	tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@echo "✓ Архивы созданы в $(BUILD_DIR)/"

# Справка
help:
	@echo "Cronodamp - Экспортер данных из Kronos в ClickHouse"
	@echo ""
	@echo "Доступные команды:"
	@echo "  build           - Собрать основной бинарник"
	@echo "  build-release   - Собрать оптимизированную версию"
	@echo "  build-all       - Собрать для всех платформ"
	@echo "  example         - Собрать пример"
	@echo "  deps            - Установить зависимости"
	@echo "  test            - Запустить тесты"
	@echo "  clean           - Очистить файлы сборки"
	@echo "  lint            - Проверить код линтером"
	@echo "  fmt             - Отформатировать код"
	@echo "  run             - Запустить с параметрами по умолчанию"
	@echo "  run-debug       - Запустить в режиме отладки"
	@echo "  run-example     - Запустить пример"
	@echo "  check-connection - Проверить подключение к ClickHouse"
	@echo "  install-tools   - Установить инструменты разработки"
	@echo "  package         - Создать архивы для релиза"
	@echo "  help            - Показать эту справку"
	@echo ""
	@echo "Переменные окружения:"
	@echo "  KRONOS_DB_PATH     - Путь к базам Kronos (по умолчанию: /home/usersamba/smb/)"
	@echo "  CLICKHOUSE_DSN     - Адрес ClickHouse (по умолчанию: localhost:9000)"
	@echo "  CLICKHOUSE_DB      - База данных (по умолчанию: default)"
	@echo "  CLICKHOUSE_USER    - Пользователь (по умолчанию: default)"
	@echo "  CLICKHOUSE_PASSWORD - Пароль (по умолчанию: default)"
	@echo "  DEBUG              - Режим отладки (по умолчанию: false)"