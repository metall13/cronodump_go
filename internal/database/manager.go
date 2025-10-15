package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type Manager struct {
	logger Logger
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

type Logger interface {
	Info(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})
	LogProgress(jobID string, message string)
	LogError(jobID string, err error)
}

func NewManager(logger Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// ConnectToDatabase подключается к базе данных
func (m *Manager) ConnectToDatabase(dbInfo *DatabaseInfo) (*sql.DB, error) {
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
	
	m.logger.Info("Успешное подключение к базе данных:", dbInfo.Name)
	return db, nil
}

// GetTables получает список таблиц из базы данных
func (m *Manager) GetTables(db *sql.DB, dbType string) ([]TableInfo, error) {
	var query string
	var tables []TableInfo
	
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
			m.logger.Error("Ошибка сканирования таблицы:", err)
			continue
		}
		
		// Получаем колонки для таблицы
		columns, err := m.GetColumns(db, table.Name, dbType)
		if err != nil {
			m.logger.Error("Ошибка получения колонок для таблицы", table.Name, ":", err)
			table.Columns = []ColumnInfo{}
		} else {
			table.Columns = columns
		}
		
		tables = append(tables, table)
	}
	
	return tables, nil
}

// GetColumns получает информацию о колонках таблицы
func (m *Manager) GetColumns(db *sql.DB, tableName, dbType string) ([]ColumnInfo, error) {
	var query string
	var columns []ColumnInfo
	
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
			m.logger.Error("Ошибка сканирования колонки:", err)
			continue
		}
		
		columns = append(columns, column)
	}
	
	return columns, nil
}

// TestConnection проверяет подключение к базе данных
func (m *Manager) TestConnection(dbInfo *DatabaseInfo) error {
	db, err := m.ConnectToDatabase(dbInfo)
	if err != nil {
		return err
	}
	defer db.Close()
	
	return nil
}