// Package httpclient provides an HTTP client with middleware support.
package httpclient

import (
	"net/http"
	"time"
)

// Client is an HTTP client that supports middleware chaining.
type Client struct {
	base       *http.Client
	middleware []Middleware
}

// Middleware wraps an http.RoundTripper to add behavior.
// Middleware is applied in order: first middleware is outermost.
type Middleware func(http.RoundTripper) http.RoundTripper

// New creates a new HTTP client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		base: &http.Client{
			Timeout: 30 * time.Second,
		},
		middleware: []Middleware{},
	}

	for _, opt := range opts {
		opt(c)
	}

	// Build middleware chain
	if len(c.middleware) > 0 {
		transport := c.base.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}

		// Apply middleware in reverse order so first middleware is outermost
		for i := len(c.middleware) - 1; i >= 0; i-- {
			transport = c.middleware[i](transport)
		}

		c.base.Transport = transport
	}

	return c
}

// Do executes an HTTP request using the configured middleware chain.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.base.Do(req)
}

// HTTPClient returns the underlying http.Client.
// This is useful when the client needs to be passed to code that expects *http.Client.
func (c *Client) HTTPClient() *http.Client {
	return c.base
}
