// Package trigger provides workflow trigger orchestration
package trigger

import (
	"context"
	"fmt"
	"sync"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// Manager orchestrates all trigger types
type Manager struct {
	// Dependencies
	triggerRepo  repository.TriggerRepository
	workflowRepo repository.WorkflowRepository
	executionMgr *engine.ExecutionManager
	cache        *cache.RedisCache

	// Trigger handlers
	cronScheduler   *CronScheduler
	eventListener   *EventListener
	webhookRegistry *WebhookRegistry

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// ManagerConfig holds configuration for trigger manager
type ManagerConfig struct {
	TriggerRepo  repository.TriggerRepository
	WorkflowRepo repository.WorkflowRepository
	ExecutionMgr *engine.ExecutionManager
	Cache        *cache.RedisCache
}

// NewManager creates a new trigger manager
func NewManager(cfg ManagerConfig) (*Manager, error) {
	if cfg.TriggerRepo == nil {
		return nil, fmt.Errorf("trigger repository is required")
	}
	if cfg.WorkflowRepo == nil {
		return nil, fmt.Errorf("workflow repository is required")
	}
	if cfg.ExecutionMgr == nil {
		return nil, fmt.Errorf("execution manager is required")
	}
	if cfg.Cache == nil {
		return nil, fmt.Errorf("redis cache is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		triggerRepo:  cfg.TriggerRepo,
		workflowRepo: cfg.WorkflowRepo,
		executionMgr: cfg.ExecutionMgr,
		cache:        cfg.Cache,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Initialize trigger handlers
	if err := m.initializeHandlers(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize handlers: %w", err)
	}

	return m, nil
}

// initializeHandlers initializes all trigger type handlers
func (m *Manager) initializeHandlers() error {
	// Initialize cron scheduler
	cronScheduler, err := NewCronScheduler(CronSchedulerConfig{
		TriggerRepo:  m.triggerRepo,
		WorkflowRepo: m.workflowRepo,
		ExecutionMgr: m.executionMgr,
		Cache:        m.cache,
	})
	if err != nil {
		return fmt.Errorf("failed to create cron scheduler: %w", err)
	}
	m.cronScheduler = cronScheduler

	// Initialize event listener
	eventListener, err := NewEventListener(EventListenerConfig{
		TriggerRepo:  m.triggerRepo,
		WorkflowRepo: m.workflowRepo,
		ExecutionMgr: m.executionMgr,
		Cache:        m.cache,
	})
	if err != nil {
		return fmt.Errorf("failed to create event listener: %w", err)
	}
	m.eventListener = eventListener

	// Initialize webhook registry
	webhookRegistry := NewWebhookRegistry(WebhookRegistryConfig{
		TriggerRepo:  m.triggerRepo,
		WorkflowRepo: m.workflowRepo,
		ExecutionMgr: m.executionMgr,
		Cache:        m.cache,
	})
	m.webhookRegistry = webhookRegistry

	return nil
}

// Start starts all trigger handlers
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load all enabled triggers
	triggers, err := m.triggerRepo.FindEnabled(m.ctx)
	if err != nil {
		return fmt.Errorf("failed to load enabled triggers: %w", err)
	}

	// Start cron scheduler
	if err := m.cronScheduler.Start(m.ctx, triggers); err != nil {
		return fmt.Errorf("failed to start cron scheduler: %w", err)
	}

	// Start event listener
	if err := m.eventListener.Start(m.ctx, triggers); err != nil {
		return fmt.Errorf("failed to start event listener: %w", err)
	}

	// Register webhooks
	if err := m.webhookRegistry.RegisterAll(m.ctx, triggers); err != nil {
		return fmt.Errorf("failed to register webhooks: %w", err)
	}

	return nil
}

// Stop gracefully shuts down all trigger handlers
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel context to signal shutdown
	m.cancel()

	// Stop cron scheduler
	if m.cronScheduler != nil {
		if err := m.cronScheduler.Stop(); err != nil {
			return fmt.Errorf("failed to stop cron scheduler: %w", err)
		}
	}

	// Stop event listener
	if m.eventListener != nil {
		if err := m.eventListener.Stop(); err != nil {
			return fmt.Errorf("failed to stop event listener: %w", err)
		}
	}

	// Wait for all goroutines to complete
	m.wg.Wait()

	return nil
}

// TriggerManual triggers a workflow manually
func (m *Manager) TriggerManual(ctx context.Context, triggerID, workflowID string, input map[string]any) (string, error) {
	// Execute workflow
	execution, err := m.executionMgr.Execute(ctx, workflowID, input, nil)
	if err != nil {
		return "", fmt.Errorf("failed to execute workflow: %w", err)
	}

	// Update trigger state
	if err := m.updateTriggerState(ctx, triggerID); err != nil {
		// Log error but don't fail execution
		fmt.Printf("failed to update trigger state: %v\n", err)
	}

	return execution.ID, nil
}

// OnTriggerCreated handles trigger creation events
func (m *Manager) OnTriggerCreated(ctx context.Context, trigger *models.Trigger) error {
	if !trigger.Enabled {
		return nil
	}

	switch trigger.Type {
	case models.TriggerTypeCron:
		return m.cronScheduler.AddTrigger(ctx, trigger)
	case models.TriggerTypeEvent:
		return m.eventListener.AddTrigger(ctx, trigger)
	case models.TriggerTypeWebhook:
		return m.webhookRegistry.RegisterWebhook(ctx, trigger)
	case models.TriggerTypeInterval:
		return m.cronScheduler.AddTrigger(ctx, trigger)
	}

	return nil
}

// OnTriggerUpdated handles trigger update events
func (m *Manager) OnTriggerUpdated(ctx context.Context, trigger *models.Trigger) error {
	// Remove old trigger
	if err := m.OnTriggerDeleted(ctx, trigger.ID); err != nil {
		return err
	}

	// Add updated trigger if enabled
	if trigger.Enabled {
		return m.OnTriggerCreated(ctx, trigger)
	}

	return nil
}

// OnTriggerDeleted handles trigger deletion events
func (m *Manager) OnTriggerDeleted(ctx context.Context, triggerID string) error {
	// Remove from cron scheduler
	if err := m.cronScheduler.RemoveTrigger(ctx, triggerID); err != nil {
		// Log error but continue
		fmt.Printf("failed to remove cron trigger: %v\n", err)
	}

	// Remove from event listener
	if err := m.eventListener.RemoveTrigger(ctx, triggerID); err != nil {
		fmt.Printf("failed to remove event trigger: %v\n", err)
	}

	// Remove from webhook registry
	if err := m.webhookRegistry.UnregisterWebhook(ctx, triggerID); err != nil {
		fmt.Printf("failed to unregister webhook: %v\n", err)
	}

	// Clear trigger state
	if err := m.clearTriggerState(ctx, triggerID); err != nil {
		fmt.Printf("failed to clear trigger state: %v\n", err)
	}

	return nil
}

// updateTriggerState updates trigger state in Redis and database
func (m *Manager) updateTriggerState(ctx context.Context, triggerID string) error {
	state, err := LoadTriggerState(ctx, m.cache, triggerID)
	if err != nil {
		state = NewTriggerState(triggerID)
	}

	state.MarkExecuted()

	if err := state.Save(ctx, m.cache); err != nil {
		return err
	}

	// Also update last triggered timestamp in database
	// This is handled by the repository's MarkTriggered method
	// which is called from the individual trigger handlers

	return nil
}

// clearTriggerState clears trigger state from Redis
func (m *Manager) clearTriggerState(ctx context.Context, triggerID string) error {
	return DeleteTriggerState(ctx, m.cache, triggerID)
}

// WebhookRegistry returns the webhook registry for HTTP webhook handling
func (m *Manager) WebhookRegistry() *WebhookRegistry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.webhookRegistry
}
