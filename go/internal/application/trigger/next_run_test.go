package trigger

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func TestNextRun(t *testing.T) {
	after := time.Date(2026, 10, 31, 10, 7, 30, 0, time.UTC)

	t.Run("cron", func(t *testing.T) {
		next := NextRun("cron", true, map[string]any{"schedule": "0 * * * *"}, after)
		require.NotNil(t, next)
		assert.True(t, next.Equal(time.Date(2026, 10, 31, 11, 0, 0, 0, time.UTC)), next.String())
	})

	t.Run("cron with timezone", func(t *testing.T) {
		next := NextRun("cron", true, map[string]any{"schedule": "0 9 * * *", "timezone": "Europe/Berlin"}, after)
		require.NotNil(t, next)
		// 09:00 Berlin on Nov 1 2026 (CET, UTC+1) is 08:00 UTC.
		assert.True(t, next.Equal(time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)), next.String())
	})

	t.Run("interval aligned like the scheduler", func(t *testing.T) {
		next := NextRun("interval", true, map[string]any{"interval": "5m"}, after)
		require.NotNil(t, next)
		assert.True(t, next.Equal(time.Date(2026, 10, 31, 10, 10, 0, 0, time.UTC)), next.String())

		sched, err := (&CronScheduler{}).parseIntervalSchedule(&models.Trigger{Config: map[string]any{"interval": "5m"}})
		require.NoError(t, err)
		assert.True(t, next.Equal(sched.Next(after)))
	})

	t.Run("interval in seconds", func(t *testing.T) {
		next := NextRun("interval", true, map[string]any{"interval": float64(60)}, after)
		require.NotNil(t, next)
		assert.True(t, next.Equal(time.Date(2026, 10, 31, 10, 8, 0, 0, time.UTC)), next.String())
	})

	t.Run("no next run", func(t *testing.T) {
		assert.Nil(t, NextRun("cron", false, map[string]any{"schedule": "0 * * * *"}, after), "disabled")
		assert.Nil(t, NextRun("manual", true, map[string]any{}, after), "manual")
		assert.Nil(t, NextRun("webhook", true, nil, after), "webhook")
		assert.Nil(t, NextRun("cron", true, nil, after), "nil config")
		assert.Nil(t, NextRun("cron", true, map[string]any{"schedule": "bogus"}, after), "invalid cron")
		assert.Nil(t, NextRun("interval", true, map[string]any{"interval": "-1s"}, after), "invalid interval")
		assert.Nil(t, NextRun("cron", true, map[string]any{"schedule": "0 0 30 2 *"}, after), "never matches")
	})
}

func TestCronScheduler_modelToDomain_NameDescription(t *testing.T) {
	cs := &CronScheduler{}
	tm := &storagemodels.TriggerModel{
		ID: uuid.New(), WorkflowID: uuid.New(), Type: "cron",
		Name: "Stored", Config: storagemodels.JSONBMap{"description": "legacy"},
	}
	got := cs.modelToDomain(tm)
	assert.Equal(t, "Stored", got.Name)
	assert.Equal(t, "legacy", got.Description)
}
