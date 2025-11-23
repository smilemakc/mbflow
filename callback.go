package mbflow

import (
	"github.com/smilemakc/mbflow/internal/application/executor"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// HTTPCallbackObserver sends execution events to an HTTP callback URL.
// It implements the ExecutionObserver interface and sends POST requests
// with JSON payloads for each execution event.
type HTTPCallbackObserver = monitoring.HTTPCallbackObserver

// HTTPCallbackObserverConfig holds configuration for HTTPCallbackObserver.
type HTTPCallbackObserverConfig = monitoring.HTTPCallbackObserverConfig

func NewHTTPCallbackObserver(config HTTPCallbackObserverConfig) (*HTTPCallbackObserver, error) {
	return monitoring.NewHTTPCallbackObserver(config)
}

// HTTPCallbackConfig configures the HTTP callback processor.
type HTTPCallbackConfig = executor.HTTPCallbackConfig
