package mbflow

import (
	"log/slog"
	"time"
)

// Option configures the Client.
type Option func(*clientOptions) error

type clientOptions struct {
	httpEndpoint string
	grpcAddress  string
	grpcInsecure bool
	apiKey       string
	systemKey    string
	onBehalfOf   string
	timeout      time.Duration
	retryCount   int
	retryDelay   time.Duration
	logger       *slog.Logger
}

func defaultOptions() clientOptions {
	return clientOptions{
		timeout:    30 * time.Second,
		retryCount: 0,
		retryDelay: time.Second,
	}
}

// WithHTTP sets the HTTP endpoint URL.
func WithHTTP(endpoint string) Option {
	return func(o *clientOptions) error { o.httpEndpoint = endpoint; return nil }
}

// WithGRPC sets the gRPC server address.
func WithGRPC(address string) Option {
	return func(o *clientOptions) error { o.grpcAddress = address; return nil }
}

// WithGRPCInsecure disables TLS for gRPC connections.
func WithGRPCInsecure() Option {
	return func(o *clientOptions) error { o.grpcInsecure = true; return nil }
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(o *clientOptions) error { o.apiKey = key; return nil }
}

// WithSystemKey sets the system key for Service API authentication.
func WithSystemKey(key string) Option {
	return func(o *clientOptions) error { o.systemKey = key; return nil }
}

// WithOnBehalfOf sets the default user ID for impersonation.
func WithOnBehalfOf(userID string) Option {
	return func(o *clientOptions) error { o.onBehalfOf = userID; return nil }
}

// WithTimeout sets the default request timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *clientOptions) error { o.timeout = d; return nil }
}

// WithRetry configures automatic retry with backoff.
func WithRetry(count int, delay time.Duration) Option {
	return func(o *clientOptions) error { o.retryCount = count; o.retryDelay = delay; return nil }
}

// WithLogger sets a structured logger.
func WithLogger(l *slog.Logger) Option {
	return func(o *clientOptions) error { o.logger = l; return nil }
}

// RequestOption configures a single request.
type RequestOption func(*requestOptions)

type requestOptions struct {
	onBehalfOf string
}

// OnBehalfOf sets user impersonation for a single request.
func OnBehalfOf(userID string) RequestOption {
	return func(o *requestOptions) { o.onBehalfOf = userID }
}

func applyRequestOptions(opts []RequestOption) requestOptions {
	var ro requestOptions
	for _, o := range opts {
		o(&ro)
	}
	return ro
}
