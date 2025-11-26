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

// GetFirewallZoneDefaults retrieves default firewall zone configurations.
func (c *APIClient) GetFirewallZoneDefaults(ctx context.Context, site Site) ([]FirewallZoneDefault, error) {
	resp, err := c.client.GetFirewallZoneDefaultsWithResponse(ctx, site)
	var dataPtr *[]FirewallZoneDefault
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get firewall zone defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetFirewallPolicyDefaults retrieves default firewall policy configuration.
func (c *APIClient) GetFirewallPolicyDefaults(ctx context.Context, site Site) (*FirewallPolicyDefaults, error) {
	resp, err := c.client.GetFirewallPolicyDefaultsWithResponse(ctx, site)
	var dataPtr *FirewallPolicyDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get firewall policy defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetDOHDefaults retrieves DNS over HTTPS default settings.
func (c *APIClient) GetDOHDefaults(ctx context.Context, site Site) (*DOHDefaults, error) {
	resp, err := c.client.GetDOHDefaultsWithResponse(ctx, site)
	var dataPtr *DOHDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get DoH defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetDOHAvailableServerNames retrieves list of available DNS over HTTPS server names.
func (c *APIClient) GetDOHAvailableServerNames(ctx context.Context, site Site) (*DOHAvailableServers, error) {
	resp, err := c.client.GetDOHAvailableServerNamesWithResponse(ctx, site)
	var dataPtr *DOHAvailableServers
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get DoH available server names for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetLANDefaults retrieves default LAN network configuration.
func (c *APIClient) GetLANDefaults(ctx context.Context, site Site) (*LANDefaults, error) {
	resp, err := c.client.GetLANDefaultsWithResponse(ctx, site)
	var dataPtr *LANDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get LAN defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetWANDefaults retrieves default WAN network configuration.
func (c *APIClient) GetWANDefaults(ctx context.Context, site Site) (*WANDefaults, error) {
	resp, err := c.client.GetWANDefaultsWithResponse(ctx, site)
	var dataPtr *WANDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WAN defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetWLANDefaults retrieves default WLAN configuration.
func (c *APIClient) GetWLANDefaults(ctx context.Context, site Site) ([]WLANDefaults, error) {
	resp, err := c.client.GetWLANDefaultsWithResponse(ctx, site)
	var dataPtr *[]WLANDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WLAN defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetWLANEnrichedConfiguration retrieves enriched WLAN configurations.
func (c *APIClient) GetWLANEnrichedConfiguration(ctx context.Context, site Site) ([]WLANEnrichedConfiguration, error) {
	resp, err := c.client.GetWLANEnrichedConfigurationWithResponse(ctx, site)
	var dataPtr *[]WLANEnrichedConfiguration
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WLAN enriched configuration for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetL2TPVPNDefaults retrieves default L2TP VPN configuration.
func (c *APIClient) GetL2TPVPNDefaults(ctx context.Context, site Site) (*L2TPVPNDefaults, error) {
	resp, err := c.client.GetL2TPVPNDefaultsWithResponse(ctx, site)
	var dataPtr *L2TPVPNDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get L2TP VPN defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetGlobalNetworkConfig retrieves global network configuration settings.
func (c *APIClient) GetGlobalNetworkConfig(ctx context.Context, site Site) (*GlobalNetworkConfig, error) {
	resp, err := c.client.GetGlobalNetworkConfigWithResponse(ctx, site)
	var dataPtr *GlobalNetworkConfig
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get global network config for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// ListMDNSServices retrieves available mDNS service definitions.
func (c *APIClient) ListMDNSServices(ctx context.Context, site Site) ([]MDNSService, error) {
	resp, err := c.client.ListMDNSServicesWithResponse(ctx, site)
	var dataPtr *[]MDNSService
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list mDNS services for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// ListCombinedFirewallRules retrieves combined traffic and firewall rules.
func (c *APIClient) ListCombinedFirewallRules(ctx context.Context, site Site) ([]CombinedFirewallRule, error) {
	resp, err := c.client.ListCombinedFirewallRulesWithResponse(ctx, site)
	var dataPtr *[]CombinedFirewallRule
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to list combined firewall rules for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetFirewallRulesDefaults retrieves default firewall rule settings.
func (c *APIClient) GetFirewallRulesDefaults(ctx context.Context, site Site) ([]FirewallRulesDefault, error) {
	resp, err := c.client.GetFirewallRulesDefaultsWithResponse(ctx, site)
	var dataPtr *[]FirewallRulesDefault
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get firewall rules defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetExcludedIPs retrieves IPs excluded from device identification.
func (c *APIClient) GetExcludedIPs(ctx context.Context, site Site) (*ExcludedIPs, error) {
	resp, err := c.client.GetExcludedIPsWithResponse(ctx, site)
	var dataPtr *ExcludedIPs
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get excluded IPs for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetNetworkSuggestions retrieves suggested IP subnets for new networks.
func (c *APIClient) GetNetworkSuggestions(ctx context.Context, site Site) ([]NetworkSuggestion, error) {
	resp, err := c.client.GetNetworkSuggestionsWithResponse(ctx, site)
	var dataPtr *[]NetworkSuggestion
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get network suggestions for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetNetworkPortSuggestions retrieves available ports for new services.
func (c *APIClient) GetNetworkPortSuggestions(ctx context.Context, site Site) (*NetworkPortSuggestions, error) {
	resp, err := c.client.GetNetworkPortSuggestionsWithResponse(ctx, site)
	var dataPtr *NetworkPortSuggestions
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get network port suggestions for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetWANLoadBalancingConfiguration retrieves WAN load balancing configuration.
func (c *APIClient) GetWANLoadBalancingConfiguration(ctx context.Context, site Site) (*WANLoadBalancingConfig, error) {
	resp, err := c.client.GetWANLoadBalancingConfigurationWithResponse(ctx, site)
	var dataPtr *WANLoadBalancingConfig
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WAN load balancing configuration for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetWANLoadBalancingStatus retrieves WAN load balancing interface status.
func (c *APIClient) GetWANLoadBalancingStatus(ctx context.Context, site Site) (*WANLoadBalancingStatus, error) {
	resp, err := c.client.GetWANLoadBalancingStatusWithResponse(ctx, site)
	var dataPtr *WANLoadBalancingStatus
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WAN load balancing status for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetWANNetworkGroups retrieves WAN network group configurations.
func (c *APIClient) GetWANNetworkGroups(ctx context.Context, site Site) (*WANNetworkGroups, error) {
	resp, err := c.client.GetWANNetworkGroupsWithResponse(ctx, site)
	var dataPtr *WANNetworkGroups
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WAN network groups for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetDDNSProviders retrieves available Dynamic DNS providers.
func (c *APIClient) GetDDNSProviders(ctx context.Context, site Site) (*DDNSProviders, error) {
	resp, err := c.client.GetDDNSProvidersWithResponse(ctx, site)
	var dataPtr *DDNSProviders
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get DDNS providers for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetNetworkStatus retrieves overall network health status.
func (c *APIClient) GetNetworkStatus(ctx context.Context, site Site) (*NetworkStatus, error) {
	resp, err := c.client.GetNetworkStatusWithResponse(ctx, site)
	var dataPtr *NetworkStatus
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get network status for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetTeleportInvitationHistory retrieves Teleport VPN invitation history.
func (c *APIClient) GetTeleportInvitationHistory(ctx context.Context, site Site) (*TeleportInvitationHistory, error) {
	resp, err := c.client.GetTeleportInvitationHistoryWithResponse(ctx, site)
	var dataPtr *TeleportInvitationHistory
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get Teleport invitation history for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetULPUsersGroups retrieves UniFi Local Portal users and groups.
func (c *APIClient) GetULPUsersGroups(ctx context.Context, site Site) (*ULPUsersGroups, error) {
	resp, err := c.client.GetULPUsersGroupsWithResponse(ctx, site)
	var dataPtr *ULPUsersGroups
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get ULP users and groups for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetWANMagicConfiguration retrieves WAN Magic (Teleport WAN) configuration.
func (c *APIClient) GetWANMagicConfiguration(ctx context.Context, site Site) (*WANMagicConfiguration, error) {
	resp, err := c.client.GetWANMagicConfigurationWithResponse(ctx, site)
	var dataPtr *WANMagicConfiguration
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WAN Magic configuration for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSystemLogSettings retrieves alert event settings for system logging.
func (c *APIClient) GetSystemLogSettings(ctx context.Context, site Site) (*SystemLogSettings, error) {
	resp, err := c.client.GetSystemLogSettingsWithResponse(ctx, site)
	var dataPtr *SystemLogSettings
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get system log settings for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSystemLogSettingsDefaults retrieves default alert event settings.
func (c *APIClient) GetSystemLogSettingsDefaults(ctx context.Context, site Site) (*SystemLogSettings, error) {
	resp, err := c.client.GetSystemLogSettingsDefaultsWithResponse(ctx, site)
	var dataPtr *SystemLogSettings
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get system log settings defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsConnectivityDefaults retrieves default connectivity settings.
func (c *APIClient) GetSettingsConnectivityDefaults(ctx context.Context, site Site) (*ConnectivityDefaults, error) {
	resp, err := c.client.GetSettingsConnectivityDefaultsWithResponse(ctx, site)
	var dataPtr *ConnectivityDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get connectivity defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsElementAdoptDefaults retrieves default element adoption settings.
func (c *APIClient) GetSettingsElementAdoptDefaults(ctx context.Context, site Site) (*ElementAdoptDefaults, error) {
	resp, err := c.client.GetSettingsElementAdoptDefaultsWithResponse(ctx, site)
	var dataPtr *ElementAdoptDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get element adopt defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsGlobalNATDefaults retrieves default global NAT settings.
func (c *APIClient) GetSettingsGlobalNATDefaults(ctx context.Context, site Site) (*GlobalNATDefaults, error) {
	resp, err := c.client.GetSettingsGlobalNATDefaultsWithResponse(ctx, site)
	var dataPtr *GlobalNATDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get global NAT defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsGlobalSwitchDefaults retrieves default global switch settings.
func (c *APIClient) GetSettingsGlobalSwitchDefaults(ctx context.Context, site Site) (*GlobalSwitchDefaults, error) {
	resp, err := c.client.GetSettingsGlobalSwitchDefaultsWithResponse(ctx, site)
	var dataPtr *GlobalSwitchDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get global switch defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsIPSAdvancedFilteringAutoValues retrieves auto-computed IPS advanced filtering settings.
func (c *APIClient) GetSettingsIPSAdvancedFilteringAutoValues(
	ctx context.Context, site Site,
) (*IPSAdvancedFilteringSettings, error) {
	resp, err := c.client.GetSettingsIPSAdvancedFilteringAutoValuesWithResponse(ctx, site)
	var dataPtr *IPSAdvancedFilteringSettings
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get IPS advanced filtering auto values for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsIPSAdvancedFilteringDefaults retrieves default IPS advanced filtering settings.
func (c *APIClient) GetSettingsIPSAdvancedFilteringDefaults(
	ctx context.Context, site Site,
) (*IPSAdvancedFilteringSettings, error) {
	resp, err := c.client.GetSettingsIPSAdvancedFilteringDefaultsWithResponse(ctx, site)
	var dataPtr *IPSAdvancedFilteringSettings
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get IPS advanced filtering defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsIPSAvailableCategories retrieves available IPS categories.
func (c *APIClient) GetSettingsIPSAvailableCategories(ctx context.Context, site Site) ([]IPSCategory, error) {
	resp, err := c.client.GetSettingsIPSAvailableCategoriesWithResponse(ctx, site)
	var dataPtr *[]IPSCategory
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get IPS available categories for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetSettingsNetflowDefaults retrieves default netflow settings.
func (c *APIClient) GetSettingsNetflowDefaults(ctx context.Context, site Site) (*NetflowDefaults, error) {
	resp, err := c.client.GetSettingsNetflowDefaultsWithResponse(ctx, site)
	var dataPtr *NetflowDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get netflow defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsNTPDefaults retrieves default NTP settings.
func (c *APIClient) GetSettingsNTPDefaults(ctx context.Context, site Site) (*NTPDefaults, error) {
	resp, err := c.client.GetSettingsNTPDefaultsWithResponse(ctx, site)
	var dataPtr *NTPDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get NTP defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsRoamingAssistantDefaults retrieves default roaming assistant settings.
func (c *APIClient) GetSettingsRoamingAssistantDefaults(
	ctx context.Context, site Site,
) (*RoamingAssistantDefaults, error) {
	resp, err := c.client.GetSettingsRoamingAssistantDefaultsWithResponse(ctx, site)
	var dataPtr *RoamingAssistantDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get roaming assistant defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsShortcuts retrieves configured shortcuts.
func (c *APIClient) GetSettingsShortcuts(ctx context.Context, site Site) ([]SettingsShortcut, error) {
	resp, err := c.client.GetSettingsShortcutsWithResponse(ctx, site)
	var dataPtr *[]SettingsShortcut
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get shortcuts for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetSettingsTeleportDefaults retrieves default teleport settings.
func (c *APIClient) GetSettingsTeleportDefaults(ctx context.Context, site Site) ([]TeleportSettingsDefaults, error) {
	resp, err := c.client.GetSettingsTeleportDefaultsWithResponse(ctx, site)
	var dataPtr *[]TeleportSettingsDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get teleport defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetSettingsTrafficFlowDefaults retrieves default traffic flow settings.
func (c *APIClient) GetSettingsTrafficFlowDefaults(ctx context.Context, site Site) (*TrafficFlowDefaults, error) {
	resp, err := c.client.GetSettingsTrafficFlowDefaultsWithResponse(ctx, site)
	var dataPtr *TrafficFlowDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get traffic flow defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsUSGDefaults retrieves default USG settings.
func (c *APIClient) GetSettingsUSGDefaults(ctx context.Context, site Site) (*USGDefaults, error) {
	resp, err := c.client.GetSettingsUSGDefaultsWithResponse(ctx, site)
	var dataPtr *USGDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get USG defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSettingsWiFiAIDefaults retrieves default WiFi AI settings.
func (c *APIClient) GetSettingsWiFiAIDefaults(ctx context.Context, site Site) ([]WiFiAIDefaults, error) {
	resp, err := c.client.GetSettingsWiFiAIDefaultsWithResponse(ctx, site)
	var dataPtr *[]WiFiAIDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WiFi AI defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetSSLInspectionCertificates retrieves SSL inspection certificates.
func (c *APIClient) GetSSLInspectionCertificates(
	ctx context.Context, site Site,
) ([]SSLInspectionCertificate, error) {
	resp, err := c.client.GetSSLInspectionCertificatesWithResponse(ctx, site)
	var dataPtr *[]SSLInspectionCertificate
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get SSL inspection certificates for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetSSLInspectionCertificateActive retrieves the active SSL inspection certificate.
func (c *APIClient) GetSSLInspectionCertificateActive(
	ctx context.Context, site Site,
) (*SSLInspectionCertificate, error) {
	resp, err := c.client.GetSSLInspectionCertificateActiveWithResponse(ctx, site)
	var dataPtr *SSLInspectionCertificate
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get active SSL inspection certificate for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSSLInspectionFileExtensions retrieves file extensions for SSL inspection.
func (c *APIClient) GetSSLInspectionFileExtensions(ctx context.Context, site Site) ([]string, error) {
	resp, err := c.client.GetSSLInspectionFileExtensionsWithResponse(ctx, site)
	var dataPtr *[]string
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get SSL inspection file extensions for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetSSLInspectionSearchEngines retrieves search engines for SSL inspection.
func (c *APIClient) GetSSLInspectionSearchEngines(
	ctx context.Context, site Site,
) ([]SSLInspectionSearchEngine, error) {
	resp, err := c.client.GetSSLInspectionSearchEnginesWithResponse(ctx, site)
	var dataPtr *[]SSLInspectionSearchEngine
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get SSL inspection search engines for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetSSLInspectionSetting retrieves SSL inspection setting.
func (c *APIClient) GetSSLInspectionSetting(ctx context.Context, site Site) (*SSLInspectionSetting, error) {
	resp, err := c.client.GetSSLInspectionSettingWithResponse(ctx, site)
	var dataPtr *SSLInspectionSetting
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get SSL inspection setting for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSSLInspectionSettingDefaults retrieves default SSL inspection settings.
func (c *APIClient) GetSSLInspectionSettingDefaults(ctx context.Context, site Site) (*SSLInspectionSetting, error) {
	resp, err := c.client.GetSSLInspectionSettingDefaultsWithResponse(ctx, site)
	var dataPtr *SSLInspectionSetting
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get SSL inspection setting defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSSLInspectionProfilesDefaults retrieves default SSL inspection profile configuration.
func (c *APIClient) GetSSLInspectionProfilesDefaults(
	ctx context.Context, site Site,
) (*SSLInspectionProfileDefaults, error) {
	resp, err := c.client.GetSSLInspectionProfilesDefaultsWithResponse(ctx, site)
	var dataPtr *SSLInspectionProfileDefaults
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get SSL inspection profiles defaults for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetShadowModeInfo retrieves shadow mode information.
func (c *APIClient) GetShadowModeInfo(ctx context.Context, site Site) (*ShadowModeInfo, error) {
	resp, err := c.client.GetShadowModeInfoWithResponse(ctx, site)
	var dataPtr *ShadowModeInfo
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get shadow mode info for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetStacking retrieves switch stacking configuration.
func (c *APIClient) GetStacking(ctx context.Context, site Site) ([]StackingConfig, error) {
	resp, err := c.client.GetStackingWithResponse(ctx, site)
	var dataPtr *[]StackingConfig
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get stacking configuration for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetMCLAGGroups retrieves MC-LAG group configurations.
func (c *APIClient) GetMCLAGGroups(ctx context.Context, site Site) ([]MCLAGGroup, error) {
	resp, err := c.client.GetMCLAGGroupsWithResponse(ctx, site)
	var dataPtr *[]MCLAGGroup
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get MC-LAG groups for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetDeviceWirelessLinks retrieves wireless mesh link configurations.
func (c *APIClient) GetDeviceWirelessLinks(ctx context.Context, site Site) ([]DeviceWirelessLink, error) {
	resp, err := c.client.GetDeviceWirelessLinksWithResponse(ctx, site)
	var dataPtr *[]DeviceWirelessLink
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get device wireless links for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetWireGuardUsersExistingSubnets retrieves WireGuard existing subnets.
func (c *APIClient) GetWireGuardUsersExistingSubnets(
	ctx context.Context, site Site,
) (*WireGuardExistingSubnets, error) {
	resp, err := c.client.GetWireGuardUsersExistingSubnetsWithResponse(ctx, site)
	var dataPtr *WireGuardExistingSubnets
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get WireGuard existing subnets for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetMagicSiteToSiteVPNConfigs retrieves Magic site-to-site VPN configurations.
func (c *APIClient) GetMagicSiteToSiteVPNConfigs(
	ctx context.Context, site Site,
) ([]MagicSiteToSiteVPNConfig, error) {
	resp, err := c.client.GetMagicSiteToSiteVPNConfigsWithResponse(ctx, site)
	var dataPtr *[]MagicSiteToSiteVPNConfig
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get Magic site-to-site VPN configs for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetObjectOrientedNetworkConfigs retrieves object-oriented network configurations.
func (c *APIClient) GetObjectOrientedNetworkConfigs(
	ctx context.Context, site Site,
) ([]ObjectOrientedNetworkConfig, error) {
	resp, err := c.client.GetObjectOrientedNetworkConfigsWithResponse(ctx, site)
	var dataPtr *[]ObjectOrientedNetworkConfig
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get OON configs for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetUtilizationLastDays retrieves utilization statistics for last days.
func (c *APIClient) GetUtilizationLastDays(ctx context.Context, site Site) (*UtilizationLastDays, error) {
	resp, err := c.client.GetUtilizationLastDaysWithResponse(ctx, site)
	var dataPtr *UtilizationLastDays
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get utilization data for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetVPNClientConnections retrieves current VPN client connections.
func (c *APIClient) GetVPNClientConnections(ctx context.Context, site Site) (*VPNClientConnections, error) {
	resp, err := c.client.GetVPNClientConnectionsWithResponse(ctx, site)
	var dataPtr *VPNClientConnections
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get VPN client connections for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSystemLogRemoteSettings retrieves remote syslog settings.
func (c *APIClient) GetSystemLogRemoteSettings(ctx context.Context, site Site) (*SystemLogRemoteSettings, error) {
	resp, err := c.client.GetSystemLogRemoteSettingsWithResponse(ctx, site)
	var dataPtr *SystemLogRemoteSettings
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get system log remote settings for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return data, nil
}

// GetSystemLogDisplayOptionsAdmins retrieves admin users for log filtering.
func (c *APIClient) GetSystemLogDisplayOptionsAdmins(ctx context.Context, site Site) ([]SystemLogAdmin, error) {
	resp, err := c.client.GetSystemLogDisplayOptionsAdminsWithResponse(ctx, site)
	var dataPtr *[]SystemLogAdmin
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get system log display options admins for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetSystemLogThreatDisplayOptionsClients retrieves clients for threat log filtering.
func (c *APIClient) GetSystemLogThreatDisplayOptionsClients(
	ctx context.Context, site Site,
) ([]SystemLogClient, error) {
	resp, err := c.client.GetSystemLogThreatDisplayOptionsClientsWithResponse(ctx, site)
	var dataPtr *[]SystemLogClient
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get system log threat display options clients for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetSystemLogTriggersDisplayOptionsHosts retrieves hosts for trigger log filtering.
func (c *APIClient) GetSystemLogTriggersDisplayOptionsHosts(
	ctx context.Context, site Site,
) ([]SystemLogHost, error) {
	resp, err := c.client.GetSystemLogTriggersDisplayOptionsHostsWithResponse(ctx, site)
	var dataPtr *[]SystemLogHost
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get system log triggers display options hosts for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetSystemLogAPLogsDisplayOptionsAPs retrieves AP MAC addresses for AP log filtering.
func (c *APIClient) GetSystemLogAPLogsDisplayOptionsAPs(ctx context.Context, site Site) ([]string, error) {
	resp, err := c.client.GetSystemLogAPLogsDisplayOptionsAPsWithResponse(ctx, site)
	var dataPtr *[]string
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get system log AP logs display options APs for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetWiFiStatsRadios retrieves WiFi statistics per radio.
func (c *APIClient) GetWiFiStatsRadios(
	ctx context.Context,
	site Site,
	params *GetWiFiStatsRadiosParams,
) (*WiFiStatsRadiosResponse, error) {
	resp, err := c.client.GetWiFiStatsRadiosWithResponse(ctx, site, params)
	var dataPtr *WiFiStatsRadiosResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get WiFi stats radios for site "+site)
}

// GetWiFiStatsChannelization retrieves WiFi channelization statistics over time.
func (c *APIClient) GetWiFiStatsChannelization(
	ctx context.Context,
	site Site,
	params *GetWiFiStatsChannelizationParams,
) (*WiFiStatsChannelizationResponse, error) {
	resp, err := c.client.GetWiFiStatsChannelizationWithResponse(ctx, site, params)
	var dataPtr *WiFiStatsChannelizationResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get WiFi stats channelization for site "+site)
}

// GetWiFiStatsDetails retrieves detailed WiFi statistics per connected client.
func (c *APIClient) GetWiFiStatsDetails(
	ctx context.Context,
	site Site,
	params *GetWiFiStatsDetailsParams,
) (*WiFiStatsDetailsResponse, error) {
	resp, err := c.client.GetWiFiStatsDetailsWithResponse(ctx, site, params)
	var dataPtr *WiFiStatsDetailsResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get WiFi stats details for site "+site)
}

// GetTrafficStats retrieves traffic statistics by client and application.
func (c *APIClient) GetTrafficStats(
	ctx context.Context,
	site Site,
	params *GetTrafficStatsParams,
) (*TrafficStatsResponse, error) {
	resp, err := c.client.GetTrafficStatsWithResponse(ctx, site, params)
	var dataPtr *TrafficStatsResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get traffic stats for site "+site)
}

// GetTrafficRate retrieves traffic rate time series data.
func (c *APIClient) GetTrafficRate(
	ctx context.Context,
	site Site,
	params *GetTrafficRateParams,
) ([]TrafficRateEntry, error) {
	resp, err := c.client.GetTrafficRateWithResponse(ctx, site, params)
	var dataPtr *[]TrafficRateEntry
	if resp != nil {
		dataPtr = resp.JSON200
	}
	data, err := response.Handle(resp, dataPtr, err, "failed to get traffic rate for site "+site)
	if err != nil {
		//nolint:wrapcheck // err is already wrapped by response.Handle
		return nil, err
	}
	return *data, nil
}

// GetUtilizationTimeRange retrieves system and ISP utilization over time range.
func (c *APIClient) GetUtilizationTimeRange(
	ctx context.Context,
	site Site,
	params *GetUtilizationTimeRangeParams,
) (*UtilizationTimeRangeResponse, error) {
	resp, err := c.client.GetUtilizationTimeRangeWithResponse(ctx, site, params)
	var dataPtr *UtilizationTimeRangeResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get utilization time range for site "+site)
}

// GetSystemLogCount retrieves system log counts by category.
func (c *APIClient) GetSystemLogCount(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogCountResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogCountWithResponse(ctx, site, *request)
	var dataPtr *SystemLogCountResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get system log count for site "+site)
}

// GetSystemLogCritical retrieves critical system logs.
func (c *APIClient) GetSystemLogCritical(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogCriticalWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get critical system logs for site "+site)
}

// GetSystemLogThreats retrieves threat detection logs.
func (c *APIClient) GetSystemLogThreats(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogThreatsWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get threat logs for site "+site)
}

// GetSystemLogAdminAccess retrieves admin access logs.
func (c *APIClient) GetSystemLogAdminAccess(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogAdminAccessWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get admin access logs for site "+site)
}

// GetSystemLogAdminActivity retrieves admin activity logs.
func (c *APIClient) GetSystemLogAdminActivity(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogAdminActivityWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get admin activity logs for site "+site)
}

// GetSystemLogClientAlert retrieves client alert logs.
func (c *APIClient) GetSystemLogClientAlert(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogClientAlertWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get client alert logs for site "+site)
}

// GetSystemLogDeviceAlert retrieves device alert logs.
func (c *APIClient) GetSystemLogDeviceAlert(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogDeviceAlertWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get device alert logs for site "+site)
}

// GetSystemLogThreatAlert retrieves security threat alert logs.
func (c *APIClient) GetSystemLogThreatAlert(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogThreatAlertWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get threat alert logs for site "+site)
}

// GetSystemLogUpdateAlert retrieves update alert logs.
func (c *APIClient) GetSystemLogUpdateAlert(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogUpdateAlertWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get update alert logs for site "+site)
}

// GetSystemLogVPNAlert retrieves VPN alert logs.
func (c *APIClient) GetSystemLogVPNAlert(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogVPNAlertWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get VPN alert logs for site "+site)
}

// GetSystemLogNextAIAlert retrieves AI-powered threat detection alert logs.
func (c *APIClient) GetSystemLogNextAIAlert(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogNextAIAlertWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get Next AI alert logs for site "+site)
}

// GetSystemLogSystemCriticalAlert retrieves system critical alert logs.
func (c *APIClient) GetSystemLogSystemCriticalAlert(
	ctx context.Context,
	site Site,
	request *SystemLogRequest,
) (*SystemLogResponse, error) {
	if request == nil {
		request = &SystemLogRequest{}
	}
	resp, err := c.client.GetSystemLogSystemCriticalAlertWithResponse(ctx, site, *request)
	var dataPtr *SystemLogResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get system critical alert logs for site "+site)
}

// GetTrafficFlows retrieves firewall traffic flow data.
func (c *APIClient) GetTrafficFlows(
	ctx context.Context,
	site Site,
	request *TrafficFlowsRequest,
) (*TrafficFlowsResponse, error) {
	if request == nil {
		request = &TrafficFlowsRequest{}
	}
	resp, err := c.client.GetTrafficFlowsWithResponse(ctx, site, *request)
	var dataPtr *TrafficFlowsResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get traffic flows for site "+site)
}

// GetOpenVPNCertificates retrieves OpenVPN certificates.
func (c *APIClient) GetOpenVPNCertificates(
	ctx context.Context,
	site Site,
) (*OpenVPNCertificatesResponse, error) {
	body := map[string]any{}
	resp, err := c.client.GetOpenVPNCertificatesWithResponse(ctx, site, body)
	var dataPtr *OpenVPNCertificatesResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get OpenVPN certificates for site "+site)
}

// GetAliases retrieves client aliases for specified MAC addresses.
func (c *APIClient) GetAliases(
	ctx context.Context,
	site Site,
	request *AliasRequest,
) (*AliasResponse, error) {
	resp, err := c.client.GetAliasesWithResponse(ctx, site, *request)
	var dataPtr *AliasResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get aliases for site "+site)
}

// GetClientsMetadata retrieves metadata for clients by their MAC addresses.
func (c *APIClient) GetClientsMetadata(
	ctx context.Context,
	site Site,
	macs []string,
) (*ClientsMetadataResponse, error) {
	resp, err := c.client.GetClientsMetadataWithResponse(ctx, site, macs)
	var dataPtr *ClientsMetadataResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get clients metadata for site "+site)
}

// GetPortSystemLogs retrieves system logs for a specific port.
func (c *APIClient) GetPortSystemLogs(
	ctx context.Context,
	site Site,
	request *PortSystemLogsRequest,
) (*PortSystemLogsResponse, error) {
	resp, err := c.client.GetPortSystemLogsWithResponse(ctx, site, *request)
	var dataPtr *PortSystemLogsResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get port system logs for site "+site)
}

// =============================================================================
// Integration API: Application Info
// =============================================================================

// GetApplicationInfo retrieves general information about the UniFi Network application.
func (c *APIClient) GetApplicationInfo(ctx context.Context) (*ApplicationInfo, error) {
	resp, err := c.client.GetApplicationInfoWithResponse(ctx)
	var dataPtr *ApplicationInfo
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, "failed to get application info")
}

// =============================================================================
// Integration API: Device Statistics
// =============================================================================

// GetDeviceLatestStatistics retrieves the latest real-time statistics of a device.
func (c *APIClient) GetDeviceLatestStatistics(
	ctx context.Context,
	siteID openapi_types.UUID,
	deviceID openapi_types.UUID,
) (*DeviceStatistics, error) {
	resp, err := c.client.GetDeviceLatestStatisticsWithResponse(ctx, siteID, deviceID)
	var dataPtr *DeviceStatistics
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, fmt.Sprintf("failed to get statistics for device %s", deviceID))
}

// =============================================================================
// Integration API: Device Actions
// =============================================================================

// ExecuteDeviceAction performs an action on a specific adopted device.
func (c *APIClient) ExecuteDeviceAction(
	ctx context.Context,
	siteID openapi_types.UUID,
	deviceID openapi_types.UUID,
	request DeviceActionRequest,
) error {
	resp, err := c.client.ExecuteDeviceActionWithResponse(ctx, siteID, deviceID, request)
	if err != nil {
		return errors.Wrapf(err, "failed to execute action %s on device %s", request.Action, deviceID)
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("failed to execute action %s on device %s: status %d",
			request.Action, deviceID, resp.StatusCode())
	}
	return nil
}

// RestartDevice restarts a specific adopted device.
func (c *APIClient) RestartDevice(
	ctx context.Context,
	siteID openapi_types.UUID,
	deviceID openapi_types.UUID,
) error {
	return c.ExecuteDeviceAction(ctx, siteID, deviceID, DeviceActionRequest{
		Action: RESTART,
	})
}

// =============================================================================
// Integration API: Port Actions
// =============================================================================

// ExecutePortAction performs an action on a specific device port.
func (c *APIClient) ExecutePortAction(
	ctx context.Context,
	siteID openapi_types.UUID,
	deviceID openapi_types.UUID,
	portIdx int32,
	request PortActionRequest,
) error {
	resp, err := c.client.ExecutePortActionWithResponse(ctx, siteID, deviceID, portIdx, request)
	if err != nil {
		return errors.Wrapf(err, "failed to execute action %s on port %d of device %s",
			request.Action, portIdx, deviceID)
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("failed to execute action %s on port %d of device %s: status %d",
			request.Action, portIdx, deviceID, resp.StatusCode())
	}
	return nil
}

// PowerCyclePort power cycles a PoE port (turns it off and on again).
func (c *APIClient) PowerCyclePort(
	ctx context.Context,
	siteID openapi_types.UUID,
	deviceID openapi_types.UUID,
	portIdx int32,
) error {
	return c.ExecutePortAction(ctx, siteID, deviceID, portIdx, PortActionRequest{
		Action: POWERCYCLE,
	})
}

// =============================================================================
// Integration API: Client Actions
// =============================================================================

// ExecuteClientAction performs an action on a specific connected client.
func (c *APIClient) ExecuteClientAction(
	ctx context.Context,
	siteID openapi_types.UUID,
	clientID openapi_types.UUID,
	request ClientActionRequest,
) (*ClientActionResponse, error) {
	resp, err := c.client.ExecuteClientActionWithResponse(ctx, clientID, siteID, request)
	var dataPtr *ClientActionResponse
	if resp != nil {
		dataPtr = resp.JSON200
	}
	//nolint:wrapcheck // response.Handle wraps errors internally
	return response.Handle(resp, dataPtr, err, fmt.Sprintf("failed to execute action on client %s", clientID))
}

// AuthorizeGuestAccess authorizes a guest client for network access.
func (c *APIClient) AuthorizeGuestAccess(
	ctx context.Context,
	siteID openapi_types.UUID,
	clientID openapi_types.UUID,
	options *GuestAccessOptions,
) (*ClientActionResponse, error) {
	request := ClientActionRequest{
		Action: ClientActionRequestActionAUTHORIZEGUESTACCESS,
	}
	if options != nil {
		request.TimeLimitMinutes = options.TimeLimitMinutes
		request.DataUsageLimitMBytes = options.DataUsageLimitMBytes
		request.RxRateLimitKbps = options.RxRateLimitKbps
		request.TxRateLimitKbps = options.TxRateLimitKbps
	}
	return c.ExecuteClientAction(ctx, siteID, clientID, request)
}

// UnauthorizeGuestAccess revokes guest authorization and disconnects the client.
func (c *APIClient) UnauthorizeGuestAccess(
	ctx context.Context,
	siteID openapi_types.UUID,
	clientID openapi_types.UUID,
) (*ClientActionResponse, error) {
	return c.ExecuteClientAction(ctx, siteID, clientID, ClientActionRequest{
		Action: ClientActionRequestActionUNAUTHORIZEGUESTACCESS,
	})
}

// GuestAccessOptions contains optional parameters for authorizing guest access.
type GuestAccessOptions struct {
	// TimeLimitMinutes is how long (in minutes) the guest will be authorized
	TimeLimitMinutes *int64
	// DataUsageLimitMBytes is the data usage limit in megabytes
	DataUsageLimitMBytes *int64
	// RxRateLimitKbps is the download rate limit in kilobits per second
	RxRateLimitKbps *int64
	// TxRateLimitKbps is the upload rate limit in kilobits per second
	TxRateLimitKbps *int64
}

// ============================================================================
// Legacy Cmd API - Device Manager Commands
// These methods use the legacy /api/s/{site}/cmd/devmgr endpoint for operations
// not available in the Integration API, such as LED blinking and PoE power cycling.
// ============================================================================

// ExecuteDeviceManagerCommand executes a device manager command on a device.
// This is a low-level method; prefer using the higher-level wrapper methods like
// LocateDevice, UnlocateDevice, PowerCyclePortByMAC, or RestartDeviceByMAC.
func (c *APIClient) ExecuteDeviceManagerCommand(
	ctx context.Context,
	site Site,
	request DeviceManagerCommandRequest,
) error {
	resp, err := c.client.ExecuteDeviceManagerCommandWithResponse(ctx, site, request)
	if err != nil {
		return errors.Wrap(err, "failed to execute device manager command")
	}

	if resp.StatusCode() != http.StatusOK {
		//nolint:wrapcheck // Creating new error for unexpected status code, no source error to wrap
		return errors.Newf("device manager command failed: status=%d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return errors.New("unexpected nil response")
	}

	// Check legacy API result code
	if resp.JSON200.Meta.Rc != "ok" {
		msg := "command failed"
		if resp.JSON200.Meta.Msg != nil {
			msg = *resp.JSON200.Meta.Msg
		}
		//nolint:wrapcheck // Creating new error for API error, no source error to wrap
		return errors.Newf("device manager command failed: %s", msg)
	}

	return nil
}

// LocateDevice starts LED blinking on a device to help locate it physically.
// The device MAC address should be in lowercase with colons (e.g., "ac:8b:a9:3c:12:5d").
func (c *APIClient) LocateDevice(ctx context.Context, site Site, mac string) error {
	return c.ExecuteDeviceManagerCommand(ctx, site, DeviceManagerCommandRequest{
		Cmd: SetLocate,
		Mac: mac,
	})
}

// UnlocateDevice stops LED blinking on a device.
// The device MAC address should be in lowercase with colons (e.g., "ac:8b:a9:3c:12:5d").
func (c *APIClient) UnlocateDevice(ctx context.Context, site Site, mac string) error {
	return c.ExecuteDeviceManagerCommand(ctx, site, DeviceManagerCommandRequest{
		Cmd: UnsetLocate,
		Mac: mac,
	})
}

// PowerCyclePortByMAC power cycles a PoE port on a device identified by MAC address.
// This will cut power to the connected device and restore it after a short delay.
//
// Parameters:
//   - site: Site reference (e.g., "default")
//   - deviceMAC: MAC address of the device with the PoE port (e.g., "94:2a:6f:ce:26:52")
//   - portIdx: Port index (1-based) of the PoE port to power cycle
//
// Note: Only devices with PoE capability on the specified port can use this command.
// This uses the legacy API which works with devices that don't have UUIDs in the Integration API.
func (c *APIClient) PowerCyclePortByMAC(ctx context.Context, site Site, deviceMAC string, portIdx int) error {
	portIdxInt := portIdx

	return c.ExecuteDeviceManagerCommand(ctx, site, DeviceManagerCommandRequest{
		Cmd:     PowerCycle,
		Mac:     deviceMAC,
		PortIdx: &portIdxInt,
	})
}

// RestartDeviceByMAC restarts a device by its MAC address using the legacy API.
// The device MAC address should be in lowercase with colons (e.g., "ac:8b:a9:3c:12:5d").
//
// Note: This is an alternative to RestartDevice which uses the Integration API with UUIDs.
// Use this for devices that don't have UUIDs in the Integration API (like gateway devices).
func (c *APIClient) RestartDeviceByMAC(ctx context.Context, site Site, mac string) error {
	return c.ExecuteDeviceManagerCommand(ctx, site, DeviceManagerCommandRequest{
		Cmd: Restart,
		Mac: mac,
	})
}

// AdoptDevice initiates adoption of a new device.
// The device MAC address should be in lowercase with colons (e.g., "ac:8b:a9:3c:12:5d").
func (c *APIClient) AdoptDevice(ctx context.Context, site Site, mac string) error {
	return c.ExecuteDeviceManagerCommand(ctx, site, DeviceManagerCommandRequest{
		Cmd: Adopt,
		Mac: mac,
	})
}

// ForceProvisionDevice forces re-provisioning of a device configuration.
// The device MAC address should be in lowercase with colons (e.g., "ac:8b:a9:3c:12:5d").
func (c *APIClient) ForceProvisionDevice(ctx context.Context, site Site, mac string) error {
	return c.ExecuteDeviceManagerCommand(ctx, site, DeviceManagerCommandRequest{
		Cmd: ForceProvision,
		Mac: mac,
	})
}
