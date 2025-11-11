package unifi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"golang.org/x/time/rate"
)

const (
	// DefaultBaseURL is the default Unifi API base URL.
	DefaultBaseURL = "https://api.ui.com"

	// Rate limits per API version
	V1RateLimit = 10000 // requests per minute for v1 endpoints
	EARateLimit = 100   // requests per minute for EA endpoints

	// Retry configuration
	DefaultMaxRetries    = 3
	DefaultRetryWaitTime = 1 * time.Second
	DefaultTimeout       = 30 * time.Second
)

// UnifiClient wraps the generated API client with rate limiting and retry logic.
type UnifiClient struct {
	client      *ClientWithResponses
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	maxRetries  int
	retryWait   time.Duration
}

// ClientConfig holds configuration for the Unifi API client.
type ClientConfig struct {
	// APIKey is the Unifi API key for authentication
	APIKey string

	// BaseURL is the base URL for the API (defaults to https://api.ui.com)
	BaseURL string

	// HTTPClient is the HTTP client to use (optional)
	HTTPClient *http.Client

	// RateLimitPerMinute sets the rate limit (defaults to 10000 for v1)
	RateLimitPerMinute int

	// MaxRetries sets maximum number of retries for failed requests
	MaxRetries int

	// RetryWaitTime sets the wait time between retries
	RetryWaitTime time.Duration

	// Timeout sets the HTTP client timeout
	Timeout time.Duration
}

// NewUnifiClient creates a new Unifi API client with rate limiting and retry logic.
func NewUnifiClient(cfg ClientConfig) (*UnifiClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	// Set defaults
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.RateLimitPerMinute == 0 {
		cfg.RateLimitPerMinute = V1RateLimit
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

	// Wrap HTTP client with rate limiter and retry logic
	rateLimitedClient := &rateLimitedHTTPClient{
		client:      httpClient,
		rateLimiter: rate.NewLimiter(rate.Limit(cfg.RateLimitPerMinute)/60.0, cfg.RateLimitPerMinute/60),
		maxRetries:  cfg.MaxRetries,
		retryWait:   cfg.RetryWaitTime,
	}

	// Create request editor to add API key header
	requestEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-API-Key", cfg.APIKey)
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
		client:      generatedClient,
		httpClient:  httpClient,
		rateLimiter: rateLimitedClient.rateLimiter,
		maxRetries:  cfg.MaxRetries,
		retryWait:   cfg.RetryWaitTime,
	}, nil
}

// rateLimitedHTTPClient wraps http.Client with rate limiting and retry logic.
type rateLimitedHTTPClient struct {
	client      *http.Client
	rateLimiter *rate.Limiter
	maxRetries  int
	retryWait   time.Duration
}

// Do executes HTTP request with rate limiting and retry logic.
func (c *rateLimitedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, errors.Wrap(err, "rate limiter wait failed")
	}

	var resp *http.Response
	var err error

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
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil {
					resp.Body.Close()
					time.Sleep(time.Duration(seconds) * time.Second)
					continue
				}
			}
			// Retry with exponential backoff
			resp.Body.Close()
			if attempt < c.maxRetries {
				continue
			}
			return nil, errors.Newf("rate limited after %d attempts", attempt+1)

		case resp.StatusCode >= 500:
			// Server error - retry
			resp.Body.Close()
			if attempt < c.maxRetries {
				continue
			}
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
			return nil, errors.Newf("API error: status=%d", resp.StatusCode())
		}
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
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// ListDevices retrieves a list of UniFi devices.
func (c *UnifiClient) ListDevices(ctx context.Context) (*DevicesResponse, error) {
	resp, err := c.client.ListDevicesWithResponse(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list devices")
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}
