package trigger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/cache"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// CronScheduler manages cron-based triggers
type CronScheduler struct {
	triggerRepo  repository.TriggerRepository
	workflowRepo repository.WorkflowRepository
	executionMgr *engine.ExecutionManager
	cache        *cache.RedisCache

	cron    *cron.Cron
	entries map[string]cron.EntryID // triggerID -> entryID
	mu      sync.RWMutex
}

// CronSchedulerConfig holds configuration for cron scheduler
type CronSchedulerConfig struct {
	TriggerRepo  repository.TriggerRepository
	WorkflowRepo repository.WorkflowRepository
	ExecutionMgr *engine.ExecutionManager
	Cache        *cache.RedisCache
}

// NewCronScheduler creates a new cron scheduler
func NewCronScheduler(cfg CronSchedulerConfig) (*CronScheduler, error) {
	// Create cron with second precision and UTC timezone
	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC))

	return &CronScheduler{
		triggerRepo:  cfg.TriggerRepo,
		workflowRepo: cfg.WorkflowRepo,
		executionMgr: cfg.ExecutionMgr,
		cache:        cfg.Cache,
		cron:         c,
		entries:      make(map[string]cron.EntryID),
	}, nil
}

// Start starts the cron scheduler with initial triggers
func (cs *CronScheduler) Start(ctx context.Context, triggers []*storagemodels.TriggerModel) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Add all cron and interval triggers
	for _, trigger := range triggers {
		if trigger.Type == string(models.TriggerTypeCron) || trigger.Type == string(models.TriggerTypeInterval) {
			domainTrigger := cs.modelToDomain(trigger)
			if err := cs.addTriggerLocked(ctx, domainTrigger); err != nil {
				fmt.Printf("failed to add trigger %s: %v\n", trigger.ID, err)
				continue
			}
		}
	}

	// Start cron scheduler
	cs.cron.Start()

	return nil
}

// Stop stops the cron scheduler
func (cs *CronScheduler) Stop() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Stop cron (waits for running jobs to complete)
	ctx := cs.cron.Stop()
	<-ctx.Done()

	return nil
}

// AddTrigger adds a new cron trigger
func (cs *CronScheduler) AddTrigger(ctx context.Context, trigger *models.Trigger) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	return cs.addTriggerLocked(ctx, trigger)
}

// addTriggerLocked adds a trigger (must hold lock)
func (cs *CronScheduler) addTriggerLocked(ctx context.Context, trigger *models.Trigger) error {
	if trigger.Type != models.TriggerTypeCron && trigger.Type != models.TriggerTypeInterval {
		return nil // Not a cron/interval trigger
	}

	// Remove existing entry if present
	if entryID, exists := cs.entries[trigger.ID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.entries, trigger.ID)
	}

	// Parse schedule
	schedule, err := cs.parseSchedule(trigger)
	if err != nil {
		return fmt.Errorf("failed to parse schedule: %w", err)
	}

	// Create job
	job := cs.createJob(trigger)

	// Add to cron
	entryID := cs.cron.Schedule(schedule, job)

	cs.entries[trigger.ID] = entryID

	// Calculate and save next execution time
	entry := cs.cron.Entry(entryID)
	if err := cs.updateNextExecution(ctx, trigger.ID, entry.Next); err != nil {
		fmt.Printf("failed to update next execution for trigger %s: %v\n", trigger.ID, err)
	}

	return nil
}

// RemoveTrigger removes a cron trigger
func (cs *CronScheduler) RemoveTrigger(ctx context.Context, triggerID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if entryID, exists := cs.entries[triggerID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.entries, triggerID)
	}

	return nil
}

// parseSchedule parses trigger schedule configuration
func (cs *CronScheduler) parseSchedule(trigger *models.Trigger) (cron.Schedule, error) {
	if trigger.Type == models.TriggerTypeCron {
		return cs.parseCronSchedule(trigger)
	} else if trigger.Type == models.TriggerTypeInterval {
		return cs.parseIntervalSchedule(trigger)
	}

	return nil, fmt.Errorf("unsupported trigger type: %s", trigger.Type)
}

// parseCronSchedule parses a cron schedule
func (cs *CronScheduler) parseCronSchedule(trigger *models.Trigger) (cron.Schedule, error) {
	scheduleStr, ok := trigger.Config["schedule"].(string)
	if !ok {
		return nil, fmt.Errorf("schedule not found in trigger config")
	}

	// Get timezone if specified
	location := time.UTC
	if tz, ok := trigger.Config["timezone"].(string); ok && tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %s: %w", tz, err)
		}
		location = loc
	}

	// Create cron with timezone
	c := cron.New(cron.WithSeconds(), cron.WithLocation(location))

	// Parse cron expression
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(scheduleStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %s: %w", scheduleStr, err)
	}

	// For testing purposes, we need to close the temporary cron instance
	_ = c

	return schedule, nil
}

// parseIntervalSchedule parses an interval schedule
func (cs *CronScheduler) parseIntervalSchedule(trigger *models.Trigger) (cron.Schedule, error) {
	intervalValue, ok := trigger.Config["interval"]
	if !ok {
		return nil, fmt.Errorf("interval not found in trigger config")
	}

	var duration time.Duration
	var err error

	switch v := intervalValue.(type) {
	case string:
		duration, err = time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid interval duration %s: %w", v, err)
		}
	case float64:
		duration = time.Duration(v) * time.Second
	case int:
		duration = time.Duration(v) * time.Second
	default:
		return nil, fmt.Errorf("invalid interval type: %T", intervalValue)
	}

	if duration <= 0 {
		return nil, fmt.Errorf("interval must be positive")
	}

	// Use cron's ConstantDelaySchedule for fixed intervals
	return cron.ConstantDelaySchedule{Delay: duration}, nil
}

// createJob creates a cron job for the trigger
func (cs *CronScheduler) createJob(trigger *models.Trigger) cron.Job {
	return cron.FuncJob(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := cs.executeTrigger(ctx, trigger); err != nil {
			fmt.Printf("trigger %s execution failed: %v\n", trigger.ID, err)
		}
	})
}

// executeTrigger executes a workflow triggered by the cron schedule
func (cs *CronScheduler) executeTrigger(ctx context.Context, trigger *models.Trigger) error {
	// Get default input from trigger config
	input := make(map[string]interface{})
	if defaultInput, ok := trigger.Config["input"].(map[string]interface{}); ok {
		input = defaultInput
	}

	// Execute workflow
	_, err := cs.executionMgr.Execute(ctx, trigger.WorkflowID, input, nil)
	if err != nil {
		return fmt.Errorf("failed to execute workflow: %w", err)
	}

	// Update trigger state
	state, err := LoadTriggerState(ctx, cs.cache, trigger.ID)
	if err != nil {
		state = NewTriggerState(trigger.ID)
	}
	state.MarkExecuted()

	// Get next execution time
	cs.mu.RLock()
	if entryID, exists := cs.entries[trigger.ID]; exists {
		entry := cs.cron.Entry(entryID)
		state.SetNextExecution(entry.Next)
	}
	cs.mu.RUnlock()

	if err := state.Save(ctx, cs.cache); err != nil {
		fmt.Printf("failed to save trigger state: %v\n", err)
	}

	// Update last triggered timestamp in database
	triggerUUID, _ := uuid.Parse(trigger.ID)
	if err := cs.triggerRepo.MarkTriggered(ctx, triggerUUID); err != nil {
		fmt.Printf("failed to mark trigger as triggered: %v\n", err)
	}

	return nil
}

// updateNextExecution updates the next execution time in trigger state
func (cs *CronScheduler) updateNextExecution(ctx context.Context, triggerID string, nextTime time.Time) error {
	// Skip state persistence if cache is not available (e.g., in unit tests)
	if cs.cache == nil {
		return nil
	}

	state, err := LoadTriggerState(ctx, cs.cache, triggerID)
	if err != nil {
		state = NewTriggerState(triggerID)
	}

	state.SetNextExecution(nextTime)
	return state.Save(ctx, cs.cache)
}

// modelToDomain converts storage model to domain model
func (cs *CronScheduler) modelToDomain(tm *storagemodels.TriggerModel) *models.Trigger {
	trigger := &models.Trigger{
		ID:         tm.ID.String(),
		WorkflowID: tm.WorkflowID.String(),
		Type:       models.TriggerType(tm.Type),
		Config:     make(map[string]interface{}),
		Enabled:    tm.Enabled,
		CreatedAt:  tm.CreatedAt,
		UpdatedAt:  tm.UpdatedAt,
	}

	if tm.Config != nil {
		trigger.Config = map[string]interface{}(tm.Config)
	}

	if tm.LastTriggeredAt != nil {
		trigger.LastRun = tm.LastTriggeredAt
	}

	return trigger
}
