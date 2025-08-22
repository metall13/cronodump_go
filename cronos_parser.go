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

// CronosParser парсер для баз данных Cronos
type CronosParser struct {
	dbPath  string
	verbose bool
	stru    *CronosFile
	bank    *CronosFile
	index   *CronosFile
}

// CronosFile представляет файл Cronos (.dat + .tad)
type CronosFile struct {
	name          string
	datFile       *os.File
	tadFile       *os.File
	header        CronosHeader
	compact       bool
	kod           *KODDecoder
	tadHdrLen     int
	tadEntrySize  int
	nrOfRecords   int
}

// CronosHeader заголовок файла .dat
type CronosHeader struct {
	Magic     [8]byte
	Unknown   uint16
	Version   [5]byte
	Encoding  uint16
	BlockSize uint16
}

// Table представляет таблицу в базе данных
type Table struct {
	ID     uint32
	Name   string
	Abbrev string
	Fields []Field
}

// Field представляет поле в таблице
type Field struct {
	Name    string
	Type    int
	MaxLen  uint32
	Flags   uint32
}

// Record представляет запись в таблице
type Record struct {
	Fields []FieldValue
}

// FieldValue значение поля в записи
type FieldValue struct {
	Type    int
	Content string
}

// NewCronosParser создает новый парсер
func NewCronosParser(dbPath string, verbose bool) (*CronosParser, error) {
	parser := &CronosParser{
		dbPath:  dbPath,
		verbose: verbose,
	}

	// Открываем основные файлы базы данных
	var err error
	parser.stru, err = parser.openCronosFile("Stru")
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия CroStru: %v", err)
	}

	parser.bank, err = parser.openCronosFile("Bank")
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия CroBank: %v", err)
	}

	parser.index, err = parser.openCronosFile("Index")
	if err != nil {
		// Index файл не обязателен
		if verbose {
			fmt.Printf("Предупреждение: CroIndex файл не найден\n")
		}
	}

	return parser, nil
}

// openCronosFile открывает пару файлов .dat/.tad
func (p *CronosParser) openCronosFile(name string) (*CronosFile, error) {
	datPath := filepath.Join(p.dbPath, fmt.Sprintf("Cro%s.dat", name))
	tadPath := filepath.Join(p.dbPath, fmt.Sprintf("Cro%s.tad", name))

	// Проверяем существование файлов (регистронезависимо)
	datPath = p.findCaseInsensitiveFile(datPath)
	tadPath = p.findCaseInsensitiveFile(tadPath)

	if datPath == "" || tadPath == "" {
		return nil, fmt.Errorf("файлы Cro%s.dat/tad не найдены", name)
	}

	datFile, err := os.Open(datPath)
	if err != nil {
		return nil, err
	}

	tadFile, err := os.Open(tadPath)
	if err != nil {
		datFile.Close()
		return nil, err
	}

	croFile := &CronosFile{
		name:    name,
		datFile: datFile,
		tadFile: tadFile,
		compact: false,
		kod:     NewKODDecoder(),
	}

	// Читаем заголовок
	if err := croFile.readHeader(); err != nil {
		croFile.Close()
		return nil, err
	}

	// Читаем TAD заголовок
	if err := croFile.readTADHeader(); err != nil {
		croFile.Close()
		return nil, err
	}

	return croFile, nil
}

// findCaseInsensitiveFile ищет файл без учета регистра
func (p *CronosParser) findCaseInsensitiveFile(targetPath string) string {
	dir := filepath.Dir(targetPath)
	fileName := filepath.Base(targetPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), fileName) {
			return filepath.Join(dir, entry.Name())
		}
	}

	return ""
}

// readHeader читает заголовок .dat файла
func (cf *CronosFile) readHeader() error {
	_, err := cf.datFile.Seek(0, 0)
	if err != nil {
		return err
	}

	err = binary.Read(cf.datFile, binary.LittleEndian, &cf.header)
	if err != nil {
		return err
	}

	// Проверяем магическое число
	if string(cf.header.Magic[:]) != "CroFile\x00" {
		return fmt.Errorf("неверное магическое число: %v", cf.header.Magic)
	}

	return nil
}

// Close закрывает все файлы
func (cf *CronosFile) Close() error {
	var errors []string
	
	if cf.datFile != nil {
		if err := cf.datFile.Close(); err != nil {
			errors = append(errors, err.Error())
		}
	}
	
	if cf.tadFile != nil {
		if err := cf.tadFile.Close(); err != nil {
			errors = append(errors, err.Error())
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("ошибки закрытия файлов: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

// Close закрывает парсер
func (p *CronosParser) Close() error {
	var errors []string

	if p.stru != nil {
		if err := p.stru.Close(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if p.bank != nil {
		if err := p.bank.Close(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if p.index != nil {
		if err := p.index.Close(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("ошибки закрытия парсера: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetTables возвращает список таблиц в базе данных
func (p *CronosParser) GetTables() ([]*Table, error) {
	if p.stru == nil {
		return nil, fmt.Errorf("файл CroStru не открыт")
	}

	// Читаем первую запись (определения таблиц)
	dbInfo, err := p.stru.readRecord(1)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения определения базы: %v", err)
	}

	if len(dbInfo) == 0 || dbInfo[0] != 0x03 {
		return nil, fmt.Errorf("неверный формат определения базы")
	}

	// Декодируем определения таблиц
	dbDef, err := p.decodeDBDefinition(dbInfo[1:])
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования определений: %v", err)
	}

	var tables []*Table

	// Ищем определения таблиц (Base000, Base001, и т.д.)
	for key, value := range dbDef {
		if strings.HasPrefix(key, "Base") && len(key) > 4 {
			tableID := key[4:]
			if tableID != "000" { // Пропускаем файловую таблицу Base000
				// Получаем изображение таблицы если есть
				imageKey := "BaseImage" + tableID
				var imageData []byte
				if img, exists := dbDef[imageKey]; exists {
					imageData = img
				}

				table, err := p.parseTableDefinition(value, imageData)
				if err != nil {
					if p.verbose {
						fmt.Printf("Ошибка парсинга таблицы %s: %v\n", key, err)
					}
					continue
				}

				tables = append(tables, table)
			}
		}
	}

	return tables, nil
}

// GetRecords возвращает записи из таблицы
func (p *CronosParser) GetRecords(table *Table, offset, limit int) ([]*Record, error) {
	if p.bank == nil {
		return nil, fmt.Errorf("файл CroBank не открыт")
	}

	var records []*Record
	processed := 0
	skipped := 0

	// Перебираем все записи в банке
	for i := 1; i <= p.bank.nrOfRecords && len(records) < limit; i++ {
		data, err := p.bank.readRecord(i)
		if err != nil {
			continue // Пропускаем поврежденные записи
		}

		if len(data) == 0 {
			continue
		}

		// Проверяем принадлежность записи к нашей таблице
		if data[0] != byte(table.ID) {
			continue
		}

		// Пропускаем записи до нужного offset
		if skipped < offset {
			skipped++
			continue
		}

		// Парсим запись
		record, err := p.parseRecord(table, data[1:])
		if err != nil {
			if p.verbose {
				fmt.Printf("Ошибка парсинга записи %d: %v\n", i, err)
			}
			continue
		}

		records = append(records, record)
		processed++
	}

	return records, nil
}

// readRecord читает запись из файла
func (cf *CronosFile) readRecord(recordID int) ([]byte, error) {
	// Читаем индекс записи из .tad файла
	tadOffset := int64(cf.tadHdrLen + (recordID-1)*cf.tadEntrySize)
	
	_, err := cf.tadFile.Seek(tadOffset, 0)
	if err != nil {
		return nil, err
	}

	var offset uint64
	var size uint32

	if cf.use64bit() {
		// 64-битные смещения
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &offset); err != nil {
			return nil, err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &size); err != nil {
			return nil, err
		}
	} else {
		// 32-битные смещения
		var offset32 uint32
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &offset32); err != nil {
			return nil, err
		}
		offset = uint64(offset32)
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &size); err != nil {
			return nil, err
		}
	}

	// Проверяем на удаленную запись
	if size == 0xFFFFFFFF {
		return nil, fmt.Errorf("запись удалена")
	}

	// Читаем данные из .dat файла
	_, err = cf.datFile.Seek(int64(offset), 0)
	if err != nil {
		return nil, err
	}

	data := make([]byte, size&0x7FFFFFFF) // Убираем флаг в старшем бите
	_, err = cf.datFile.Read(data)
	if err != nil {
		return nil, err
	}

	// Декодируем с помощью KOD если нужно
	if cf.isEncoded() && cf.kod != nil {
		data = cf.kod.Decode(int(offset), data)
	}

	return data, nil
}

// decodeDBDefinition декодирует определения базы данных
func (p *CronosParser) decodeDBDefinition(data []byte) (map[string][]byte, error) {
	result := make(map[string][]byte)
	rd := NewByteReader(data)

	for !rd.EOF() {
		keyName, err := rd.ReadName()
		if err != nil {
			break
		}

		indexOrLength, err := rd.ReadDWord()
		if err != nil {
			return nil, err
		}

		if indexOrLength&0x80000000 != 0 {
			// Прямые данные
			length := indexOrLength & 0x7FFFFFFF
			value, err := rd.ReadBytes(int(length))
			if err != nil {
				return nil, err
			}
			result[keyName] = value
		} else {
			// Ссылка на другую запись
			refData, err := p.stru.readRecord(int(indexOrLength))
			if err != nil {
				return nil, err
			}
			if len(refData) > 0 && refData[0] == 0x04 {
				result[keyName] = refData[1:]
			} else {
				result[keyName] = refData
			}
		}
	}

	return result, nil
}

// parseTableDefinition парсит определение таблицы
func (p *CronosParser) parseTableDefinition(data, imageData []byte) (*Table, error) {
	rd := NewByteReader(data)

	// Читаем заголовок таблицы
	unk1, err := rd.ReadWord()
	if err != nil {
		return nil, err
	}

	version, err := rd.ReadByte()
	if err != nil {
		return nil, err
	}

	if version > 1 {
		_, err := rd.ReadByte() // Пропускаем байт
		if err != nil {
			return nil, err
		}
	}

	unk2, err := rd.ReadByte()
	if err != nil {
		return nil, err
	}

	unk3, err := rd.ReadByte()
	if err != nil {
		return nil, err
	}

	if unk2 > 5 {
		_, err := rd.ReadDWord() // Дополнительные данные
		if err != nil {
			return nil, err
		}
	}

	unk4, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}

	tableID, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}

	tableName, err := rd.ReadName()
	if err != nil {
		return nil, err
	}

	abbrev, err := rd.ReadName()
	if err != nil {
		return nil, err
	}

	unk7, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}

	nrFields, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}

	// Читаем определения полей
	var fields []Field
	for i := 0; i < int(nrFields); i++ {
		defLen, err := rd.ReadWord()
		if err != nil {
			return nil, err
		}

		fieldDefData, err := rd.ReadBytes(int(defLen))
		if err != nil {
			return nil, err
		}

		field, err := p.parseFieldDefinition(fieldDefData)
		if err != nil {
			if p.verbose {
				fmt.Printf("Ошибка парсинга поля %d: %v\n", i, err)
			}
			continue
		}

		fields = append(fields, *field)
	}

	table := &Table{
		ID:     tableID,
		Name:   tableName,
		Abbrev: abbrev,
		Fields: fields,
	}

	return table, nil
}

// parseFieldDefinition парсит определение поля
func (p *CronosParser) parseFieldDefinition(data []byte) (*Field, error) {
	rd := NewByteReader(data)

	typ, err := rd.ReadWord()
	if err != nil {
		return nil, err
	}

	idx1, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}

	name, err := rd.ReadName()
	if err != nil {
		return nil, err
	}

	flags, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}

	minVal, err := rd.ReadByte()
	if err != nil {
		return nil, err
	}

	var maxLen uint32
	if typ != 0 {
		idx2, err := rd.ReadDWord()
		if err != nil {
			return nil, err
		}
		_ = idx2 // Пока не используем

		maxLen, err = rd.ReadDWord()
		if err != nil {
			return nil, err
		}

		unk4, err := rd.ReadDWord()
		if err != nil {
			return nil, err
		}
		_ = unk4 // Пока не используем
	}

	field := &Field{
		Name:   name,
		Type:   int(typ),
		MaxLen: maxLen,
		Flags:  flags,
	}

	return field, nil
}

// parseRecord парсит запись таблицы
func (p *CronosParser) parseRecord(table *Table, data []byte) (*Record, error) {
	rd := NewByteReader(data)
	var fieldValues []FieldValue

	for _, field := range table.Fields {
		value, err := p.parseFieldValue(field, rd)
		if err != nil {
			// Если не удалось прочитать поле, добавляем пустое значение
			fieldValues = append(fieldValues, FieldValue{
				Type:    field.Type,
				Content: "",
			})
			continue
		}

		fieldValues = append(fieldValues, *value)
	}

	return &Record{Fields: fieldValues}, nil
}

// parseFieldValue парсит значение поля
func (p *CronosParser) parseFieldValue(field Field, rd *ByteReader) (*FieldValue, error) {
	var content string

	switch field.Type {
	case 0: // Системный номер (ID)
		// Читаем как число
		if !rd.EOF() {
			data, err := rd.ReadBytes(4)
			if err == nil && len(data) == 4 {
				value := binary.LittleEndian.Uint32(data)
				content = fmt.Sprintf("%d", value)
			}
		}

	case 1: // Целое число
		if !rd.EOF() {
			data, err := rd.ReadBytes(4)
			if err == nil && len(data) == 4 {
				value := binary.LittleEndian.Uint32(data)
				content = fmt.Sprintf("%d", value)
			}
		}

	case 2: // Строка (VARCHAR)
		if !rd.EOF() {
			str, err := rd.ReadName()
			if err == nil {
				content = str
			}
		}

	case 3: // Текст (словарь)
		if !rd.EOF() {
			str, err := rd.ReadLongString()
			if err == nil {
				content = str
			}
		}

	case 4: // Дата
		if !rd.EOF() {
			dateBytes, err := rd.ReadBytes(6) // Формат: YYMMDD
			if err == nil && len(dateBytes) >= 6 {
				// Парсим дату в формате YYMMDD
				year := 1900 + int(dateBytes[0]) + int(dateBytes[1])*256
				month := int(dateBytes[2]) + int(dateBytes[3])*256
				day := int(dateBytes[4]) + int(dateBytes[5])*256
				content = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
			}
		}

	case 5: // Время
		if !rd.EOF() {
			timeBytes, err := rd.ReadBytes(4) // Формат: HHMM
			if err == nil && len(timeBytes) >= 4 {
				hour := int(timeBytes[0]) + int(timeBytes[1])*256
				minute := int(timeBytes[2]) + int(timeBytes[3])*256
				content = fmt.Sprintf("%02d:%02d:00", hour, minute)
			}
		}

	case 6: // Ссылка на файл
		if !rd.EOF() {
			str, err := rd.ReadName()
			if err == nil {
				content = str
			}
		}

	default:
		// Неизвестный тип, читаем как строку
		if !rd.EOF() {
			remaining := rd.Remaining()
			content = string(remaining)
		}
	}

	return &FieldValue{
		Type:    field.Type,
		Content: content,
	}, nil
}

// readTADHeader читает заголовок .tad файла
func (cf *CronosFile) readTADHeader() error {
	_, err := cf.tadFile.Seek(0, 0)
	if err != nil {
		return err
	}

	if cf.isV3() {
		// V3 формат: 2 uint32
		var nrDeleted, firstDeleted uint32
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &nrDeleted); err != nil {
			return err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &firstDeleted); err != nil {
			return err
		}
		cf.tadHdrLen = 8
	} else if cf.isV4() {
		// V4 формат: 4 uint32
		var unk1, nrDeleted, firstDeleted, unk2 uint32
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &unk1); err != nil {
			return err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &nrDeleted); err != nil {
			return err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &firstDeleted); err != nil {
			return err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &unk2); err != nil {
			return err
		}
		cf.tadHdrLen = 16
	} else {
		return fmt.Errorf("неподдерживаемая версия TAD файла")
	}

	// Определяем размер записи в TAD
	if cf.use64bit() {
		cf.tadEntrySize = 16 // 8 bytes offset + 4 bytes size + 4 bytes checksum
	} else {
		cf.tadEntrySize = 12 // 4 bytes offset + 4 bytes size + 4 bytes checksum
	}

	// Вычисляем количество записей
	tadFileSize, err := cf.tadFile.Seek(0, 2) // Переходим в конец файла
	if err != nil {
		return err
	}

	tadDataSize := int(tadFileSize) - cf.tadHdrLen
	cf.nrOfRecords = tadDataSize / cf.tadEntrySize

	return nil
}

// use64bit проверяет использует ли файл 64-битные смещения
func (cf *CronosFile) use64bit() bool {
	version := string(cf.header.Version[:])
	return version == "01.03" || version == "01.05" || version == "01.11"
}

// isV3 проверяет версию 3
func (cf *CronosFile) isV3() bool {
	version := string(cf.header.Version[:])
	return version == "01.02" || version == "01.03" || version == "01.04" || version == "01.05"
}

// isV4 проверяет версию 4
func (cf *CronosFile) isV4() bool {
	version := string(cf.header.Version[:])
	return version == "01.11" || version == "01.13" || version == "01.14"
}

// isEncoded проверяет закодированы ли данные
func (cf *CronosFile) isEncoded() bool {
	return cf.header.Encoding&1 != 0
}