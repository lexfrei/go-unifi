package middleware

import (
	"crypto/tls"
	"net/http"
)

// TLSConfig returns a middleware that configures TLS for HTTPS connections.
// This is useful for:
// - Disabling certificate verification for self-signed certificates (development/testing).
// - Custom certificate validation.
// - Minimum TLS version enforcement.
func TLSConfig(config *tls.Config) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		// Get underlying transport or create default
		transport, ok := next.(*http.Transport)
		if !ok {
			defaultTransport, ok := http.DefaultTransport.(*http.Transport)
			if !ok {
				// Should never happen, but handle gracefully
				return next
			}
			transport = defaultTransport.Clone()
			transport.ForceAttemptHTTP2 = true
		} else {
			transport = transport.Clone()
		}

		// Apply TLS config
		transport.TLSClientConfig = config

		return transport
	}
}

// InsecureSkipVerify returns a TLS config that skips certificate verification.
// WARNING: This should only be used in development/testing with local controllers
// that use self-signed certificates. Never use in production.
func InsecureSkipVerify() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // This is an opt-in feature for dev/test environments
	}
}
