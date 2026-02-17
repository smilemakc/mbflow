package internal

import (
	"context"
	"io"
)

// Method represents an HTTP-like method for transport operations.
type Method string

const (
	MethodGet    Method = "GET"
	MethodPost   Method = "POST"
	MethodPut    Method = "PUT"
	MethodDelete Method = "DELETE"
)

// Request represents a transport-agnostic API request.
type Request struct {
	Method  Method
	Path    string
	Body    interface{}
	Query   map[string]string
	Headers map[string]string
}

// Response represents a transport-agnostic API response.
type Response struct {
	StatusCode int
	Body       io.ReadCloser
	Data       interface{}
}

// Transport is the interface that HTTP and gRPC clients implement.
type Transport interface {
	Do(ctx context.Context, req *Request) (*Response, error)
	Close() error
}
