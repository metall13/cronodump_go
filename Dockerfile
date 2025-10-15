# Многоэтапная сборка
FROM golang:1.21-alpine AS builder

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git ca-certificates tzdata

# Создаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cronodump-go .

# Финальный образ
FROM alpine:latest

# Устанавливаем необходимые пакеты
RUN apk --no-cache add ca-certificates tzdata

# Создаем пользователя для безопасности
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Создаем рабочую директорию
WORKDIR /app

# Копируем бинарный файл из builder
COPY --from=builder /app/cronodump-go .

# Копируем веб-файлы
COPY --from=builder /app/web ./web

# Создаем директорию для временных файлов
RUN mkdir -p /tmp/cronodump && \
    chown -R appuser:appgroup /tmp/cronodump && \
    chown -R appuser:appgroup /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт
EXPOSE 9999

# Устанавливаем переменные окружения по умолчанию
ENV LOG_LEVEL=info
ENV TEMP_DIR=/tmp/cronodump
ENV MAX_FILE_SIZE=104857600

# Запускаем приложение
CMD ["./cronodump-go"]