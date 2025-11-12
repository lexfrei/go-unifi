package network

//go:generate oapi-codegen -config .oapi-codegen.yaml openapi.yaml

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/lexfrei/go-unifi/internal/httpclient"
	"github.com/lexfrei/go-unifi/internal/middleware"
	"github.com/lexfrei/go-unifi/internal/observability"
	"github.com/lexfrei/go-unifi/internal/ratelimit"
	"github.com/lexfrei/go-unifi/internal/response"
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

// APIClient wraps the generated API client with composable middleware.
type APIClient struct {
	client *ClientWithResponses
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

	// Logger for observability (optional, uses noop logger if nil)
	Logger observability.Logger

	// Metrics recorder for observability (optional, uses noop recorder if nil)
	Metrics observability.MetricsRecorder
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
//	    Logger:             myLogger,
//	    Metrics:            myMetrics,
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

	// Create rate limiter
	rateLimiter := ratelimit.NewRateLimiter(cfg.RateLimitPerMinute)

	// Build middleware chain (applied in reverse order: last = innermost, applied first)
	// Order from outside to inside: Observability -> TLS -> RateLimit -> Retry
	httpClient := httpclient.New(
		httpclient.WithTimeout(cfg.Timeout),
		httpclient.WithMiddleware(
			middleware.Observability(cfg.Logger, cfg.Metrics),
			middleware.TLSConfig(&tls.Config{
				InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // User-configurable
			}),
			middleware.RateLimit(middleware.RateLimitConfig{
				Limiter: rateLimiter,
				Logger:  cfg.Logger,
				Metrics: cfg.Metrics,
			}),
			middleware.Retry(middleware.RetryConfig{
				MaxRetries:  cfg.MaxRetries,
				InitialWait: cfg.RetryWaitTime,
				Logger:      cfg.Logger,
				Metrics:     cfg.Metrics,
			}),
		),
	)

	// Build base URL (paths like /integration/v1/sites are added by generated client)
	baseURL := cfg.ControllerURL + "/proxy/network"

	// Create request editor to add API key and Accept headers
	requestEditor := func(_ context.Context, req *http.Request) error {
		//nolint:canonicalheader // X-API-KEY is the correct header name per UniFi API specification
		req.Header.Set("X-API-KEY", cfg.APIKey)
		req.Header.Set("Accept", "application/json")
		return nil
	}

	// Create generated client
	generatedClient, err := NewClientWithResponses(
		baseURL,
		WithHTTPClient(httpClient.HTTPClient()),
		WithRequestEditorFn(requestEditor),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create API client")
	}

	return &APIClient{
		client: generatedClient,
	}, nil
}

// ListSites retrieves a list of all sites configured on the controller.
func (c *APIClient) ListSites(ctx context.Context, params *ListSitesParams) (*SitesResponse, error) {
	resp, err := c.client.ListSitesWithResponse(ctx, params)
	var data *SitesResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list sites")
}

// ListSiteDevices retrieves a list of all devices for a specific site.
func (c *APIClient) ListSiteDevices(ctx context.Context, siteID SiteId, params *ListSiteDevicesParams) (*DevicesResponse, error) {
	resp, err := c.client.ListSiteDevicesWithResponse(ctx, siteID, params)
	var data *DevicesResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to list devices for site %s", siteID).Error())
}

// GetDeviceByID retrieves detailed information about a specific device.
func (c *APIClient) GetDeviceByID(ctx context.Context, siteID SiteId, deviceID DeviceId) (*Device, error) {
	resp, err := c.client.GetDeviceByIdWithResponse(ctx, siteID, deviceID)
	var data *Device
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to get device %s in site %s", deviceID, siteID).Error())
}

// ListSiteClients retrieves a list of all clients for a specific site.
func (c *APIClient) ListSiteClients(ctx context.Context, siteID SiteId, params *ListSiteClientsParams) (*ClientsResponse, error) {
	resp, err := c.client.ListSiteClientsWithResponse(ctx, siteID, params)
	var data *ClientsResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to list clients for site %s", siteID).Error())
}

// GetClientByID retrieves detailed information about a specific client.
func (c *APIClient) GetClientByID(ctx context.Context, siteID SiteId, clientID ClientId) (*NetworkClient, error) {
	resp, err := c.client.GetClientByIdWithResponse(ctx, siteID, clientID)
	var data *NetworkClient
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to get client %s in site %s", clientID, siteID).Error())
}

// ListHotspotVouchers retrieves a list of all hotspot vouchers for a specific site.
func (c *APIClient) ListHotspotVouchers(ctx context.Context, siteID SiteId, params *ListHotspotVouchersParams) (*HotspotVouchersResponse, error) {
	resp, err := c.client.ListHotspotVouchersWithResponse(ctx, siteID, params)
	var data *HotspotVouchersResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to list hotspot vouchers for site %s", siteID).Error())
}

// CreateHotspotVouchers creates one or more hotspot vouchers for temporary guest access.
func (c *APIClient) CreateHotspotVouchers(ctx context.Context, siteID SiteId, request *CreateVouchersRequest) (*HotspotVouchersResponse, error) {
	resp, err := c.client.CreateHotspotVouchersWithResponse(ctx, siteID, *request)
	var data *HotspotVouchersResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to create hotspot vouchers for site %s", siteID).Error())
}

// GetHotspotVoucher retrieves detailed information about a specific hotspot voucher.
func (c *APIClient) GetHotspotVoucher(ctx context.Context, siteID SiteId, voucherID openapi_types.UUID) (*HotspotVoucher, error) {
	resp, err := c.client.GetHotspotVoucherWithResponse(ctx, siteID, voucherID)
	var data *HotspotVoucher
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to get hotspot voucher %s in site %s", voucherID, siteID).Error())
}

// DeleteHotspotVoucher permanently deletes a hotspot voucher.
func (c *APIClient) DeleteHotspotVoucher(ctx context.Context, siteID SiteId, voucherID openapi_types.UUID) error {
	resp, err := c.client.DeleteHotspotVoucherWithResponse(ctx, siteID, voucherID)
	//nolint:wrapcheck // response.HandleNoContent wraps errors internally
	return response.HandleNoContent(resp, err, errors.Newf("failed to delete hotspot voucher %s in site %s", voucherID, siteID).Error())
}

// ListDNSRecords lists all static DNS records for a site.
func (c *APIClient) ListDNSRecords(ctx context.Context, site Site) ([]DNSRecord, error) {
	resp, err := c.client.ListDNSRecordsWithResponse(ctx, site)
	var dataPtr *[]DNSRecord
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, errors.Newf("failed to list DNS records for site %s", site).Error())
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// CreateDNSRecord creates a new static DNS record.
func (c *APIClient) CreateDNSRecord(ctx context.Context, site Site, record *DNSRecordInput) (*DNSRecord, error) {
	resp, err := c.client.CreateDNSRecordWithResponse(ctx, site, *record)
	var data *DNSRecord
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to create DNS record %s in site %s", record.Key, site).Error())
}

// UpdateDNSRecord updates an existing DNS record.
func (c *APIClient) UpdateDNSRecord(ctx context.Context, site Site, recordID RecordId, record *DNSRecordInput) (*DNSRecord, error) {
	resp, err := c.client.UpdateDNSRecordWithResponse(ctx, site, recordID, *record)
	var data *DNSRecord
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to update DNS record %s in site %s", recordID, site).Error())
}

// DeleteDNSRecord deletes a DNS record.
func (c *APIClient) DeleteDNSRecord(ctx context.Context, site Site, recordID RecordId) error {
	resp, err := c.client.DeleteDNSRecordWithResponse(ctx, site, recordID)
	//nolint:wrapcheck // response.HandleNoContent wraps errors internally
	return response.HandleNoContent(resp, err, errors.Newf("failed to delete DNS record %s in site %s", recordID, site).Error())
}

// ListFirewallPolicies lists all firewall policies for a site.
func (c *APIClient) ListFirewallPolicies(ctx context.Context, site Site) ([]FirewallPolicy, error) {
	resp, err := c.client.ListFirewallPoliciesWithResponse(ctx, site)
	var dataPtr *[]FirewallPolicy
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, errors.Newf("failed to list firewall policies for site %s", site).Error())
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// UpdateFirewallPolicy updates an existing firewall policy.
func (c *APIClient) UpdateFirewallPolicy(ctx context.Context, site Site, policyID PolicyId, policy *FirewallPolicyInput) (*FirewallPolicy, error) {
	resp, err := c.client.UpdateFirewallPolicyWithResponse(ctx, site, policyID, *policy)
	var data *FirewallPolicy
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to update firewall policy %s in site %s", policyID, site).Error())
}

// CreateFirewallPolicy creates a new firewall policy.
func (c *APIClient) CreateFirewallPolicy(ctx context.Context, site Site, policy *FirewallPolicyInput) (*FirewallPolicy, error) {
	resp, err := c.client.CreateFirewallPolicyWithResponse(ctx, site, *policy)
	var data *FirewallPolicy
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to create firewall policy in site %s", site).Error())
}

// DeleteFirewallPolicy permanently deletes a firewall policy.
func (c *APIClient) DeleteFirewallPolicy(ctx context.Context, site Site, policyID PolicyId) error {
	resp, err := c.client.DeleteFirewallPolicyWithResponse(ctx, site, policyID)
	//nolint:wrapcheck // response.HandleNoContent wraps errors internally
	return response.HandleNoContent(resp, err, errors.Newf("failed to delete firewall policy %s in site %s", policyID, site).Error())
}

// ListTrafficRules lists all traffic rules for a site.
func (c *APIClient) ListTrafficRules(ctx context.Context, site Site) ([]TrafficRule, error) {
	resp, err := c.client.ListTrafficRulesWithResponse(ctx, site)
	var dataPtr *[]TrafficRule
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, errors.Newf("failed to list traffic rules for site %s", site).Error())
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// UpdateTrafficRule updates an existing traffic rule.
func (c *APIClient) UpdateTrafficRule(ctx context.Context, site Site, ruleID RuleId, rule *TrafficRuleInput) (*TrafficRule, error) {
	resp, err := c.client.UpdateTrafficRuleWithResponse(ctx, site, ruleID, *rule)
	var data *TrafficRule
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to update traffic rule %s in site %s", ruleID, site).Error())
}

// CreateTrafficRule creates a new traffic rule.
func (c *APIClient) CreateTrafficRule(ctx context.Context, site Site, rule *TrafficRuleInput) (*TrafficRule, error) {
	resp, err := c.client.CreateTrafficRuleWithResponse(ctx, site, *rule)
	var data *TrafficRule
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to create traffic rule in site %s", site).Error())
}

// DeleteTrafficRule permanently deletes a traffic rule.
func (c *APIClient) DeleteTrafficRule(ctx context.Context, site Site, ruleID RuleId) error {
	resp, err := c.client.DeleteTrafficRuleWithResponse(ctx, site, ruleID)
	//nolint:wrapcheck // response.HandleNoContent wraps errors internally
	return response.HandleNoContent(resp, err, errors.Newf("failed to delete traffic rule %s in site %s", ruleID, site).Error())
}

// GetAggregatedDashboard retrieves aggregated dashboard statistics.
func (c *APIClient) GetAggregatedDashboard(ctx context.Context, site Site, params *GetAggregatedDashboardParams) (*AggregatedDashboard, error) {
	resp, err := c.client.GetAggregatedDashboardWithResponse(ctx, site, params)
	var data *AggregatedDashboard
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, errors.Newf("failed to get aggregated dashboard for site %s", site).Error())
}
