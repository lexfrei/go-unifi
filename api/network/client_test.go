package network

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexfrei/go-unifi/api/network/testdata"
	"github.com/lexfrei/go-unifi/internal/testutil"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants.
const (
	testAPIKey       = "test-api-key"
	testSiteInternal = "default"
	testRecordID     = "6913a4964a990741124a6d94"
	testHostKey      = "testhost1.local"
	testHostValue    = "192.168.100.1"
	testModelUDR7    = "UDR7"
	testTypeWired    = "WIRED"
	testPolicyName   = "test-policy-1"
	testPolicyID     = "507f1f77bcf86cd799439011"
	testRuleName     = "test-rule-1"
	testRuleID       = "507f1f77bcf86cd799439012"
)

var testSiteID = types.UUID{0x88, 0xf7, 0xaf, 0x54, 0x98, 0xf8, 0x30, 0x6a, 0xa1, 0xc7, 0xc9, 0x34, 0x97, 0x22, 0xb1, 0xf6}

func TestNew(t *testing.T) {
	t.Parallel()

	client, err := New("https://test.local", testAPIKey)
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestNewWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *ClientConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ClientConfig{
				ControllerURL: "https://test.local",
				APIKey:        testAPIKey,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty API key",
			config: &ClientConfig{
				ControllerURL: "https://test.local",
				APIKey:        "",
			},
			wantErr: true,
		},
		{
			name: "empty controller URL",
			config: &ClientConfig{
				ControllerURL: "",
				APIKey:        testAPIKey,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewWithConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
		})
	}
}

func TestListSites(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *SitesResponse)
	}{
		{
			name:           "success",
			mockResponse:   testdata.LoadFixture(t, "sites/list_success.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *SitesResponse) {
				t.Helper()
				assert.Equal(t, 1, resp.Count)
				assert.Equal(t, 1, resp.TotalCount)
				assert.Len(t, resp.Data, 1)

				site := resp.Data[0]
				assert.Equal(t, testSiteID, site.Id)
				assert.Equal(t, testSiteInternal, site.InternalReference)
				assert.Equal(t, "Default", site.Name)
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "rate limit",
			mockResponse:   testdata.LoadFixture(t, "errors/rate_limit.json"),
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/integration/v1/sites"
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.ListSites(context.Background(), nil)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestListDNSRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp []DNSRecord)
	}{
		{
			name:           "success with records",
			mockResponse:   testdata.LoadFixture(t, "dns/list_success.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp []DNSRecord) {
				t.Helper()
				assert.Len(t, resp, 3)

				// Check first record
				assert.Equal(t, testHostKey, resp[0].Key)
				assert.Equal(t, testHostValue, resp[0].Value)
				assert.Equal(t, "A", string(resp[0].RecordType))
				assert.True(t, resp[0].Enabled)
			},
		},
		{
			name:           "empty list",
			mockResponse:   testdata.LoadFixture(t, "dns/empty_list.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp []DNSRecord) {
				t.Helper()
				assert.Empty(t, resp)
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/static-dns"
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.ListDNSRecords(context.Background(), testSiteInternal)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestCreateDNSRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *DNSRecord)
	}{
		{
			name:           "success",
			mockResponse:   testdata.LoadFixture(t, "dns/single_record.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *DNSRecord) {
				t.Helper()
				assert.Equal(t, testHostKey, resp.Key)
				assert.Equal(t, testHostValue, resp.Value)
			},
		},
		{
			name:           "bad request",
			mockResponse:   testdata.LoadFixture(t, "errors/bad_request.json"),
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/static-dns"
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			input := &DNSRecordInput{
				Key:        testHostKey,
				RecordType: DNSRecordInputRecordTypeA,
				Value:      testHostValue,
			}

			resp, err := client.CreateDNSRecord(context.Background(), testSiteInternal, input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestUpdateDNSRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *DNSRecord)
	}{
		{
			name:           "success",
			mockResponse:   testdata.LoadFixture(t, "dns/single_record.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *DNSRecord) {
				t.Helper()
				assert.Equal(t, testHostKey, resp.Key)
			},
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "bad request",
			mockResponse:   testdata.LoadFixture(t, "errors/bad_request.json"),
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/static-dns/" + testRecordID
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodPut, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			input := &DNSRecordInput{
				Key:        testHostKey,
				RecordType: DNSRecordInputRecordTypeA,
				Value:      testHostValue,
			}

			resp, err := client.UpdateDNSRecord(context.Background(), testSiteInternal, testRecordID, input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestDeleteDNSRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
		{
			name:           "success",
			mockResponse:   `{}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/static-dns/" + testRecordID
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodDelete, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			err = client.DeleteDNSRecord(context.Background(), testSiteInternal, testRecordID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestListSiteDevices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *DevicesResponse)
	}{
		{
			name:           "success with devices",
			mockResponse:   testdata.LoadFixture(t, "devices/list_success.json"),
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *DevicesResponse) {
				t.Helper()
				assert.Equal(t, 2, resp.Count)
				assert.Len(t, resp.Data, 2)
				assert.Equal(t, testModelUDR7, resp.Data[0].Model)
				assert.Equal(t, "USW Flex Mini", resp.Data[1].Model)
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/devices"
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.ListSiteDevices(context.Background(), testSiteID, nil)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestGetDeviceByID(t *testing.T) {
	t.Parallel()

	testDeviceID := types.UUID{0x62, 0x04, 0xb5, 0x87, 0x72, 0x15, 0x23, 0x5b, 0xd0, 0x68, 0xf9, 0x6c, 0xa1, 0x2e, 0xab, 0x52}

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *Device)
	}{
		{
			name:           "success with full device",
			mockResponse:   testdata.LoadFixture(t, "devices/single_device.json"),
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *Device) {
				t.Helper()
				assert.Equal(t, testModelUDR7, resp.Model)
				assert.Equal(t, "4.3.9", resp.FirmwareVersion)
				require.NotNil(t, resp.Interfaces.Ports)
				assert.Len(t, *resp.Interfaces.Ports, 2)
				require.NotNil(t, resp.Interfaces.Radios)
				assert.Len(t, *resp.Interfaces.Radios, 2)
			},
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/devices/" + testDeviceID.String()
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.GetDeviceByID(context.Background(), testSiteID, testDeviceID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestListSiteClients(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *ClientsResponse)
	}{
		{
			name:           "success with clients",
			mockResponse:   testdata.LoadFixture(t, "clients/list_success.json"),
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *ClientsResponse) {
				t.Helper()
				assert.Equal(t, 3, resp.Count)
				assert.Len(t, resp.Data, 3)
				assert.Equal(t, testTypeWired, string(resp.Data[0].Type))
				assert.Equal(t, "WIRELESS", string(resp.Data[1].Type))
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/clients"
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.ListSiteClients(context.Background(), testSiteID, nil)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestGetClientByID(t *testing.T) {
	t.Parallel()

	testClientID := types.UUID{0x7f, 0xe0, 0x38, 0xe8, 0x94, 0x6b, 0xfa, 0x53, 0x73, 0x35, 0x6c, 0x00, 0xbe, 0xe8, 0x46, 0x57}

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *NetworkClient)
	}{
		{
			name:           "success with client details",
			mockResponse:   testdata.LoadFixture(t, "clients/single_client.json"),
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *NetworkClient) {
				t.Helper()
				assert.Equal(t, testTypeWired, string(resp.Type))
				assert.Equal(t, "client-1", resp.Name)
				assert.Equal(t, "aa:bb:cc:14:01:56", resp.MacAddress)
			},
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/clients/" + testClientID.String()
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.GetClientByID(context.Background(), testSiteID, testClientID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestGetAggregatedDashboard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *AggregatedDashboard)
	}{
		{
			name:           "success with real data",
			mockResponse:   testdata.LoadFixture(t, "dashboard/aggregated.json"),
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *AggregatedDashboard) {
				t.Helper()
				require.NotNil(t, resp.DashboardMeta)
				require.NotNil(t, resp.DashboardMeta.Layout)
				assert.Equal(t, "wireless", *resp.DashboardMeta.Layout)
				require.NotNil(t, resp.WifiChannels)
				require.NotNil(t, resp.WifiChannels.RadioChannels)
				assert.Len(t, *resp.WifiChannels.RadioChannels, 2)
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/aggregated-dashboard"
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.GetAggregatedDashboard(context.Background(), testSiteInternal, nil)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestListFirewallPolicies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp []FirewallPolicy)
	}{
		{
			name:           "success with empty list",
			mockResponse:   testdata.LoadFixture(t, "firewall/empty_list.json"),
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp []FirewallPolicy) {
				t.Helper()
				assert.Empty(t, resp)
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/firewall-policies"
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.ListFirewallPolicies(context.Background(), testSiteInternal)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestCreateFirewallPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *FirewallPolicy)
	}{
		{
			name:           "success",
			mockResponse:   testdata.LoadFixture(t, "firewall/single_policy.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *FirewallPolicy) {
				t.Helper()
				assert.Equal(t, testPolicyName, resp.Name)
			},
		},
		{
			name:           "bad request",
			mockResponse:   testdata.LoadFixture(t, "errors/bad_request.json"),
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/firewall-policies"
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			input := &FirewallPolicyInput{
				Action:  FirewallPolicyInputActionALLOW,
				Enabled: true,
				Name:    testPolicyName,
			}

			resp, err := client.CreateFirewallPolicy(context.Background(), testSiteInternal, input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestUpdateFirewallPolicy(t *testing.T) {
	t.Parallel()

	policyID := testPolicyID

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *FirewallPolicy)
	}{
		{
			name:           "success",
			mockResponse:   testdata.LoadFixture(t, "firewall/single_policy.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *FirewallPolicy) {
				t.Helper()
				assert.Equal(t, testPolicyName, resp.Name)
			},
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/firewall-policies/" + policyID
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodPut, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			input := &FirewallPolicyInput{
				Action:  FirewallPolicyInputActionALLOW,
				Enabled: true,
				Name:    testPolicyName,
			}

			resp, err := client.UpdateFirewallPolicy(context.Background(), testSiteInternal, policyID, input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestDeleteFirewallPolicy(t *testing.T) {
	t.Parallel()

	policyID := testPolicyID

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
		{
			name:           "success",
			mockResponse:   `{}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/firewall-policies/" + policyID
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodDelete, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			err = client.DeleteFirewallPolicy(context.Background(), testSiteInternal, policyID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestListTrafficRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp []TrafficRule)
	}{
		{
			name:           "success with empty list",
			mockResponse:   testdata.LoadFixture(t, "traffic/empty_list.json"),
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp []TrafficRule) {
				t.Helper()
				assert.Empty(t, resp)
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/trafficrules"
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.ListTrafficRules(context.Background(), testSiteInternal)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestCreateTrafficRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *TrafficRule)
	}{
		{
			name:           "success",
			mockResponse:   testdata.LoadFixture(t, "traffic/single_rule.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *TrafficRule) {
				t.Helper()
				require.NotNil(t, resp.Description)
				assert.Equal(t, testRuleName, *resp.Description)
			},
		},
		{
			name:           "bad request",
			mockResponse:   testdata.LoadFixture(t, "errors/bad_request.json"),
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/trafficrules"
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			desc := testRuleName
			input := &TrafficRuleInput{
				Enabled:        true,
				MatchingTarget: TrafficRuleInputMatchingTargetINTERNET,
				Description:    &desc,
			}

			resp, err := client.CreateTrafficRule(context.Background(), testSiteInternal, input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestUpdateTrafficRule(t *testing.T) {
	t.Parallel()

	ruleID := testRuleID

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *TrafficRule)
	}{
		{
			name:           "success",
			mockResponse:   testdata.LoadFixture(t, "traffic/single_rule.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *TrafficRule) {
				t.Helper()
				require.NotNil(t, resp.Description)
				assert.Equal(t, testRuleName, *resp.Description)
			},
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/trafficrules/" + ruleID
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodPut, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			desc := testRuleName
			input := &TrafficRuleInput{
				Enabled:        true,
				MatchingTarget: TrafficRuleInputMatchingTargetINTERNET,
				Description:    &desc,
			}

			resp, err := client.UpdateTrafficRule(context.Background(), testSiteInternal, ruleID, input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestDeleteTrafficRule(t *testing.T) {
	t.Parallel()

	ruleID := testRuleID

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
		{
			name:           "success",
			mockResponse:   `{}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/trafficrules/" + ruleID
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodDelete, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			err = client.DeleteTrafficRule(context.Background(), testSiteInternal, ruleID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestListHotspotVouchers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *HotspotVouchersResponse)
	}{
		{
			name:           "success with vouchers",
			mockResponse:   testdata.LoadFixture(t, "hotspot/list_vouchers_success.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HotspotVouchersResponse) {
				t.Helper()
				assert.Equal(t, 2, resp.Count)
				assert.Len(t, resp.Data, 2)
			},
		},
		{
			name:           "success with empty list",
			mockResponse:   testdata.LoadFixture(t, "hotspot/empty_list.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HotspotVouchersResponse) {
				t.Helper()
				assert.Equal(t, 0, resp.Count)
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/hotspot/vouchers"
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.ListHotspotVouchers(context.Background(), testSiteID, nil)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestCreateHotspotVouchers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
		{
			name:           "success",
			mockResponse:   testdata.LoadFixture(t, "hotspot/list_vouchers_success.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "bad request",
			mockResponse:   testdata.LoadFixture(t, "errors/bad_request.json"),
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/hotspot/vouchers"
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			quota := 1
			input := &CreateVouchersRequest{
				Count: 1,
				Quota: &quota,
			}

			_, err = client.CreateHotspotVouchers(context.Background(), testSiteID, input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetHotspotVoucher(t *testing.T) {
	t.Parallel()

	testVoucherID := types.UUID{0x50, 0x7f, 0x1f, 0x77, 0xbc, 0xf8, 0x6c, 0xd7, 0x99, 0x43, 0x90, 0x13, 0x00, 0x00, 0x00, 0x00}

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *HotspotVoucher)
	}{
		{
			name:           "success",
			mockResponse:   testdata.LoadFixture(t, "hotspot/single_voucher.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HotspotVoucher) {
				t.Helper()
				assert.Equal(t, "12345-67890", resp.Code)
			},
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/hotspot/vouchers/" + testVoucherID.String()
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			resp, err := client.GetHotspotVoucher(context.Background(), testSiteID, testVoucherID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestDeleteHotspotVoucher(t *testing.T) {
	t.Parallel()

	testVoucherID := types.UUID{0x50, 0x7f, 0x1f, 0x77, 0xbc, 0xf8, 0x6c, 0xd7, 0x99, 0x43, 0x90, 0x13, 0x00, 0x00, 0x00, 0x00}

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
		{
			name:           "success",
			mockResponse:   `{}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "not found",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/hotspot/vouchers/" + testVoucherID.String()
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodDelete, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			require.NoError(t, err)

			err = client.DeleteHotspotVoucher(context.Background(), testSiteID, testVoucherID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
