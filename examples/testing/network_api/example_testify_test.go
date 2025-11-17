package network_api_test

import (
	"context"
	"fmt"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/lexfrei/go-unifi/api/network"
)

// MockNetworkClient is a mock implementation of network.NetworkAPIClient for testing.
type MockNetworkClient struct {
	mock.Mock
}

// ListDNSRecords mocks the ListDNSRecords method.
func (m *MockNetworkClient) ListDNSRecords(ctx context.Context, site network.Site) ([]network.DNSRecord, error) {
	args := m.Called(ctx, site)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]network.DNSRecord), args.Error(1)
}

// CreateDNSRecord mocks the CreateDNSRecord method.
func (m *MockNetworkClient) CreateDNSRecord(ctx context.Context, site network.Site, record *network.DNSRecordInput) (*network.DNSRecord, error) {
	args := m.Called(ctx, site, record)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*network.DNSRecord), args.Error(1)
}

// DeleteDNSRecord mocks the DeleteDNSRecord method.
func (m *MockNetworkClient) DeleteDNSRecord(ctx context.Context, site network.Site, recordID network.RecordId) error {
	args := m.Called(ctx, site, recordID)
	return args.Error(0)
}

// Implement other methods as no-op (not used in these tests)
func (m *MockNetworkClient) ListSites(ctx context.Context, params *network.ListSitesParams) (*network.SitesResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) ListSiteDevices(ctx context.Context, siteID network.SiteId, params *network.ListSiteDevicesParams) (*network.DevicesResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) GetDeviceByID(ctx context.Context, siteID network.SiteId, deviceID network.DeviceId) (*network.Device, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) ListSiteClients(ctx context.Context, siteID network.SiteId, params *network.ListSiteClientsParams) (*network.ClientsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) GetClientByID(ctx context.Context, siteID network.SiteId, clientID network.ClientId) (*network.NetworkClient, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) ListHotspotVouchers(ctx context.Context, siteID network.SiteId, params *network.ListHotspotVouchersParams) (*network.HotspotVouchersResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) CreateHotspotVouchers(ctx context.Context, siteID network.SiteId, request *network.CreateVouchersRequest) (*network.HotspotVouchersResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) GetHotspotVoucher(ctx context.Context, siteID network.SiteId, voucherID openapi_types.UUID) (*network.HotspotVoucher, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) DeleteHotspotVoucher(ctx context.Context, siteID network.SiteId, voucherID openapi_types.UUID) error {
	return fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) UpdateDNSRecord(ctx context.Context, site network.Site, recordID network.RecordId, record *network.DNSRecordInput) (*network.DNSRecord, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) ListFirewallPolicies(ctx context.Context, site network.Site) ([]network.FirewallPolicy, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) CreateFirewallPolicy(ctx context.Context, site network.Site, policy *network.FirewallPolicyInput) (*network.FirewallPolicy, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) UpdateFirewallPolicy(ctx context.Context, site network.Site, policyID network.PolicyId, policy *network.FirewallPolicyInput) (*network.FirewallPolicy, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) DeleteFirewallPolicy(ctx context.Context, site network.Site, policyID network.PolicyId) error {
	return fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) ListTrafficRules(ctx context.Context, site network.Site) ([]network.TrafficRule, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) CreateTrafficRule(ctx context.Context, site network.Site, rule *network.TrafficRuleInput) (*network.TrafficRule, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) UpdateTrafficRule(ctx context.Context, site network.Site, ruleID network.RuleId, rule *network.TrafficRuleInput) (*network.TrafficRule, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) DeleteTrafficRule(ctx context.Context, site network.Site, ruleID network.RuleId) error {
	return fmt.Errorf("not implemented")
}
func (m *MockNetworkClient) GetAggregatedDashboard(ctx context.Context, site network.Site, params *network.GetAggregatedDashboardParams) (*network.AggregatedDashboard, error) {
	return nil, fmt.Errorf("not implemented")
}

// Example application code that uses the Network API client

// DNSManager manages DNS records using the Network API.
type DNSManager struct {
	client network.NetworkAPIClient
}

// NewDNSManager creates a new DNS manager.
func NewDNSManager(client network.NetworkAPIClient) *DNSManager {
	return &DNSManager{client: client}
}

// GetRecordCount returns the number of DNS records for a site.
func (dm *DNSManager) GetRecordCount(ctx context.Context, site string) (int, error) {
	records, err := dm.client.ListDNSRecords(ctx, network.Site(site))
	if err != nil {
		return 0, fmt.Errorf("failed to list DNS records: %w", err)
	}
	return len(records), nil
}

// FindRecordByKey finds a DNS record by its key (domain name).
func (dm *DNSManager) FindRecordByKey(ctx context.Context, site, key string) (*network.DNSRecord, error) {
	records, err := dm.client.ListDNSRecords(ctx, network.Site(site))
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}

	for _, record := range records {
		if record.Key == key {
			return &record, nil
		}
	}

	return nil, fmt.Errorf("record not found: %s", key)
}

// Tests

func TestDNSManager_GetRecordCount_Success(t *testing.T) {
	mockClient := new(MockNetworkClient)

	// Setup expectations
	expectedRecords := []network.DNSRecord{
		{Key: "example.com", Value: "192.168.1.1"},
		{Key: "test.com", Value: "192.168.1.2"},
	}

	mockClient.On("ListDNSRecords", mock.Anything, network.Site("default")).
		Return(expectedRecords, nil)

	// Create manager and test
	manager := NewDNSManager(mockClient)
	count, err := manager.GetRecordCount(context.Background(), "default")

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
	mockClient.AssertExpectations(t)
}

func TestDNSManager_GetRecordCount_Error(t *testing.T) {
	mockClient := new(MockNetworkClient)

	// Setup expectations with error
	mockClient.On("ListDNSRecords", mock.Anything, network.Site("default")).
		Return(nil, fmt.Errorf("API error: connection timeout"))

	// Create manager and test
	manager := NewDNSManager(mockClient)
	count, err := manager.GetRecordCount(context.Background(), "default")

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "failed to list DNS records")
	mockClient.AssertExpectations(t)
}

func TestDNSManager_FindRecordByKey_Found(t *testing.T) {
	mockClient := new(MockNetworkClient)

	// Setup expectations
	expectedRecords := []network.DNSRecord{
		{Key: "example.com", Value: "192.168.1.1"},
		{Key: "test.com", Value: "192.168.1.2"},
	}

	mockClient.On("ListDNSRecords", mock.Anything, network.Site("default")).
		Return(expectedRecords, nil)

	// Create manager and test
	manager := NewDNSManager(mockClient)
	record, err := manager.FindRecordByKey(context.Background(), "default", "test.com")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, "test.com", record.Key)
	assert.Equal(t, "192.168.1.2", record.Value)
	mockClient.AssertExpectations(t)
}

func TestDNSManager_FindRecordByKey_NotFound(t *testing.T) {
	mockClient := new(MockNetworkClient)

	// Setup expectations
	expectedRecords := []network.DNSRecord{
		{Key: "example.com", Value: "192.168.1.1"},
	}

	mockClient.On("ListDNSRecords", mock.Anything, network.Site("default")).
		Return(expectedRecords, nil)

	// Create manager and test
	manager := NewDNSManager(mockClient)
	record, err := manager.FindRecordByKey(context.Background(), "default", "notfound.com")

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.Contains(t, err.Error(), "record not found")
	mockClient.AssertExpectations(t)
}
