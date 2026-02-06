package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestNewProvider_Disabled(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Enabled: false,
	}

	provider, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.Nil(t, provider)
}

func TestProvider_Tracer_NilProvider(t *testing.T) {
	t.Parallel()

	var p *Provider
	tracer := p.Tracer()
	assert.NotNil(t, tracer)
}

func TestProvider_Shutdown_NilProvider(t *testing.T) {
	t.Parallel()

	var p *Provider
	err := p.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Enabled:     true,
		ServiceName: "test-service",
		Endpoint:    "localhost:4318",
		Insecure:    true,
		SampleRate:  1.0,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Equal(t, "localhost:4318", cfg.Endpoint)
	assert.True(t, cfg.Insecure)
	assert.Equal(t, 1.0, cfg.SampleRate)
}

func TestStartSpan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-span")
	defer span.End()

	assert.NotNil(t, span)
	assert.NotNil(t, ctx)
}

func TestSpanFromContext_NoSpan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	span := SpanFromContext(ctx)

	// Should return a no-op span
	assert.NotNil(t, span)
	assert.False(t, span.IsRecording())
}

func TestSpanFromContext_WithSpan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx, originalSpan := StartSpan(ctx, "test-span")
	defer originalSpan.End()

	retrievedSpan := SpanFromContext(ctx)
	assert.NotNil(t, retrievedSpan)
}

func TestAddSpanEvent_NoOp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Should not panic with no span
	AddSpanEvent(ctx, "test-event")
}

func TestRecordError_NoOp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Should not panic with no span
	RecordError(ctx, assert.AnError)
}

func TestRecordError_WithSpan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-span")
	defer span.End()

	// Should not panic
	RecordError(ctx, assert.AnError)
}

func TestAddSpanEvent_WithRecordingSpan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-span")
	defer span.End()

	// Should not panic even if span is not recording
	AddSpanEvent(ctx, "test-event")
}

func TestNewProvider_SampleRates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		sampleRate float64
	}{
		{"always sample", 1.0},
		{"never sample", 0.0},
		{"ratio based", 0.5},
		{"over 1.0", 1.5},
		{"negative", -0.5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				Enabled:     false, // Keep disabled to avoid actual connection
				SampleRate:  tc.sampleRate,
				ServiceName: "test",
			}

			// Should not error even with edge case sample rates
			provider, err := NewProvider(context.Background(), cfg)
			require.NoError(t, err)
			assert.Nil(t, provider)
		})
	}
}

func TestProvider_TracerInterface(t *testing.T) {
	t.Parallel()

	var p *Provider
	tracer := p.Tracer()

	// Ensure the tracer implements the interface
	var _ trace.Tracer = tracer
}
