.PHONY: build run clean test deps

# Переменные
BINARY_NAME=cronodump-go
BUILD_DIR=build
TEMP_DIR=/tmp/cronodump

# Сборка приложения
build:
	@echo "Сборка приложения..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Приложение собрано: $(BUILD_DIR)/$(BINARY_NAME)"

# Запуск приложения
run: build
	@echo "Запуск приложения..."
	@mkdir -p $(TEMP_DIR)
	@$(BUILD_DIR)/$(BINARY_NAME)

# Установка зависимостей
deps:
	@echo "Установка зависимостей..."
	@go mod download
	@go mod tidy

# Очистка
clean:
	@echo "Очистка..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(TEMP_DIR)

# Тестирование
test:
	@echo "Запуск тестов..."
	@go test ./...

# Сборка для Linux
build-linux:
	@echo "Сборка для Linux..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux .
	@echo "Linux версия собрана: $(BUILD_DIR)/$(BINARY_NAME)-linux"

# Сборка для Windows
build-windows:
	@echo "Сборка для Windows..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe .
	@echo "Windows версия собрана: $(BUILD_DIR)/$(BINARY_NAME)-windows.exe"

# Сборка для macOS
build-macos:
	@echo "Сборка для macOS..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-macos .
	@echo "macOS версия собрана: $(BUILD_DIR)/$(BINARY_NAME)-macos"

# Сборка всех платформ
build-all: build-linux build-windows build-macos
	@echo "Все версии собраны"

# Установка
install: build
	@echo "Установка приложения..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Приложение установлено в /usr/local/bin/$(BINARY_NAME)"

# Создание systemd сервиса
install-service: install
	@echo "Создание systemd сервиса..."
	@sudo tee /etc/systemd/system/cronodump-go.service > /dev/null <<EOF
[Unit]
Description=Cronodump Go Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/workspace
ExecStart=/usr/local/bin/$(BINARY_NAME)
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
	@sudo systemctl daemon-reload
	@sudo systemctl enable cronodump-go
	@echo "Сервис создан и включен"

# Запуск сервиса
start-service:
	@echo "Запуск сервиса..."
	@sudo systemctl start cronodump-go
	@echo "Сервис запущен"

# Остановка сервиса
stop-service:
	@echo "Остановка сервиса..."
	@sudo systemctl stop cronodump-go
	@echo "Сервис остановлен"

# Статус сервиса
status-service:
	@sudo systemctl status cronodump-go

# Логи сервиса
logs-service:
	@sudo journalctl -u cronodump-go -f

# Помощь
help:
	@echo "Доступные команды:"
	@echo "  build          - Сборка приложения"
	@echo "  run            - Запуск приложения"
	@echo "  deps           - Установка зависимостей"
	@echo "  clean          - Очистка"
	@echo "  test           - Запуск тестов"
	@echo "  build-linux    - Сборка для Linux"
	@echo "  build-windows  - Сборка для Windows"
	@echo "  build-macos    - Сборка для macOS"
	@echo "  build-all      - Сборка для всех платформ"
	@echo "  install        - Установка приложения"
	@echo "  install-service- Создание systemd сервиса"
	@echo "  start-service  - Запуск сервиса"
	@echo "  stop-service   - Остановка сервиса"
	@echo "  status-service - Статус сервиса"
	@echo "  logs-service   - Логи сервиса"
	@echo "  help           - Показать эту справку"