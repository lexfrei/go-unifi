package sitemanager

import "context"

// SiteManagerAPIClient defines the interface for UniFi Site Manager API operations.
// This interface enables consumers to create mock implementations for testing.
//
// The Site Manager API provides cloud-based access to UniFi infrastructure for managing:
//   - Hosts (controllers/consoles)
//   - Sites across multiple locations
//   - Devices across all sites
//   - ISP performance metrics
//   - SD-WAN configurations
//
// All methods mirror the corresponding methods in UnifiClient to ensure
// compatibility and ease of use.
//
// Example usage with mocking frameworks:
//
//	// Using gomock:
//	//go:generate mockgen -destination=mocks/sitemanager_client.go -package=mocks github.com/lexfrei/go-unifi/api/sitemanager SiteManagerAPIClient
//
//	// Using testify/mock:
//	type MockClient struct {
//	    mock.Mock
//	}
//
//	func (m *MockClient) ListHosts(ctx context.Context, params *ListHostsParams) (*HostsResponse, error) {
//	    args := m.Called(ctx, params)
//	    return args.Get(0).(*HostsResponse), args.Error(1)
//	}
//
//nolint:revive // SiteManagerAPIClient is intentionally explicit to avoid confusion with UnifiClient struct
type SiteManagerAPIClient interface {
	// Hosts operations

	// ListHosts retrieves a list of all hosts across all sites.
	ListHosts(ctx context.Context, params *ListHostsParams) (*HostsResponse, error)

	// GetHostByID retrieves detailed information about a specific host.
	GetHostByID(ctx context.Context, hostID string) (*HostResponse, error)

	// Sites operations

	// ListSites retrieves a list of all sites configured on the controller.
	ListSites(ctx context.Context) (*SitesResponse, error)

	// Devices operations

	// ListDevices retrieves a list of all devices across all sites.
	ListDevices(ctx context.Context, params *ListDevicesParams) (*DevicesResponse, error)

	// ISP metrics operations

	// GetISPMetrics retrieves ISP performance metrics.
	GetISPMetrics(ctx context.Context, metricType GetISPMetricsParamsType, params *GetISPMetricsParams) (*ISPMetricsResponse, error)

	// QueryISPMetrics queries ISP metrics with custom parameters.
	QueryISPMetrics(ctx context.Context, metricType string, query ISPMetricsQuery) (*ISPMetricsQueryResponse, error)

	// SD-WAN operations

	// ListSDWANConfigs retrieves a list of all SD-WAN configurations.
	ListSDWANConfigs(ctx context.Context) (*SDWANConfigsResponse, error)

	// GetSDWANConfigByID retrieves detailed information about a specific SD-WAN configuration.
	GetSDWANConfigByID(ctx context.Context, configID string) (*SDWANConfigResponse, error)

	// GetSDWANConfigStatus retrieves the status of a specific SD-WAN configuration.
	GetSDWANConfigStatus(ctx context.Context, configID string) (*SDWANConfigStatusResponse, error)
}
