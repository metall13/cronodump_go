package main

import (
	"testing"
)

func TestTransliteration(t *testing.T) {
	transliterator := NewTransliterator()

	tests := []struct {
		input    string
		expected string
	}{
		{"Пользователи", "polzovateli"},
		{"Дата создания", "data_sozdaniya"},
		{"Системный номер", "sistemnyi_nomer"},
		{"Тестовая таблица", "testovaya_tablitsa"},
		{"", ""},
		{"123", "123"},
		{"test", "test"},
	}

	for _, test := range tests {
		result := transliterator.Transliterate(test.input)
		if result != test.expected {
			t.Errorf("Transliterate(%q) = %q, ожидалось %q", test.input, result, test.expected)
		}
	}
}

func TestTableNameTransliteration(t *testing.T) {
	transliterator := NewTransliterator()

	tests := []struct {
		input    string
		expected string
	}{
		{"Пользователи", "polzovateli"},
		{"Справочники", "spravochniki"},
		{"Документы", "dokumenty"},
	}

	for _, test := range tests {
		result := transliterator.TransliterateTableName(test.input)
		if result != test.expected {
			t.Errorf("TransliterateTableName(%q) = %q, ожидалось %q", test.input, result, test.expected)
		}
	}
}

func TestFieldNameTransliteration(t *testing.T) {
	transliterator := NewTransliterator()

	tests := []struct {
		input    string
		expected string
	}{
		{"Наименование", "naimenovanie"},
		{"Дата создания", "data_sozdaniya"},
		{"Количество", "kolichestvo"},
	}

	for _, test := range tests {
		result := transliterator.TransliterateFieldName(test.input)
		if result != test.expected {
			t.Errorf("TransliterateFieldName(%q) = %q, ожидалось %q", test.input, result, test.expected)
		}
	}
}

func TestConfig(t *testing.T) {
	config := &Config{
		InputDir:   "/test/input",
		OutputDir:  "/test/output",
		ClickHouse: "tcp://localhost:9000",
		BatchSize:  1000,
		Cleanup:    true,
	}

	if config.InputDir != "/test/input" {
		t.Errorf("InputDir = %q, ожидалось %q", config.InputDir, "/test/input")
	}

	if config.OutputDir != "/test/output" {
		t.Errorf("OutputDir = %q, ожидалось %q", config.OutputDir, "/test/output")
	}

	if config.ClickHouse != "tcp://localhost:9000" {
		t.Errorf("ClickHouse = %q, ожидалось %q", config.ClickHouse, "tcp://localhost:9000")
	}

	if config.BatchSize != 1000 {
		t.Errorf("BatchSize = %d, ожидалось %d", config.BatchSize, 1000)
	}

	if !config.Cleanup {
		t.Errorf("Cleanup = %v, ожидалось %v", config.Cleanup, true)
	}
}