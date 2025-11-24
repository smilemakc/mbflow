package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// TriggerManager manages trigger activations and enforces cooldowns/concurrency limits
type TriggerManager struct {
	mu sync.RWMutex

	// Track last activation time for cooldown enforcement
	lastActivation map[uuid.UUID]time.Time

	// Track concurrent executions per trigger
	activeExecutions map[uuid.UUID]int

	// Event callbacks
	onTriggerActivated func(triggerID uuid.UUID, input map[string]any)
	onTriggerRejected  func(triggerID uuid.UUID, reason string)
}

// NewTriggerManager creates a new trigger manager
func NewTriggerManager() *TriggerManager {
	return &TriggerManager{
		lastActivation:   make(map[uuid.UUID]time.Time),
		activeExecutions: make(map[uuid.UUID]int),
	}
}

// CanActivate checks if a trigger can be activated
func (tm *TriggerManager) CanActivate(trigger domain.Trigger, input map[string]any) (bool, string) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	triggerID := trigger.ID()

	// Check if trigger is active
	if !trigger.IsActive() {
		return false, "trigger is not active"
	}

	// Check cooldown
	cooldown := trigger.Cooldown()
	if cooldown > 0 {
		if lastTime, exists := tm.lastActivation[triggerID]; exists {
			elapsed := time.Since(lastTime)
			if elapsed < cooldown {
				remaining := cooldown - elapsed
				return false, fmt.Sprintf("trigger is in cooldown, retry in %v", remaining)
			}
		}
	}

	// Check concurrent execution limit
	maxConcurrent := trigger.MaxConcurrentExecutions()
	if maxConcurrent > 0 {
		if count := tm.activeExecutions[triggerID]; count >= maxConcurrent {
			return false, fmt.Sprintf("max concurrent executions reached (%d)", maxConcurrent)
		}
	}

	// Check trigger condition
	if !trigger.ShouldTrigger(input) {
		return false, "trigger condition not met"
	}

	// Validate input
	if err := trigger.ValidateInput(input); err != nil {
		return false, fmt.Sprintf("input validation failed: %v", err)
	}

	return true, ""
}

// Activate activates a trigger and records the activation
func (tm *TriggerManager) Activate(trigger domain.Trigger, input map[string]any) error {
	canActivate, reason := tm.CanActivate(trigger, input)
	if !canActivate {
		if tm.onTriggerRejected != nil {
			tm.onTriggerRejected(trigger.ID(), reason)
		}
		return fmt.Errorf("cannot activate trigger: %s", reason)
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	triggerID := trigger.ID()

	// Record activation
	tm.lastActivation[triggerID] = time.Now()
	tm.activeExecutions[triggerID]++

	if tm.onTriggerActivated != nil {
		tm.onTriggerActivated(triggerID, input)
	}

	return nil
}

// CompleteExecution marks an execution as completed, decrementing the counter
func (tm *TriggerManager) CompleteExecution(triggerID uuid.UUID) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if count := tm.activeExecutions[triggerID]; count > 0 {
		tm.activeExecutions[triggerID]--
	}
}

// GetActiveExecutions returns the number of active executions for a trigger
func (tm *TriggerManager) GetActiveExecutions(triggerID uuid.UUID) int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.activeExecutions[triggerID]
}

// GetLastActivation returns the last activation time for a trigger
func (tm *TriggerManager) GetLastActivation(triggerID uuid.UUID) *time.Time {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if t, exists := tm.lastActivation[triggerID]; exists {
		return &t
	}
	return nil
}

// Reset resets all trigger state (useful for testing)
func (tm *TriggerManager) Reset() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.lastActivation = make(map[uuid.UUID]time.Time)
	tm.activeExecutions = make(map[uuid.UUID]int)
}

// SetOnTriggerActivated sets the callback for trigger activation
func (tm *TriggerManager) SetOnTriggerActivated(callback func(triggerID uuid.UUID, input map[string]any)) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.onTriggerActivated = callback
}

// SetOnTriggerRejected sets the callback for trigger rejection
func (tm *TriggerManager) SetOnTriggerRejected(callback func(triggerID uuid.UUID, reason string)) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.onTriggerRejected = callback
}

// TriggerRegistry manages multiple triggers for a workflow
type TriggerRegistry struct {
	mu       sync.RWMutex
	triggers map[uuid.UUID]domain.Trigger
	manager  *TriggerManager
}

// NewTriggerRegistry creates a new trigger registry
func NewTriggerRegistry(manager *TriggerManager) *TriggerRegistry {
	return &TriggerRegistry{
		triggers: make(map[uuid.UUID]domain.Trigger),
		manager:  manager,
	}
}

// Register registers a trigger
func (tr *TriggerRegistry) Register(trigger domain.Trigger) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.triggers[trigger.ID()] = trigger
}

// Unregister removes a trigger
func (tr *TriggerRegistry) Unregister(triggerID uuid.UUID) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	delete(tr.triggers, triggerID)
}

// Get retrieves a trigger by ID
func (tr *TriggerRegistry) Get(triggerID uuid.UUID) (domain.Trigger, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	trigger, exists := tr.triggers[triggerID]
	return trigger, exists
}

// GetAll returns all registered triggers
func (tr *TriggerRegistry) GetAll() []domain.Trigger {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make([]domain.Trigger, 0, len(tr.triggers))
	for _, trigger := range tr.triggers {
		result = append(result, trigger)
	}
	return result
}

// GetByType returns all triggers of a specific type
func (tr *TriggerRegistry) GetByType(triggerType domain.TriggerType) []domain.Trigger {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make([]domain.Trigger, 0)
	for _, trigger := range tr.triggers {
		if trigger.Type() == triggerType {
			result = append(result, trigger)
		}
	}
	return result
}

// FindMatchingTriggers finds triggers that match the given input
func (tr *TriggerRegistry) FindMatchingTriggers(input map[string]any) []domain.Trigger {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make([]domain.Trigger, 0)
	for _, trigger := range tr.triggers {
		if canActivate, _ := tr.manager.CanActivate(trigger, input); canActivate {
			result = append(result, trigger)
		}
	}
	return result
}

// AutoTriggerScheduler manages automatic trigger scheduling
type AutoTriggerScheduler struct {
	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc

	registry *TriggerRegistry
	executor *WorkflowEngine

	// Scheduling configuration
	checkInterval time.Duration

	// Callbacks
	onAutoTriggerFired func(trigger domain.Trigger)
}

// NewAutoTriggerScheduler creates a new auto-trigger scheduler
func NewAutoTriggerScheduler(
	registry *TriggerRegistry,
	executor *WorkflowEngine,
	checkInterval time.Duration,
) *AutoTriggerScheduler {
	return &AutoTriggerScheduler{
		registry:      registry,
		executor:      executor,
		checkInterval: checkInterval,
	}
}

// Start starts the scheduler
func (ats *AutoTriggerScheduler) Start(ctx context.Context) error {
	ats.mu.Lock()
	defer ats.mu.Unlock()

	if ats.running {
		return fmt.Errorf("scheduler is already running")
	}

	ats.ctx, ats.cancel = context.WithCancel(ctx)
	ats.running = true

	go ats.run()

	return nil
}

// Stop stops the scheduler
func (ats *AutoTriggerScheduler) Stop() {
	ats.mu.Lock()
	defer ats.mu.Unlock()

	if !ats.running {
		return
	}

	if ats.cancel != nil {
		ats.cancel()
	}

	ats.running = false
}

// run is the main scheduler loop
func (ats *AutoTriggerScheduler) run() {
	ticker := time.NewTicker(ats.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ats.ctx.Done():
			return

		case <-ticker.C:
			ats.checkAutoTriggers()
		}
	}
}

// checkAutoTriggers checks and fires auto triggers
func (ats *AutoTriggerScheduler) checkAutoTriggers() {
	// Get all auto triggers
	autoTriggers := ats.registry.GetByType(domain.TriggerTypeAuto)

	for _, trigger := range autoTriggers {
		// Check if trigger can be activated
		canActivate, _ := ats.registry.manager.CanActivate(trigger, nil)
		if !canActivate {
			continue
		}

		// Fire callback
		if ats.onAutoTriggerFired != nil {
			ats.onAutoTriggerFired(trigger)
		}
	}
}

// SetOnAutoTriggerFired sets the callback for auto trigger firing
func (ats *AutoTriggerScheduler) SetOnAutoTriggerFired(callback func(trigger domain.Trigger)) {
	ats.mu.Lock()
	defer ats.mu.Unlock()

	ats.onAutoTriggerFired = callback
}

// IsRunning returns whether the scheduler is running
func (ats *AutoTriggerScheduler) IsRunning() bool {
	ats.mu.RLock()
	defer ats.mu.RUnlock()

	return ats.running
}

// ManualTriggerExecutor handles manual trigger execution
type ManualTriggerExecutor struct {
	manager  *TriggerManager
	engine   *WorkflowEngine
	registry *TriggerRegistry
}

// NewManualTriggerExecutor creates a new manual trigger executor
func NewManualTriggerExecutor(
	manager *TriggerManager,
	engine *WorkflowEngine,
	registry *TriggerRegistry,
) *ManualTriggerExecutor {
	return &ManualTriggerExecutor{
		manager:  manager,
		engine:   engine,
		registry: registry,
	}
}

// Execute executes a workflow via manual trigger
func (mte *ManualTriggerExecutor) Execute(
	ctx context.Context,
	workflow domain.Workflow,
	triggerID uuid.UUID,
	input map[string]any,
) (domain.Execution, error) {
	// Get trigger
	trigger, exists := mte.registry.Get(triggerID)
	if !exists {
		return nil, fmt.Errorf("trigger %s not found", triggerID)
	}

	// Verify it's a manual trigger
	if trigger.Type() != domain.TriggerTypeManual {
		return nil, fmt.Errorf("trigger %s is not a manual trigger", triggerID)
	}

	// Try to activate
	if err := mte.manager.Activate(trigger, input); err != nil {
		return nil, fmt.Errorf("failed to activate trigger: %w", err)
	}

	// Execute workflow
	execution, err := mte.engine.ExecuteWorkflow(ctx, workflow, trigger, input)

	// Mark execution as completed
	mte.manager.CompleteExecution(triggerID)

	return execution, err
}
