package network

import (
	"context"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// NetworkAPIClient defines the interface for UniFi Network API operations.
// This interface enables consumers to create mock implementations for testing.
//
// The Network API provides access to a local UniFi controller for managing:
//   - Sites and devices
//   - Network clients
//   - DNS records
//   - Firewall policies
//   - Traffic rules (QoS)
//   - Hotspot vouchers
//   - Dashboard statistics
//
// All methods mirror the corresponding methods in APIClient to ensure
// compatibility and ease of use.
//
// Example usage with mocking frameworks:
//
//	// Using gomock:
//	//go:generate mockgen -destination=mocks/network_client.go -package=mocks github.com/lexfrei/go-unifi/api/network NetworkAPIClient
//
//	// Using testify/mock:
//	type MockClient struct {
//	    mock.Mock
//	}
//
//	func (m *MockClient) ListDNSRecords(ctx context.Context, site Site) ([]DNSRecord, error) {
//	    args := m.Called(ctx, site)
//	    return args.Get(0).([]DNSRecord), args.Error(1)
//	}
//
//nolint:revive // NetworkAPIClient is intentionally explicit to avoid confusion with APIClient struct
type NetworkAPIClient interface { //nolint:interfacebloat // This interface mirrors the full API client with 22 methods
	// Sites operations

	// ListSites retrieves a list of all sites configured on the controller.
	ListSites(ctx context.Context, params *ListSitesParams) (*SitesResponse, error)

	// Devices operations

	// ListSiteDevices retrieves a list of all devices for a specific site.
	ListSiteDevices(ctx context.Context, siteID SiteId, params *ListSiteDevicesParams) (*DevicesResponse, error)

	// GetDeviceByID retrieves detailed information about a specific device.
	GetDeviceByID(ctx context.Context, siteID SiteId, deviceID DeviceId) (*Device, error)

	// Clients operations

	// ListSiteClients retrieves a list of all clients for a specific site.
	ListSiteClients(ctx context.Context, siteID SiteId, params *ListSiteClientsParams) (*ClientsResponse, error)

	// GetClientByID retrieves detailed information about a specific client.
	GetClientByID(ctx context.Context, siteID SiteId, clientID ClientId) (*NetworkClient, error)

	// Hotspot vouchers operations

	// ListHotspotVouchers retrieves a list of all hotspot vouchers for a specific site.
	ListHotspotVouchers(ctx context.Context, siteID SiteId, params *ListHotspotVouchersParams) (*HotspotVouchersResponse, error)

	// CreateHotspotVouchers creates one or more hotspot vouchers for temporary guest access.
	CreateHotspotVouchers(ctx context.Context, siteID SiteId, request *CreateVouchersRequest) (*HotspotVouchersResponse, error)

	// GetHotspotVoucher retrieves detailed information about a specific hotspot voucher.
	GetHotspotVoucher(ctx context.Context, siteID SiteId, voucherID openapi_types.UUID) (*HotspotVoucher, error)

	// DeleteHotspotVoucher permanently deletes a hotspot voucher.
	DeleteHotspotVoucher(ctx context.Context, siteID SiteId, voucherID openapi_types.UUID) error

	// DNS records operations

	// ListDNSRecords lists all static DNS records for a site.
	ListDNSRecords(ctx context.Context, site Site) ([]DNSRecord, error)

	// CreateDNSRecord creates a new static DNS record.
	CreateDNSRecord(ctx context.Context, site Site, record *DNSRecordInput) (*DNSRecord, error)

	// UpdateDNSRecord updates an existing DNS record.
	UpdateDNSRecord(ctx context.Context, site Site, recordID RecordId, record *DNSRecordInput) (*DNSRecord, error)

	// DeleteDNSRecord deletes a DNS record.
	DeleteDNSRecord(ctx context.Context, site Site, recordID RecordId) error

	// Firewall policies operations

	// ListFirewallPolicies lists all firewall policies for a site.
	ListFirewallPolicies(ctx context.Context, site Site) ([]FirewallPolicy, error)

	// CreateFirewallPolicy creates a new firewall policy.
	CreateFirewallPolicy(ctx context.Context, site Site, policy *FirewallPolicyInput) (*FirewallPolicy, error)

	// UpdateFirewallPolicy updates an existing firewall policy.
	UpdateFirewallPolicy(ctx context.Context, site Site, policyID PolicyId, policy *FirewallPolicyInput) (*FirewallPolicy, error)

	// DeleteFirewallPolicy permanently deletes a firewall policy.
	DeleteFirewallPolicy(ctx context.Context, site Site, policyID PolicyId) error

	// Traffic rules operations

	// ListTrafficRules lists all traffic rules for a site.
	ListTrafficRules(ctx context.Context, site Site) ([]TrafficRule, error)

	// CreateTrafficRule creates a new traffic rule.
	CreateTrafficRule(ctx context.Context, site Site, rule *TrafficRuleInput) (*TrafficRule, error)

	// UpdateTrafficRule updates an existing traffic rule.
	UpdateTrafficRule(ctx context.Context, site Site, ruleID RuleId, rule *TrafficRuleInput) (*TrafficRule, error)

	// DeleteTrafficRule permanently deletes a traffic rule.
	DeleteTrafficRule(ctx context.Context, site Site, ruleID RuleId) error

	// Dashboard operations

	// GetAggregatedDashboard retrieves aggregated dashboard statistics.
	GetAggregatedDashboard(ctx context.Context, site Site, params *GetAggregatedDashboardParams) (*AggregatedDashboard, error)
}
