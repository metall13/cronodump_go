package main

import (
	"fmt"
)

func testTransliteration() {
	fmt.Println("=== Тестирование модуля транслитерации ===")
	
	testCases := []struct {
		input    string
		expected string
		testType string
	}{
		{"Пользователи", "polzovateli", "table"},
		{"ЗАКАЗЫ", "zakazy", "table"},
		{"Номер_телефона", "nomer_telefona", "column"},
		{"Дата создания", "data_sozdaniya", "column"},
		{"select", "col_select", "column"}, // зарезервированное слово
		{"123_test", "col_123_test", "column"}, // начинается с цифры
		{"Многоэтажка №1", "mnogoetazhka_1", "table"},
		{"Фамилия И.О.", "familiya_i_o", "column"},
	}
	
	for _, tc := range testCases {
		var result string
		if tc.testType == "table" {
			result = TransliterateTableName(tc.input)
		} else {
			result = TransliterateColumnName(tc.input)
		}
		
		status := "✓"
		if result != tc.expected {
			status = "✗"
		}
		
		fmt.Printf("%s %s (%s): '%s' -> '%s' (ожидалось: '%s')\n", 
			status, tc.testType, tc.input, tc.input, result, tc.expected)
	}
	
	fmt.Println()
}

func testValidation() {
	fmt.Println("=== Тестирование валидации имен ===")
	
	testCases := []struct {
		input    string
		expected bool
	}{
		{"valid_name", true},
		{"123invalid", false},
		{"_valid", true},
		{"", false},
		{"name with spaces", false},
		{"name-with-dash", false},
		{"UPPERCASE", true},
		{"mixed_Case_123", true},
	}
	
	for _, tc := range testCases {
		result := ValidateName(tc.input)
		status := "✓"
		if result != tc.expected {
			status = "✗"
		}
		
		fmt.Printf("%s ValidateName('%s') = %v (ожидалось: %v)\n", 
			status, tc.input, result, tc.expected)
	}
	
	fmt.Println()
}

func main() {
	fmt.Println("Тестирование функций cronodump\n")
	
	testTransliteration()
	testValidation()
	
	fmt.Println("=== Тестирование конфигурации ===")
	
	config, err := LoadConfig("config.json")
	if err != nil {
		fmt.Printf("✗ Ошибка загрузки конфигурации: %v\n", err)
		return
	}
	
	fmt.Printf("✓ Конфигурация загружена успешно\n")
	fmt.Printf("  - База данных: %s\n", config.DatabasePath)
	fmt.Printf("  - Вывод: %s\n", config.OutputPath)
	fmt.Printf("  - ClickHouse: %s:%d\n", config.ClickHouseHost, config.ClickHousePort)
	fmt.Printf("  - Транслитерация: %v\n", config.TransliterateNames)
	
	if err := config.Validate(); err != nil {
		fmt.Printf("✗ Валидация конфигурации: %v\n", err)
	} else {
		fmt.Printf("✓ Конфигурация прошла валидацию\n")
	}
	
	fmt.Println("\n=== Симуляция работы с базой данных ===")
	fmt.Println("Примечание: для полного тестирования необходим Firebird сервер и файл базы данных")
	fmt.Println("Текущая реализация поддерживает:")
	fmt.Println("  - Подключение к Firebird через go-firebirdsql")
	fmt.Println("  - Экспорт в CSV файлы")
	fmt.Println("  - Экспорт в ClickHouse")
	fmt.Println("  - Транслитерацию русских названий")
	fmt.Println("  - Пакетную обработку данных")
}