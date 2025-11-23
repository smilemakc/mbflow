package monitoring

// ExecutionLogger defines the interface for logging workflow execution events.
// Implementations can log to console, files, databases (ClickHouse), or other destinations.
type ExecutionLogger interface {
	// Log logs a single event. This is the main method for all logging.
	Log(event *LogEvent)
}
