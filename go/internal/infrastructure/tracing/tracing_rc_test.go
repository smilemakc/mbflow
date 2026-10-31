package tracing

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// rcCollector is a fake OTLP/HTTP collector.
type rcCollector struct {
	mu       sync.Mutex
	requests int
	paths    []string
	ctype    string
	bodies   [][]byte
}

func rcStartCollector(t *testing.T) (*rcCollector, string) {
	t.Helper()
	c := &rcCollector{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		c.mu.Lock()
		c.requests++
		c.paths = append(c.paths, r.URL.Path)
		c.ctype = r.Header.Get("Content-Type")
		c.bodies = append(c.bodies, body)
		c.mu.Unlock()
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return c, strings.TrimPrefix(srv.URL, "http://")
}

// rcRestoreGlobals restores the global tracer provider and propagator that
// NewProvider replaces.
func rcRestoreGlobals(t *testing.T) {
	t.Helper()
	tp := otel.GetTracerProvider()
	prop := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(prop)
	})
}

func TestRCNewProvider_ExportsSpansToCollector(t *testing.T) {
	rcRestoreGlobals(t)
	collector, endpoint := rcStartCollector(t)

	p, err := NewProvider(context.Background(), Config{
		Enabled:     true,
		ServiceName: "rc-service",
		Endpoint:    endpoint,
		Insecure:    true,
		SampleRate:  1.0,
	})
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Same(t, p.tp, otel.GetTracerProvider(), "provider is installed globally")
	fields := otel.GetTextMapPropagator().Fields()
	assert.Contains(t, fields, "traceparent")
	assert.Contains(t, fields, "baggage")

	ctx, span := p.Tracer().Start(context.Background(), "rc-operation")
	assert.True(t, span.IsRecording(), "AlwaysSample records spans")
	AddSpanEvent(ctx, "rc-event")
	RecordError(ctx, errors.New("rc-error"))
	assert.Equal(t, span.SpanContext().TraceID(), SpanFromContext(ctx).SpanContext().TraceID())

	// The package-level helper uses the global provider.
	_, helperSpan := StartSpan(ctx, "rc-child")
	assert.True(t, helperSpan.IsRecording())
	assert.Equal(t, span.SpanContext().TraceID(), helperSpan.SpanContext().TraceID())
	helperSpan.End()
	span.End()

	// Propagation round-trip through HTTP headers.
	carrier := propagation.HeaderCarrier(http.Header{})
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	assert.NotEmpty(t, carrier.Get("traceparent"))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, p.Shutdown(shutdownCtx), "shutdown flushes the batcher")

	collector.mu.Lock()
	defer collector.mu.Unlock()
	require.GreaterOrEqual(t, collector.requests, 1)
	assert.Equal(t, "/v1/traces", collector.paths[0])
	assert.Equal(t, "application/x-protobuf", collector.ctype)
	var all []byte
	for _, b := range collector.bodies {
		all = append(all, b...)
	}
	assert.Contains(t, string(all), "rc-operation")
	assert.Contains(t, string(all), "rc-child")
	assert.Contains(t, string(all), "rc-service")
}

func TestRCNewProvider_Samplers(t *testing.T) {
	rcRestoreGlobals(t)
	_, endpoint := rcStartCollector(t)

	cases := []struct {
		name      string
		rate      float64
		recording *bool
	}{
		{"never", 0, rcBool(false)},
		{"negative", -1, rcBool(false)},
		{"always above one", 2, rcBool(true)},
		{"ratio", 0.5, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewProvider(context.Background(), Config{
				Enabled: true, ServiceName: "rc", Endpoint: endpoint, Insecure: true, SampleRate: tc.rate,
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = p.Shutdown(context.Background()) })

			_, span := p.Tracer().Start(context.Background(), "s")
			defer span.End()
			if tc.recording != nil {
				assert.Equal(t, *tc.recording, span.IsRecording())
			}
		})
	}
}

func rcBool(b bool) *bool { return &b }

func TestRCNewProvider_SecureEndpointOption(t *testing.T) {
	rcRestoreGlobals(t)
	// TLS exporter configuration is created lazily; construction must succeed
	// without contacting the endpoint.
	p, err := NewProvider(context.Background(), Config{
		Enabled: true, ServiceName: "rc", Endpoint: "127.0.0.1:1", Insecure: false, SampleRate: 1,
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = p.Shutdown(ctx) // nothing exported, nothing to flush
}

func TestRCProvider_ShutdownWithoutTracerProvider(t *testing.T) {
	p := &Provider{}
	assert.NoError(t, p.Shutdown(context.Background()))
}

func TestRCHelpers_NonRecordingSpanIgnoresEvents(t *testing.T) {
	ctx := trace.ContextWithSpanContext(context.Background(), trace.SpanContext{})
	// Must be no-ops on a non-recording span.
	AddSpanEvent(ctx, "e")
	RecordError(ctx, errors.New("x"))
	assert.False(t, SpanFromContext(ctx).IsRecording())
}
