package main

import (
	"encoding/binary"
	"fmt"
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

	if p.verbose {
		fmt.Printf("Размер dbInfo: %d байт\n", len(dbInfo))
		if len(dbInfo) > 0 {
			fmt.Printf("Первый байт: 0x%02x\n", dbInfo[0])
		}
	}

	if len(dbInfo) == 0 || dbInfo[0] != 0x03 {
		return nil, fmt.Errorf("неверный формат определения базы: ожидался 0x03, получен 0x%02x", dbInfo[0])
	}

	// Декодируем определения таблиц
	dbDef, err := p.decodeDBDefinition(dbInfo[1:])
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования определений: %v", err)
	}

	var tables []*Table

	// Ищем определения таблиц (Base000, Base001, и т.д.)
	if p.verbose {
		fmt.Printf("Найдено ключей в dbDef: %d\n", len(dbDef))
		for key := range dbDef {
			fmt.Printf("Ключ: %s\n", key)
		}
	}

	for key, value := range dbDef {
		if strings.HasPrefix(key, "Base") && len(key) > 4 {
			tableID := key[4:]
			if p.verbose {
				fmt.Printf("Обрабатываем таблицу: %s (ID: %s)\n", key, tableID)
			}
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

				if p.verbose {
					fmt.Printf("Успешно спарсили таблицу: %s (%d полей)\n", table.Name, len(table.Fields))
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

	if p.verbose {
		fmt.Printf("Всего записей в банке: %d\n", p.bank.nrOfRecords)
		fmt.Printf("Ищем записи для таблицы ID: %d\n", table.ID)
	}

	// Перебираем все записи в банке
	for i := 1; i <= p.bank.nrOfRecords && len(records) < limit; i++ {
		data, err := p.bank.readRecord(i)
		if err != nil {
			if p.verbose {
				fmt.Printf("Ошибка чтения записи %d: %v\n", i, err)
			}
			continue // Пропускаем поврежденные записи
		}

		if len(data) == 0 {
			if p.verbose {
				fmt.Printf("Запись %d пустая\n", i)
			}
			continue
		}

		if p.verbose {
			fmt.Printf("Запись %d: размер %d, первый байт 0x%02x\n", i, len(data), data[0])
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

		if p.verbose {
			fmt.Printf("Успешно спарсили запись %d\n", i)
		}

		records = append(records, record)
		processed++
	}

	return records, nil
}