package mbflow

import "mbflow/internal/infrastructure/monitoring"

type EventType = monitoring.EventType
type LogEvent = monitoring.LogEvent

type LogLevel = monitoring.LogLevel

const (
	LevelDebug   = monitoring.LevelDebug
	LevelInfo    = monitoring.LevelInfo
	LevelWarning = monitoring.LevelWarning
	LevelError   = monitoring.LevelError
)

const (
	EventExecutionStarted   = monitoring.EventExecutionStarted
	EventExecutionCompleted = monitoring.EventExecutionCompleted
	EventExecutionFailed    = monitoring.EventExecutionFailed
	EventNodeStarted        = monitoring.EventNodeStarted
	EventNodeCompleted      = monitoring.EventNodeCompleted
	EventNodeFailed         = monitoring.EventNodeFailed
	EventNodeRetrying       = monitoring.EventNodeRetrying
	EventNodeSkipped        = monitoring.EventNodeSkipped
	EventVariableSet        = monitoring.EventVariableSet
	EventStateTransition    = monitoring.EventStateTransition
	EventCallbackStarted    = monitoring.EventCallbackStarted
	EventCallbackCompleted  = monitoring.EventCallbackCompleted
	EventInfo               = monitoring.EventInfo
	EventDebug              = monitoring.EventDebug
	EventError              = monitoring.EventError
)

type ConsoleLogger = monitoring.ConsoleLogger
type ConsoleLoggerConfig = monitoring.ConsoleLoggerConfig

func NewConsoleLogger(cfg ConsoleLoggerConfig) *ConsoleLogger {
	return monitoring.NewConsoleLogger(cfg)
}

func NewDefaultConsoleLogger(prefix string) *ConsoleLogger {
	return monitoring.NewDefaultConsoleLogger(prefix)
}

type ClickHouseLogger = monitoring.ClickHouseLogger
type ClickHouseLoggerConfig = monitoring.ClickHouseLoggerConfig

func NewClickHouseLogger(cfg ClickHouseLoggerConfig) (*ClickHouseLogger, error) {
	return monitoring.NewClickHouseLogger(cfg)
}
