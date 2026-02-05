package rest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/engine"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
)

// BenchmarkEventProcessingLatency measures the latency from event publication to execution start
func BenchmarkEventProcessingLatency(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	env := setupE2EEnvironment(&testing.T{})
	defer env.cleanup(&testing.T{})

	ctx := context.Background()

	// Create workflow
	workflowModel := env.createWorkflowFromFixture(&testing.T{}, ctx)

	// Reset mocks
	env.Mocks.ExampleAPI.Reset()
	env.Mocks.SendGridAPI.Reset()
	env.Mocks.SegmentAPI.Reset()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		eventData := map[string]interface{}{
			"user_id":    fmt.Sprintf("usr_bench_%d", i),
			"email":      fmt.Sprintf("bench%d@example.com", i),
			"name":       fmt.Sprintf("Bench User %d", i),
			"status":     "active",
			"source":     "api",
			"created_at": time.Now().Format(time.RFC3339),
		}

		startTime := time.Now()

		execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
		if err != nil {
			b.Fatalf("failed to publish event: %v", err)
		}

		// Measure time until execution starts
		execUUID, err := uuid.Parse(execution.ID)
		if err != nil {
			b.Fatalf("invalid execution ID: %v", err)
		}

		// Wait for execution to start (status changes from pending)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				b.Fatal("context cancelled")
			case <-ticker.C:
				exec, err := env.ExecutionRepo.FindByIDWithRelations(ctx, execUUID)
				if err != nil {
					continue
				}

				if exec.Status != "pending" {
					latency := time.Since(startTime)
					b.ReportMetric(float64(latency.Microseconds()), "μs/event-to-start")
					goto nextIteration
				}
			}
		}
	nextIteration:
	}
}

// BenchmarkWorkflowExecutionDuration measures the total duration of workflow execution
func BenchmarkWorkflowExecutionDuration(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	env := setupE2EEnvironment(&testing.T{})
	defer env.cleanup(&testing.T{})

	ctx := context.Background()

	// Create workflow
	workflowModel := env.createWorkflowFromFixture(&testing.T{}, ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Reset mocks before each iteration
		env.Mocks.ExampleAPI.Reset()
		env.Mocks.SendGridAPI.Reset()
		env.Mocks.SegmentAPI.Reset()

		eventData := map[string]interface{}{
			"user_id":    fmt.Sprintf("usr_duration_%d", i),
			"email":      fmt.Sprintf("duration%d@example.com", i),
			"name":       fmt.Sprintf("Duration User %d", i),
			"status":     "active",
			"source":     "api",
			"created_at": time.Now().Format(time.RFC3339),
		}

		startTime := time.Now()

		execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
		if err != nil {
			b.Fatalf("failed to publish event: %v", err)
		}

		execUUID, err := uuid.Parse(execution.ID)
		if err != nil {
			b.Fatalf("invalid execution ID: %v", err)
		}

		// Wait for execution to complete
		execModel := waitForExecution(&testing.T{}, ctx, env.ExecutionRepo, execUUID, 30*time.Second)
		if execModel.Status != "completed" {
			b.Fatalf("execution failed: %s - %s", execModel.Status, execModel.Error)
		}

		duration := time.Since(startTime)
		b.ReportMetric(float64(duration.Milliseconds()), "ms/execution")
	}
}

// BenchmarkConcurrentThroughput measures throughput under concurrent load
func BenchmarkConcurrentThroughput(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	concurrencyLevels := []int{10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency_%d", concurrency), func(b *testing.B) {
			env := setupE2EEnvironment(&testing.T{})
			defer env.cleanup(&testing.T{})

			ctx := context.Background()

			// Create workflow
			workflowModel := env.createWorkflowFromFixture(&testing.T{}, ctx)

			b.ResetTimer()

			// Track total events processed
			totalEvents := 0
			startTime := time.Now()

			for i := 0; i < b.N; i++ {
				// Reset mocks
				env.Mocks.ExampleAPI.Reset()
				env.Mocks.SendGridAPI.Reset()
				env.Mocks.SegmentAPI.Reset()

				var wg sync.WaitGroup
				executions := make([]*uuid.UUID, concurrency)
				errors := make([]error, concurrency)

				// Publish events concurrently
				for j := 0; j < concurrency; j++ {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()

						eventData := map[string]interface{}{
							"user_id":    fmt.Sprintf("usr_throughput_%d_%d", i, idx),
							"email":      fmt.Sprintf("throughput%d_%d@example.com", i, idx),
							"name":       fmt.Sprintf("Throughput User %d-%d", i, idx),
							"status":     "active",
							"source":     "api",
							"created_at": time.Now().Format(time.RFC3339),
						}

						execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
						if err != nil {
							errors[idx] = err
							return
						}

						execUUID, err := uuid.Parse(execution.ID)
						if err != nil {
							errors[idx] = err
							return
						}

						executions[idx] = &execUUID
					}(j)
				}

				wg.Wait()

				// Check for errors
				for idx, err := range errors {
					if err != nil {
						b.Fatalf("event %d failed: %v", idx, err)
					}
				}

				// Wait for all executions to complete
				for idx, execUUID := range executions {
					if execUUID == nil {
						continue
					}

					execModel := waitForExecution(&testing.T{}, ctx, env.ExecutionRepo, *execUUID, 30*time.Second)
					if execModel.Status != "completed" {
						b.Fatalf("execution %d failed: %s - %s", idx, execModel.Status, execModel.Error)
					}
				}

				totalEvents += concurrency
			}

			duration := time.Since(startTime)

			// Report throughput metrics
			throughput := float64(totalEvents) / duration.Seconds()
			b.ReportMetric(throughput, "events/sec")
			b.ReportMetric(float64(duration.Milliseconds())/float64(totalEvents), "ms/event")
		})
	}
}

// BenchmarkWorkflowCreation measures the performance of workflow creation
func BenchmarkWorkflowCreation(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	env := setupE2EEnvironment(&testing.T{})
	defer env.cleanup(&testing.T{})

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		startTime := time.Now()

		_ = env.createWorkflowFromFixture(&testing.T{}, ctx)

		duration := time.Since(startTime)
		b.ReportMetric(float64(duration.Microseconds()), "μs/workflow-create")
	}
}

// BenchmarkDAGValidation measures the performance of DAG validation
func BenchmarkDAGValidation(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	env := setupE2EEnvironment(&testing.T{})
	defer env.cleanup(&testing.T{})

	ctx := context.Background()

	// Create workflow once
	workflowModel := env.createWorkflowFromFixture(&testing.T{}, ctx)

	// Convert to domain model
	workflow := storagemodels.WorkflowModelToDomain(workflowModel)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := workflow.Validate()
		if err != nil {
			b.Fatalf("validation failed: %v", err)
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/validation")
}

// BenchmarkTemplateResolution measures the performance of template variable resolution
func BenchmarkTemplateResolution(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	env := setupE2EEnvironment(&testing.T{})
	defer env.cleanup(&testing.T{})

	ctx := context.Background()

	// Create workflow
	workflowModel := env.createWorkflowFromFixture(&testing.T{}, ctx)

	eventData := map[string]interface{}{
		"user_id":    "usr_template_bench",
		"email":      "template@example.com",
		"name":       "Template User",
		"status":     "active",
		"source":     "api",
		"created_at": time.Now().Format(time.RFC3339),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Reset mocks
		env.Mocks.ExampleAPI.Reset()
		env.Mocks.SendGridAPI.Reset()
		env.Mocks.SegmentAPI.Reset()

		execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
		if err != nil {
			b.Fatalf("failed to execute: %v", err)
		}

		execUUID, err := uuid.Parse(execution.ID)
		if err != nil {
			b.Fatalf("invalid execution ID: %v", err)
		}

		// Wait for completion
		_ = waitForExecution(&testing.T{}, ctx, env.ExecutionRepo, execUUID, 30*time.Second)
	}
}

// BenchmarkNodeExecutionRetry measures the performance of node retry logic
func BenchmarkNodeExecutionRetry(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	env := setupE2EEnvironment(&testing.T{})
	defer env.cleanup(&testing.T{})

	ctx := context.Background()

	// Create a simple workflow with retry configuration
	workflowModel := env.createWorkflowFromFixture(&testing.T{}, ctx)

	// Modify first node to fail initially (using invalid URL)
	for _, node := range workflowModel.Nodes {
		if node.NodeID == "create_profile" {
			config := node.Config
			// Use a URL that will fail quickly
			config["url"] = "http://invalid-host-for-bench.local/profiles"
			config["max_attempts"] = 3
			config["initial_delay"] = "10ms"
			config["max_delay"] = "100ms"
			node.Config = config
			_, err := env.DB.NewUpdate().
				Model(node).
				Column("config").
				Where("id = ?", node.ID).
				Exec(ctx)
			if err != nil {
				b.Fatalf("failed to update node: %v", err)
			}
			break
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		eventData := map[string]interface{}{
			"user_id":    fmt.Sprintf("usr_retry_%d", i),
			"email":      fmt.Sprintf("retry%d@example.com", i),
			"name":       fmt.Sprintf("Retry User %d", i),
			"status":     "active",
			"source":     "api",
			"created_at": time.Now().Format(time.RFC3339),
		}

		startTime := time.Now()

		execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
		if err != nil {
			b.Fatalf("failed to publish event: %v", err)
		}

		execUUID, err := uuid.Parse(execution.ID)
		if err != nil {
			b.Fatalf("invalid execution ID: %v", err)
		}

		// Wait for execution to fail (after retries)
		_ = waitForExecution(&testing.T{}, ctx, env.ExecutionRepo, execUUID, 10*time.Second)

		duration := time.Since(startTime)
		b.ReportMetric(float64(duration.Milliseconds()), "ms/retry-cycle")
	}
}

// BenchmarkMemoryUsage measures memory allocation during workflow execution
func BenchmarkMemoryUsage(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	env := setupE2EEnvironment(&testing.T{})
	defer env.cleanup(&testing.T{})

	ctx := context.Background()

	// Create workflow
	workflowModel := env.createWorkflowFromFixture(&testing.T{}, ctx)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Reset mocks
		env.Mocks.ExampleAPI.Reset()
		env.Mocks.SendGridAPI.Reset()
		env.Mocks.SegmentAPI.Reset()

		eventData := map[string]interface{}{
			"user_id":    fmt.Sprintf("usr_mem_%d", i),
			"email":      fmt.Sprintf("mem%d@example.com", i),
			"name":       fmt.Sprintf("Memory User %d", i),
			"status":     "active",
			"source":     "api",
			"created_at": time.Now().Format(time.RFC3339),
		}

		execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
		if err != nil {
			b.Fatalf("failed to publish event: %v", err)
		}

		execUUID, err := uuid.Parse(execution.ID)
		if err != nil {
			b.Fatalf("invalid execution ID: %v", err)
		}

		_ = waitForExecution(&testing.T{}, ctx, env.ExecutionRepo, execUUID, 30*time.Second)
	}
}
