package httpclient

import (
	"net/http"
	"time"
)

// Option is a functional option for configuring the HTTP client.
type Option func(*Client)

// WithHTTPClient sets the underlying http.Client.
// If not provided, a default client with 30s timeout is used.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.base = client
		}
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.base.Timeout = timeout
	}
}

// WithTransport sets the HTTP transport.
// Note: If middleware is also configured, the transport will be wrapped.
func WithTransport(transport http.RoundTripper) Option {
	return func(c *Client) {
		c.base.Transport = transport
	}
}

// WithMiddleware adds middleware to the client.
// Middleware is applied in reverse order to create the chain:
// first middleware in the slice becomes the outermost layer.
//
// Example:
//
//	WithMiddleware(A, B, C) creates chain: A(B(C(transport)))
//	Request flow: A -> B -> C -> transport -> server
//	Response flow: server -> transport -> C -> B -> A
//
// This allows natural reading order where outer concerns (logging, observability)
// come first, followed by inner concerns (rate limiting, retries).
func WithMiddleware(middleware ...Middleware) Option {
	return func(c *Client) {
		c.middleware = append(c.middleware, middleware...)
	}
}
