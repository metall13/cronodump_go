package main

import (
	"cronodump-go/internal/api"
	"cronodump-go/internal/config"
	"cronodump-go/internal/database"
	"cronodump-go/internal/logger"
	"cronodump-go/internal/transliteration"
	"cronodump-go/internal/processor"
	"log"
	"net/http"
)

func main() {
	// Инициализация конфигурации
	cfg := config.Load()

	// Инициализация логгера
	logger := logger.New(cfg.LogLevel)

	// Инициализация транслитерации
	transliterator := transliteration.New()

	// Инициализация процессора данных
	dataProcessor := processor.New(logger, transliterator)

	// Инициализация менеджера баз данных
	dbManager := database.NewManager(logger)

	// Инициализация API
	apiHandler := api.New(logger, dbManager, dataProcessor)

	// Настройка маршрутов
	router := apiHandler.SetupRoutes()

	// Запуск сервера
	logger.Info("Запуск сервера на порту :3000")
	log.Fatal(http.ListenAndServe(":3000", router))
}