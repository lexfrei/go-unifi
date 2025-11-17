package sitemanager

//go:generate oapi-codegen -config .oapi-codegen.yaml openapi.yaml

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"golang.org/x/time/rate"

	"github.com/lexfrei/go-unifi/internal/httpclient"
	"github.com/lexfrei/go-unifi/internal/middleware"
	"github.com/lexfrei/go-unifi/internal/ratelimit"
	"github.com/lexfrei/go-unifi/internal/response"
	"github.com/lexfrei/go-unifi/observability"
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

// UnifiClient wraps the generated API client with composable middleware.
// It uses separate rate limiters for v1 and Early Access endpoints.
type UnifiClient struct {
	client *ClientWithResponses
}

// Compile-time check to ensure UnifiClient implements SiteManagerAPIClient interface.
var _ SiteManagerAPIClient = (*UnifiClient)(nil)

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

	// Logger for observability (optional, uses noop logger if nil)
	Logger observability.Logger

	// Metrics recorder for observability (optional, uses noop recorder if nil)
	Metrics observability.MetricsRecorder
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
//	    Logger:               myLogger,
//	    Metrics:              myMetrics,
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

	// Create separate rate limiters for v1 and EA endpoints
	v1RateLimiter := ratelimit.NewRateLimiter(cfg.V1RateLimitPerMinute)
	eaRateLimiter := ratelimit.NewRateLimiter(cfg.EARateLimitPerMinute)

	// Create selector function for dual rate limiters
	// EA endpoints start with /api/ea/, all others use v1 limiter
	rateLimiterSelector := func(req *http.Request) (*rate.Limiter, string) {
		if strings.HasPrefix(req.URL.Path, "/api/ea/") {
			return eaRateLimiter, "ea"
		}
		return v1RateLimiter, "v1"
	}

	// Build middleware chain (applied in reverse order: last = innermost, applied first)
	// Order from outside to inside: Observability -> RateLimit -> Retry
	httpClient := httpclient.New(
		httpclient.WithTimeout(cfg.Timeout),
		httpclient.WithMiddleware(
			middleware.Observability(cfg.Logger, cfg.Metrics),
			middleware.RateLimit(middleware.RateLimitConfig{
				Selector: rateLimiterSelector,
				Logger:   cfg.Logger,
				Metrics:  cfg.Metrics,
			}),
			middleware.Retry(middleware.RetryConfig{
				MaxRetries:  cfg.MaxRetries,
				InitialWait: cfg.RetryWaitTime,
				Logger:      cfg.Logger,
				Metrics:     cfg.Metrics,
			}),
		),
	)

	// Create request editor to add API key and Accept headers
	requestEditor := func(_ context.Context, req *http.Request) error {
		req.Header.Set("X-Api-Key", cfg.APIKey)
		req.Header.Set("Accept", "application/json")
		return nil
	}

	// Create generated client
	generatedClient, err := NewClientWithResponses(
		cfg.BaseURL,
		WithHTTPClient(httpClient.HTTPClient()),
		WithRequestEditorFn(requestEditor),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create API client")
	}

	return &UnifiClient{
		client: generatedClient,
	}, nil
}

// ListHosts retrieves a list of all hosts across all sites.
func (c *UnifiClient) ListHosts(ctx context.Context, params *ListHostsParams) (*HostsResponse, error) {
	resp, err := c.client.ListHostsWithResponse(ctx, params)
	var data *HostsResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list hosts")
}

// GetHostByID retrieves detailed information about a specific host.
func (c *UnifiClient) GetHostByID(ctx context.Context, hostID string) (*HostResponse, error) {
	resp, err := c.client.GetHostByIdWithResponse(ctx, hostID)
	var data *HostResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get host "+hostID)
}

// ListSites retrieves a list of all sites configured on the controller.
func (c *UnifiClient) ListSites(ctx context.Context) (*SitesResponse, error) {
	resp, err := c.client.ListSitesWithResponse(ctx)
	var data *SitesResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list sites")
}

// ListDevices retrieves a list of all devices across all sites.
func (c *UnifiClient) ListDevices(ctx context.Context, params *ListDevicesParams) (*DevicesResponse, error) {
	resp, err := c.client.ListDevicesWithResponse(ctx, params)
	var data *DevicesResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list devices")
}

// GetISPMetrics retrieves ISP performance metrics.
func (c *UnifiClient) GetISPMetrics(ctx context.Context, metricType GetISPMetricsParamsType, params *GetISPMetricsParams) (*ISPMetricsResponse, error) {
	resp, err := c.client.GetISPMetricsWithResponse(ctx, metricType, params)
	var data *ISPMetricsResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, fmt.Sprintf("failed to get ISP metrics of type %s", metricType))
}

// QueryISPMetrics queries ISP metrics with custom parameters.
func (c *UnifiClient) QueryISPMetrics(ctx context.Context, metricType string, query ISPMetricsQuery) (*ISPMetricsQueryResponse, error) {
	resp, err := c.client.QueryISPMetricsWithResponse(ctx, metricType, query)
	var data *ISPMetricsQueryResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to query ISP metrics of type "+metricType)
}

// ListSDWANConfigs retrieves a list of all SD-WAN configurations.
func (c *UnifiClient) ListSDWANConfigs(ctx context.Context) (*SDWANConfigsResponse, error) {
	resp, err := c.client.ListSDWANConfigsWithResponse(ctx)
	var data *SDWANConfigsResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list SD-WAN configs")
}

// GetSDWANConfigByID retrieves detailed information about a specific SD-WAN configuration.
func (c *UnifiClient) GetSDWANConfigByID(ctx context.Context, configID string) (*SDWANConfigResponse, error) {
	resp, err := c.client.GetSDWANConfigByIdWithResponse(ctx, configID)
	var data *SDWANConfigResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get SD-WAN config "+configID)
}

// GetSDWANConfigStatus retrieves the status of a specific SD-WAN configuration.
func (c *UnifiClient) GetSDWANConfigStatus(ctx context.Context, configID string) (*SDWANConfigStatusResponse, error) {
	resp, err := c.client.GetSDWANConfigStatusWithResponse(ctx, configID)
	var data *SDWANConfigStatusResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get SD-WAN config status for "+configID)
}
