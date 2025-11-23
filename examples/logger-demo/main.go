package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"mbflow/internal/application/executor"
	"mbflow/internal/infrastructure/monitoring"
)

// Этот пример демонстрирует использование различных реализаций ExecutionLogger:
// 1. ConsoleLogger с настраиваемым writer (stdout, stderr, file, buffer)
// 2. ClickHouseLogger для записи в ClickHouse (требует подключения к БД)

func main() {
	fmt.Println("=== ExecutionLogger Interface Demo ===\n")

	// Пример 1: ConsoleLogger с stdout (по умолчанию)
	demoConsoleLoggerStdout()

	// Пример 2: ConsoleLogger с buffer
	demoConsoleLoggerBuffer()

	// Пример 3: ConsoleLogger с файлом
	demoConsoleLoggerFile()

	// Пример 4: ClickHouseLogger (закомментирован, требует подключения к БД)
	// demoClickHouseLogger()

	// Пример 5: Использование в WorkflowEngine
	demoWorkflowEngineWithLogger()

	// Пример 6: Использование нового метода Log() с LogEvent
	demoNewLogMethod()
}

// demoConsoleLoggerStdout демонстрирует использование ConsoleLogger с stdout
func demoConsoleLoggerStdout() {
	fmt.Println("--- 1. ConsoleLogger with stdout ---")

	logger := monitoring.NewConsoleLogger(monitoring.ConsoleLoggerConfig{
		Prefix:  "STDOUT",
		Verbose: true,
		Writer:  os.Stdout,
	})

	logger.LogExecutionStarted("workflow-1", "exec-1")
	logger.LogInfo("exec-1", "This is a test message")
	logger.LogExecutionCompleted("workflow-1", "exec-1", 100*time.Millisecond)

	fmt.Println()
}

// demoConsoleLoggerBuffer демонстрирует использование ConsoleLogger с buffer
func demoConsoleLoggerBuffer() {
	fmt.Println("--- 2. ConsoleLogger with buffer ---")

	var buffer bytes.Buffer
	logger := monitoring.NewConsoleLogger(monitoring.ConsoleLoggerConfig{
		Prefix:  "BUFFER",
		Verbose: false,
		Writer:  &buffer,
	})

	logger.LogExecutionStarted("workflow-2", "exec-2")
	logger.LogInfo("exec-2", "Logging to buffer")
	logger.LogExecutionCompleted("workflow-2", "exec-2", 200*time.Millisecond)

	fmt.Println("Buffer contents:")
	fmt.Println(buffer.String())
}

// demoConsoleLoggerFile демонстрирует использование ConsoleLogger с файлом
func demoConsoleLoggerFile() {
	fmt.Println("--- 3. ConsoleLogger with file ---")

	file, err := os.CreateTemp("", "workflow-log-*.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	logger := monitoring.NewConsoleLogger(monitoring.ConsoleLoggerConfig{
		Prefix:  "FILE",
		Verbose: true,
		Writer:  file,
	})

	logger.LogExecutionStarted("workflow-3", "exec-3")
	logger.LogInfo("exec-3", "Logging to file")
	logger.LogDebug("exec-3", "This is a debug message")
	logger.LogExecutionCompleted("workflow-3", "exec-3", 300*time.Millisecond)

	// Прочитаем файл и выведем содержимое
	file.Seek(0, 0)
	content := make([]byte, 1024)
	n, _ := file.Read(content)

	fmt.Printf("File: %s\n", file.Name())
	fmt.Println("File contents:")
	fmt.Println(string(content[:n]))
}

// demoClickHouseLogger демонстрирует использование ClickHouseLogger
// Раскомментируйте и настройте подключение к ClickHouse для тестирования
/*
func demoClickHouseLogger() {
	fmt.Println("--- 4. ClickHouseLogger ---")

	// Подключение к ClickHouse
	db, err := sql.Open("clickhouse", "tcp://localhost:9000?database=default")
	if err != nil {
		log.Printf("Failed to connect to ClickHouse: %v", err)
		return
	}
	defer db.Close()

	// Создание логера с автоматическим созданием таблицы
	logger, err := monitoring.NewClickHouseLogger(monitoring.ClickHouseLoggerConfig{
		DB:            db,
		TableName:     "workflow_logs_demo",
		BatchSize:     10,
		FlushInterval: 2 * time.Second,
		Verbose:       true,
		CreateTable:   true,
	})
	if err != nil {
		log.Printf("Failed to create ClickHouse logger: %v", err)
		return
	}
	defer logger.Close()

	// Логирование событий
	logger.LogExecutionStarted("workflow-4", "exec-4")
	logger.LogInfo("exec-4", "Logging to ClickHouse")
	logger.LogVariableSet("exec-4", "input", map[string]interface{}{"user": "alice"})
	logger.LogExecutionCompleted("workflow-4", "exec-4", 400*time.Millisecond)

	// Ждем, пока логер запишет события
	time.Sleep(3 * time.Second)

	fmt.Println("Events logged to ClickHouse")
}
*/

// demoWorkflowEngineWithLogger демонстрирует использование логеров в WorkflowEngine
func demoWorkflowEngineWithLogger() {
	fmt.Println("--- 5. WorkflowEngine with custom logger ---")

	// Создаем буфер для сбора логов
	var buffer bytes.Buffer

	// Создаем ConsoleLogger с буфером
	logger := monitoring.NewConsoleLogger(monitoring.ConsoleLoggerConfig{
		Prefix:  "ENGINE",
		Verbose: true,
		Writer:  &buffer,
	})

	// Создаем WorkflowEngine с мониторингом
	engine := executor.NewWorkflowEngine(&executor.EngineConfig{
		EnableMonitoring: false, // Отключаем встроенный мониторинг
		VerboseLogging:   true,
	})

	// Добавляем наш кастомный логер через CompositeObserver
	observer := monitoring.NewCompositeObserver(logger, nil, nil)
	engine.AddObserver(observer)

	// Демонстрируем прямое использование логера для симуляции событий
	logger.LogExecutionStarted("workflow-5", "exec-5")
	logger.LogNodeStartedFromConfig("exec-5", "node-1", "workflow-5", "http", "API Call",
		map[string]any{"url": "https://api.example.com"}, 1)
	logger.LogNodeCompletedFromConfig("exec-5", "node-1", "workflow-5", "http", "API Call",
		map[string]any{"url": "https://api.example.com"}, 150*time.Millisecond)
	logger.LogExecutionCompleted("workflow-5", "exec-5", 200*time.Millisecond)

	fmt.Println("Simulated workflow execution logged:")
	fmt.Println(buffer.String())
}

// demoNewLogMethod демонстрирует использование нового метода Log() с LogEvent
func demoNewLogMethod() {
	fmt.Println("--- 6. Using new Log() method with LogEvent ---")

	// Создаем logger с буфером
	var buffer bytes.Buffer
	logger := monitoring.NewConsoleLogger(monitoring.ConsoleLoggerConfig{
		Prefix:  "NEW-API",
		Verbose: true,
		Writer:  &buffer,
	})

	// Используем новый API - метод Log() с helper функциями
	logger.Log(monitoring.NewExecutionStartedEvent("workflow-6", "exec-6"))

	logger.Log(monitoring.NewNodeStartedEventFromConfig(
		"exec-6",
		"node-1",
		"workflow-6",
		"http",
		"API Request",
		map[string]any{"url": "https://api.example.com/users"},
		1,
	))

	logger.Log(monitoring.NewNodeCompletedEventFromConfig(
		"exec-6",
		"node-1",
		"workflow-6",
		"http",
		"API Request",
		map[string]any{"url": "https://api.example.com/users"},
		120*time.Millisecond,
	))

	// Пример с ошибкой и retry
	logger.Log(monitoring.NewNodeFailedEventFromConfig(
		"exec-6",
		"node-2",
		"workflow-6",
		"transform",
		"Data Transform",
		map[string]any{"expression": "invalid"},
		fmt.Errorf("syntax error in expression"),
		50*time.Millisecond,
		true, // will retry
	))

	logger.Log(monitoring.NewNodeRetryingEventFromConfig(
		"exec-6",
		"node-2",
		"workflow-6",
		"transform",
		"Data Transform",
		map[string]any{"expression": "fixed"},
		2, // attempt number
		time.Second,
	))

	// Переменные и state transitions (verbose mode)
	logger.Log(monitoring.NewVariableSetEvent("exec-6", "user_count", 42))
	logger.Log(monitoring.NewStateTransitionEvent("exec-6", "node-2", "running", "completed"))

	// Общие события
	logger.Log(monitoring.NewInfoEvent("exec-6", "All nodes processed successfully"))
	logger.Log(monitoring.NewDebugEvent("exec-6", "Cache hit for user data"))

	logger.Log(monitoring.NewExecutionCompletedEvent("workflow-6", "exec-6", 300*time.Millisecond))

	fmt.Println("Events logged using new Log() API:")
	fmt.Println(buffer.String())
	fmt.Println()

	// Демонстрация создания custom LogEvent
	fmt.Println("--- Custom LogEvent ---")

	buffer.Reset()

	customEvent := &monitoring.LogEvent{
		Timestamp:   time.Now(),
		Type:        monitoring.EventInfo,
		Level:       monitoring.LevelInfo,
		Message:     "Custom event with metadata",
		ExecutionID: "exec-custom",
		WorkflowID:  "workflow-custom",
		NodeID:      "custom-node",
		Metadata: map[string]interface{}{
			"custom_field_1": "value1",
			"custom_field_2": 123,
			"tags":           []string{"important", "custom"},
		},
	}

	logger.Log(customEvent)

	fmt.Println("Custom event logged:")
	fmt.Println(buffer.String())
}
