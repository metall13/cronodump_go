package cronodamp

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Column представляет колонку в таблице
type Column struct {
	Name string
	Type string
}

// KronosTable представляет таблицу из базы Kronos
type KronosTable struct {
	FolderName string   // Имя папки в которой лежит таблица
	TableName  string   // Имя таблицы
	FilePath   string   // Полный путь к файлу
	Columns    []Column // Колонки таблицы
}

// KronosReader читатель данных из баз Kronos
type KronosReader struct {
	dbPath string
	debug  bool
}

// NewKronosReader создает новый читатель для баз Kronos
func NewKronosReader(dbPath string, debug bool) *KronosReader {
	return &KronosReader{
		dbPath: dbPath,
		debug:  debug,
	}
}

// DiscoverTables находит все таблицы в указанной папке
func (kr *KronosReader) DiscoverTables() ([]KronosTable, error) {
	var tables []KronosTable
	
	err := filepath.WalkDir(kr.dbPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// Ищем файлы с данными (предполагаем что это .txt, .csv, .dat файлы)
		if !d.IsDir() && (strings.HasSuffix(path, ".txt") || 
			strings.HasSuffix(path, ".csv") || 
			strings.HasSuffix(path, ".dat")) {
			
			// Получаем имя папки и файла
			relPath, err := filepath.Rel(kr.dbPath, path)
			if err != nil {
				return fmt.Errorf("не удалось получить относительный путь: %w", err)
			}
			
			folderName := filepath.Dir(relPath)
			if folderName == "." {
				folderName = "root" // Для файлов в корневой папке
			}
			
			fileName := filepath.Base(path)
			tableName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
			
			// Читаем структуру таблицы
			columns, err := kr.readTableStructure(path)
			if err != nil {
				if kr.debug {
					fmt.Printf("Предупреждение: не удалось прочитать структуру таблицы %s: %v\n", path, err)
				}
				// Продолжаем, создавая базовую структуру
				columns = []Column{{Name: "data", Type: "String"}}
			}
			
			table := KronosTable{
				FolderName: folderName,
				TableName:  tableName,
				FilePath:   path,
				Columns:    columns,
			}
			
			tables = append(tables, table)
			
			if kr.debug {
				fmt.Printf("Найдена таблица: %s/%s (%d колонок)\n", 
					folderName, tableName, len(columns))
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("ошибка при поиске таблиц: %w", err)
	}
	
	// Сортируем таблицы для предсказуемого порядка
	sort.Slice(tables, func(i, j int) bool {
		if tables[i].FolderName != tables[j].FolderName {
			return tables[i].FolderName < tables[j].FolderName
		}
		return tables[i].TableName < tables[j].TableName
	})
	
	return tables, nil
}

// ReadTableData читает данные из таблицы
func (kr *KronosReader) ReadTableData(table KronosTable) ([][]string, error) {
	file, err := os.Open(table.FilePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл %s: %w", table.FilePath, err)
	}
	defer file.Close()
	
	var rows [][]string
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Пропускаем пустые строки
		if line == "" {
			continue
		}
		
		// Пропускаем комментарии
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		
		// Определяем разделитель (табуляция, точка с запятой, запятая)
		var row []string
		if strings.Contains(line, "\t") {
			row = strings.Split(line, "\t")
		} else if strings.Contains(line, ";") {
			row = strings.Split(line, ";")
		} else if strings.Contains(line, ",") {
			row = strings.Split(line, ",")
		} else {
			// Если разделитель не найден, считаем всю строку одним полем
			row = []string{line}
		}
		
		// Очищаем данные от лишних пробелов
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}
		
		// Приводим количество полей к количеству колонок
		adjustedRow := make([]string, len(table.Columns))
		for i := range adjustedRow {
			if i < len(row) {
				adjustedRow[i] = row[i]
			} else {
				adjustedRow[i] = "" // Заполняем пустыми строками
			}
		}
		
		rows = append(rows, adjustedRow)
		
		if kr.debug && lineNum <= 3 {
			fmt.Printf("Строка %d: %v\n", lineNum, adjustedRow)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при чтении файла %s: %w", table.FilePath, err)
	}
	
	if kr.debug {
		fmt.Printf("Прочитано %d строк из %s\n", len(rows), table.FilePath)
	}
	
	return rows, nil
}

// readTableStructure пытается определить структуру таблицы
func (kr *KronosReader) readTableStructure(filePath string) ([]Column, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	
	// Читаем первые несколько строк для определения структуры
	var sampleLines []string
	for scanner.Scan() && len(sampleLines) < 5 {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "//") {
			sampleLines = append(sampleLines, line)
		}
	}
	
	if len(sampleLines) == 0 {
		return nil, fmt.Errorf("файл пустой или содержит только комментарии")
	}
	
	// Анализируем первую строку для определения количества колонок
	firstLine := sampleLines[0]
	var fieldCount int
	
	if strings.Contains(firstLine, "\t") {
		fieldCount = len(strings.Split(firstLine, "\t"))
	} else if strings.Contains(firstLine, ";") {
		fieldCount = len(strings.Split(firstLine, ";"))
	} else if strings.Contains(firstLine, ",") {
		fieldCount = len(strings.Split(firstLine, ","))
	} else {
		fieldCount = 1
	}
	
	// Создаем колонки с автогенерированными именами
	var columns []Column
	for i := 0; i < fieldCount; i++ {
		columnName := fmt.Sprintf("column_%d", i+1)
		columns = append(columns, Column{
			Name: columnName,
			Type: "String", // Все данные из Kronos сохраняем как строки
		})
	}
	
	return columns, nil
}

// GetTableByPath находит таблицу по пути к файлу
func (kr *KronosReader) GetTableByPath(filePath string) (*KronosTable, error) {
	tables, err := kr.DiscoverTables()
	if err != nil {
		return nil, err
	}
	
	for _, table := range tables {
		if table.FilePath == filePath {
			return &table, nil
		}
	}
	
	return nil, fmt.Errorf("таблица не найдена по пути: %s", filePath)
}