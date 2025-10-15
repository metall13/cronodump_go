package api

import (
	"cronodump-go/internal/database"
	"cronodump-go/internal/processor"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type API struct {
	logger        Logger
	dbManager     *database.Manager
	dataProcessor *processor.Processor
	upgrader      websocket.Upgrader
	jobs          map[string]*processor.JobStatus
}

type Logger interface {
	Info(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})
	LogProgress(jobID string, message string)
	LogError(jobID string, err error)
}

type DatabaseRequest struct {
	Databases []database.DatabaseInfo `json:"databases"`
	TempDir   string                  `json:"temp_dir,omitempty"`
}

type JobResponse struct {
	JobID string `json:"job_id"`
	Status string `json:"status"`
	Message string `json:"message"`
}

func New(logger Logger, dbManager *database.Manager, dataProcessor *processor.Processor) *API {
	return &API{
		logger:        logger,
		dbManager:     dbManager,
		dataProcessor: dataProcessor,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		jobs: make(map[string]*processor.JobStatus),
	}
}

func (a *API) SetupRoutes() *mux.Router {
	router := mux.NewRouter()
	
	// CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})
	
	// API routes
	router.HandleFunc("/api/databases/test", a.testDatabaseConnection).Methods("POST")
	router.HandleFunc("/api/databases/tables", a.getDatabaseTables).Methods("POST")
	router.HandleFunc("/api/process", a.processDatabases).Methods("POST")
	router.HandleFunc("/api/jobs/{jobId}", a.getJobStatus).Methods("GET")
	router.HandleFunc("/api/jobs/{jobId}/download", a.downloadResult).Methods("GET")
	router.HandleFunc("/api/jobs", a.listJobs).Methods("GET")
	router.HandleFunc("/ws", a.handleWebSocket).Methods("GET")
	
	// Serve static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/")))
	
	return router
}

// testDatabaseConnection тестирует подключение к базе данных
func (a *API) testDatabaseConnection(w http.ResponseWriter, r *http.Request) {
	var dbInfo database.DatabaseInfo
	if err := json.NewDecoder(r.Body).Decode(&dbInfo); err != nil {
		http.Error(w, "Ошибка декодирования JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := a.dbManager.TestConnection(&dbInfo); err != nil {
		http.Error(w, "Ошибка подключения: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Подключение успешно"})
}

// getDatabaseTables получает список таблиц из базы данных
func (a *API) getDatabaseTables(w http.ResponseWriter, r *http.Request) {
	var dbInfo database.DatabaseInfo
	if err := json.NewDecoder(r.Body).Decode(&dbInfo); err != nil {
		http.Error(w, "Ошибка декодирования JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	db, err := a.dbManager.ConnectToDatabase(&dbInfo)
	if err != nil {
		http.Error(w, "Ошибка подключения: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	
	tables, err := a.dbManager.GetTables(db, dbInfo.Type)
	if err != nil {
		http.Error(w, "Ошибка получения таблиц: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tables)
}

// processDatabases запускает обработку баз данных
func (a *API) processDatabases(w http.ResponseWriter, r *http.Request) {
	var req DatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ошибка декодирования JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	if len(req.Databases) == 0 {
		http.Error(w, "Список баз данных не может быть пустым", http.StatusBadRequest)
		return
	}
	
	// Генерируем ID задачи
	jobID := fmt.Sprintf("job_%d", time.Now().Unix())
	
	// Устанавливаем временную директорию по умолчанию
	if req.TempDir == "" {
		req.TempDir = "/tmp/cronodump"
	}
	
	// Запускаем обработку в отдельной горутине
	go func() {
		status, err := a.dataProcessor.ProcessDatabases(jobID, req.Databases, req.TempDir)
		if err != nil {
			status.Status = "failed"
			status.Error = err.Error()
			a.logger.LogError(jobID, err)
		}
		a.jobs[jobID] = status
	}()
	
	// Создаем начальный статус задачи
	initialStatus := &processor.JobStatus{
		ID:        jobID,
		Status:    "pending",
		Progress:  0,
		Message:   "Задача создана, ожидает запуска",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	a.jobs[jobID] = initialStatus
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JobResponse{
		JobID:   jobID,
		Status:  "pending",
		Message: "Задача создана успешно",
	})
}

// getJobStatus получает статус задачи
func (a *API) getJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobId"]
	
	status, exists := a.jobs[jobID]
	if !exists {
		http.Error(w, "Задача не найдена", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// downloadResult скачивает результат обработки
func (a *API) downloadResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobId"]
	
	status, exists := a.jobs[jobID]
	if !exists {
		http.Error(w, "Задача не найдена", http.StatusNotFound)
		return
	}
	
	if status.Status != "completed" {
		http.Error(w, "Задача еще не завершена", http.StatusBadRequest)
		return
	}
	
	if status.OutputFile == "" {
		http.Error(w, "Файл результата не найден", http.StatusNotFound)
		return
	}
	
	// Проверяем существование файла
	if _, err := os.Stat(status.OutputFile); os.IsNotExist(err) {
		http.Error(w, "Файл не существует", http.StatusNotFound)
		return
	}
	
	// Устанавливаем заголовки для скачивания
	filename := filepath.Base(status.OutputFile)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "text/csv")
	
	// Отправляем файл
	http.ServeFile(w, r, status.OutputFile)
}

// listJobs получает список всех задач
func (a *API) listJobs(w http.ResponseWriter, r *http.Request) {
	jobs := make([]*processor.JobStatus, 0, len(a.jobs))
	for _, job := range a.jobs {
		jobs = append(jobs, job)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

// handleWebSocket обрабатывает WebSocket соединения для real-time обновлений
func (a *API) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		a.logger.Error("Ошибка обновления WebSocket:", err)
		return
	}
	defer conn.Close()
	
	// Отправляем обновления каждые 5 секунд
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Отправляем все активные задачи
			activeJobs := make([]*processor.JobStatus, 0)
			for _, job := range a.jobs {
				if job.Status == "running" || job.Status == "pending" {
					activeJobs = append(activeJobs, job)
				}
			}
			
			if err := conn.WriteJSON(activeJobs); err != nil {
				a.logger.Error("Ошибка отправки WebSocket сообщения:", err)
				return
			}
		}
	}
}