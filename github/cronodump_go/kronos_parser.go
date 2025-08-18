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

// KronosParser парсер для баз данных Kronos
type KronosParser struct {
	dbPath    string
	struFile  *DataFile
	indexFile *DataFile
	bankFile  *DataFile
	kod       *KODDecoder
}

// DataFile представляет файл данных Kronos (.dat + .tad)
type DataFile struct {
	name        string
	datFile     *os.File
	tadFile     *os.File
	version     []byte
	blockSize   uint16
	encoding    uint16
	nrRecords   int
	use64bit    bool
	isEncrypted bool
	tadHdrSize  int
	kod         *KODDecoder
}

// NewKronosParser создает новый парсер для базы Kronos
func NewKronosParser(dbPath string) (*KronosParser, error) {
	parser := &KronosParser{
		dbPath: dbPath,
		kod:    NewKODDecoder(),
	}

	var err error

	// Открываем основные файлы базы данных
	parser.struFile, err = parser.openDataFile("Stru")
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл CroStru: %w", err)
	}

	parser.indexFile, err = parser.openDataFile("Index")
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл CroIndex: %w", err)
	}

	parser.bankFile, err = parser.openDataFile("Bank")
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл CroBank: %w", err)
	}

	return parser, nil
}

// openDataFile открывает пару файлов .dat и .tad
func (p *KronosParser) openDataFile(name string) (*DataFile, error) {
	datPath := p.findFile(fmt.Sprintf("Cro%s.dat", name))
	tadPath := p.findFile(fmt.Sprintf("Cro%s.tad", name))

	if datPath == "" || tadPath == "" {
		return nil, fmt.Errorf("файлы Cro%s.dat или Cro%s.tad не найдены", name, name)
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

	dataFile := &DataFile{
		name:    name,
		datFile: datFile,
		tadFile: tadFile,
		kod:     p.kod,
	}

	err = dataFile.readHeaders()
	if err != nil {
		dataFile.Close()
		return nil, err
	}

	return dataFile, nil
}

// findFile ищет файл без учета регистра
func (p *KronosParser) findFile(filename string) string {
	entries, err := os.ReadDir(p.dbPath)
	if err != nil {
		return ""
	}

	lowerFilename := strings.ToLower(filename)
	for _, entry := range entries {
		if strings.ToLower(entry.Name()) == lowerFilename {
			return filepath.Join(p.dbPath, entry.Name())
		}
	}
	return ""
}

// readHeaders читает заголовки файлов .dat и .tad
func (df *DataFile) readHeaders() error {
	// Читаем заголовок .dat файла
	df.datFile.Seek(0, io.SeekStart)
	hdrData := make([]byte, 19)
	_, err := df.datFile.Read(hdrData)
	if err != nil {
		return err
	}

	magic := hdrData[:8]
	if !bytes.Equal(magic, []byte("CroFile\x00")) {
		return fmt.Errorf("неверная сигнатура файла: %x", magic)
	}

	df.version = hdrData[10:15]
	df.encoding = binary.LittleEndian.Uint16(hdrData[15:17])
	df.blockSize = binary.LittleEndian.Uint16(hdrData[17:19])

	// Определяем версию и шифрование
	versionStr := string(df.version)
	df.use64bit = versionStr == "01.03" || versionStr == "01.05" || versionStr == "01.11"
	df.isEncrypted = versionStr == "01.04" || versionStr == "01.05" || versionStr == "01.14"

	// Читаем заголовок .tad файла
	df.tadFile.Seek(0, io.SeekStart)
	
	if versionStr == "01.02" || versionStr == "01.03" || versionStr == "01.04" || versionStr == "01.05" {
		// v3
		tadHdr := make([]byte, 8)
		_, err = df.tadFile.Read(tadHdr)
		if err != nil {
			return err
		}
		df.tadHdrSize = 8
	} else {
		// v4
		tadHdr := make([]byte, 16)
		_, err = df.tadFile.Read(tadHdr)
		if err != nil {
			return err
		}
		df.tadHdrSize = 16
	}

	// Вычисляем количество записей
	tadFileInfo, err := df.tadFile.Stat()
	if err != nil {
		return err
	}

	tadSize := int(tadFileInfo.Size()) - df.tadHdrSize
	entrySize := 12
	if df.use64bit {
		entrySize = 16
	}
	df.nrRecords = tadSize / entrySize

	fmt.Printf("Файл %s: версия %s, записей %d, 64bit: %v, зашифрован: %v\n", 
		df.name, df.version, df.nrRecords, df.use64bit, df.isEncrypted)

	return nil
}

// readRecord читает запись по индексу
func (df *DataFile) readRecord(index int) ([]byte, error) {
	if index < 1 || index > df.nrRecords {
		return nil, fmt.Errorf("индекс записи вне диапазона: %d", index)
	}

	entrySize := 12
	if df.use64bit {
		entrySize = 16
	}

	tadOffset := int64(df.tadHdrSize + (index-1)*entrySize)
	df.tadFile.Seek(tadOffset, io.SeekStart)

	tadEntry := make([]byte, entrySize)
	_, err := df.tadFile.Read(tadEntry)
	if err != nil {
		return nil, err
	}

	var offset, size uint64
	if df.use64bit {
		offset = binary.LittleEndian.Uint64(tadEntry[:8])
		size = binary.LittleEndian.Uint64(tadEntry[8:16])
	} else {
		offset = uint64(binary.LittleEndian.Uint32(tadEntry[:4]))
		size = uint64(binary.LittleEndian.Uint32(tadEntry[4:8]))
	}

	if size == 0 {
		return nil, nil // Удаленная запись
	}

	// Читаем данные из .dat файла
	df.datFile.Seek(int64(offset), io.SeekStart)
	data := make([]byte, size)
	_, err = df.datFile.Read(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Close закрывает все открытые файлы
func (df *DataFile) Close() error {
	var errs []error
	if df.datFile != nil {
		if err := df.datFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if df.tadFile != nil {
		if err := df.tadFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("ошибки при закрытии файлов: %v", errs)
	}
	return nil
}

// Close закрывает парсер
func (p *KronosParser) Close() error {
	var errs []error
	
	if p.struFile != nil {
		if err := p.struFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if p.indexFile != nil {
		if err := p.indexFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if p.bankFile != nil {
		if err := p.bankFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("ошибки при закрытии файлов: %v", errs)
	}
	return nil
}

// GetTables возвращает список таблиц в базе данных
func (p *KronosParser) GetTables() ([]string, error) {
	// Ищем запись с определениями таблиц (обычно это запись с типом 0x03)
	var dbInfo []byte
	
	for i := 1; i <= p.struFile.nrRecords; i++ {
		record, err := p.struFile.readRecord(i)
		if err != nil {
			fmt.Printf("Ошибка чтения записи %d: %v\n", i, err)
			continue
		}
		
		if record == nil {
			fmt.Printf("Запись %d: удалена\n", i)
			continue
		}
		
		if len(record) == 0 {
			fmt.Printf("Запись %d: пустая\n", i)
			continue
		}
		
		fmt.Printf("Запись %d: тип=%02x, длина=%d\n", i, record[0], len(record))
		
		if record[0] == 0x03 {
			dbInfo = record
			fmt.Printf("Найдена запись с определениями таблиц: %d\n", i)
			break
		}
	}

	if len(dbInfo) == 0 {
		return nil, fmt.Errorf("не найдены определения таблиц в базе данных")
	}

	// Парсим определения таблиц
	tables, err := p.parseTableDefinitions(dbInfo[1:])
	if err != nil {
		return nil, err
	}

	var tableNames []string
	for _, table := range tables {
		if table.TableName != "" {
			tableNames = append(tableNames, table.TableName)
		}
	}

	return tableNames, nil
}

// parseTableDefinitions парсит определения таблиц из данных CroStru
func (p *KronosParser) parseTableDefinitions(data []byte) ([]TableDefinition, error) {
	reader := bytes.NewReader(data)
	dbDef := make(map[string][]byte)

	fmt.Printf("Парсим определения таблиц, размер данных: %d\n", len(data))

	// Читаем определения
	count := 0
	for reader.Len() > 0 && count < 20 { // Ограничиваем для отладки
		count++
		
		keyName, err := p.readName(reader)
		if err != nil {
			fmt.Printf("Ошибка чтения имени ключа: %v\n", err)
			break
		}

		if keyName == "" {
			break
		}

		var indexOrLength uint32
		err = binary.Read(reader, binary.LittleEndian, &indexOrLength)
		if err != nil {
			fmt.Printf("Ошибка чтения длины для ключа %s: %v\n", keyName, err)
			break
		}

		fmt.Printf("Ключ: '%s', индекс/длина: %08x\n", keyName, indexOrLength)

		if (indexOrLength >> 31) != 0 {
			// Прямые данные
			length := indexOrLength & 0x7FFFFFFF
			value := make([]byte, length)
			_, err = reader.Read(value)
			if err != nil {
				fmt.Printf("Ошибка чтения данных для ключа %s: %v\n", keyName, err)
				break
			}
			dbDef[keyName] = value
			fmt.Printf("  Прямые данные длиной: %d\n", length)
		} else {
			// Ссылка на запись в CroStru
			refData, err := p.struFile.readRecord(int(indexOrLength))
			if err != nil {
				fmt.Printf("Ошибка чтения ссылочных данных для ключа %s: %v\n", keyName, err)
				continue
			}
			if refData != nil && len(refData) > 0 && refData[0] == 0x04 {
				dbDef[keyName] = refData[1:]
				fmt.Printf("  Ссылочные данные длиной: %d\n", len(refData)-1)
			}
		}
	}

	// Извлекаем определения таблиц
	var tables []TableDefinition
	for key, value := range dbDef {
		if strings.HasPrefix(key, "Base") && len(key) > 4 {
			tableID := key[4:]
			fmt.Printf("Обрабатываем таблицу: %s (ID: %s)\n", key, tableID)
			
			table, err := p.parseTableDefinition(value)
			if err != nil {
				fmt.Printf("Ошибка парсинга таблицы %s: %v\n", key, err)
				continue
			}
			
			fmt.Printf("  Найдена таблица: '%s' (ID: %d)\n", table.TableName, table.TableID)
			tables = append(tables, table)
		}
	}

	return tables, nil
}

// readName читает имя с префиксом длины (1 байт)
func (p *KronosParser) readName(reader *bytes.Reader) (string, error) {
	length, err := reader.ReadByte()
	if err != nil {
		return "", err
	}
	
	if length == 0 {
		return "", nil
	}
	
	nameBytes := make([]byte, length)
	_, err = reader.Read(nameBytes)
	if err != nil {
		return "", err
	}
	
	return DecodeCP1251(nameBytes), nil
}

// parseTableDefinition парсит определение таблицы
func (p *KronosParser) parseTableDefinition(data []byte) (TableDefinition, error) {
	reader := bytes.NewReader(data)
	table := TableDefinition{}

	// Читаем заголовок таблицы
	var unk1 uint16
	binary.Read(reader, binary.LittleEndian, &unk1)
	
	version, _ := reader.ReadByte()
	table.Version = version
	
	if version > 1 {
		reader.ReadByte() // пропускаем байт
	}

	unk2, _ := reader.ReadByte()
	_, _ = reader.ReadByte() // unk3
	
	if unk2 > 5 {
		var skip uint32
		binary.Read(reader, binary.LittleEndian, &skip)
	}

	var unk4 uint32
	binary.Read(reader, binary.LittleEndian, &unk4)
	binary.Read(reader, binary.LittleEndian, &table.TableID)

	tableName, err := p.readName(reader)
	if err != nil {
		return table, err
	}
	table.TableName = tableName

	abbrev, err := p.readName(reader)
	if err != nil {
		return table, err
	}
	table.Abbrev = abbrev

	var unk7, nrFields uint32
	binary.Read(reader, binary.LittleEndian, &unk7)
	binary.Read(reader, binary.LittleEndian, &nrFields)

	fmt.Printf("    Полей в таблице: %d\n", nrFields)

	// Читаем определения полей
	for i := uint32(0); i < nrFields; i++ {
		var defLen uint16
		err = binary.Read(reader, binary.LittleEndian, &defLen)
		if err != nil {
			fmt.Printf("    Ошибка чтения длины поля %d: %v\n", i, err)
			break
		}
		
		fieldData := make([]byte, defLen)
		_, err = reader.Read(fieldData)
		if err != nil {
			fmt.Printf("    Ошибка чтения данных поля %d: %v\n", i, err)
			break
		}
		
		field, err := p.parseFieldDefinition(fieldData)
		if err != nil {
			fmt.Printf("    Ошибка парсинга поля %d: %v\n", i, err)
			continue
		}
		
		fmt.Printf("    Поле %d: '%s' (тип: %d)\n", i, field.Name, field.Type)
		table.Fields = append(table.Fields, field)
	}

	return table, nil
}

// parseFieldDefinition парсит определение поля
func (p *KronosParser) parseFieldDefinition(data []byte) (FieldDefinition, error) {
	reader := bytes.NewReader(data)
	field := FieldDefinition{}

	var typ uint16
	binary.Read(reader, binary.LittleEndian, &typ)
	field.Type = FieldType(typ)

	binary.Read(reader, binary.LittleEndian, &field.Index1)

	name, err := p.readName(reader)
	if err != nil {
		return field, err
	}
	field.Name = name

	binary.Read(reader, binary.LittleEndian, &field.Flags)
	
	minVal, _ := reader.ReadByte()
	field.MinVal = minVal

	if field.Type != FieldTypeSystemNumber {
		binary.Read(reader, binary.LittleEndian, &field.Index2)
		binary.Read(reader, binary.LittleEndian, &field.MaxVal)
		binary.Read(reader, binary.LittleEndian, &field.Unknown4)
	}

	return field, nil
}

// GetTableStructure возвращает структуру таблицы
func (p *KronosParser) GetTableStructure(tableName string) ([]FieldDefinition, error) {
	tables, err := p.GetTables()
	if err != nil {
		return nil, err
	}

	for _, table := range tables {
		if table.TableName == tableName {
			return table.Fields, nil
		}
	}

	return nil, fmt.Errorf("таблица '%s' не найдена", tableName)
}

// GetTableData возвращает данные таблицы
func (p *KronosParser) GetTableData(tableName string) ([][]string, error) {
	// Получаем структуру таблицы
	tables, err := p.GetTables()
	if err != nil {
		return nil, err
	}

	var targetTable *TableDefinition
	for _, table := range tables {
		if table.TableName == tableName {
			targetTable = &table
			break
		}
	}

	if targetTable == nil {
		return nil, fmt.Errorf("таблица '%s' не найдена", tableName)
	}

	// Читаем записи из CroBank
	var rows [][]string
	
	for i := 1; i <= p.bankFile.nrRecords; i++ {
		recordData, err := p.bankFile.readRecord(i)
		if err != nil || recordData == nil || len(recordData) == 0 {
			continue
		}

		// Проверяем что запись принадлежит нашей таблице
		if recordData[0] != byte(targetTable.TableID) {
			continue
		}

		// Парсим поля записи
		record, err := p.parseRecord(targetTable.Fields, recordData[1:], uint32(i))
		if err != nil {
			continue
		}

		rows = append(rows, record.Fields)
	}

	return rows, nil
}

// parseRecord парсит запись
func (p *KronosParser) parseRecord(fields []FieldDefinition, data []byte, systemNumber uint32) (Record, error) {
	record := Record{
		SystemNumber: systemNumber,
		Fields:       make([]string, len(fields)),
	}

	// Первое поле всегда системный номер
	record.Fields[0] = fmt.Sprintf("%d", systemNumber)

	reader := bytes.NewReader(data)
	
	for i := 1; i < len(fields); i++ {
		if reader.Len() == 0 {
			break
		}

		var fieldData []byte
		
		// Проверяем на сложную запись (начинается с 0x1b)
		if reader.Len() > 0 {
			firstByte, _ := reader.ReadByte()
			reader.UnreadByte()
			
			if firstByte == 0x1b {
				reader.ReadByte() // пропускаем 0x1b
				var size uint32
				binary.Read(reader, binary.LittleEndian, &size)
				fieldData = make([]byte, size)
				reader.Read(fieldData)
			} else {
				// Читаем до разделителя 0x1e
				fieldData = p.readToSeparator(reader, 0x1e)
			}
		}

		record.Fields[i] = DecodeField(fields[i], fieldData)
	}

	return record, nil
}

// readToSeparator читает данные до указанного разделителя
func (p *KronosParser) readToSeparator(reader *bytes.Reader, separator byte) []byte {
	var result []byte
	for reader.Len() > 0 {
		b, err := reader.ReadByte()
		if err != nil || b == separator {
			break
		}
		result = append(result, b)
	}
	return result
}