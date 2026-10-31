package trigger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// CronScheduler manages cron-based triggers
type CronScheduler struct {
	triggerRepo  repository.TriggerRepository
	workflowRepo repository.WorkflowRepository
	executionMgr WorkflowExecutor
	cache        *cache.RedisCache

	cron    *cron.Cron
	entries map[string]cron.EntryID // triggerID -> entryID
	mu      sync.RWMutex
}

// CronSchedulerConfig holds configuration for cron scheduler
type CronSchedulerConfig struct {
	TriggerRepo  repository.TriggerRepository
	WorkflowRepo repository.WorkflowRepository
	ExecutionMgr WorkflowExecutor
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

// Stop stops the cron scheduler and waits for running jobs to complete.
func (cs *CronScheduler) Stop() error {
	// Do not hold cs.mu while waiting: running jobs take cs.mu.RLock, so
	// waiting for them under the write lock would deadlock.
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
	if err := cs.updateNextExecution(ctx, trigger.ID, cs.nextRun(entryID)); err != nil {
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
	switch trigger.Type {
	case models.TriggerTypeCron:
		return cs.parseCronSchedule(trigger)
	case models.TriggerTypeInterval:
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

	tz, _ := trigger.Config["timezone"].(string)
	return ParseCronSchedule(scheduleStr, tz)
}

// alignedIntervalSchedule fires every Delay at instants aligned to multiples
// of Delay (counted from the zero time). Unlike cron.ConstantDelaySchedule,
// whose ticks depend on when the process started, every replica computes the
// same ticks, which is what makes per-tick deduplication across replicas
// work.
type alignedIntervalSchedule struct {
	Delay time.Duration
}

// Next returns the first aligned tick strictly after t.
func (s alignedIntervalSchedule) Next(t time.Time) time.Time {
	return t.Truncate(s.Delay).Add(s.Delay)
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

	// The scheduler has second precision: round like cron.Every does.
	if duration < time.Second {
		duration = time.Second
	}
	duration = duration.Round(time.Second)

	return alignedIntervalSchedule{Delay: duration}, nil
}

// createJob creates a cron job for the trigger
func (cs *CronScheduler) createJob(trigger *models.Trigger) cron.Job {
	return cron.FuncJob(func() {
		cs.runTick(trigger, cs.scheduledTick(trigger.ID, time.Now()))
	})
}

// runTick fires the trigger for one scheduled tick if this replica wins the
// claim for that tick. It reports whether the trigger was fired.
func (cs *CronScheduler) runTick(trigger *models.Trigger, tick time.Time) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if !claimOccurrence(ctx, cs.cache, cronClaimKey(trigger.ID, tick)) {
		// Another replica fires this tick.
		return false
	}

	if err := cs.executeTrigger(ctx, trigger); err != nil {
		fmt.Printf("trigger %s execution failed: %v\n", trigger.ID, err)
	}
	return true
}

// scheduledTick returns the scheduled time of the tick that is running now.
// The cron runner records it as the entry's Prev before the job can observe
// the entry, so it is identical on every replica regardless of when the job
// goroutine actually starts. If it is unavailable (e.g. the trigger was
// replaced meanwhile), now rounded to the scheduler's one-second precision
// is used.
func (cs *CronScheduler) scheduledTick(triggerID string, now time.Time) time.Time {
	cs.mu.RLock()
	entryID, ok := cs.entries[triggerID]
	cs.mu.RUnlock()

	if ok {
		prev := cs.cron.Entry(entryID).Prev
		if !prev.IsZero() && !prev.After(now) && now.Sub(prev) < time.Minute {
			return prev
		}
	}
	return now.Round(time.Second)
}

// executeTrigger executes a workflow triggered by the cron schedule
func (cs *CronScheduler) executeTrigger(ctx context.Context, trigger *models.Trigger) error {
	// Get default input from trigger config
	input := make(map[string]any)
	if defaultInput, ok := trigger.Config["input"].(map[string]any); ok {
		input = defaultInput
	}

	// Execute workflow
	_, err := cs.executionMgr.Execute(ctx, trigger.WorkflowID, input, executionOptionsFor(trigger.ID))
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
		state.SetNextExecution(cs.nextRun(entryID))
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

// nextRun returns the next activation of a scheduled entry. Before the
// scheduler is started the runner has not computed it yet, so it is derived
// from the schedule.
func (cs *CronScheduler) nextRun(entryID cron.EntryID) time.Time {
	entry := cs.cron.Entry(entryID)
	if entry.Next.IsZero() && entry.Schedule != nil {
		return entry.Schedule.Next(time.Now())
	}
	return entry.Next
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
		Config:     make(map[string]any),
		Enabled:    tm.Enabled,
		CreatedAt:  tm.CreatedAt,
		UpdatedAt:  tm.UpdatedAt,
	}

	if tm.Config != nil {
		trigger.Config = map[string]any(tm.Config)
	}

	if tm.LastTriggeredAt != nil {
		trigger.LastRun = tm.LastTriggeredAt
	}

	return trigger
}

// cronParser accepts standard 5-field expressions ("0 9 * * *", as used in
// the docs), 6-field expressions with a leading seconds field, descriptors
// such as @hourly, and an optional CRON_TZ=/TZ= prefix.
var cronParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// ParseCronSchedule parses a cron expression evaluated in the given IANA
// timezone (UTC when empty).
func ParseCronSchedule(expression, timezone string) (cron.Schedule, error) {
	spec := expression
	if timezone != "" {
		if _, err := time.LoadLocation(timezone); err != nil {
			return nil, fmt.Errorf("invalid timezone %s: %w", timezone, err)
		}
		// The schedule itself must carry the location: the scheduler's own
		// location only applies to specs without one.
		spec = "CRON_TZ=" + timezone + " " + expression
	}
	schedule, err := cronParser.Parse(spec)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %s: %w", expression, err)
	}
	return schedule, nil
}

// ValidateScheduleConfig checks that a cron or interval trigger config can
// be scheduled. Other trigger types are accepted as-is.
func ValidateScheduleConfig(triggerType string, config map[string]any) error {
	t := &models.Trigger{Type: models.TriggerType(triggerType), Config: config}
	if t.Config == nil {
		t.Config = map[string]any{}
	}
	cs := &CronScheduler{}
	switch t.Type {
	case models.TriggerTypeCron:
		_, err := cs.parseCronSchedule(t)
		return err
	case models.TriggerTypeInterval:
		_, err := cs.parseIntervalSchedule(t)
		return err
	}
	return nil
}
