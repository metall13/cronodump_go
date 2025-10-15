class CronodumpApp {
    constructor() {
        this.databases = [];
        this.currentJob = null;
        this.ws = null;
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadJobs();
        this.connectWebSocket();
    }

    setupEventListeners() {
        // Форма добавления базы данных
        document.getElementById('addDatabaseBtn').addEventListener('click', () => this.addDatabase());
        document.getElementById('testConnectionBtn').addEventListener('click', () => this.testConnection());
        document.getElementById('startProcessBtn').addEventListener('click', () => this.startProcess());
        
        // Обработка Enter в форме
        document.getElementById('databaseForm').addEventListener('submit', (e) => {
            e.preventDefault();
            this.addDatabase();
        });
    }

    async addDatabase() {
        const form = document.getElementById('databaseForm');
        const formData = new FormData(form);
        
        const database = {
            name: formData.get('name'),
            type: formData.get('type'),
            host: formData.get('host'),
            port: parseInt(formData.get('port')),
            database: formData.get('database'),
            username: formData.get('username'),
            password: formData.get('password'),
            status: 'pending'
        };

        // Проверяем, что все поля заполнены
        if (!database.name || !database.type || !database.host || !database.port || !database.database || !database.username) {
            this.showNotification('Пожалуйста, заполните все обязательные поля', 'error');
            return;
        }

        // Проверяем, что база данных с таким именем не добавлена
        if (this.databases.find(db => db.name === database.name)) {
            this.showNotification('База данных с таким именем уже добавлена', 'error');
            return;
        }

        this.databases.push(database);
        this.updateDatabasesList();
        this.updateStartButton();
        form.reset();
        
        // Устанавливаем порт по умолчанию в зависимости от типа БД
        this.setDefaultPort();
        
        this.showNotification('База данных добавлена', 'success');
    }

    setDefaultPort() {
        const typeSelect = document.getElementById('dbType');
        const portInput = document.getElementById('dbPort');
        
        const defaultPorts = {
            'postgres': 5432,
            'mysql': 3306,
            'clickhouse': 9000
        };
        
        typeSelect.addEventListener('change', () => {
            portInput.value = defaultPorts[typeSelect.value] || '';
        });
    }

    async testConnection() {
        const form = document.getElementById('databaseForm');
        const formData = new FormData(form);
        
        const database = {
            name: formData.get('name'),
            type: formData.get('type'),
            host: formData.get('host'),
            port: parseInt(formData.get('port')),
            database: formData.get('database'),
            username: formData.get('username'),
            password: formData.get('password')
        };

        if (!database.name || !database.type || !database.host || !database.port || !database.database || !database.username) {
            this.showNotification('Пожалуйста, заполните все обязательные поля', 'error');
            return;
        }

        const testBtn = document.getElementById('testConnectionBtn');
        testBtn.disabled = true;
        testBtn.textContent = 'Тестирование...';

        try {
            const response = await fetch('/api/databases/test', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(database)
            });

            if (response.ok) {
                this.showNotification('Подключение успешно!', 'success');
            } else {
                const error = await response.text();
                this.showNotification('Ошибка подключения: ' + error, 'error');
            }
        } catch (error) {
            this.showNotification('Ошибка: ' + error.message, 'error');
        } finally {
            testBtn.disabled = false;
            testBtn.textContent = 'Тестировать подключение';
        }
    }

    updateDatabasesList() {
        const container = document.getElementById('databasesList');
        container.innerHTML = '';

        this.databases.forEach((db, index) => {
            const card = document.createElement('div');
            card.className = 'database-card';
            card.innerHTML = `
                <h3>${db.name}</h3>
                <div class="database-info">
                    <span><strong>Тип:</strong> ${db.type}</span>
                    <span><strong>Хост:</strong> ${db.host}:${db.port}</span>
                    <span><strong>База данных:</strong> ${db.database}</span>
                    <span><strong>Пользователь:</strong> ${db.username}</span>
                </div>
                <div class="database-actions">
                    <button class="btn btn-danger" onclick="app.removeDatabase(${index})">Удалить</button>
                </div>
            `;
            container.appendChild(card);
        });
    }

    removeDatabase(index) {
        this.databases.splice(index, 1);
        this.updateDatabasesList();
        this.updateStartButton();
        this.showNotification('База данных удалена', 'info');
    }

    updateStartButton() {
        const startBtn = document.getElementById('startProcessBtn');
        startBtn.disabled = this.databases.length === 0;
    }

    async startProcess() {
        if (this.databases.length === 0) {
            this.showNotification('Добавьте хотя бы одну базу данных', 'error');
            return;
        }

        const startBtn = document.getElementById('startProcessBtn');
        startBtn.disabled = true;
        startBtn.textContent = 'Запуск...';

        try {
            const response = await fetch('/api/process', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    databases: this.databases,
                    temp_dir: '/tmp/cronodump'
                })
            });

            if (response.ok) {
                const result = await response.json();
                this.currentJob = result.jobId;
                this.showProgressSection();
                this.showNotification('Обработка запущена', 'success');
            } else {
                const error = await response.text();
                this.showNotification('Ошибка запуска: ' + error, 'error');
            }
        } catch (error) {
            this.showNotification('Ошибка: ' + error.message, 'error');
        } finally {
            startBtn.disabled = false;
            startBtn.textContent = 'Начать обработку';
        }
    }

    showProgressSection() {
        document.getElementById('progressSection').style.display = 'block';
        document.getElementById('progressSection').scrollIntoView({ behavior: 'smooth' });
    }

    updateProgress(jobStatus) {
        if (!jobStatus) return;

        document.getElementById('databasesFound').textContent = jobStatus.databases_found || 0;
        document.getElementById('databasesProcessed').textContent = jobStatus.databases_processed || 0;
        document.getElementById('tablesFound').textContent = jobStatus.tables_found || 0;
        document.getElementById('tablesProcessed').textContent = jobStatus.tables_processed || 0;
        document.getElementById('recordsProcessed').textContent = jobStatus.records_processed || 0;
        
        const progressFill = document.getElementById('progressFill');
        progressFill.style.width = (jobStatus.progress || 0) + '%';
        
        document.getElementById('progressText').textContent = jobStatus.message || 'Обработка...';

        // Добавляем лог
        this.addLog(jobStatus.message, 'info');

        // Если задача завершена
        if (jobStatus.status === 'completed') {
            this.showNotification('Обработка завершена успешно!', 'success');
            this.loadJobs();
        } else if (jobStatus.status === 'failed') {
            this.showNotification('Ошибка обработки: ' + (jobStatus.error || 'Неизвестная ошибка'), 'error');
            this.loadJobs();
        }
    }

    addLog(message, level = 'info') {
        const logsContainer = document.getElementById('logsContainer');
        const logEntry = document.createElement('div');
        logEntry.className = 'log-entry';
        
        const timestamp = new Date().toLocaleTimeString();
        logEntry.innerHTML = `
            <span class="log-timestamp">[${timestamp}]</span>
            <span class="log-level-${level}">[${level.toUpperCase()}]</span>
            ${message}
        `;
        
        logsContainer.appendChild(logEntry);
        logsContainer.scrollTop = logsContainer.scrollHeight;
    }

    async loadJobs() {
        try {
            const response = await fetch('/api/jobs');
            if (response.ok) {
                const jobs = await response.json();
                this.updateJobsList(jobs);
            }
        } catch (error) {
            console.error('Ошибка загрузки задач:', error);
        }
    }

    updateJobsList(jobs) {
        const container = document.getElementById('jobsList');
        container.innerHTML = '';

        // Сортируем задачи по дате создания (новые сверху)
        jobs.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));

        jobs.forEach(job => {
            const card = document.createElement('div');
            card.className = 'job-card';
            
            const statusClass = job.status;
            const statusText = this.getStatusText(job.status);
            
            card.innerHTML = `
                <div class="job-info">
                    <h3>Задача ${job.id}</h3>
                    <p><strong>Статус:</strong> <span class="job-status ${statusClass}">${statusText}</span></p>
                    <p><strong>Прогресс:</strong> ${job.progress || 0}%</p>
                    <p><strong>Сообщение:</strong> ${job.message || 'Нет сообщений'}</p>
                    <p><strong>Создана:</strong> ${new Date(job.created_at).toLocaleString()}</p>
                    ${job.error ? `<p><strong>Ошибка:</strong> ${job.error}</p>` : ''}
                </div>
                <div class="job-actions">
                    ${job.status === 'completed' && job.output_file ? 
                        `<button class="btn btn-primary" onclick="app.downloadResult('${job.id}')">Скачать</button>` : 
                        ''
                    }
                </div>
            `;
            container.appendChild(card);
        });
    }

    getStatusText(status) {
        const statusMap = {
            'pending': 'Ожидание',
            'running': 'Выполняется',
            'completed': 'Завершена',
            'failed': 'Ошибка'
        };
        return statusMap[status] || status;
    }

    async downloadResult(jobId) {
        try {
            const response = await fetch(`/api/jobs/${jobId}/download`);
            if (response.ok) {
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `cronodump_${jobId}.csv`;
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                document.body.removeChild(a);
                this.showNotification('Файл скачан', 'success');
            } else {
                this.showNotification('Ошибка скачивания файла', 'error');
            }
        } catch (error) {
            this.showNotification('Ошибка: ' + error.message, 'error');
        }
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.hostname}:9999/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('WebSocket подключен');
            this.addLog('WebSocket подключен', 'info');
        };
        
        this.ws.onmessage = (event) => {
            try {
                const jobs = JSON.parse(event.data);
                jobs.forEach(job => {
                    if (job.id === this.currentJob) {
                        this.updateProgress(job);
                    }
                });
            } catch (error) {
                console.error('Ошибка парсинга WebSocket сообщения:', error);
            }
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket отключен');
            this.addLog('WebSocket отключен', 'warn');
            // Переподключаемся через 5 секунд
            setTimeout(() => this.connectWebSocket(), 5000);
        };
        
        this.ws.onerror = (error) => {
            console.error('Ошибка WebSocket:', error);
            this.addLog('Ошибка WebSocket', 'error');
        };
    }

    showNotification(message, type = 'info') {
        // Создаем уведомление
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;
        
        // Стили для уведомления
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 15px 20px;
            border-radius: 5px;
            color: white;
            font-weight: 600;
            z-index: 1000;
            max-width: 400px;
            word-wrap: break-word;
        `;
        
        // Цвета в зависимости от типа
        const colors = {
            'success': '#28a745',
            'error': '#dc3545',
            'warning': '#ffc107',
            'info': '#17a2b8'
        };
        
        notification.style.backgroundColor = colors[type] || colors.info;
        
        document.body.appendChild(notification);
        
        // Удаляем уведомление через 5 секунд
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);
    }
}

// Инициализируем приложение
const app = new CronodumpApp();

// Устанавливаем порты по умолчанию при загрузке
document.addEventListener('DOMContentLoaded', () => {
    app.setDefaultPort();
});