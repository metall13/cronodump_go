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

// CronosField представляет поле в таблице Cronos
type CronosField struct {
	Name     string
	Type     string
	Length   int
	Position int
}

// CronosTable представляет таблицу базы данных Cronos
type CronosTable struct {
	Name    string
	Fields  []CronosField
	Records [][]string
}

// CronosDatabase представляет базу данных Cronos
type CronosDatabase struct {
	Path    string
	Name    string
	Tables  []CronosTable
}

// CronosParser парсер для баз данных Cronos
type CronosParser struct {
	logger func(format string, args ...interface{})
}

// NewCronosParser создает новый парсер Cronos
func NewCronosParser() *CronosParser {
	return &CronosParser{
		logger: func(format string, args ...interface{}) {
			fmt.Printf("[CronosParser] "+format+"\n", args...)
		},
	}
}

// SetLogger устанавливает функцию логирования
func (p *CronosParser) SetLogger(logger func(format string, args ...interface{})) {
	p.logger = logger
}

// ParseDatabase парсит базу данных Cronos по указанному пути
func (p *CronosParser) ParseDatabase(dbPath string) (*CronosDatabase, error) {
	p.logger("Начинаем парсинг базы данных: %s", dbPath)
	
	// Проверяем существование пути
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("путь к базе данных не существует: %s", dbPath)
	}
	
	// Получаем имя базы данных из пути
	dbName := filepath.Base(dbPath)
	
	database := &CronosDatabase{
		Path:   dbPath,
		Name:   dbName,
		Tables: []CronosTable{},
	}
	
	// Ищем файлы структуры (.dat файлы)
	tables, err := p.findTables(dbPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска таблиц: %w", err)
	}
	
	p.logger("Найдено таблиц: %d", len(tables))
	
	// Парсим каждую таблицу
	for _, tablePath := range tables {
		table, err := p.parseTable(tablePath)
		if err != nil {
			p.logger("Ошибка парсинга таблицы %s: %v", tablePath, err)
			continue
		}
		
		if table != nil {
			database.Tables = append(database.Tables, *table)
		}
	}
	
	p.logger("Успешно распарсено таблиц: %d", len(database.Tables))
	return database, nil
}

// findTables находит все таблицы в базе данных
func (p *CronosParser) findTables(dbPath string) ([]string, error) {
	var tables []string
	
	err := filepath.Walk(dbPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Ищем файлы с расширением .dat (структура таблицы)
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".dat" {
			// Исключаем системные файлы
			fileName := strings.ToLower(info.Name())
			if !strings.Contains(fileName, "crostru") && 
			   !strings.Contains(fileName, "system") &&
			   !strings.Contains(fileName, "index") {
				tables = append(tables, path)
			}
		}
		
		return nil
	})
	
	return tables, err
}

// parseTable парсит отдельную таблицу
func (p *CronosParser) parseTable(tablePath string) (*CronosTable, error) {
	p.logger("Парсинг таблицы: %s", tablePath)
	
	// Определяем имя таблицы из пути файла
	fileName := filepath.Base(tablePath)
	tableName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	
	table := &CronosTable{
		Name:    tableName,
		Fields:  []CronosField{},
		Records: [][]string{},
	}
	
	// Пытаемся найти файл структуры
	structFile := p.findStructureFile(tablePath)
	if structFile != "" {
		fields, err := p.parseStructure(structFile)
		if err != nil {
			p.logger("Ошибка парсинга структуры %s: %v", structFile, err)
		} else {
			table.Fields = fields
		}
	}
	
	// Если не удалось получить структуру, создаем базовую
	if len(table.Fields) == 0 {
		p.logger("Создаем базовую структуру для таблицы %s", tableName)
		table.Fields = p.createDefaultFields()
	}
	
	// Парсим данные
	records, err := p.parseData(tablePath, table.Fields)
	if err != nil {
		p.logger("Ошибка парсинга данных %s: %v", tablePath, err)
		// Возвращаем таблицу даже если данные не удалось прочитать
	} else {
		table.Records = records
	}
	
	p.logger("Таблица %s: полей=%d, записей=%d", tableName, len(table.Fields), len(table.Records))
	return table, nil
}

// findStructureFile ищет файл структуры для таблицы
func (p *CronosParser) findStructureFile(tablePath string) string {
	dir := filepath.Dir(tablePath)
	baseName := strings.TrimSuffix(filepath.Base(tablePath), filepath.Ext(tablePath))
	
	// Возможные варианты имен файлов структуры
	variants := []string{
		filepath.Join(dir, baseName+".def"),
		filepath.Join(dir, baseName+".str"),
		filepath.Join(dir, baseName+".struct"),
		filepath.Join(dir, "CroStru.dat"),
		filepath.Join(dir, "structure.dat"),
	}
	
	for _, variant := range variants {
		if _, err := os.Stat(variant); err == nil {
			return variant
		}
	}
	
	return ""
}

// parseStructure парсит файл структуры таблицы
func (p *CronosParser) parseStructure(structPath string) ([]CronosField, error) {
	file, err := os.Open(structPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл структуры: %w", err)
	}
	defer file.Close()
	
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл структуры: %w", err)
	}
	
	// Пытаемся парсить структуру разными способами
	if fields := p.tryParseStructureFormat1(data); len(fields) > 0 {
		return fields, nil
	}
	
	if fields := p.tryParseStructureFormat2(data); len(fields) > 0 {
		return fields, nil
	}
	
	// Если ничего не получилось, возвращаем базовую структуру
	return p.createDefaultFields(), nil
}

// tryParseStructureFormat1 пытается парсить структуру в формате 1
func (p *CronosParser) tryParseStructureFormat1(data []byte) []CronosField {
	var fields []CronosField
	
	// Простейший парсинг - ищем текстовые описания полей
	lines := strings.Split(string(data), "\n")
	fieldNum := 1
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Пытаемся извлечь информацию о поле
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			fieldName := parts[0]
			if fieldName != "" {
				field := CronosField{
					Name:     fieldName,
					Type:     "String",
					Length:   255,
					Position: fieldNum - 1,
				}
				
				// Пытаемся определить тип по имени
				lowerName := strings.ToLower(fieldName)
				if strings.Contains(lowerName, "дата") || strings.Contains(lowerName, "date") {
					field.Type = "Date"
				} else if strings.Contains(lowerName, "номер") || strings.Contains(lowerName, "number") || strings.Contains(lowerName, "id") {
					field.Type = "Number"
				}
				
				fields = append(fields, field)
				fieldNum++
			}
		}
	}
	
	return fields
}

// tryParseStructureFormat2 пытается парсить бинарную структуру
func (p *CronosParser) tryParseStructureFormat2(data []byte) []CronosField {
	var fields []CronosField
	
	if len(data) < 4 {
		return fields
	}
	
	reader := bytes.NewReader(data)
	
	// Пытаемся прочитать количество полей
	var fieldCount uint32
	if err := binary.Read(reader, binary.LittleEndian, &fieldCount); err != nil {
		return fields
	}
	
	// Санитарная проверка
	if fieldCount > 1000 || fieldCount == 0 {
		return fields
	}
	
	// Читаем описания полей
	for i := uint32(0); i < fieldCount && reader.Len() > 0; i++ {
		field := CronosField{
			Position: int(i),
			Type:     "String",
			Length:   255,
		}
		
		// Пытаемся прочитать имя поля (предполагаем фиксированную длину)
		nameBytes := make([]byte, 32)
		if _, err := reader.Read(nameBytes); err != nil {
			break
		}
		
		// Убираем null-терминаторы и лишние байты
		nameEnd := bytes.IndexByte(nameBytes, 0)
		if nameEnd == -1 {
			nameEnd = len(nameBytes)
		}
		
		field.Name = strings.TrimSpace(string(nameBytes[:nameEnd]))
		if field.Name == "" {
			field.Name = fmt.Sprintf("field_%d", i+1)
		}
		
		fields = append(fields, field)
	}
	
	return fields
}

// createDefaultFields создает стандартный набор полей
func (p *CronosParser) createDefaultFields() []CronosField {
	return []CronosField{
		{Name: "id", Type: "String", Length: 50, Position: 0},
		{Name: "data", Type: "String", Length: 1000, Position: 1},
	}
}

// parseData парсит данные таблицы
func (p *CronosParser) parseData(tablePath string, fields []CronosField) ([][]string, error) {
	// Ищем файл с данными
	dataFile := p.findDataFile(tablePath)
	if dataFile == "" {
		return [][]string{}, nil
	}
	
	file, err := os.Open(dataFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл данных: %w", err)
	}
	defer file.Close()
	
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл данных: %w", err)
	}
	
	// Парсим данные
	records := p.extractRecords(data, fields)
	return records, nil
}

// findDataFile ищет файл с данными для таблицы
func (p *CronosParser) findDataFile(tablePath string) string {
	dir := filepath.Dir(tablePath)
	baseName := strings.TrimSuffix(filepath.Base(tablePath), filepath.Ext(tablePath))
	
	// Возможные варианты файлов данных
	variants := []string{
		tablePath,                                    // сам файл может содержать данные
		filepath.Join(dir, baseName+".db"),
		filepath.Join(dir, baseName+".data"),
		filepath.Join(dir, baseName+".idx"),
		strings.Replace(tablePath, ".dat", ".db", 1),
	}
	
	for _, variant := range variants {
		if info, err := os.Stat(variant); err == nil && info.Size() > 0 {
			return variant
		}
	}
	
	return tablePath
}

// extractRecords извлекает записи из бинарных данных
func (p *CronosParser) extractRecords(data []byte, fields []CronosField) [][]string {
	var records [][]string
	
	if len(data) == 0 {
		return records
	}
	
	// Пытаемся разные методы извлечения данных
	
	// Метод 1: Попытка текстового парсинга
	if textRecords := p.tryParseAsText(data, fields); len(textRecords) > 0 {
		return textRecords
	}
	
	// Метод 2: Попытка бинарного парсинга с фиксированной длиной записи
	if binaryRecords := p.tryParseAsBinary(data, fields); len(binaryRecords) > 0 {
		return binaryRecords
	}
	
	// Метод 3: Создаем одну запись со всеми данными как строкой
	if len(data) > 0 {
		record := make([]string, len(fields))
		
		// Очищаем данные от непечатаемых символов
		cleanData := p.cleanBinaryData(data)
		
		// Распределяем данные по полям
		if len(fields) > 0 {
			record[0] = cleanData
			for i := 1; i < len(fields); i++ {
				record[i] = ""
			}
		}
		
		records = append(records, record)
	}
	
	return records
}

// tryParseAsText пытается парсить данные как текст
func (p *CronosParser) tryParseAsText(data []byte, fields []CronosField) [][]string {
	var records [][]string
	
	// Конвертируем в строку
	text := string(data)
	
	// Разделяем по строкам
	lines := strings.Split(text, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Пытаемся разделить строку на поля
		var record []string
		
		// Пробуем разные разделители
		separators := []string{"\t", "|", ";", ","}
		for _, sep := range separators {
			parts := strings.Split(line, sep)
			if len(parts) >= len(fields) {
				record = parts[:len(fields)]
				break
			}
		}
		
		// Если не удалось разделить, используем всю строку как первое поле
		if len(record) == 0 {
			record = make([]string, len(fields))
			record[0] = line
			for i := 1; i < len(fields); i++ {
				record[i] = ""
			}
		}
		
		// Дополняем запись до нужного количества полей
		for len(record) < len(fields) {
			record = append(record, "")
		}
		
		records = append(records, record)
	}
	
	return records
}

// tryParseAsBinary пытается парсить бинарные данные
func (p *CronosParser) tryParseAsBinary(data []byte, fields []CronosField) [][]string {
	var records [][]string
	
	// Пытаемся определить размер записи
	recordSize := 0
	for _, field := range fields {
		recordSize += field.Length
		if recordSize == 0 {
			recordSize += 100 // значение по умолчанию
		}
	}
	
	// Если размер записи слишком большой или маленький, используем эвристику
	if recordSize > 10000 || recordSize < 10 {
		recordSize = 256 // разумное значение по умолчанию
	}
	
	// Читаем записи фиксированного размера
	for offset := 0; offset+recordSize <= len(data); offset += recordSize {
		recordData := data[offset : offset+recordSize]
		
		// Извлекаем поля из записи
		record := make([]string, len(fields))
		fieldOffset := 0
		
		for i, field := range fields {
			fieldLength := field.Length
			if fieldLength <= 0 {
				fieldLength = 50
			}
			
			if fieldOffset+fieldLength <= len(recordData) {
				fieldData := recordData[fieldOffset : fieldOffset+fieldLength]
				record[i] = p.cleanBinaryData(fieldData)
				fieldOffset += fieldLength
			} else {
				record[i] = ""
			}
		}
		
		records = append(records, record)
		
		// Ограничиваем количество записей для безопасности
		if len(records) > 10000 {
			break
		}
	}
	
	return records
}

// cleanBinaryData очищает бинарные данные для использования как строки
func (p *CronosParser) cleanBinaryData(data []byte) string {
	// Убираем null-байты и непечатаемые символы
	var result []byte
	
	for _, b := range data {
		if b == 0 {
			break // останавливаемся на null-терминаторе
		}
		if b >= 32 && b <= 126 {
			result = append(result, b)
		} else if b >= 192 { // кириллица в CP1251/CP866
			result = append(result, b)
		}
	}
	
	return strings.TrimSpace(string(result))
}