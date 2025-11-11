package network

//go:generate oapi-codegen -config .oapi-codegen.yaml openapi.yaml

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"golang.org/x/time/rate"

	"github.com/lexfrei/go-unifi/internal/ratelimit"
	"github.com/lexfrei/go-unifi/internal/retry"
)

const (
	// DefaultRateLimit is the default rate limit for the Network API (requests per minute).
	DefaultRateLimit = 1000

	// DefaultMaxRetries is the default number of retries for failed requests.
	DefaultMaxRetries = 3
	// DefaultRetryWaitTime is the default wait time between retries.
	DefaultRetryWaitTime = 1 * time.Second
	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second
)

// APIClient wraps the generated API client with rate limiting and retry logic.
type APIClient struct {
	client      *ClientWithResponses
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	maxRetries  int
	retryWait   time.Duration
}

// ClientConfig holds configuration for the Network API client.
type ClientConfig struct {
	// ControllerURL is the base URL of the UniFi controller (e.g., "https://unifi.local" or "https://192.168.1.1")
	ControllerURL string

	// APIKey is the API key for authentication
	APIKey string

	// HTTPClient is the HTTP client to use (optional)
	HTTPClient *http.Client

	// InsecureSkipVerify disables TLS certificate verification (useful for self-signed certs)
	InsecureSkipVerify bool

	// RateLimitPerMinute sets the rate limit (defaults to 1000)
	RateLimitPerMinute int

	// MaxRetries sets maximum number of retries for failed requests
	MaxRetries int

	// RetryWaitTime sets the wait time between retries
	RetryWaitTime time.Duration

	// Timeout sets the HTTP client timeout
	Timeout time.Duration
}

// New creates a new UniFi Network API client with default settings.
// This is the recommended way to create a client for most use cases.
//
// The client automatically handles rate limiting (1000 requests/minute by default)
// and retries failed requests with exponential backoff.
//
// Default settings:
//   - Rate limit: 1000 requests/minute
//   - Max retries: 3
//   - Retry wait time: 1 second
//   - Timeout: 30 seconds
//   - TLS verification: disabled (for self-signed certificates)
//
// For custom configuration, use NewWithConfig.
//
// Example:
//
//	client, err := network.New("https://unifi.local", "your-api-key")
func New(controllerURL, apiKey string) (*APIClient, error) {
	return NewWithConfig(&ClientConfig{
		ControllerURL:      controllerURL,
		APIKey:             apiKey,
		InsecureSkipVerify: true, // Default to true for self-signed certs
	})
}

// NewWithConfig creates a new UniFi Network API client with custom configuration.
// Use this when you need to customize rate limits, timeouts, or other settings.
//
// Example:
//
//	client, err := network.NewWithConfig(&network.ClientConfig{
//	    ControllerURL:      "https://unifi.local",
//	    APIKey:             "your-api-key",
//	    InsecureSkipVerify: true,
//	    RateLimitPerMinute: 500,
//	})
func NewWithConfig(cfg *ClientConfig) (*APIClient, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.ControllerURL == "" {
		return nil, errors.New("controller URL is required")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	// Set defaults
	if cfg.RateLimitPerMinute == 0 {
		cfg.RateLimitPerMinute = DefaultRateLimit
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
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // User-configurable
			},
		}
		httpClient = &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		}
	}

	// Create rate limiter
	rateLimiter := ratelimit.NewRateLimiter(cfg.RateLimitPerMinute)

	// Wrap HTTP client with rate limiting and retry logic
	rateLimitedClient := &rateLimitedHTTPClient{
		client:      httpClient,
		rateLimiter: rateLimiter,
		maxRetries:  cfg.MaxRetries,
		retryWait:   cfg.RetryWaitTime,
	}

	// Create request editor to add API key header
	requestEditor := func(_ context.Context, req *http.Request) error {
		//nolint:canonicalheader // X-API-KEY is the correct header name per UniFi API spec
		req.Header.Set("X-API-KEY", cfg.APIKey)
		req.Header.Set("Accept", "application/json")
		return nil
	}

	// Build full base URL
	baseURL := cfg.ControllerURL + "/proxy/network/integration/v1"

	// Create generated client
	generatedClient, err := NewClientWithResponses(
		baseURL,
		WithHTTPClient(rateLimitedClient),
		WithRequestEditorFn(requestEditor),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create API client")
	}

	return &APIClient{
		client:      generatedClient,
		httpClient:  httpClient,
		rateLimiter: rateLimiter,
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

	var resp *http.Response
	var err error

	// Apply rate limiting
	err = c.rateLimiter.Wait(ctx)
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

// ListSites retrieves a list of all sites configured on the controller.
func (c *APIClient) ListSites(ctx context.Context, params *ListSitesParams) (*SitesResponse, error) {
	resp, err := c.client.ListSitesWithResponse(ctx, params)
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

// ListSiteDevices retrieves a list of all devices for a specific site.
func (c *APIClient) ListSiteDevices(ctx context.Context, siteID SiteId, params *ListSiteDevicesParams) (*DevicesResponse, error) {
	resp, err := c.client.ListSiteDevicesWithResponse(ctx, siteID, params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list devices for site %s", siteID)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// GetDeviceByID retrieves detailed information about a specific device.
func (c *APIClient) GetDeviceByID(ctx context.Context, siteID SiteId, deviceID DeviceId) (*Device, error) {
	resp, err := c.client.GetDeviceByIdWithResponse(ctx, siteID, deviceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get device %s in site %s", deviceID, siteID)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// ListSiteClients retrieves a list of all clients for a specific site.
func (c *APIClient) ListSiteClients(ctx context.Context, siteID SiteId, params *ListSiteClientsParams) (*ClientsResponse, error) {
	resp, err := c.client.ListSiteClientsWithResponse(ctx, siteID, params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list clients for site %s", siteID)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// GetClientByID retrieves detailed information about a specific client.
func (c *APIClient) GetClientByID(ctx context.Context, siteID SiteId, clientID ClientId) (*NetworkClient, error) {
	resp, err := c.client.GetClientByIdWithResponse(ctx, siteID, clientID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get client %s in site %s", clientID, siteID)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// ListDNSRecords lists all static DNS records for a site.
func (c *APIClient) ListDNSRecords(ctx context.Context, site Site) ([]DNSRecord, error) {
	resp, err := c.client.ListDNSRecordsWithResponse(ctx, site)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list DNS records for site %s", site)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return *resp.JSON200, nil
}

// CreateDNSRecord creates a new static DNS record.
func (c *APIClient) CreateDNSRecord(ctx context.Context, site Site, record *DNSRecordInput) (*DNSRecord, error) {
	resp, err := c.client.CreateDNSRecordWithResponse(ctx, site, *record)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create DNS record %s in site %s", record.Key, site)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// GetDNSRecordByID retrieves a specific DNS record by ID.
func (c *APIClient) GetDNSRecordByID(ctx context.Context, site Site, recordID RecordId) (*DNSRecord, error) {
	resp, err := c.client.GetDNSRecordByIdWithResponse(ctx, site, recordID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get DNS record %s in site %s", recordID, site)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// UpdateDNSRecord updates an existing DNS record.
func (c *APIClient) UpdateDNSRecord(ctx context.Context, site Site, recordID RecordId, record *DNSRecordInput) (*DNSRecord, error) {
	resp, err := c.client.UpdateDNSRecordWithResponse(ctx, site, recordID, *record)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update DNS record %s in site %s", recordID, site)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// DeleteDNSRecord deletes a DNS record.
func (c *APIClient) DeleteDNSRecord(ctx context.Context, site Site, recordID RecordId) error {
	resp, err := c.client.DeleteDNSRecordWithResponse(ctx, site, recordID)
	if err != nil {
		return errors.Wrapf(err, "failed to delete DNS record %s in site %s", recordID, site)
	}

	if resp.StatusCode() != http.StatusNoContent {
		return errors.Newf("API error: status=%d", resp.StatusCode())
	}

	return nil
}

// ListFirewallPolicies lists all firewall policies for a site.
func (c *APIClient) ListFirewallPolicies(ctx context.Context, site Site) ([]FirewallPolicy, error) {
	resp, err := c.client.ListFirewallPoliciesWithResponse(ctx, site)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list firewall policies for site %s", site)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return *resp.JSON200, nil
}

// UpdateFirewallPolicy updates an existing firewall policy.
func (c *APIClient) UpdateFirewallPolicy(ctx context.Context, site Site, policyID PolicyId, policy *FirewallPolicyInput) (*FirewallPolicy, error) {
	resp, err := c.client.UpdateFirewallPolicyWithResponse(ctx, site, policyID, *policy)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update firewall policy %s in site %s", policyID, site)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// ListTrafficRules lists all traffic rules for a site.
func (c *APIClient) ListTrafficRules(ctx context.Context, site Site) ([]TrafficRule, error) {
	resp, err := c.client.ListTrafficRulesWithResponse(ctx, site)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list traffic rules for site %s", site)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return *resp.JSON200, nil
}

// UpdateTrafficRule updates an existing traffic rule.
func (c *APIClient) UpdateTrafficRule(ctx context.Context, site Site, ruleID RuleId, rule *TrafficRuleInput) (*TrafficRule, error) {
	resp, err := c.client.UpdateTrafficRuleWithResponse(ctx, site, ruleID, *rule)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update traffic rule %s in site %s", ruleID, site)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}

// GetAggregatedDashboard retrieves aggregated dashboard statistics.
func (c *APIClient) GetAggregatedDashboard(ctx context.Context, site Site, params *GetAggregatedDashboardParams) (*AggregatedDashboard, error) {
	resp, err := c.client.GetAggregatedDashboardWithResponse(ctx, site, params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get aggregated dashboard for site %s", site)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, errors.New("empty response from API")
	}

	return resp.JSON200, nil
}
