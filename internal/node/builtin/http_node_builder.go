package builtin

import (
	core "mbflow/internal/node"
	"net/http"
	"time"
)

type HTTPRequestNodeBuilder[T any] struct {
	id, name, version string
	method            string
	url               string
	headers           map[string]string
	timeout           time.Duration
	client            HTTPClient
	adapter           core.DataAdapter[*http.Response, T]
	failOnStatus      func(code int) bool
}

func NewHTTPRequestNodeBuilder[T any]() *HTTPRequestNodeBuilder[T] {
	return &HTTPRequestNodeBuilder[T]{headers: map[string]string{}}
}
func (b *HTTPRequestNodeBuilder[T]) ID(id string) *HTTPRequestNodeBuilder[T] { b.id = id; return b }
func (b *HTTPRequestNodeBuilder[T]) Name(name string) *HTTPRequestNodeBuilder[T] {
	b.name = name
	return b
}
func (b *HTTPRequestNodeBuilder[T]) Version(v string) *HTTPRequestNodeBuilder[T] {
	b.version = v
	return b
}
func (b *HTTPRequestNodeBuilder[T]) Method(m string) *HTTPRequestNodeBuilder[T] {
	b.method = m
	return b
}
func (b *HTTPRequestNodeBuilder[T]) URL(u string) *HTTPRequestNodeBuilder[T] { b.url = u; return b }
func (b *HTTPRequestNodeBuilder[T]) Header(k, v string) *HTTPRequestNodeBuilder[T] {
	b.headers[k] = v
	return b
}
func (b *HTTPRequestNodeBuilder[T]) Timeout(d time.Duration) *HTTPRequestNodeBuilder[T] {
	b.timeout = d
	return b
}
func (b *HTTPRequestNodeBuilder[T]) Client(c HTTPClient) *HTTPRequestNodeBuilder[T] {
	b.client = c
	return b
}
func (b *HTTPRequestNodeBuilder[T]) Adapter(a core.DataAdapter[*http.Response, T]) *HTTPRequestNodeBuilder[T] {
	b.adapter = a
	return b
}
func (b *HTTPRequestNodeBuilder[T]) FailOnStatus(fn func(code int) bool) *HTTPRequestNodeBuilder[T] {
	b.failOnStatus = fn
	return b
}

func (b *HTTPRequestNodeBuilder[T]) Build() *HTTPRequestNode[T] {
	cfg := HTTPRequestConfig{Method: b.method, URL: b.url, Headers: b.headers, Timeout: b.timeout}
	n := NewHTTPRequestNode[T](b.id, b.name, b.version, cfg, b.client, b.adapter)
	n.failOnStatus = b.failOnStatus
	return n
}
