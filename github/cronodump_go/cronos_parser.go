package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CronosParser парсит файлы базы данных Cronos
type CronosParser struct {
	verbose bool
}

// NewCronosParser создает новый парсер
func NewCronosParser(verbose bool) *CronosParser {
	return &CronosParser{
		verbose: verbose,
	}
}

// ParseDatabase парсит базу данных Cronos из указанной папки
func (p *CronosParser) ParseDatabase(dbPath string) (*CronosDatabase, error) {
	if p.verbose {
		fmt.Printf("Парсинг базы данных: %s\n", dbPath)
	}

	db := &CronosDatabase{
		Path:   dbPath,
		Tables: []CronosTable{},
	}

	// Парсим основные файлы базы данных
	struData, err := p.parseStruFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга CroStru: %v", err)
	}

	// Извлекаем определения таблиц
	tableDefs, err := p.extractTableDefinitions(struData)
	if err != nil {
		return nil, fmt.Errorf("ошибка извлечения определений таблиц: %v", err)
	}

	// Парсим данные из CroBank
	bankData, err := p.parseBankFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга CroBank: %v", err)
	}

	// Создаем таблицы
	for _, tableDef := range tableDefs {
		table, err := p.createTable(tableDef, bankData)
		if err != nil {
			if p.verbose {
				fmt.Printf("Предупреждение: ошибка создания таблицы %s: %v\n", tableDef.Name, err)
			}
			continue
		}
		db.Tables = append(db.Tables, *table)
	}

	if p.verbose {
		fmt.Printf("Найдено таблиц: %d\n", len(db.Tables))
	}

	return db, nil
}

// parseStruFile парсит файл CroStru.dat
func (p *CronosParser) parseStruFile(dbPath string) ([]byte, error) {
	struPath := filepath.Join(dbPath, "CroStru.dat")
	if _, err := os.Stat(struPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("файл CroStru.dat не найден")
	}

	data, err := os.ReadFile(struPath)
	if err != nil {
		return nil, err
	}

	// Проверяем заголовок файла
	if len(data) < 19 {
		return nil, fmt.Errorf("файл CroStru.dat слишком короткий")
	}

	magic := string(data[0:8])
	if magic != "CroFile\x00" {
		return nil, fmt.Errorf("неверный формат файла CroStru.dat")
	}

	// Пропускаем заголовок и случайные данные
	headerSize := 19 + 0xE9 // 19 байт заголовка + 0xE9 случайных байт
	if len(data) <= headerSize {
		return nil, fmt.Errorf("файл CroStru.dat не содержит данных")
	}

	return data[headerSize:], nil
}

// parseBankFile парсит файл CroBank.dat
func (p *CronosParser) parseBankFile(dbPath string) ([]byte, error) {
	bankPath := filepath.Join(dbPath, "CroBank.dat")
	if _, err := os.Stat(bankPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("файл CroBank.dat не найден")
	}

	data, err := os.ReadFile(bankPath)
	if err != nil {
		return nil, err
	}

	// Проверяем заголовок файла
	if len(data) < 19 {
		return nil, fmt.Errorf("файл CroBank.dat слишком короткий")
	}

	magic := string(data[0:8])
	if magic != "CroFile\x00" {
		return nil, fmt.Errorf("неверный формат файла CroBank.dat")
	}

	// Пропускаем заголовок и случайные данные
	headerSize := 19 + 0xE9
	if len(data) <= headerSize {
		return nil, fmt.Errorf("файл CroBank.dat не содержит данных")
	}

	return data[headerSize:], nil
}

// extractTableDefinitions извлекает определения таблиц из данных CroStru
func (p *CronosParser) extractTableDefinitions(struData []byte) ([]*CronosTable, error) {
	var tables []*CronosTable

	// Упрощенная версия - создаем базовую структуру
	// В реальной реализации здесь должен быть полный парсинг CroStru
	
	// Создаем пример таблицы для демонстрации
	table := &CronosTable{
		Name:       "example_table",
		FolderName: "ПримерТаблицы",
		TableID:    1,
		Fields: []CronosField{
			{
				Name:         "system_number",
				OriginalName: "Системный номер",
				Type:         FieldTypeSystemNumber,
				IsPrimary:    true,
			},
			{
				Name:         "name",
				OriginalName: "Наименование",
				Type:         FieldTypeText,
				Length:       255,
			},
			{
				Name:         "date_created",
				OriginalName: "Дата создания",
				Type:         FieldTypeDate,
			},
		},
		Records: []CronosRecord{},
	}

	tables = append(tables, table)

	return tables, nil
}

// createTable создает таблицу на основе определения и данных
func (p *CronosParser) createTable(tableDef *CronosTable, bankData []byte) (*CronosTable, error) {
	// В реальной реализации здесь должен быть парсинг записей из CroBank
	// Пока создаем пустую таблицу с примерами записей
	
	table := *tableDef
	table.Records = []CronosRecord{
		{
			SystemNumber: 1,
			Fields: []CronosFieldValue{
				{Field: table.Fields[0], Value: uint32(1)},
				{Field: table.Fields[1], Value: "Пример записи 1"},
				{Field: table.Fields[2], Value: "2025-01-01"},
			},
		},
		{
			SystemNumber: 2,
			Fields: []CronosFieldValue{
				{Field: table.Fields[0], Value: uint32(2)},
				{Field: table.Fields[1], Value: "Пример записи 2"},
				{Field: table.Fields[2], Value: "2025-01-02"},
			},
		},
	}

	return &table, nil
}

// readString читает строку из данных (формат Cronos: длина + данные)
func (p *CronosParser) readString(data []byte, offset int) (string, int, error) {
	if offset >= len(data) {
		return "", offset, io.EOF
	}

	length := int(data[offset])
	offset++

	if offset+length > len(data) {
		return "", offset, fmt.Errorf("недостаточно данных для чтения строки")
	}

	str := string(data[offset : offset+length])
	offset += length

	return str, offset, nil
}

// readUint32 читает 32-битное беззнаковое число
func (p *CronosParser) readUint32(data []byte, offset int) (uint32, int, error) {
	if offset+4 > len(data) {
		return 0, offset, fmt.Errorf("недостаточно данных для чтения uint32")
	}

	value := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	return value, offset, nil
}

// readUint16 читает 16-битное беззнаковое число
func (p *CronosParser) readUint16(data []byte, offset int) (uint16, int, error) {
	if offset+2 > len(data) {
		return 0, offset, fmt.Errorf("недостаточно данных для чтения uint16")
	}

	value := binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2

	return value, offset, nil
}

// readByte читает байт
func (p *CronosParser) readByte(data []byte, offset int) (byte, int, error) {
	if offset >= len(data) {
		return 0, offset, io.EOF
	}

	value := data[offset]
	offset++

	return value, offset, nil
}