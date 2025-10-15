package processor

import (
	"cronodump-go/internal/transliteration"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Processor struct {
	logger         Logger
	transliterator *transliteration.Transliterator
}

type Logger interface {
	Info(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})
	LogProgress(jobID string, message string)
	LogError(jobID string, err error)
}

type JobStatus struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"` // pending, running, completed, failed
	Progress     int       `json:"progress"`
	Message      string    `json:"message"`
	DatabasesFound int     `json:"databases_found"`
	DatabasesProcessed int `json:"databases_processed"`
	TablesFound  int       `json:"tables_found"`
	TablesProcessed int    `json:"tables_processed"`
	RecordsProcessed int64 `json:"records_processed"`
	OutputFile   string    `json:"output_file"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Error        string    `json:"error,omitempty"`
}

type DatabaseInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Database    string    `json:"database"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	Status      string    `json:"status"`
	TablesCount int       `json:"tables_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type TableInfo struct {
	Name      string            `json:"name"`
	Columns   []ColumnInfo      `json:"columns"`
	RowCount  int64             `json:"row_count"`
	Size      int64             `json:"size"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	IsNullable   bool   `json:"is_nullable"`
	IsPrimaryKey bool   `json:"is_primary_key"`
	IsForeignKey bool   `json:"is_foreign_key"`
	DefaultValue string `json:"default_value"`
}

func New(logger Logger, transliterator *transliteration.Transliterator) *Processor {
	return &Processor{
		logger:         logger,
		transliterator: transliterator,
	}
}

// ProcessDatabases обрабатывает список баз данных и объединяет их в одну таблицу
func (p *Processor) ProcessDatabases(jobID string, databases []DatabaseInfo, tempDir string) (*JobStatus, error) {
	status := &JobStatus{
		ID:        jobID,
		Status:    "running",
		Progress:  0,
		Message:   "Начинаем обработку баз данных",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	p.logger.LogProgress(jobID, "Начинаем обработку баз данных")
	
	// Создаем временную директорию
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		status.Status = "failed"
		status.Error = fmt.Sprintf("Ошибка создания временной директории: %v", err)
		return status, err
	}
	
	// Создаем выходной файл
	outputFile := filepath.Join(tempDir, fmt.Sprintf("cronodump_%s_%d.csv", jobID, time.Now().Unix()))
	file, err := os.Create(outputFile)
	if err != nil {
		status.Status = "failed"
		status.Error = fmt.Sprintf("Ошибка создания выходного файла: %v", err)
		return status, err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Собираем все уникальные колонки из всех таблиц
	allColumns := make(map[string]bool)
	allTables := make([]TableInfo, 0)
	
	status.DatabasesFound = len(databases)
	
	for i, dbInfo := range databases {
		p.logger.LogProgress(jobID, fmt.Sprintf("Обрабатываем базу данных %d/%d: %s", i+1, len(databases), dbInfo.Name))
		
		// Подключаемся к базе данных
		db, err := p.connectToDatabase(dbInfo)
		if err != nil {
			p.logger.LogError(jobID, fmt.Errorf("ошибка подключения к базе %s: %v", dbInfo.Name, err))
			continue
		}
		defer db.Close()
		
		// Получаем список таблиц
		tables, err := p.getTables(db, dbInfo.Type)
		if err != nil {
			p.logger.LogError(jobID, fmt.Errorf("ошибка получения таблиц из базы %s: %v", dbInfo.Name, err))
			continue
		}
		
		status.TablesFound += len(tables)
		allTables = append(allTables, tables...)
		
		// Собираем колонки
		for _, table := range tables {
			for _, column := range table.Columns {
				transliteratedName := p.transliterator.TransliterateFieldName(column.Name)
				allColumns[transliteratedName] = true
			}
		}
		
		status.DatabasesProcessed++
		status.Progress = int((float64(i+1) / float64(len(databases))) * 100)
		status.UpdatedAt = time.Now()
	}
	
	// Создаем заголовки CSV
	headers := make([]string, 0, len(allColumns)+3) // +3 для source_db, source_table, record_id
	headers = append(headers, "source_db", "source_table", "record_id")
	
	for columnName := range allColumns {
		headers = append(headers, columnName)
	}
	
	if err := writer.Write(headers); err != nil {
		status.Status = "failed"
		status.Error = fmt.Sprintf("Ошибка записи заголовков: %v", err)
		return status, err
	}
	
	// Обрабатываем каждую таблицу
	recordID := int64(1)
	for _, table := range allTables {
		p.logger.LogProgress(jobID, fmt.Sprintf("Обрабатываем таблицу: %s", table.Name))
		
		// Подключаемся к базе данных таблицы
		var db *sql.DB
		for _, dbInfo := range databases {
			if dbInfo.Name == table.Name { // Это упрощение, в реальности нужно отслеживать связь
				db, err = p.connectToDatabase(dbInfo)
				if err != nil {
					p.logger.LogError(jobID, fmt.Errorf("ошибка подключения: %v", err))
					continue
				}
				break
			}
		}
		
		if db == nil {
			continue
		}
		defer db.Close()
		
		// Получаем данные из таблицы
		rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", table.Name))
		if err != nil {
			p.logger.LogError(jobID, fmt.Errorf("ошибка получения данных из таблицы %s: %v", table.Name, err))
			continue
		}
		defer rows.Close()
		
		// Получаем колонки
		columns, err := rows.Columns()
		if err != nil {
			p.logger.LogError(jobID, fmt.Errorf("ошибка получения колонок: %v", err))
			continue
		}
		
		// Создаем мапу колонок для быстрого поиска
		columnMap := make(map[string]int)
		for i, col := range columns {
			transliteratedName := p.transliterator.TransliterateFieldName(col)
			columnMap[transliteratedName] = i
		}
		
		// Обрабатываем каждую строку
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			
			if err := rows.Scan(valuePtrs...); err != nil {
				p.logger.LogError(jobID, fmt.Errorf("ошибка сканирования строки: %v", err))
				continue
			}
			
			// Создаем строку для CSV
			row := make([]string, len(headers))
			row[0] = "unknown_db" // source_db
			row[1] = p.transliterator.TransliterateTableName(table.Name) // source_table
			row[2] = strconv.FormatInt(recordID, 10) // record_id
			
			// Заполняем данные
			for i := 3; i < len(headers); i++ {
				columnName := headers[i]
				if colIndex, exists := columnMap[columnName]; exists {
					row[i] = p.convertToString(values[colIndex])
				} else {
					row[i] = ""
				}
			}
			
			if err := writer.Write(row); err != nil {
				p.logger.LogError(jobID, fmt.Errorf("ошибка записи строки: %v", err))
				continue
			}
			
			recordID++
			status.RecordsProcessed++
			
			// Обновляем прогресс
			if recordID%1000 == 0 {
				status.Progress = int((float64(recordID) / float64(status.RecordsProcessed+1000)) * 100)
				status.UpdatedAt = time.Now()
			}
		}
		
		status.TablesProcessed++
	}
	
	status.Status = "completed"
	status.Progress = 100
	status.Message = "Обработка завершена успешно"
	status.OutputFile = outputFile
	status.UpdatedAt = time.Now()
	
	p.logger.LogProgress(jobID, "Обработка завершена успешно")
	
	return status, nil
}

// convertToString конвертирует значение в строку с учетом типа данных
func (p *Processor) convertToString(value interface{}) string {
	if value == nil {
		return ""
	}
	
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(v)
	case time.Time:
		// Форматируем дату в ДД.ММ.ГГГГ
		return v.Format("02.01.2006")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// connectToDatabase подключается к базе данных
func (p *Processor) connectToDatabase(dbInfo DatabaseInfo) (*sql.DB, error) {
	var dsn string
	
	switch dbInfo.Type {
	case "postgres":
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			dbInfo.Host, dbInfo.Port, dbInfo.Username, dbInfo.Password, dbInfo.Database)
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbInfo.Username, dbInfo.Password, dbInfo.Host, dbInfo.Port, dbInfo.Database)
	case "clickhouse":
		dsn = fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s",
			dbInfo.Username, dbInfo.Password, dbInfo.Host, dbInfo.Port, dbInfo.Database)
	default:
		return nil, fmt.Errorf("неподдерживаемый тип базы данных: %s", dbInfo.Type)
	}
	
	db, err := sql.Open(dbInfo.Type, dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}
	
	// Проверяем подключение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка проверки подключения: %v", err)
	}
	
	return db, nil
}

// getTables получает список таблиц из базы данных
func (p *Processor) getTables(db *sql.DB, dbType string) ([]TableInfo, error) {
	var query string
	
	switch dbType {
	case "postgres":
		query = `
			SELECT 
				table_name,
				COALESCE(n_tup_ins + n_tup_upd + n_tup_del, 0) as row_count,
				COALESCE(pg_total_relation_size(quote_ident(table_name)), 0) as size,
				COALESCE(created_at, NOW()) as created_at,
				COALESCE(updated_at, NOW()) as updated_at
			FROM information_schema.tables t
			LEFT JOIN pg_stat_user_tables s ON t.table_name = s.relname
			WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
			ORDER BY table_name`
	case "mysql":
		query = `
			SELECT 
				table_name,
				COALESCE(table_rows, 0) as row_count,
				COALESCE(data_length + index_length, 0) as size,
				COALESCE(create_time, NOW()) as created_at,
				COALESCE(update_time, NOW()) as updated_at
			FROM information_schema.tables
			WHERE table_schema = DATABASE()
			ORDER BY table_name`
	case "clickhouse":
		query = `
			SELECT 
				name as table_name,
				COALESCE(total_rows, 0) as row_count,
				COALESCE(total_bytes, 0) as size,
				COALESCE(created_at, NOW()) as created_at,
				COALESCE(updated_at, NOW()) as updated_at
			FROM system.tables
			WHERE database = currentDatabase()
			ORDER BY name`
	default:
		return nil, fmt.Errorf("неподдерживаемый тип базы данных: %s", dbType)
	}
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка таблиц: %v", err)
	}
	defer rows.Close()
	
	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		err := rows.Scan(
			&table.Name,
			&table.RowCount,
			&table.Size,
			&table.CreatedAt,
			&table.UpdatedAt,
		)
		if err != nil {
			p.logger.Error("Ошибка сканирования таблицы:", err)
			continue
		}
		
		// Получаем колонки для таблицы
		columns, err := p.getColumns(db, table.Name, dbType)
		if err != nil {
			p.logger.Error("Ошибка получения колонок для таблицы", table.Name, ":", err)
			table.Columns = []ColumnInfo{}
		} else {
			table.Columns = columns
		}
		
		tables = append(tables, table)
	}
	
	return tables, nil
}

// getColumns получает информацию о колонках таблицы
func (p *Processor) getColumns(db *sql.DB, tableName, dbType string) ([]ColumnInfo, error) {
	var query string
	
	switch dbType {
	case "postgres":
		query = `
			SELECT 
				column_name,
				data_type,
				is_nullable = 'YES' as is_nullable,
				column_name IN (
					SELECT column_name 
					FROM information_schema.key_column_usage 
					WHERE table_name = $1 AND constraint_name LIKE '%_pkey'
				) as is_primary_key,
				column_name IN (
					SELECT column_name 
					FROM information_schema.key_column_usage 
					WHERE table_name = $1 AND constraint_name LIKE '%_fkey'
				) as is_foreign_key,
				COALESCE(column_default, '') as default_value
			FROM information_schema.columns
			WHERE table_name = $1
			ORDER BY ordinal_position`
	case "mysql":
		query = `
			SELECT 
				column_name,
				data_type,
				is_nullable = 'YES' as is_nullable,
				column_key = 'PRI' as is_primary_key,
				column_key = 'MUL' as is_foreign_key,
				COALESCE(column_default, '') as default_value
			FROM information_schema.columns
			WHERE table_name = ?
			ORDER BY ordinal_position`
	case "clickhouse":
		query = `
			SELECT 
				name as column_name,
				type as data_type,
				null as is_nullable,
				position IN (
					SELECT position 
					FROM system.columns 
					WHERE table = ? AND is_in_primary_key = 1
				) as is_primary_key,
				0 as is_foreign_key,
				COALESCE(default_expression, '') as default_value
			FROM system.columns
			WHERE table = ?
			ORDER BY position`
	default:
		return nil, fmt.Errorf("неподдерживаемый тип базы данных: %s", dbType)
	}
	
	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения колонок: %v", err)
	}
	defer rows.Close()
	
	var columns []ColumnInfo
	for rows.Next() {
		var column ColumnInfo
		err := rows.Scan(
			&column.Name,
			&column.Type,
			&column.IsNullable,
			&column.IsPrimaryKey,
			&column.IsForeignKey,
			&column.DefaultValue,
		)
		if err != nil {
			p.logger.Error("Ошибка сканирования колонки:", err)
			continue
		}
		
		columns = append(columns, column)
	}
	
	return columns, nil
}