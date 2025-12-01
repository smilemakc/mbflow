package builder

import (
	"fmt"
	"strings"
	"time"
)

// HTTPMethod sets the HTTP method for an HTTP node.
func HTTPMethod(method string) NodeOption {
	return func(nb *NodeBuilder) error {
		method = strings.ToUpper(method)
		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true, "DELETE": true,
			"PATCH": true, "HEAD": true, "OPTIONS": true,
		}
		if !validMethods[method] {
			return fmt.Errorf("invalid HTTP method: %s", method)
		}
		nb.config["method"] = method
		return nil
	}
}

// HTTPURL sets the URL for an HTTP node.
func HTTPURL(url string) NodeOption {
	return func(nb *NodeBuilder) error {
		if url == "" {
			return fmt.Errorf("HTTP URL cannot be empty")
		}
		nb.config["url"] = url
		return nil
	}
}

// HTTPBody sets the request body for an HTTP node.
func HTTPBody(body map[string]interface{}) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["body"] = body
		return nil
	}
}

// HTTPHeaders sets all headers for an HTTP node.
func HTTPHeaders(headers map[string]string) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["headers"] = headers
		return nil
	}
}

// HTTPHeader adds a single header to an HTTP node.
func HTTPHeader(key, value string) NodeOption {
	return func(nb *NodeBuilder) error {
		if key == "" {
			return fmt.Errorf("header key cannot be empty")
		}

		headers, ok := nb.config["headers"].(map[string]string)
		if !ok {
			headers = make(map[string]string)
			nb.config["headers"] = headers
		}
		headers[key] = value
		return nil
	}
}

// HTTPTimeout sets the timeout for an HTTP request.
func HTTPTimeout(timeout time.Duration) NodeOption {
	return func(nb *NodeBuilder) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
		nb.config["timeout"] = timeout.String()
		return nil
	}
}

// HTTPQueryParam adds a query parameter to the URL.
func HTTPQueryParam(key, value string) NodeOption {
	return func(nb *NodeBuilder) error {
		if key == "" {
			return fmt.Errorf("query param key cannot be empty")
		}

		params, ok := nb.config["query_params"].(map[string]string)
		if !ok {
			params = make(map[string]string)
			nb.config["query_params"] = params
		}
		params[key] = value
		return nil
	}
}

// NewHTTPGetNode creates a new HTTP GET node builder.
func NewHTTPGetNode(id, name, url string, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{
		HTTPMethod("GET"),
		HTTPURL(url),
	}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "http", name, allOpts...)
}

// NewHTTPPostNode creates a new HTTP POST node builder.
func NewHTTPPostNode(id, name, url string, body map[string]interface{}, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{
		HTTPMethod("POST"),
		HTTPURL(url),
	}
	if body != nil {
		allOpts = append(allOpts, HTTPBody(body))
	}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "http", name, allOpts...)
}

// NewHTTPPutNode creates a new HTTP PUT node builder.
func NewHTTPPutNode(id, name, url string, body map[string]interface{}, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{
		HTTPMethod("PUT"),
		HTTPURL(url),
	}
	if body != nil {
		allOpts = append(allOpts, HTTPBody(body))
	}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "http", name, allOpts...)
}

// NewHTTPDeleteNode creates a new HTTP DELETE node builder.
func NewHTTPDeleteNode(id, name, url string, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{
		HTTPMethod("DELETE"),
		HTTPURL(url),
	}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "http", name, allOpts...)
}

// NewHTTPPatchNode creates a new HTTP PATCH node builder.
func NewHTTPPatchNode(id, name, url string, body map[string]interface{}, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{
		HTTPMethod("PATCH"),
		HTTPURL(url),
	}
	if body != nil {
		allOpts = append(allOpts, HTTPBody(body))
	}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "http", name, allOpts...)
}
