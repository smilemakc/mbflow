package trigger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// WorkflowExecutor runs a workflow on behalf of a trigger.
// *engine.ExecutionManager implements it.
type WorkflowExecutor interface {
	Execute(ctx context.Context, workflowID string, input map[string]any, opts *engine.ExecutionOptions) (*models.Execution, error)
}

// Multi-replica deduplication.
//
// Every API replica runs its own cron scheduler and subscribes to the same
// Redis event channels, so without coordination each replica would start its
// own execution for every cron tick and every published event. Before
// starting an execution a replica claims the occurrence in Redis with
// SET NX; only the replica that wins the claim fires.
//
// The claim key identifies the occurrence the same way on every replica:
// trigger ID plus the scheduled tick (cron/interval) or the event ID (event
// triggers). Claims expire after claimTTL, which only has to outlast the
// clock skew and delivery delay between replicas.
const (
	claimKeyPrefix = "mbflow:trigger:claim:"
	claimTTL       = 10 * time.Minute
)

// replicaID identifies this process in claim values (debugging aid only).
var replicaID = func() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}
	return fmt.Sprintf("%s:%d", host, os.Getpid())
}()

// cronClaimKey returns the claim key for a scheduled tick of a trigger.
func cronClaimKey(triggerID string, tick time.Time) string {
	return fmt.Sprintf("%s%s:tick:%d", claimKeyPrefix, triggerID, tick.Unix())
}

// eventClaimKey returns the claim key for an event delivered to a trigger.
func eventClaimKey(triggerID, eventID string) string {
	return fmt.Sprintf("%s%s:event:%s", claimKeyPrefix, triggerID, eventID)
}

// eventDedupID returns the identifier used to deduplicate an event across
// replicas: the publisher-assigned event ID, or, for events published
// without one, a hash of the raw payload (pub/sub delivers byte-identical
// payloads to every subscriber).
func eventDedupID(event Event, rawPayload string) string {
	if event.ID != "" {
		return event.ID
	}
	sum := sha256.Sum256([]byte(rawPayload))
	return "sha256-" + hex.EncodeToString(sum[:])
}

// claimOccurrence reports whether this replica should fire the occurrence
// identified by key.
//
// It fails open: when Redis cannot be reached the occurrence is fired
// anyway (and the error is logged). Redis is a hard dependency of the
// trigger subsystem (state, event delivery), so an error here means an
// outage or a transient failure; failing closed would silently drop
// scheduled runs on every replica, which is undetectable, whereas failing
// open at worst produces a duplicate run that is visible in the execution
// history. With a single replica fail-open is exactly the old behavior.
func claimOccurrence(ctx context.Context, c *cache.RedisCache, key string) bool {
	if c == nil {
		return true
	}
	ok, err := c.SetNX(ctx, key, replicaID, claimTTL)
	if err != nil {
		slog.Warn("trigger claim failed, firing without deduplication",
			"key", key, "error", err)
		return true
	}
	return ok
}
