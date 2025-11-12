package middleware

import (
	"maps"
	"net/http"
)

// Auth returns a middleware that adds an authentication header to all requests.
// Common header names:
// - "X-Api-Key" for Site Manager API.
// - "X-API-KEY" for Network API.
// - "Authorization" for Bearer tokens.
func Auth(headerName, headerValue string) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return &authTransport{
			next:        next,
			headerName:  headerName,
			headerValue: headerValue,
		}
	}
}

type authTransport struct {
	next        http.RoundTripper
	headerName  string
	headerValue string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request to avoid modifying original
	req = cloneRequest(req)

	// Add auth header
	req.Header.Set(t.headerName, t.headerValue)

	//nolint:wrapcheck // Middleware passes through errors from next handler in chain
	return t.next.RoundTrip(req)
}

// cloneRequest creates a shallow copy of the request with a cloned header map.
func cloneRequest(req *http.Request) *http.Request {
	r := new(http.Request)
	*r = *req
	r.Header = make(http.Header, len(req.Header))
	maps.Copy(r.Header, req.Header)
	return r
}
