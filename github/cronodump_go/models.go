package main

import (
	"time"
)

// CronosDatabase представляет базу данных Cronos
type CronosDatabase struct {
	Path   string
	Tables []CronosTable
}

// CronosTable представляет таблицу в базе данных Cronos
type CronosTable struct {
	Name       string          // Имя таблицы (транслитерированное)
	FolderName string          // Оригинальное имя папки
	TableID    uint32          // ID таблицы в Cronos
	Fields     []CronosField   // Поля таблицы
	Records    []CronosRecord  // Записи таблицы
	Relations  []TableRelation // Связи с другими таблицами
}

// CronosField представляет поле таблицы
type CronosField struct {
	Name        string // Имя поля (транслитерированное)
	OriginalName string // Оригинальное имя поля
	Type        int    // Тип поля (0-9, 17, 29)
	Length      int    // Длина поля
	Index1      uint32 // Индекс отображения
	Index2      uint32 // Индекс сериализации
	Flags       uint32 // Флаги поля
	IsPrimary   bool   // Является ли первичным ключом
}

// CronosRecord представляет запись в таблице
type CronosRecord struct {
	SystemNumber uint32              // Системный номер (первичный ключ)
	Fields       []CronosFieldValue  // Значения полей
}

// CronosFieldValue представляет значение поля в записи
type CronosFieldValue struct {
	Field CronosField
	Value interface{}
}

// TableRelation представляет связь между таблицами
type TableRelation struct {
	SourceTable    string // Имя исходной таблицы
	TargetTable    string // Имя целевой таблицы
	SourceField    string // Поле в исходной таблице
	TargetField    string // Поле в целевой таблице
	RelationType   int    // Тип связи (7=прямая, 8=обратная, 9=прямо-обратная, 17=по полю)
	IsRequired     bool   // Обязательная ли связь
}

// MergedTable представляет объединенную таблицу
type MergedTable struct {
	Name        string
	SourceTable string
	Fields      []CronosField
	Records     []CronosRecord
	JoinedData  map[uint32]map[string]interface{} // Данные из связанных таблиц
}

// FieldType константы типов полей Cronos
const (
	FieldTypeSystemNumber = 0  // Системный номер (первичный ключ)
	FieldTypeNumeric      = 1  // Числовое
	FieldTypeText         = 2  // Текстовое
	FieldTypeDictionary   = 3  // Словарное
	FieldTypeDate         = 4  // Дата
	FieldTypeTime         = 5  // Время
	FieldTypeFile         = 6  // Файл (внутренний)
	FieldTypeDirectLink   = 7  // Прямая ссылка
	FieldTypeBackLink     = 8  // Обратная ссылка
	FieldTypeDirectBack   = 9  // Прямо-обратная ссылка
	FieldTypeFieldLink    = 17 // Связь по полю
	FieldTypeExternalFile = 29 // Внешний файл
)

// FieldFlags константы флагов полей
const (
	FlagMultiple     = 0x2000 // Множественное
	FlagInformative  = 0x0800 // Информативное
	FlagUncorrectable = 0x0040 // Некорректируемое
	FlagInputSearch  = 0x1000 // Поиск на вводе
	FlagReplaceNonEmpty = 0x0200 // Замена непустого значения
	FlagReplaceValue = 0x0100 // Замена значения
	FlagAutocomplete = 0x0004 // Автозаполнение
	FlagObligatory   = 0x0002 // Обязательное
)

// ParsedData представляет распарсенные данные из файлов Cronos
type ParsedData struct {
	DatabaseInfo map[string]interface{}
	TableDefs    map[string]*CronosTable
	Records      map[string][]CronosRecord
	Files        map[string][]byte
}

// ConversionStats содержит статистику конвертации
type ConversionStats struct {
	StartTime       time.Time
	EndTime         time.Time
	TablesProcessed int
	RecordsProcessed int
	FilesProcessed  int
	Errors          []string
}