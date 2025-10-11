package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CronosFieldType тип поля в Cronos
type CronosFieldType int

const (
	FieldTypeString   CronosFieldType = 0
	FieldTypeInteger  CronosFieldType = 1
	FieldTypeFloat    CronosFieldType = 2
	FieldTypeDate     CronosFieldType = 3
	FieldTypeBoolean  CronosFieldType = 4
	FieldTypeMemo     CronosFieldType = 5
	FieldTypeBinary   CronosFieldType = 6
	FieldTypeUnknown  CronosFieldType = 99
)

// CronosField описание поля в таблице Cronos
type CronosField struct {
	Name        string
	Type        CronosFieldType
	Size        int
	Offset      int
	Description string
}

// CronosTable структура таблицы Cronos
type CronosTable struct {
	Name        string
	Fields      []CronosField
	Records     [][]string
	RecordCount int
	FilePath    string
}

// CronosDatabase структура базы данных Cronos
type CronosDatabase struct {
	Name    string
	Path    string
	Tables  map[string]*CronosTable
	Version string
}

// CronosParser парсер для работы с базами данных Cronos
type CronosParser struct {
	dbPath string
	logger func(format string, args ...interface{})
}

// NewCronosParser создает новый парсер Cronos
func NewCronosParser(dbPath string) *CronosParser {
	return &CronosParser{
		dbPath: dbPath,
		logger: func(format string, args ...interface{}) {
			fmt.Printf("[CronosParser] "+format+"\n", args...)
		},
	}
}

// SetLogger устанавливает функцию логирования
func (p *CronosParser) SetLogger(logger func(format string, args ...interface{})) {
	p.logger = logger
}

// ParseDatabase парсит базу данных Cronos
func (p *CronosParser) ParseDatabase() (*CronosDatabase, error) {
	db := &CronosDatabase{
		Name:   filepath.Base(p.dbPath),
		Path:   p.dbPath,
		Tables: make(map[string]*CronosTable),
	}

	// Ищем DBF файлы напрямую
	tables := make(map[string]*CronosTable)
	if err := p.findDBFFiles(tables); err != nil {
		return nil, err
	}

	// Парсим данные для каждой таблицы
	for tableName, table := range tables {
		if err := p.parseTableData(table); err != nil {
			p.logger("Предупреждение: не удалось загрузить данные для таблицы %s: %v", tableName, err)
			continue
		}
		db.Tables[tableName] = table
	}

	return db, nil
}

// findDBFFiles ищет DBF файлы в директории
func (p *CronosParser) findDBFFiles(tables map[string]*CronosTable) error {
	return filepath.Walk(p.dbPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".dbf") {
			tableName := strings.ToLower(strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())))
			
			table := &CronosTable{
				Name:     tableName,
				Fields:   []CronosField{},
				Records:  [][]string{},
				FilePath: path,
			}
			tables[tableName] = table
			p.logger("Найден DBF файл: %s", tableName)
		}
		return nil
	})
}

// parseTableData парсит данные таблицы из DBF файла
func (p *CronosParser) parseTableData(table *CronosTable) error {
	file, err := os.Open(table.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Читаем заголовок DBF
	header, err := p.readDBFHeader(file)
	if err != nil {
		return err
	}

	// Парсим описания полей
	fields, err := p.readDBFFields(file, header)
	if err != nil {
		return err
	}
	table.Fields = fields

	// Читаем записи
	records, err := p.readDBFRecords(file, header, fields)
	if err != nil {
		return err
	}
	table.Records = records
	table.RecordCount = len(records)

	p.logger("Таблица %s: %d полей, %d записей", table.Name, len(fields), len(records))
	return nil
}

// DBFHeader структура заголовка DBF файла
type DBFHeader struct {
	Version     byte
	Year        byte
	Month       byte
	Day         byte
	RecordCount uint32
	HeaderSize  uint16
	RecordSize  uint16
}

// readDBFHeader читает заголовок DBF файла
func (p *CronosParser) readDBFHeader(file *os.File) (*DBFHeader, error) {
	header := &DBFHeader{}
	
	// Читаем основные поля заголовка
	data := make([]byte, 32)
	if _, err := file.Read(data); err != nil {
		return nil, err
	}

	header.Version = data[0]
	header.Year = data[1]
	header.Month = data[2]
	header.Day = data[3]
	header.RecordCount = binary.LittleEndian.Uint32(data[4:8])
	header.HeaderSize = binary.LittleEndian.Uint16(data[8:10])
	header.RecordSize = binary.LittleEndian.Uint16(data[10:12])

	return header, nil
}

// readDBFFields читает описания полей из DBF файла
func (p *CronosParser) readDBFFields(file *os.File, header *DBFHeader) ([]CronosField, error) {
	var fields []CronosField
	
	// Перемещаемся к началу описаний полей
	if _, err := file.Seek(32, io.SeekStart); err != nil {
		return nil, err
	}

	// Читаем поля до конца заголовка
	fieldDescSize := 32
	fieldsCount := (int(header.HeaderSize) - 33) / fieldDescSize
	
	for i := 0; i < fieldsCount; i++ {
		fieldData := make([]byte, fieldDescSize)
		if _, err := file.Read(fieldData); err != nil {
			break
		}

		// Извлекаем имя поля (первые 11 байт, завершенные нулем)
		nameBytes := fieldData[0:11]
		nameEnd := bytes.IndexByte(nameBytes, 0)
		if nameEnd == -1 {
			nameEnd = 11
		}
		name := string(nameBytes[:nameEnd])

		if name == "" {
			break
		}

		// Тип поля
		fieldType := fieldData[11]
		size := int(fieldData[16])

		var cronosType CronosFieldType
		switch fieldType {
		case 'C': // Character
			cronosType = FieldTypeString
		case 'N': // Numeric
			cronosType = FieldTypeInteger
		case 'F': // Float
			cronosType = FieldTypeFloat
		case 'D': // Date
			cronosType = FieldTypeDate
		case 'L': // Logical
			cronosType = FieldTypeBoolean
		case 'M': // Memo
			cronosType = FieldTypeMemo
		default:
			cronosType = FieldTypeUnknown
		}

		field := CronosField{
			Name:   name,
			Type:   cronosType,
			Size:   size,
			Offset: 0, // Будет вычислен позже
		}

		fields = append(fields, field)
	}

	// Вычисляем offset'ы полей
	offset := 1 // Первый байт - флаг удаления записи
	for i := range fields {
		fields[i].Offset = offset
		offset += fields[i].Size
	}

	return fields, nil
}

// readDBFRecords читает записи из DBF файла
func (p *CronosParser) readDBFRecords(file *os.File, header *DBFHeader, fields []CronosField) ([][]string, error) {
	// Перемещаемся к началу данных
	if _, err := file.Seek(int64(header.HeaderSize), io.SeekStart); err != nil {
		return nil, err
	}

	var records [][]string
	recordSize := int(header.RecordSize)
	
	for i := 0; i < int(header.RecordCount); i++ {
		recordData := make([]byte, recordSize)
		if _, err := file.Read(recordData); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Проверяем флаг удаления
		if recordData[0] == '*' {
			continue // Пропускаем удаленные записи
		}

		var record []string
		for _, field := range fields {
			if field.Offset+field.Size > len(recordData) {
				record = append(record, "")
				continue
			}

			fieldData := recordData[field.Offset : field.Offset+field.Size]
			
			// Конвертируем в строку
			value := strings.TrimSpace(string(fieldData))
			record = append(record, value)
		}

		records = append(records, record)
	}

	return records, nil
}