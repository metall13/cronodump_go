package cronodamp

import (
	"time"
)

// Column представляет колонку в таблице
type Column struct {
	Name string
	Type string
}

// Table представляет таблицу в базе данных
type Table struct {
	Name        string
	Columns     []Column
	Data        [][]string
	SourcePath  string
	FolderName  string
	TableName   string
}

// Database представляет базу данных Cronos
type Database struct {
	Path   string
	Tables []Table
}

// ExportResult содержит результат экспорта
type ExportResult struct {
	TableName    string
	RowsExported int
	CreatedAt    time.Time
	Success      bool
	Error        error
}

// CronosReader интерфейс для чтения данных из Cronos
type CronosReader interface {
	ReadDatabase(dbPath string) (*Database, error)
	ReadTable(tablePath string) (*Table, error)
	GetTableNames(dbPath string) ([]string, error)
}

// ClickHouseExporter интерфейс для экспорта в ClickHouse
type ClickHouseExporter interface {
	ExportTable(table *Table) (*ExportResult, error)
	ExportDatabase(db *Database) ([]*ExportResult, error)
}

// FileInfo представляет информацию о файле в базе данных
type FileInfo struct {
	Name         string
	Size         int64
	Path         string
	Referenced   bool
	Content      []byte
}

// DatabaseInfo содержит метаинформацию о базе данных
type DatabaseInfo struct {
	Path            string
	CreatedAt       time.Time
	ModifiedAt      time.Time
	Version         string
	IsPasswordProtected bool
	TableCount      int
	FileCount       int
}