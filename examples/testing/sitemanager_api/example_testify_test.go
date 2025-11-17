package sitemanager_api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/lexfrei/go-unifi/api/sitemanager"
)

// MockSiteManagerClient is a mock implementation of sitemanager.SiteManagerAPIClient for testing.
type MockSiteManagerClient struct {
	mock.Mock
}

// ListHosts mocks the ListHosts method.
func (m *MockSiteManagerClient) ListHosts(ctx context.Context, params *sitemanager.ListHostsParams) (*sitemanager.HostsResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sitemanager.HostsResponse), args.Error(1)
}

// GetHostByID mocks the GetHostByID method.
func (m *MockSiteManagerClient) GetHostByID(ctx context.Context, hostID string) (*sitemanager.HostResponse, error) {
	args := m.Called(ctx, hostID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sitemanager.HostResponse), args.Error(1)
}

// Implement other methods as no-op (not used in these tests)
func (m *MockSiteManagerClient) ListSites(ctx context.Context) (*sitemanager.SitesResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockSiteManagerClient) ListDevices(ctx context.Context, params *sitemanager.ListDevicesParams) (*sitemanager.DevicesResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockSiteManagerClient) GetISPMetrics(ctx context.Context, metricType sitemanager.GetISPMetricsParamsType, params *sitemanager.GetISPMetricsParams) (*sitemanager.ISPMetricsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockSiteManagerClient) QueryISPMetrics(ctx context.Context, metricType string, query sitemanager.ISPMetricsQuery) (*sitemanager.ISPMetricsQueryResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockSiteManagerClient) ListSDWANConfigs(ctx context.Context) (*sitemanager.SDWANConfigsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockSiteManagerClient) GetSDWANConfigByID(ctx context.Context, configID string) (*sitemanager.SDWANConfigResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockSiteManagerClient) GetSDWANConfigStatus(ctx context.Context, configID string) (*sitemanager.SDWANConfigStatusResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// Example application code that uses the Site Manager API client

// HostMonitor monitors UniFi hosts across all sites.
type HostMonitor struct {
	client sitemanager.SiteManagerAPIClient
}

// NewHostMonitor creates a new host monitor.
func NewHostMonitor(client sitemanager.SiteManagerAPIClient) *HostMonitor {
	return &HostMonitor{client: client}
}

// GetTotalHosts returns the total number of hosts.
func (hm *HostMonitor) GetTotalHosts(ctx context.Context) (int, error) {
	resp, err := hm.client.ListHosts(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to list hosts: %w", err)
	}
	if resp.Data == nil {
		return 0, nil
	}
	return len(resp.Data), nil
}

// GetHostByID finds a host by its ID.
func (hm *HostMonitor) GetHostByID(ctx context.Context, id string) (*sitemanager.Host, error) {
	resp, err := hm.client.ListHosts(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list hosts: %w", err)
	}

	if resp.Data == nil {
		return nil, fmt.Errorf("host not found: %s", id)
	}

	for i := range resp.Data {
		host := &resp.Data[i]
		if host.Id == id {
			return host, nil
		}
	}

	return nil, fmt.Errorf("host not found: %s", id)
}

// Tests

func TestHostMonitor_GetTotalHosts_Success(t *testing.T) {
	mockClient := new(MockSiteManagerClient)

	// Setup expectations
	expectedResponse := &sitemanager.HostsResponse{
		Data: []sitemanager.Host{
			{Id: "host1", HardwareId: "UDM-001"},
			{Id: "host2", HardwareId: "UDM-002"},
			{Id: "host3", HardwareId: "UDR-001"},
		},
	}

	mockClient.On("ListHosts", mock.Anything, (*sitemanager.ListHostsParams)(nil)).
		Return(expectedResponse, nil)

	// Create monitor and test
	monitor := NewHostMonitor(mockClient)
	count, err := monitor.GetTotalHosts(context.Background())

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
	mockClient.AssertExpectations(t)
}

func TestHostMonitor_GetTotalHosts_Empty(t *testing.T) {
	mockClient := new(MockSiteManagerClient)

	// Setup expectations with empty response
	expectedResponse := &sitemanager.HostsResponse{
		Data: []sitemanager.Host{},
	}

	mockClient.On("ListHosts", mock.Anything, (*sitemanager.ListHostsParams)(nil)).
		Return(expectedResponse, nil)

	// Create monitor and test
	monitor := NewHostMonitor(mockClient)
	count, err := monitor.GetTotalHosts(context.Background())

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	mockClient.AssertExpectations(t)
}

func TestHostMonitor_GetTotalHosts_Error(t *testing.T) {
	mockClient := new(MockSiteManagerClient)

	// Setup expectations with error
	mockClient.On("ListHosts", mock.Anything, (*sitemanager.ListHostsParams)(nil)).
		Return(nil, fmt.Errorf("API error: authentication failed"))

	// Create monitor and test
	monitor := NewHostMonitor(mockClient)
	count, err := monitor.GetTotalHosts(context.Background())

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "failed to list hosts")
	mockClient.AssertExpectations(t)
}

func TestHostMonitor_GetHostByID_Found(t *testing.T) {
	mockClient := new(MockSiteManagerClient)

	// Setup expectations
	expectedResponse := &sitemanager.HostsResponse{
		Data: []sitemanager.Host{
			{Id: "id1", HardwareId: "UDM-001"},
			{Id: "id2", HardwareId: "UDM-002"},
			{Id: "target-id", HardwareId: "UDR-001"},
		},
	}

	mockClient.On("ListHosts", mock.Anything, (*sitemanager.ListHostsParams)(nil)).
		Return(expectedResponse, nil)

	// Create monitor and test
	monitor := NewHostMonitor(mockClient)
	host, err := monitor.GetHostByID(context.Background(), "target-id")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, host)
	assert.Equal(t, "target-id", host.Id)
	assert.Equal(t, "UDR-001", host.HardwareId)
	mockClient.AssertExpectations(t)
}

func TestHostMonitor_GetHostByID_NotFound(t *testing.T) {
	mockClient := new(MockSiteManagerClient)

	// Setup expectations
	expectedResponse := &sitemanager.HostsResponse{
		Data: []sitemanager.Host{
			{Id: "id1", HardwareId: "UDM-001"},
			{Id: "id2", HardwareId: "UDM-002"},
		},
	}

	mockClient.On("ListHosts", mock.Anything, (*sitemanager.ListHostsParams)(nil)).
		Return(expectedResponse, nil)

	// Create monitor and test
	monitor := NewHostMonitor(mockClient)
	host, err := monitor.GetHostByID(context.Background(), "notfound")

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, host)
	assert.Contains(t, err.Error(), "host not found")
	mockClient.AssertExpectations(t)
}
