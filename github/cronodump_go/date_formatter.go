package main

import (
	"time"
)

// formatDateString форматирует дату в формат дд.мм.гггг
func (e *ClickHouseExporter) formatDateString(dateStr string) string {
	if dateStr == "" || dateStr == "NULL" {
		return ""
	}
	
	// Пытаемся распарсить различные форматы дат
	formats := []string{
		"2006-01-02",     // YYYY-MM-DD
		"2006/01/02",     // YYYY/MM/DD
		"02.01.2006",     // DD.MM.YYYY (уже правильный формат)
		"02-01-2006",     // DD-MM-YYYY
		"01/02/2006",     // MM/DD/YYYY
		"2006-01-02 15:04:05", // YYYY-MM-DD HH:MM:SS
		"02.01.2006 15:04:05", // DD.MM.YYYY HH:MM:SS
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			// Форматируем в дд.мм.гггг
			return t.Format("02.01.2006")
		}
	}
	
	// Если не удалось распарсить, возвращаем как есть
	return dateStr
}