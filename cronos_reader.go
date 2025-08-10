package cronodamp

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CronosFileReader реализует CronosReader для чтения баз данных Cronos
type CronosFileReader struct {
	config Config
}

// NewCronosReader создает новый ридер для баз данных Cronos
func NewCronosReader(cfg Config) *CronosFileReader {
	return &CronosFileReader{
		config: cfg,
	}
}

// ReadDatabase читает всю базу данных Cronos из указанной директории
func (r *CronosFileReader) ReadDatabase(dbPath string) (*Database, error) {
	fmt.Printf("Читаем базу данных Cronos из: %s\n", dbPath)
	
	db := &Database{
		Path:   dbPath,
		Tables: []Table{},
	}
	
	// Рекурсивно ищем все файлы баз данных в директории
	err := filepath.Walk(dbPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Ищем файлы с расширениями, характерными для Cronos
		if !info.IsDir() && r.isCronosFile(path) {
			fmt.Printf("Найден файл базы данных: %s\n", path)
			
			table, readErr := r.ReadTable(path)
			if readErr != nil {
				fmt.Printf("Предупреждение: не удалось прочитать таблицу %s: %v\n", path, readErr)
				return nil // Продолжаем обработку других файлов
			}
			
			if table != nil {
				db.Tables = append(db.Tables, *table)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("ошибка при сканировании директории: %w", err)
	}
	
	fmt.Printf("Найдено %d таблиц в базе данных\n", len(db.Tables))
	return db, nil
}

// ReadTable читает отдельную таблицу из файла
func (r *CronosFileReader) ReadTable(tablePath string) (*Table, error) {
	fmt.Printf("Читаем таблицу из файла: %s\n", tablePath)
	
	// Определяем тип файла и соответствующий метод чтения
	if strings.HasSuffix(strings.ToLower(tablePath), ".csv") {
		return r.readCSVTable(tablePath)
	} else if strings.HasSuffix(strings.ToLower(tablePath), ".txt") {
		return r.readTextTable(tablePath)
	} else if r.isSQLiteFile(tablePath) {
		return r.readSQLiteTable(tablePath)
	} else {
		// Пытаемся читать как бинарный файл Cronos
		return r.readCronosBinaryTable(tablePath)
	}
}

// GetTableNames возвращает список имен таблиц в базе данных
func (r *CronosFileReader) GetTableNames(dbPath string) ([]string, error) {
	var tableNames []string
	
	err := filepath.Walk(dbPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && r.isCronosFile(path) {
			tableName := GetTableName(path)
			tableNames = append(tableNames, tableName)
		}
		
		return nil
	})
	
	return tableNames, err
}

// isCronosFile проверяет, является ли файл файлом базы данных Cronos
func (r *CronosFileReader) isCronosFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	baseName := strings.ToLower(filepath.Base(path))
	
	// Проверяем характерные расширения и имена файлов Cronos
	cronosExtensions := []string{".db", ".dat", ".idx", ".fld", ".csv", ".txt"}
	for _, cronExt := range cronosExtensions {
		if ext == cronExt {
			return true
		}
	}
	
	// Проверяем характерные имена файлов Cronos
	cronosNames := []string{"crostru.dat", "crobank.dat", "croindex.dat"}
	for _, cronName := range cronosNames {
		if baseName == cronName {
			return true
		}
	}
	
	return false
}

// isSQLiteFile проверяет, является ли файл SQLite базой данных
func (r *CronosFileReader) isSQLiteFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	
	// Читаем первые 16 байт для проверки SQLite магического числа
	header := make([]byte, 16)
	_, err = file.Read(header)
	if err != nil {
		return false
	}
	
	return string(header[:16]) == "SQLite format 3\x00"
}

// readCSVTable читает таблицу из CSV файла
func (r *CronosFileReader) readCSVTable(tablePath string) (*Table, error) {
	file, err := os.Open(tablePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть CSV файл: %w", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	var lines []string
	
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при чтении CSV файла: %w", err)
	}
	
	if len(lines) == 0 {
		return nil, fmt.Errorf("CSV файл пуст")
	}
	
	// Первая строка - заголовки
	headers := strings.Split(lines[0], ",")
	columns := make([]Column, len(headers))
	for i, header := range headers {
		columns[i] = Column{
			Name: strings.TrimSpace(header),
			Type: "String",
		}
	}
	
	// Остальные строки - данные
	var data [][]string
	for i := 1; i < len(lines); i++ {
		row := strings.Split(lines[i], ",")
		// Приводим все значения к строковому типу
		stringRow := make([]string, len(row))
		for j, value := range row {
			stringRow[j] = strings.TrimSpace(value)
		}
		data = append(data, stringRow)
	}
	
	folderName, tableName := r.extractFolderAndTableName(tablePath)
	
	return &Table{
		Name:       GetTableName(tablePath),
		Columns:    columns,
		Data:       data,
		SourcePath: tablePath,
		FolderName: folderName,
		TableName:  tableName,
	}, nil
}

// readTextTable читает таблицу из текстового файла
func (r *CronosFileReader) readTextTable(tablePath string) (*Table, error) {
	file, err := os.Open(tablePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть текстовый файл: %w", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	var lines []string
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при чтении текстового файла: %w", err)
	}
	
	if len(lines) == 0 {
		return nil, fmt.Errorf("текстовый файл пуст")
	}
	
	// Для текстовых файлов создаем простую структуру с одной колонкой
	columns := []Column{
		{Name: "content", Type: "String"},
	}
	
	var data [][]string
	for _, line := range lines {
		data = append(data, []string{line})
	}
	
	folderName, tableName := r.extractFolderAndTableName(tablePath)
	
	return &Table{
		Name:       GetTableName(tablePath),
		Columns:    columns,
		Data:       data,
		SourcePath: tablePath,
		FolderName: folderName,
		TableName:  tableName,
	}, nil
}

// readSQLiteTable читает таблицу из SQLite файла (если есть SQLite драйвер)
func (r *CronosFileReader) readSQLiteTable(tablePath string) (*Table, error) {
	// Для SQLite нужен специальный драйвер
	// Пока что возвращаем ошибку, можно реализовать позже
	return nil, fmt.Errorf("чтение SQLite файлов пока не поддерживается")
}

// readCronosBinaryTable читает бинарную таблицу Cronos
func (r *CronosFileReader) readCronosBinaryTable(tablePath string) (*Table, error) {
	file, err := os.Open(tablePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть бинарный файл Cronos: %w", err)
	}
	defer file.Close()
	
	// Простая реализация для чтения бинарных файлов Cronos
	// В реальности формат более сложный и требует детального анализа
	return r.readBinaryAsHexDump(tablePath, file)
}

// readBinaryAsHexDump читает бинарный файл как hex dump для отладки
func (r *CronosFileReader) readBinaryAsHexDump(tablePath string, file *os.File) (*Table, error) {
	// Читаем первые несколько байт для анализа
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("ошибка при чтении бинарного файла: %w", err)
	}
	
	if n == 0 {
		return nil, fmt.Errorf("бинарный файл пуст")
	}
	
	// Создаем простую структуру для отображения hex данных
	columns := []Column{
		{Name: "offset", Type: "String"},
		{Name: "hex_data", Type: "String"},
		{Name: "ascii_data", Type: "String"},
	}
	
	var data [][]string
	for i := 0; i < n; i += 16 {
		end := i + 16
		if end > n {
			end = n
		}
		
		chunk := buffer[i:end]
		
		// Hex представление
		hexStr := ""
		asciiStr := ""
		for _, b := range chunk {
			hexStr += fmt.Sprintf("%02x ", b)
			if b >= 32 && b <= 126 {
				asciiStr += string(b)
			} else {
				asciiStr += "."
			}
		}
		
		row := []string{
			fmt.Sprintf("%08x", i),
			hexStr,
			asciiStr,
		}
		data = append(data, row)
	}
	
	folderName, tableName := r.extractFolderAndTableName(tablePath)
	
	return &Table{
		Name:       GetTableName(tablePath),
		Columns:    columns,
		Data:       data,
		SourcePath: tablePath,
		FolderName: folderName,
		TableName:  tableName,
	}, nil
}

// extractFolderAndTableName извлекает имя папки и таблицы из пути
func (r *CronosFileReader) extractFolderAndTableName(tablePath string) (folderName, tableName string) {
	// Убираем расширение
	base := strings.TrimSuffix(filepath.Base(tablePath), filepath.Ext(tablePath))
	dir := filepath.Base(filepath.Dir(tablePath))
	
	return dir, base
}

// readBinaryUint32 читает 32-битное число из бинарного потока
func readBinaryUint32(reader io.Reader) (uint32, error) {
	var value uint32
	err := binary.Read(reader, binary.LittleEndian, &value)
	return value, err
}

// readBinaryString читает строку из бинарного потока
func readBinaryString(reader io.Reader, length int) (string, error) {
	buffer := make([]byte, length)
	_, err := io.ReadFull(reader, buffer)
	if err != nil {
		return "", err
	}
	
	// Убираем null-терминаторы
	nullIndex := strings.Index(string(buffer), "\x00")
	if nullIndex != -1 {
		return string(buffer[:nullIndex]), nil
	}
	
	return string(buffer), nil
}