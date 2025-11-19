package mbflow

import (
	"mbflow/internal/domain"
)

// executionWrapper wraps domain Execution entity for public API.
type executionWrapper struct {
	*domain.Execution
}

func (e *executionWrapper) Status() string {
	return string(e.Execution.Status())
}

// wrapExecution wraps domain entity into public interface.
func wrapExecution(e *domain.Execution) Execution {
	return &executionWrapper{Execution: e}
}

// unwrapExecution extracts domain entity from wrapper.
func unwrapExecution(e Execution) *domain.Execution {
	if wrapper, ok := e.(*executionWrapper); ok {
		return wrapper.Execution
	}
	// If it's not our wrapper, reconstruct
	return domain.ReconstructExecution(
		e.ID(),
		e.WorkflowID(),
		domain.ExecutionStatus(e.Status()),
		e.StartedAt(),
		e.FinishedAt(),
	)
}

// Similarly for other entities if needed
