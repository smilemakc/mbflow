// Package tracing provides OpenTelemetry distributed tracing support.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Config holds tracing configuration.
type Config struct {
	Enabled     bool    `env:"OTEL_ENABLED" default:"false"`
	ServiceName string  `env:"OTEL_SERVICE_NAME" default:"mbflow"`
	Endpoint    string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT" default:"localhost:4318"`
	Insecure    bool    `env:"OTEL_EXPORTER_OTLP_INSECURE" default:"true"`
	SampleRate  float64 `env:"OTEL_SAMPLE_RATE" default:"1.0"`
}

// Provider wraps the OpenTelemetry TracerProvider for lifecycle management.
type Provider struct {
	tp     *sdktrace.TracerProvider
	tracer trace.Tracer
}

// NewProvider creates a new tracing provider with the given configuration.
// Returns nil if tracing is disabled.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// Create OTLP exporter
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler based on sample rate
	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// Create TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global TracerProvider and propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{
		tp:     tp,
		tracer: tp.Tracer(cfg.ServiceName),
	}, nil
}

// Tracer returns the tracer instance.
func (p *Provider) Tracer() trace.Tracer {
	if p == nil {
		return noop.NewTracerProvider().Tracer("")
	}
	return p.tracer
}

// Shutdown gracefully shuts down the tracing provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.tp == nil {
		return nil
	}
	return p.tp.Shutdown(ctx)
}

// StartSpan starts a new span from context.
// This is a convenience function for common span creation.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("mbflow").Start(ctx, name, opts...)
}

// SpanFromContext returns the span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanEvent adds an event to the current span.
func AddSpanEvent(ctx context.Context, name string, attrs ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, attrs...)
	}
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err, opts...)
	}
}
