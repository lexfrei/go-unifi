package network

//go:generate oapi-codegen -config .oapi-codegen.yaml openapi.yaml

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/lexfrei/go-unifi/internal/httpclient"
	"github.com/lexfrei/go-unifi/internal/middleware"
	"github.com/lexfrei/go-unifi/internal/ratelimit"
	"github.com/lexfrei/go-unifi/internal/response"
	"github.com/lexfrei/go-unifi/observability"
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

// Compile-time check to ensure APIClient implements NetworkAPIClient interface.
var _ NetworkAPIClient = (*APIClient)(nil)

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
	return response.Handle(resp, data, err, fmt.Sprintf("failed to list devices for site %s", siteID))
}

// GetDeviceByID retrieves detailed information about a specific device.
func (c *APIClient) GetDeviceByID(ctx context.Context, siteID SiteId, deviceID DeviceId) (*Device, error) {
	resp, err := c.client.GetDeviceByIdWithResponse(ctx, siteID, deviceID)
	var data *Device
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, fmt.Sprintf("failed to get device %s in site %s", deviceID, siteID))
}

// ListSiteClients retrieves a list of all clients for a specific site.
func (c *APIClient) ListSiteClients(ctx context.Context, siteID SiteId, params *ListSiteClientsParams) (*ClientsResponse, error) {
	resp, err := c.client.ListSiteClientsWithResponse(ctx, siteID, params)
	var data *ClientsResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, fmt.Sprintf("failed to list clients for site %s", siteID))
}

// GetClientByID retrieves detailed information about a specific client.
func (c *APIClient) GetClientByID(ctx context.Context, siteID SiteId, clientID ClientId) (*NetworkClient, error) {
	resp, err := c.client.GetClientByIdWithResponse(ctx, siteID, clientID)
	var data *NetworkClient
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, fmt.Sprintf("failed to get client %s in site %s", clientID, siteID))
}

// ListHotspotVouchers retrieves a list of all hotspot vouchers for a specific site.
func (c *APIClient) ListHotspotVouchers(ctx context.Context, siteID SiteId, params *ListHotspotVouchersParams) (*HotspotVouchersResponse, error) {
	resp, err := c.client.ListHotspotVouchersWithResponse(ctx, siteID, params)
	var data *HotspotVouchersResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, fmt.Sprintf("failed to list hotspot vouchers for site %s", siteID))
}

// CreateHotspotVouchers creates one or more hotspot vouchers for temporary guest access.
func (c *APIClient) CreateHotspotVouchers(ctx context.Context, siteID SiteId, request *CreateVouchersRequest) (*HotspotVouchersResponse, error) {
	resp, err := c.client.CreateHotspotVouchersWithResponse(ctx, siteID, *request)
	var data *HotspotVouchersResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, fmt.Sprintf("failed to create hotspot vouchers for site %s", siteID))
}

// GetHotspotVoucher retrieves detailed information about a specific hotspot voucher.
func (c *APIClient) GetHotspotVoucher(ctx context.Context, siteID SiteId, voucherID openapi_types.UUID) (*HotspotVoucher, error) {
	resp, err := c.client.GetHotspotVoucherWithResponse(ctx, siteID, voucherID)
	var data *HotspotVoucher
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, fmt.Sprintf("failed to get hotspot voucher %s in site %s", voucherID, siteID))
}

// DeleteHotspotVoucher permanently deletes a hotspot voucher.
func (c *APIClient) DeleteHotspotVoucher(ctx context.Context, siteID SiteId, voucherID openapi_types.UUID) error {
	resp, err := c.client.DeleteHotspotVoucherWithResponse(ctx, siteID, voucherID)
	//nolint:wrapcheck // response.HandleNoContent wraps errors internally
	return response.HandleNoContent(resp, err, fmt.Sprintf("failed to delete hotspot voucher %s in site %s", voucherID, siteID))
}

// ListDNSRecords lists all static DNS records for a site.
func (c *APIClient) ListDNSRecords(ctx context.Context, site Site) ([]DNSRecord, error) {
	resp, err := c.client.ListDNSRecordsWithResponse(ctx, site)
	var dataPtr *[]DNSRecord
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list DNS records for site "+site)
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
	return response.Handle(resp, data, err, fmt.Sprintf("failed to create DNS record %s in site %s", record.Key, site))
}

// UpdateDNSRecord updates an existing DNS record.
func (c *APIClient) UpdateDNSRecord(ctx context.Context, site Site, recordID RecordId, record *DNSRecordInput) (*DNSRecord, error) {
	resp, err := c.client.UpdateDNSRecordWithResponse(ctx, site, recordID, *record)
	var data *DNSRecord
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, fmt.Sprintf("failed to update DNS record %s in site %s", recordID, site))
}

// DeleteDNSRecord deletes a DNS record.
func (c *APIClient) DeleteDNSRecord(ctx context.Context, site Site, recordID RecordId) error {
	resp, err := c.client.DeleteDNSRecordWithResponse(ctx, site, recordID)
	//nolint:wrapcheck // response.HandleNoContent wraps errors internally
	return response.HandleNoContent(resp, err, fmt.Sprintf("failed to delete DNS record %s in site %s", recordID, site))
}

// ListFirewallPolicies lists all firewall policies for a site.
func (c *APIClient) ListFirewallPolicies(ctx context.Context, site Site) ([]FirewallPolicy, error) {
	resp, err := c.client.ListFirewallPoliciesWithResponse(ctx, site)
	var dataPtr *[]FirewallPolicy
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list firewall policies for site "+site)
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
	return response.Handle(resp, data, err, fmt.Sprintf("failed to update firewall policy %s in site %s", policyID, site))
}

// CreateFirewallPolicy creates a new firewall policy.
func (c *APIClient) CreateFirewallPolicy(ctx context.Context, site Site, policy *FirewallPolicyInput) (*FirewallPolicy, error) {
	resp, err := c.client.CreateFirewallPolicyWithResponse(ctx, site, *policy)
	var data *FirewallPolicy
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to create firewall policy in site "+site)
}

// DeleteFirewallPolicy permanently deletes a firewall policy.
func (c *APIClient) DeleteFirewallPolicy(ctx context.Context, site Site, policyID PolicyId) error {
	resp, err := c.client.DeleteFirewallPolicyWithResponse(ctx, site, policyID)
	//nolint:wrapcheck // response.HandleNoContent wraps errors internally
	return response.HandleNoContent(resp, err, fmt.Sprintf("failed to delete firewall policy %s in site %s", policyID, site))
}

// ListTrafficRules lists all traffic rules for a site.
func (c *APIClient) ListTrafficRules(ctx context.Context, site Site) ([]TrafficRule, error) {
	resp, err := c.client.ListTrafficRulesWithResponse(ctx, site)
	var dataPtr *[]TrafficRule
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list traffic rules for site "+site)
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
	return response.Handle(resp, data, err, fmt.Sprintf("failed to update traffic rule %s in site %s", ruleID, site))
}

// CreateTrafficRule creates a new traffic rule.
func (c *APIClient) CreateTrafficRule(ctx context.Context, site Site, rule *TrafficRuleInput) (*TrafficRule, error) {
	resp, err := c.client.CreateTrafficRuleWithResponse(ctx, site, *rule)
	var data *TrafficRule
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to create traffic rule in site "+site)
}

// DeleteTrafficRule permanently deletes a traffic rule.
func (c *APIClient) DeleteTrafficRule(ctx context.Context, site Site, ruleID RuleId) error {
	resp, err := c.client.DeleteTrafficRuleWithResponse(ctx, site, ruleID)
	//nolint:wrapcheck // response.HandleNoContent wraps errors internally
	return response.HandleNoContent(resp, err, fmt.Sprintf("failed to delete traffic rule %s in site %s", ruleID, site))
}

// GetAggregatedDashboard retrieves aggregated dashboard statistics.
func (c *APIClient) GetAggregatedDashboard(ctx context.Context, site Site, params *GetAggregatedDashboardParams) (*AggregatedDashboard, error) {
	resp, err := c.client.GetAggregatedDashboardWithResponse(ctx, site, params)
	var data *AggregatedDashboard
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get aggregated dashboard for site "+site)
}

// GetTopology retrieves the network topology graph for a site.
func (c *APIClient) GetTopology(ctx context.Context, site Site) (*Topology, error) {
	resp, err := c.client.GetTopologyWithResponse(ctx, site)
	var data *Topology
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get topology for site "+site)
}

// ListActiveClients retrieves all currently connected clients with detailed connection information.
func (c *APIClient) ListActiveClients(ctx context.Context, site Site) ([]ActiveClient, error) {
	resp, err := c.client.ListActiveClientsWithResponse(ctx, site)
	var dataPtr *[]ActiveClient
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list active clients for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListAllDevices retrieves all devices across all UniFi applications for a site.
func (c *APIClient) ListAllDevices(ctx context.Context, site Site) (*AllDevicesResponse, error) {
	resp, err := c.client.ListAllDevicesWithResponse(ctx, site)
	var data *AllDevicesResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list all devices for site "+site)
}

// GetActiveClientByMac retrieves detailed information about a specific active client by MAC address.
func (c *APIClient) GetActiveClientByMac(ctx context.Context, site Site, clientMac ClientMac) (*ActiveClientDetails, error) {
	resp, err := c.client.GetActiveClientByMacWithResponse(ctx, site, clientMac)
	var data *ActiveClientDetails
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, fmt.Sprintf("failed to get active client %s in site %s", clientMac, site))
}

// GetWiFiStatsAPs retrieves WiFi statistics for all access points within a time range.
func (c *APIClient) GetWiFiStatsAPs(ctx context.Context, site Site, params *GetWiFiStatsAPsParams) (*WiFiStatsAPsResponse, error) {
	resp, err := c.client.GetWiFiStatsAPsWithResponse(ctx, site, params)
	var data *WiFiStatsAPsResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get WiFi AP stats for site "+site)
}

// GetSystemLogs retrieves paginated system logs.
func (c *APIClient) GetSystemLogs(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogsWithResponse(ctx, site, *request)
	var data *SystemLogResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get system logs for site "+site)
}

// ListFirewallZones retrieves all firewall zones for a site.
func (c *APIClient) ListFirewallZones(ctx context.Context, site Site) ([]FirewallZone, error) {
	resp, err := c.client.ListFirewallZonesWithResponse(ctx, site)
	var dataPtr *[]FirewallZone
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list firewall zones for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetFirewallZoneMatrix retrieves the firewall zone policy matrix for a site.
func (c *APIClient) GetFirewallZoneMatrix(ctx context.Context, site Site) ([]FirewallZoneMatrixEntry, error) {
	resp, err := c.client.GetFirewallZoneMatrixWithResponse(ctx, site)
	var dataPtr *[]FirewallZoneMatrixEntry
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get firewall zone matrix for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetISPStatus retrieves comprehensive ISP status with metrics and history.
func (c *APIClient) GetISPStatus(ctx context.Context, site Site) (*ISPStatus, error) {
	resp, err := c.client.GetISPStatusWithResponse(ctx, site)
	var data *ISPStatus
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get ISP status for site "+site)
}

// ListClientHistory retrieves historical client connection information.
func (c *APIClient) ListClientHistory(ctx context.Context, site Site) ([]ClientHistoryEntry, error) {
	resp, err := c.client.ListClientHistoryWithResponse(ctx, site)
	var dataPtr *[]ClientHistoryEntry
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list client history for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListSpeedtestHistory retrieves historical speedtest results.
func (c *APIClient) ListSpeedtestHistory(ctx context.Context, site Site) (*SpeedtestHistoryResponse, error) {
	resp, err := c.client.ListSpeedtestHistoryWithResponse(ctx, site)
	var data *SpeedtestHistoryResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list speedtest history for site "+site)
}

// GetSpeedtestLatestPerWan retrieves the latest speedtest per WAN interface.
func (c *APIClient) GetSpeedtestLatestPerWan(ctx context.Context, site Site) (*SpeedtestLatestPerWanResponse, error) {
	resp, err := c.client.GetSpeedtestLatestPerWanWithResponse(ctx, site)
	var data *SpeedtestLatestPerWanResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get latest speedtest per WAN for site "+site)
}

// ListVPNConnections retrieves all VPN connections.
func (c *APIClient) ListVPNConnections(ctx context.Context, site Site) ([]VPNConnection, error) {
	resp, err := c.client.ListVPNConnectionsWithResponse(ctx, site)
	var data *VPNConnectionsResponse
	if resp != nil {
		data = resp.JSON200
	}
	result, err := response.Handle(resp, data, err, "failed to list VPN connections for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	if result.Connections == nil {
		return []VPNConnection{}, nil
	}
	return *result.Connections, nil
}

// ListTrafficRoutes retrieves all traffic routes.
func (c *APIClient) ListTrafficRoutes(ctx context.Context, site Site) ([]TrafficRoute, error) {
	resp, err := c.client.ListTrafficRoutesWithResponse(ctx, site)
	var dataPtr *[]TrafficRoute
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list traffic routes for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListActiveDHCPLeases retrieves all active DHCP leases with device fingerprint information.
func (c *APIClient) ListActiveDHCPLeases(ctx context.Context, site Site) (*DHCPLeasesResponse, error) {
	resp, err := c.client.ListActiveDHCPLeasesWithResponse(ctx, site)
	var data *DHCPLeasesResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list DHCP leases for site "+site)
}

// GetISPHealth retrieves ISP health status with historical stats.
func (c *APIClient) GetISPHealth(ctx context.Context, site Site) (*ISPHealth, error) {
	resp, err := c.client.GetISPHealthWithResponse(ctx, site)
	var data *ISPHealth
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get ISP health for site "+site)
}

// GetISPHealthCompact retrieves compact ISP health status.
func (c *APIClient) GetISPHealthCompact(ctx context.Context, site Site) (*ISPHealthCompact, error) {
	resp, err := c.client.GetISPHealthCompactWithResponse(ctx, site)
	var data *ISPHealthCompact
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get compact ISP health for site "+site)
}

// ListNATRules retrieves all NAT (port forwarding) rules.
func (c *APIClient) ListNATRules(ctx context.Context, site Site) ([]NATRule, error) {
	resp, err := c.client.ListNATRulesWithResponse(ctx, site)
	var dataPtr *[]NATRule
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list NAT rules for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListAlerts retrieves all alerts for the site.
func (c *APIClient) ListAlerts(ctx context.Context, site Site) (*AlertsResponse, error) {
	resp, err := c.client.ListAlertsWithResponse(ctx, site)
	var data *AlertsResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list alerts for site "+site)
}

// ListWarnings retrieves security warnings for the site.
func (c *APIClient) ListWarnings(ctx context.Context, site Site) (*WarningsResponse, error) {
	resp, err := c.client.ListWarningsWithResponse(ctx, site)
	var data *WarningsResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list warnings for site "+site)
}

// ListAPGroups retrieves all access point groups.
func (c *APIClient) ListAPGroups(ctx context.Context, site Site) ([]APGroup, error) {
	resp, err := c.client.ListAPGroupsWithResponse(ctx, site)
	var dataPtr *[]APGroup
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list AP groups for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListRADIUSProfiles retrieves all RADIUS authentication profiles.
func (c *APIClient) ListRADIUSProfiles(ctx context.Context, site Site) ([]RADIUSProfile, error) {
	resp, err := c.client.ListRADIUSProfilesWithResponse(ctx, site)
	var dataPtr *[]RADIUSProfile
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list RADIUS profiles for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListIPSAlerts retrieves intrusion prevention system alerts.
func (c *APIClient) ListIPSAlerts(ctx context.Context, site Site) (*IPSAlertsResponse, error) {
	resp, err := c.client.ListIPSAlertsWithResponse(ctx, site)
	var data *IPSAlertsResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list IPS alerts for site "+site)
}

// ListContentFilteringRules retrieves content filtering rules for the site.
func (c *APIClient) ListContentFilteringRules(ctx context.Context, site Site) ([]ContentFilteringRule, error) {
	resp, err := c.client.ListContentFilteringRulesWithResponse(ctx, site)
	var dataPtr *[]ContentFilteringRule
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list content filtering rules for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListContentFilteringCategories retrieves all available content filtering categories.
func (c *APIClient) ListContentFilteringCategories(ctx context.Context, site Site) ([]string, error) {
	resp, err := c.client.ListContentFilteringCategoriesWithResponse(ctx, site)
	var dataPtr *[]string
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list content filtering categories for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetWiFiConnectivity retrieves WiFi connection attempts and latency statistics.
func (c *APIClient) GetWiFiConnectivity(ctx context.Context, site Site) (*WiFiConnectivity, error) {
	resp, err := c.client.GetWiFiConnectivityWithResponse(ctx, site)
	var data *WiFiConnectivity
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get WiFi connectivity for site "+site)
}

// GetWLANCapabilities retrieves WLAN capabilities like 6GHz and WPA3 support.
func (c *APIClient) GetWLANCapabilities(ctx context.Context, site Site) (*WLANCapabilities, error) {
	resp, err := c.client.GetWLANCapabilitiesWithResponse(ctx, site)
	var data *WLANCapabilities
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get WLAN capabilities for site "+site)
}

// ListFeatures retrieves list of features supported by the controller.
func (c *APIClient) ListFeatures(ctx context.Context, site Site) ([]string, error) {
	resp, err := c.client.ListFeaturesWithResponse(ctx, site)
	var dataPtr *[]string
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list features for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListTeleportInvitationHistory retrieves teleport invitation history.
func (c *APIClient) ListTeleportInvitationHistory(ctx context.Context, site Site) (*TeleportInvitationHistoryResponse, error) {
	resp, err := c.client.ListTeleportInvitationHistoryWithResponse(ctx, site)
	var data *TeleportInvitationHistoryResponse
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to list teleport invitation history for site "+site)
}

// ListNotifications retrieves notifications for the site.
func (c *APIClient) ListNotifications(ctx context.Context, site Site) ([]Notification, error) {
	resp, err := c.client.ListNotificationsWithResponse(ctx, site)
	var dataPtr *[]Notification
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list notifications for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListACLRules retrieves access control list rules.
func (c *APIClient) ListACLRules(ctx context.Context, site Site) ([]ACLRule, error) {
	resp, err := c.client.ListACLRulesWithResponse(ctx, site)
	var dataPtr *[]ACLRule
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list ACL rules for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListRADIUSUsers retrieves RADIUS users.
func (c *APIClient) ListRADIUSUsers(ctx context.Context, site Site) ([]RADIUSUser, error) {
	resp, err := c.client.ListRADIUSUsersWithResponse(ctx, site)
	var dataPtr *[]RADIUSUser
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list RADIUS users for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetHotspotInfo retrieves hotspot configuration status.
func (c *APIClient) GetHotspotInfo(ctx context.Context, site Site) (*HotspotInfo, error) {
	resp, err := c.client.GetHotspotInfoWithResponse(ctx, site)
	var data *HotspotInfo
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get hotspot info for site "+site)
}

// GetClientsTrafficControl retrieves traffic rule counts applied to clients.
func (c *APIClient) GetClientsTrafficControl(ctx context.Context, site Site) (*ClientsTrafficControl, error) {
	resp, err := c.client.GetClientsTrafficControlWithResponse(ctx, site)
	var data *ClientsTrafficControl
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get clients traffic control for site "+site)
}

// ListWireGuardUsers retrieves all WireGuard VPN users.
func (c *APIClient) ListWireGuardUsers(ctx context.Context, site Site) ([]WireGuardUser, error) {
	resp, err := c.client.ListWireGuardUsersWithResponse(ctx, site)
	var dataPtr *[]WireGuardUser
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list WireGuard users for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetWireGuardExistingSubnets retrieves subnets already in use by WireGuard.
func (c *APIClient) GetWireGuardExistingSubnets(ctx context.Context, site Site) (*WireGuardSubnets, error) {
	resp, err := c.client.GetWireGuardExistingSubnetsWithResponse(ctx, site)
	var data *WireGuardSubnets
	if resp != nil {
		data = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, data, err, "failed to get WireGuard subnets for site "+site)
}

// ListVPNClientConnections retrieves all VPN client connections.
func (c *APIClient) ListVPNClientConnections(ctx context.Context, site Site) ([]VPNConnection, error) {
	resp, err := c.client.ListVPNClientConnectionsWithResponse(ctx, site)
	var data *VPNConnectionsResponse
	if resp != nil {
		data = resp.JSON200
	}
	result, err := response.Handle(resp, data, err, "failed to list VPN client connections for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	if result.Connections == nil {
		return []VPNConnection{}, nil
	}
	return *result.Connections, nil
}

// GetBGPConfig retrieves BGP routing configuration.
func (c *APIClient) GetBGPConfig(ctx context.Context, site Site) ([]BGPConfig, error) {
	resp, err := c.client.GetBGPConfigWithResponse(ctx, site)
	var dataPtr *[]BGPConfig
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get BGP config for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetAllBGPConfig retrieves all BGP configurations across all devices.
func (c *APIClient) GetAllBGPConfig(ctx context.Context, site Site) ([]BGPConfig, error) {
	resp, err := c.client.GetAllBGPConfigWithResponse(ctx, site)
	var dataPtr *[]BGPConfig
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get all BGP configs for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListOSPFRouters retrieves OSPF router configurations.
func (c *APIClient) ListOSPFRouters(ctx context.Context, site Site) ([]OSPFRouter, error) {
	resp, err := c.client.ListOSPFRoutersWithResponse(ctx, site)
	var dataPtr *[]OSPFRouter
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list OSPF routers for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListOSPFNeighbors retrieves OSPF neighbor relationships.
func (c *APIClient) ListOSPFNeighbors(ctx context.Context, site Site) ([]OSPFNeighbor, error) {
	resp, err := c.client.ListOSPFNeighborsWithResponse(ctx, site)
	var dataPtr *[]OSPFNeighbor
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list OSPF neighbors for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListQoSRules retrieves Quality of Service rules.
func (c *APIClient) ListQoSRules(ctx context.Context, site Site) ([]QoSRule, error) {
	resp, err := c.client.ListQoSRulesWithResponse(ctx, site)
	var dataPtr *[]QoSRule
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list QoS rules for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListWANSLAs retrieves WAN Service Level Agreement configurations.
func (c *APIClient) ListWANSLAs(ctx context.Context, site Site) ([]WANSLA, error) {
	resp, err := c.client.ListWANSLAsWithResponse(ctx, site)
	var dataPtr *[]WANSLA
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list WAN SLAs for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListDescribedFeatures retrieves detailed list of features with availability and limitations.
func (c *APIClient) ListDescribedFeatures(ctx context.Context, site Site) ([]DescribedFeature, error) {
	resp, err := c.client.ListDescribedFeaturesWithResponse(ctx, site)
	var dataPtr *[]DescribedFeature
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list described features for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListVendorIDs retrieves list of known UniFi device vendor MAC address prefixes.
func (c *APIClient) ListVendorIDs(ctx context.Context, site Site) ([]string, error) {
	resp, err := c.client.ListVendorIDsWithResponse(ctx, site)
	var dataPtr *[]string
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list vendor IDs for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetGatewayEngineFeatures retrieves status of gateway engine features and their utilization.
func (c *APIClient) GetGatewayEngineFeatures(ctx context.Context, site Site) ([]GatewayEngineFeature, error) {
	resp, err := c.client.GetGatewayEngineFeaturesWithResponse(ctx, site)
	var dataPtr *[]GatewayEngineFeature
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get gateway engine features for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetGatewayEngineMostActiveNetworks retrieves list of most active networks by client count.
func (c *APIClient) GetGatewayEngineMostActiveNetworks(ctx context.Context, site Site) ([]MostActiveNetwork, error) {
	resp, err := c.client.GetGatewayEngineMostActiveNetworksWithResponse(ctx, site)
	var dataPtr *[]MostActiveNetwork
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get most active networks for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetGatewayEngineUtilization retrieves CPU and memory utilization of the gateway.
func (c *APIClient) GetGatewayEngineUtilization(ctx context.Context, site Site) (*GatewayEngineUtilization, error) {
	resp, err := c.client.GetGatewayEngineUtilizationWithResponse(ctx, site)
	var dataPtr *GatewayEngineUtilization
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get gateway utilization for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// ListHotspotClients retrieves all clients connected via hotspot or regular network.
func (c *APIClient) ListHotspotClients(ctx context.Context, site Site) ([]HotspotClient, error) {
	resp, err := c.client.ListHotspotClientsWithResponse(ctx, site)
	var dataPtr *[]HotspotClient
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list hotspot clients for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetLoopDetectionInfo retrieves information about detected network loops.
func (c *APIClient) GetLoopDetectionInfo(ctx context.Context, site Site) (*LoopDetectionInfo, error) {
	resp, err := c.client.GetLoopDetectionInfoWithResponse(ctx, site)
	var dataPtr *LoopDetectionInfo
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get loop detection info for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetPortProfileDefaults retrieves default port profile configuration.
func (c *APIClient) GetPortProfileDefaults(ctx context.Context, site Site) (*PortProfileDefaults, error) {
	resp, err := c.client.GetPortProfileDefaultsWithResponse(ctx, site)
	var dataPtr *PortProfileDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get port profile defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// ListSSLInspectionCategories retrieves available SSL inspection content categories.
func (c *APIClient) ListSSLInspectionCategories(ctx context.Context, site Site) ([]SSLInspectionCategory, error) {
	resp, err := c.client.ListSSLInspectionCategoriesWithResponse(ctx, site)
	var dataPtr *[]SSLInspectionCategory
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list SSL inspection categories for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListSSLInspectionProfiles retrieves SSL inspection profile configurations.
func (c *APIClient) ListSSLInspectionProfiles(ctx context.Context, site Site) ([]SSLInspectionProfile, error) {
	resp, err := c.client.ListSSLInspectionProfilesWithResponse(ctx, site)
	var dataPtr *[]SSLInspectionProfile
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list SSL inspection profiles for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetLANEnrichedConfiguration retrieves enriched LAN network configurations.
func (c *APIClient) GetLANEnrichedConfiguration(ctx context.Context, site Site) ([]LANEnrichedConfiguration, error) {
	resp, err := c.client.GetLANEnrichedConfigurationWithResponse(ctx, site)
	var dataPtr *[]LANEnrichedConfiguration
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get LAN enriched configuration for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetWANEnrichedConfiguration retrieves enriched WAN network configurations.
func (c *APIClient) GetWANEnrichedConfiguration(ctx context.Context, site Site) ([]WANEnrichedConfiguration, error) {
	resp, err := c.client.GetWANEnrichedConfigurationWithResponse(ctx, site)
	var dataPtr *[]WANEnrichedConfiguration
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WAN enriched configuration for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetNetworkScore retrieves overall network health score and subscores.
func (c *APIClient) GetNetworkScore(ctx context.Context, site Site) (*NetworkScore, error) {
	resp, err := c.client.GetNetworkScoreWithResponse(ctx, site)
	var dataPtr *NetworkScore
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get network score for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// ListStaticDNSDevices retrieves device mappings for static DNS.
func (c *APIClient) ListStaticDNSDevices(ctx context.Context, site Site) ([]StaticDNSDevice, error) {
	resp, err := c.client.ListStaticDNSDevicesWithResponse(ctx, site)
	var dataPtr *[]StaticDNSDevice
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list static DNS devices for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetTeleportTokens retrieves Teleport VPN tokens.
func (c *APIClient) GetTeleportTokens(ctx context.Context, site Site) (*TeleportTokenResponse, error) {
	resp, err := c.client.GetTeleportTokensWithResponse(ctx, site)
	var dataPtr *TeleportTokenResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get teleport tokens for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetWiFiManData retrieves WiFiMan diagnostic data for connected clients.
func (c *APIClient) GetWiFiManData(ctx context.Context, site Site) ([]WiFiManEntry, error) {
	resp, err := c.client.GetWiFiManDataWithResponse(ctx, site)
	var dataPtr *[]WiFiManEntry
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WiFiMan data for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}
