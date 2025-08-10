package cronodamp

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// Exporter основной экспортер данных из Kronos в ClickHouse
type Exporter struct {
	config        Config
	kronosReader  *KronosReader
	clickHouse    *ClickHouseClient
	ctx           context.Context
	cancel        context.CancelFunc
	
	// Статистика
	stats         ExportStats
	statsMutex    sync.Mutex
}

// ExportStats статистика экспорта
type ExportStats struct {
	TablesProcessed int
	TablesTotal     int
	RowsProcessed   int64
	ErrorsCount     int
	StartTime       time.Time
	Duration        time.Duration
	LastError       error
}

// NewExporter создает новый экспортер
func NewExporter(config Config) (*Exporter, error) {
	// Валидируем конфигурацию
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("ошибка конфигурации: %w", err)
	}
	
	// Создаем читатель Kronos
	kronosReader := NewKronosReader(config.KronosDBPath, config.Debug)
	
	// Создаем клиент ClickHouse
	clickHouse, err := NewClickHouseClient(config)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать клиент ClickHouse: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Exporter{
		config:       config,
		kronosReader: kronosReader,
		clickHouse:   clickHouse,
		ctx:          ctx,
		cancel:       cancel,
		stats: ExportStats{
			StartTime: time.Now(),
		},
	}, nil
}

// ExportAll экспортирует все найденные таблицы Kronos в ClickHouse
func (e *Exporter) ExportAll() error {
	e.stats.StartTime = time.Now()
	defer func() {
		e.stats.Duration = time.Since(e.stats.StartTime)
	}()
	
	// Находим все таблицы
	tables, err := e.kronosReader.DiscoverTables()
	if err != nil {
		return fmt.Errorf("не удалось найти таблицы: %w", err)
	}
	
	e.stats.TablesTotal = len(tables)
	
	if e.config.Debug {
		fmt.Printf("Найдено %d таблиц для экспорта\n", len(tables))
	}
	
	// Экспортируем каждую таблицу
	for i, table := range tables {
		select {
		case <-e.ctx.Done():
			return fmt.Errorf("экспорт отменен")
		default:
		}
		
		fmt.Printf("[%d/%d] Экспортируем таблицу: %s/%s\n", 
			i+1, len(tables), table.FolderName, table.TableName)
		
		if err := e.ExportTable(table); err != nil {
			e.updateStats(0, 1, err)
			if e.config.Debug {
				fmt.Printf("Ошибка экспорта таблицы %s/%s: %v\n", 
					table.FolderName, table.TableName, err)
			}
			continue // Продолжаем с остальными таблицами
		}
		
		e.updateStats(1, 0, nil)
	}
	
	return nil
}

// ExportTable экспортирует конкретную таблицу
func (e *Exporter) ExportTable(table KronosTable) error {
	// Создаем таблицу в ClickHouse
	if err := e.clickHouse.CreateTable(e.ctx, table); err != nil {
		return fmt.Errorf("не удалось создать таблицу: %w", err)
	}
	
	// Читаем данные из Kronos
	rows, err := e.kronosReader.ReadTableData(table)
	if err != nil {
		return fmt.Errorf("не удалось прочитать данные: %w", err)
	}
	
	if len(rows) == 0 {
		if e.config.Debug {
			fmt.Printf("Таблица %s/%s пустая, пропускаем\n", 
				table.FolderName, table.TableName)
		}
		return nil
	}
	
	// Вставляем данные батчами
	batchSize := e.config.BatchSize
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		
		batch := rows[i:end]
		if err := e.clickHouse.InsertData(e.ctx, table, batch); err != nil {
			return fmt.Errorf("не удалось вставить данные (batch %d-%d): %w", i, end, err)
		}
		
		e.statsMutex.Lock()
		e.stats.RowsProcessed += int64(len(batch))
		e.statsMutex.Unlock()
		
		if e.config.Debug {
			fmt.Printf("Вставлено строк: %d-%d из %d\n", i+1, end, len(rows))
		}
	}
	
	fmt.Printf("✓ Таблица %s/%s: %d строк экспортировано\n", 
		table.FolderName, table.TableName, len(rows))
	
	return nil
}

// ExportSpecific экспортирует конкретные таблицы по путям файлов
func (e *Exporter) ExportSpecific(filePaths []string) error {
	e.stats.StartTime = time.Now()
	defer func() {
		e.stats.Duration = time.Since(e.stats.StartTime)
	}()
	
	e.stats.TablesTotal = len(filePaths)
	
	for i, filePath := range filePaths {
		select {
		case <-e.ctx.Done():
			return fmt.Errorf("экспорт отменен")
		default:
		}
		
		// Проверяем существование файла
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			e.updateStats(0, 1, fmt.Errorf("файл не найден: %s", filePath))
			continue
		}
		
		// Получаем информацию о таблице
		table, err := e.kronosReader.GetTableByPath(filePath)
		if err != nil {
			e.updateStats(0, 1, err)
			continue
		}
		
		fmt.Printf("[%d/%d] Экспортируем файл: %s\n", i+1, len(filePaths), filePath)
		
		if err := e.ExportTable(*table); err != nil {
			e.updateStats(0, 1, err)
			continue
		}
		
		e.updateStats(1, 0, nil)
	}
	
	return nil
}

// GetStats возвращает статистику экспорта
func (e *Exporter) GetStats() ExportStats {
	e.statsMutex.Lock()
	defer e.statsMutex.Unlock()
	
	stats := e.stats
	if !stats.StartTime.IsZero() && stats.Duration == 0 {
		stats.Duration = time.Since(stats.StartTime)
	}
	
	return stats
}

// PrintStats выводит статистику экспорта
func (e *Exporter) PrintStats() {
	stats := e.GetStats()
	
	fmt.Println("\n=== Статистика экспорта ===")
	fmt.Printf("Таблиц обработано: %d из %d\n", stats.TablesProcessed, stats.TablesTotal)
	fmt.Printf("Строк обработано: %d\n", stats.RowsProcessed)
	fmt.Printf("Ошибок: %d\n", stats.ErrorsCount)
	fmt.Printf("Время выполнения: %v\n", stats.Duration.Round(time.Second))
	
	if stats.RowsProcessed > 0 && stats.Duration > 0 {
		rps := float64(stats.RowsProcessed) / stats.Duration.Seconds()
		fmt.Printf("Скорость: %.0f строк/сек\n", rps)
	}
	
	if stats.LastError != nil {
		fmt.Printf("Последняя ошибка: %v\n", stats.LastError)
	}
	
	fmt.Println("===========================")
}

// Close закрывает экспортер и освобождает ресурсы
func (e *Exporter) Close() error {
	e.cancel()
	if e.clickHouse != nil {
		return e.clickHouse.Close()
	}
	return nil
}

// Cancel отменяет текущий экспорт
func (e *Exporter) Cancel() {
	e.cancel()
}

// updateStats обновляет статистику (потокобезопасно)
func (e *Exporter) updateStats(processedTables, errors int, lastError error) {
	e.statsMutex.Lock()
	defer e.statsMutex.Unlock()
	
	e.stats.TablesProcessed += processedTables
	e.stats.ErrorsCount += errors
	if lastError != nil {
		e.stats.LastError = lastError
	}
}

// ListTables возвращает список всех найденных таблиц без экспорта
func (e *Exporter) ListTables() ([]KronosTable, error) {
	return e.kronosReader.DiscoverTables()
}

// ExportFolder экспортирует все таблицы из указанной папки
func (e *Exporter) ExportFolder(folderName string) error {
	tables, err := e.kronosReader.DiscoverTables()
	if err != nil {
		return err
	}
	
	var selectedTables []KronosTable
	for _, table := range tables {
		if table.FolderName == folderName {
			selectedTables = append(selectedTables, table)
		}
	}
	
	if len(selectedTables) == 0 {
		return fmt.Errorf("не найдено таблиц в папке: %s", folderName)
	}
	
	e.stats.StartTime = time.Now()
	e.stats.TablesTotal = len(selectedTables)
	defer func() {
		e.stats.Duration = time.Since(e.stats.StartTime)
	}()
	
	fmt.Printf("Экспортируем %d таблиц из папки: %s\n", len(selectedTables), folderName)
	
	for i, table := range selectedTables {
		select {
		case <-e.ctx.Done():
			return fmt.Errorf("экспорт отменен")
		default:
		}
		
		fmt.Printf("[%d/%d] Экспортируем таблицу: %s\n", 
			i+1, len(selectedTables), table.TableName)
		
		if err := e.ExportTable(table); err != nil {
			e.updateStats(0, 1, err)
			continue
		}
		
		e.updateStats(1, 0, nil)
	}
	
	return nil
}

// ValidateConnection проверяет подключение к ClickHouse
func (e *Exporter) ValidateConnection() error {
	return e.clickHouse.Ping(e.ctx)
}

// GetClickHouseTableName возвращает имя таблицы в ClickHouse для заданной таблицы Kronos
func (e *Exporter) GetClickHouseTableName(table KronosTable) string {
	return e.clickHouse.GetTableName(table)
}