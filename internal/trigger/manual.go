package trigger

import (
	"context"
)

type ManualTrigger struct{}

func NewManual() *ManualTrigger { return &ManualTrigger{} }

func (t *ManualTrigger) Fire(ctx context.Context, payload map[string]any) (context.Context, map[string]any) {
	return ctx, payload
}
