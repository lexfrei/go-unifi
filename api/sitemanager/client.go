package sitemanager

//go:generate oapi-codegen -config .oapi-codegen.yaml openapi.yaml

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"golang.org/x/time/rate"

	"github.com/lexfrei/go-unifi/internal/ratelimit"
	"github.com/lexfrei/go-unifi/internal/retry"
)

const (
	// DefaultBaseURL is the default Unifi API base URL.
	DefaultBaseURL = "https://api.ui.com"

	// V1RateLimit is the rate limit for v1 endpoints (requests per minute).
	V1RateLimit = 10000
	// EARateLimit is the rate limit for EA endpoints (requests per minute).
	EARateLimit = 100

	// DefaultMaxRetries is the default number of retries for failed requests.
	DefaultMaxRetries = 3
	// DefaultRetryWaitTime is the default wait time between retries.
	DefaultRetryWaitTime = 1 * time.Second
	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second
)

// UnifiClient wraps the generated API client with rate limiting and retry logic.
// It uses separate rate limiters for v1 and Early Access endpoints.
type UnifiClient struct {
	client        *ClientWithResponses
	httpClient    *http.Client
	v1RateLimiter *rate.Limiter
	eaRateLimiter *rate.Limiter
	maxRetries    int
	retryWait     time.Duration
}

// ClientConfig holds configuration for the Unifi API client.
type ClientConfig struct {
	// APIKey is the Unifi API key for authentication
	APIKey string

	// BaseURL is the base URL for the API (defaults to https://api.ui.com)
	BaseURL string

	// HTTPClient is the HTTP client to use (optional)
	HTTPClient *http.Client

	// V1RateLimitPerMinute sets the rate limit for v1 endpoints (defaults to 10000)
	V1RateLimitPerMinute int

	// EARateLimitPerMinute sets the rate limit for Early Access endpoints (defaults to 100)
	EARateLimitPerMinute int

	// MaxRetries sets maximum number of retries for failed requests
	MaxRetries int

	// RetryWaitTime sets the wait time between retries
	RetryWaitTime time.Duration

	// Timeout sets the HTTP client timeout
	Timeout time.Duration
}

// New creates a new Unifi API client with default settings.
// This is the recommended way to create a client for most use cases.
//
// The client automatically handles rate limiting for both v1 and Early Access endpoints:
//   - v1 endpoints: 10,000 requests/minute
//   - Early Access endpoints: 100 requests/minute
//
// Other default settings:
//   - Base URL: https://api.ui.com
//   - Max retries: 3
//   - Retry wait time: 1 second
//   - Timeout: 30 seconds
//
// For custom configuration, use NewWithConfig.
//
// Example:
//
//	client, err := sitemanager.New("your-api-key")
func New(apiKey string) (*UnifiClient, error) {
	return NewWithConfig(&ClientConfig{
		APIKey: apiKey,
	})
}

// NewWithConfig creates a new Unifi API client with custom configuration.
// Use this when you need to customize rate limits, timeouts, or other settings.
//
// The client uses separate rate limiters for v1 and Early Access endpoints,
// automatically selecting the appropriate limiter based on the request URL.
//
// Example:
//
//	client, err := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
//	    APIKey:               "your-api-key",
//	    V1RateLimitPerMinute: 5000,  // Custom v1 rate limit
//	    EARateLimitPerMinute: 50,    // Custom EA rate limit
//	})
func NewWithConfig(cfg *ClientConfig) (*UnifiClient, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	// Set defaults
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.V1RateLimitPerMinute == 0 {
		cfg.V1RateLimitPerMinute = V1RateLimit
	}
	if cfg.EARateLimitPerMinute == 0 {
		cfg.EARateLimitPerMinute = EARateLimit
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = DefaultMaxRetries
	}
	if cfg.RetryWaitTime == 0 {
		cfg.RetryWaitTime = DefaultRetryWaitTime
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	// Create HTTP client if not provided
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	// Create separate rate limiters for v1 and EA endpoints
	v1RateLimiter := ratelimit.NewRateLimiter(cfg.V1RateLimitPerMinute)
	eaRateLimiter := ratelimit.NewRateLimiter(cfg.EARateLimitPerMinute)

	// Wrap HTTP client with rate limiters and retry logic
	rateLimitedClient := &rateLimitedHTTPClient{
		client:        httpClient,
		v1RateLimiter: v1RateLimiter,
		eaRateLimiter: eaRateLimiter,
		maxRetries:    cfg.MaxRetries,
		retryWait:     cfg.RetryWaitTime,
	}

	// Create request editor to add API key header
	requestEditor := func(_ context.Context, req *http.Request) error {
		req.Header.Set("X-Api-Key", cfg.APIKey)
		req.Header.Set("Accept", "application/json")
		return nil
	}

	// Create generated client
	generatedClient, err := NewClientWithResponses(
		cfg.BaseURL,
		WithHTTPClient(rateLimitedClient),
		WithRequestEditorFn(requestEditor),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create API client")
	}

	return &UnifiClient{
		client:        generatedClient,
		httpClient:    httpClient,
		v1RateLimiter: v1RateLimiter,
		eaRateLimiter: eaRateLimiter,
		maxRetries:    cfg.MaxRetries,
		retryWait:     cfg.RetryWaitTime,
	}, nil
}

// rateLimitedHTTPClient wraps http.Client with rate limiting and retry logic.
// It uses separate rate limiters for v1 and Early Access endpoints.
type rateLimitedHTTPClient struct {
	client        *http.Client
	v1RateLimiter *rate.Limiter
	eaRateLimiter *rate.Limiter
	maxRetries    int
	retryWait     time.Duration
}

// isEAEndpoint checks if the request URL is an Early Access endpoint.
func (c *rateLimitedHTTPClient) isEAEndpoint(req *http.Request) bool {
	path := req.URL.Path
	return strings.HasPrefix(path, "/api/ea/")
}

// Do executes HTTP request with rate limiting and retry logic.
func (c *rateLimitedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	var resp *http.Response
	var err error

	// Select appropriate rate limiter based on endpoint
	rateLimiter := c.v1RateLimiter
	if c.isEAEndpoint(req) {
		rateLimiter = c.eaRateLimiter
	}

	// Apply rate limiting
	err = rateLimiter.Wait(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "rate limiter wait failed")
	}

	// Retry loop
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-time.After(c.retryWait * time.Duration(attempt)):
			case <-ctx.Done():
				return nil, errors.Wrap(ctx.Err(), "context cancelled during retry wait")
			}
		}

		resp, err = c.client.Do(req)
		if err != nil {
			// Network error - retry
			if attempt < c.maxRetries {
				continue
			}
			return nil, errors.Wrapf(err, "request failed after %d attempts", attempt+1)
		}

		// Check status code
		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			// Success
			return resp, nil

		case resp.StatusCode == http.StatusTooManyRequests:
			// Rate limited - check Retry-After header
			resp.Body.Close()
			if retryAfter := retry.ParseRetryAfter(resp.Header.Get("Retry-After")); retryAfter > 0 {
				time.Sleep(retryAfter)
				continue
			}
			// Retry with exponential backoff
			if attempt < c.maxRetries {
				continue
			}
			//nolint:wrapcheck // Creating new error for rate limit exhaustion, no source error to wrap
			return nil, errors.Newf("rate limited after %d attempts", attempt+1)

		case resp.StatusCode >= 500:
			// Server error - retry
			resp.Body.Close()
			if attempt < c.maxRetries {
				continue
			}
			//nolint:wrapcheck // Creating new error for server error exhaustion, no source error to wrap
			return nil, errors.Newf("server error %d after %d attempts", resp.StatusCode, attempt+1)

		default:
			// Client error or other - don't retry
			return resp, nil
		}
	}

	return resp, errors.New("unexpected retry loop exit")
}

// ListHosts retrieves a list of all hosts associated with the UI account.
func (c *UnifiClient) ListHosts(ctx context.Context, params *ListHostsParams) (*HostsResponse, error) {
	resp, err := c.client.ListHostsWithResponse(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list hosts")
	}

	if resp.StatusCode() != http.StatusOK {
		if resp.JSON200 != nil {
			//nolint:wrapcheck // Creating new error for non-OK status, no source error to wrap
			return nil, errors.Newf("API error: status=%d", resp.StatusCode())
		}
		//nolint:wrapcheck // Creating new error for non-OK status, no source error to wrap
		return nil, errors.Newf("unexpected status code: %d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// GetHostByID retrieves detailed information about a specific host by ID.
func (c *UnifiClient) GetHostByID(ctx context.Context, id string) (*HostResponse, error) {
	resp, err := c.client.GetHostByIdWithResponse(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get host %s", id)
	}

	if resp.StatusCode() != http.StatusOK {
		//nolint:wrapcheck // Creating new error for non-OK status, no source error to wrap
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// ListSites retrieves a list of all sites associated with the UI account.
func (c *UnifiClient) ListSites(ctx context.Context) (*SitesResponse, error) {
	resp, err := c.client.ListSitesWithResponse(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list sites")
	}

	if resp.StatusCode() != http.StatusOK {
		//nolint:wrapcheck // Creating new error for non-OK status, no source error to wrap
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// ListDevices retrieves a list of UniFi devices.
func (c *UnifiClient) ListDevices(ctx context.Context, params *ListDevicesParams) (*DevicesResponse, error) {
	resp, err := c.client.ListDevicesWithResponse(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list devices")
	}

	if resp.StatusCode() != http.StatusOK {
		//nolint:wrapcheck // Creating new error for non-OK status, no source error to wrap
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// GetISPMetrics retrieves ISP metrics data across all sites.
func (c *UnifiClient) GetISPMetrics(ctx context.Context, metricType GetISPMetricsParamsType, params *GetISPMetricsParams) (*ISPMetricsResponse, error) {
	resp, err := c.client.GetISPMetricsWithResponse(ctx, metricType, params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ISP metrics for type %s", string(metricType))
	}

	if resp.StatusCode() != http.StatusOK {
		//nolint:wrapcheck // Creating new error for non-OK status, no source error to wrap
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// QueryISPMetrics queries ISP metrics for specific sites with custom parameters.
func (c *UnifiClient) QueryISPMetrics(ctx context.Context, metricType string, query ISPMetricsQuery) (*ISPMetricsQueryResponse, error) {
	resp, err := c.client.QueryISPMetricsWithResponse(ctx, metricType, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query ISP metrics for type %s", metricType)
	}

	if resp.StatusCode() != http.StatusOK {
		//nolint:wrapcheck // Creating new error for non-OK status, no source error to wrap
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// ListSDWANConfigs retrieves a list of all SD-WAN configurations.
func (c *UnifiClient) ListSDWANConfigs(ctx context.Context) (*SDWANConfigsResponse, error) {
	resp, err := c.client.ListSDWANConfigsWithResponse(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list SD-WAN configs")
	}

	if resp.StatusCode() != http.StatusOK {
		//nolint:wrapcheck // Creating new error for non-OK status, no source error to wrap
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// GetSDWANConfigByID retrieves detailed information about a specific SD-WAN configuration by ID.
func (c *UnifiClient) GetSDWANConfigByID(ctx context.Context, id string) (*SDWANConfigResponse, error) {
	resp, err := c.client.GetSDWANConfigByIdWithResponse(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get SD-WAN config %s", id)
	}

	if resp.StatusCode() != http.StatusOK {
		//nolint:wrapcheck // Creating new error for non-OK status, no source error to wrap
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// GetSDWANConfigStatus retrieves the deployment status of a specific SD-WAN configuration.
func (c *UnifiClient) GetSDWANConfigStatus(ctx context.Context, id string) (*SDWANConfigStatusResponse, error) {
	resp, err := c.client.GetSDWANConfigStatusWithResponse(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get SD-WAN config status %s", id)
	}

	if resp.StatusCode() != http.StatusOK {
		//nolint:wrapcheck // Creating new error for non-OK status, no source error to wrap
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}
