package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/nakagami/firebirdsql"
)

// Column представляет колонку таблицы
type Column struct {
	Name string
	Type string
}

// KronosParser парсер для баз данных Kronos (Firebird)
type KronosParser struct {
	db       *sql.DB
	dbPath   string
}

// NewKronosParser создает новый парсер для базы данных Kronos
func NewKronosParser(dbPath string) (*KronosParser, error) {
	// Строка подключения для Firebird
	dsn := fmt.Sprintf("firebirdsql://sysdba:masterkey@localhost:3050/%s", dbPath)
	
	db, err := sql.Open("firebirdsql", dsn)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть базу данных: %w", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	return &KronosParser{
		db:     db,
		dbPath: dbPath,
	}, nil
}

// Close закрывает подключение к базе данных
func (kp *KronosParser) Close() error {
	if kp.db != nil {
		return kp.db.Close()
	}
	return nil
}

// GetTables возвращает список всех таблиц в базе данных
func (kp *KronosParser) GetTables() ([]string, error) {
	query := `
		SELECT DISTINCT RDB$RELATION_NAME
		FROM RDB$RELATIONS
		WHERE RDB$SYSTEM_FLAG = 0
		AND RDB$RELATION_TYPE = 0
		ORDER BY RDB$RELATION_NAME
	`

	rows, err := kp.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список таблиц: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("не удалось прочитать имя таблицы: %w", err)
		}
		// Убираем пробелы из имени таблицы (особенность Firebird)
		tableName = strings.TrimSpace(tableName)
		if tableName != "" {
			tables = append(tables, tableName)
		}
	}

	return tables, rows.Err()
}

// GetTableStructure возвращает структуру таблицы (список колонок)
func (kp *KronosParser) GetTableStructure(tableName string) ([]Column, error) {
	query := `
		SELECT 
			RF.RDB$FIELD_NAME,
			CASE F.RDB$FIELD_TYPE
				WHEN 7 THEN 'SMALLINT'
				WHEN 8 THEN 'INTEGER'
				WHEN 9 THEN 'QUAD'
				WHEN 10 THEN 'FLOAT'
				WHEN 11 THEN 'D_FLOAT'
				WHEN 12 THEN 'DATE'
				WHEN 13 THEN 'TIME'
				WHEN 14 THEN 'CHAR'
				WHEN 16 THEN 'BIGINT'
				WHEN 17 THEN 'BOOLEAN'
				WHEN 18 THEN 'DECFLOAT'
				WHEN 19 THEN 'DECFLOAT'
				WHEN 23 THEN 'BOOLEAN'
				WHEN 24 THEN 'DECFLOAT'
				WHEN 25 THEN 'DECFLOAT'
				WHEN 26 THEN 'INT128'
				WHEN 27 THEN 'DOUBLE'
				WHEN 35 THEN 'TIMESTAMP'
				WHEN 37 THEN 'VARCHAR'
				WHEN 40 THEN 'CSTRING'
				WHEN 45 THEN 'BLOB_ID'
				WHEN 261 THEN 'BLOB'
				ELSE 'UNKNOWN'
			END AS FIELD_TYPE
		FROM RDB$RELATION_FIELDS RF
		JOIN RDB$FIELDS F ON RF.RDB$FIELD_SOURCE = F.RDB$FIELD_NAME
		WHERE RF.RDB$RELATION_NAME = ?
		ORDER BY RF.RDB$FIELD_POSITION
	`

	rows, err := kp.db.Query(query, strings.ToUpper(tableName))
	if err != nil {
		return nil, fmt.Errorf("не удалось получить структуру таблицы %s: %w", tableName, err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var fieldName, fieldType string
		if err := rows.Scan(&fieldName, &fieldType); err != nil {
			return nil, fmt.Errorf("не удалось прочитать информацию о поле: %w", err)
		}
		
		columns = append(columns, Column{
			Name: strings.TrimSpace(fieldName),
			Type: fieldType,
		})
	}

	return columns, rows.Err()
}

// GetTableData возвращает все данные из таблицы
func (kp *KronosParser) GetTableData(tableName string) ([][]string, error) {
	// Сначала получаем структуру таблицы для формирования запроса
	columns, err := kp.GetTableStructure(tableName)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить структуру таблицы: %w", err)
	}

	if len(columns) == 0 {
		return [][]string{}, nil
	}

	// Формируем список колонок для SELECT
	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, fmt.Sprintf(`"%s"`, col.Name))
	}

	query := fmt.Sprintf(`SELECT %s FROM "%s"`, strings.Join(columnNames, ", "), tableName)

	rows, err := kp.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("не удалось выполнить запрос к таблице %s: %w", tableName, err)
	}
	defer rows.Close()

	var data [][]string
	for rows.Next() {
		// Создаем слайс для значений
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("не удалось прочитать строку: %w", err)
		}

		// Преобразуем значения в строки
		row := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = ""
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}

		data = append(data, row)
	}

	return data, rows.Err()
}

// GetRowCount возвращает количество записей в таблице
func (kp *KronosParser) GetRowCount(tableName string) (int64, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, tableName)
	
	var count int64
	err := kp.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("не удалось получить количество записей в таблице %s: %w", tableName, err)
	}

	return count, nil
}

// TestConnection проверяет подключение к базе данных
func (kp *KronosParser) TestConnection() error {
	return kp.db.Ping()
}

// GetDatabaseInfo возвращает информацию о базе данных
func (kp *KronosParser) GetDatabaseInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})
	
	// Получаем версию Firebird
	var version string
	err := kp.db.QueryRow("SELECT RDB$GET_CONTEXT('SYSTEM', 'ENGINE_VERSION') FROM RDB$DATABASE").Scan(&version)
	if err == nil {
		info["firebird_version"] = strings.TrimSpace(version)
	}

	// Получаем количество таблиц
	tables, err := kp.GetTables()
	if err == nil {
		info["table_count"] = len(tables)
		info["tables"] = tables
	}

	info["database_path"] = kp.dbPath

	return info, nil
}