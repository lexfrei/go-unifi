package network

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oapi-codegen/runtime/types"
)

// Real API responses from test controller.
const (
	// Real response from GET /v2/sites.
	listSitesSuccess = `{
  "count": 1,
  "data": [
    {
      "id": "88f7af54-98f8-306a-a1c7-c9349722b1f6",
      "internalReference": "default",
      "name": "Default"
    }
  ],
  "limit": 25,
  "offset": 0,
  "totalCount": 1
}`

	// Real response from GET /v2/site/default/static-dns.
	listDNSRecordsSuccess = `[
  {
    "_id": "6913a4964a990741124a6d94",
    "enabled": true,
    "key": "testhost1.local",
    "port": 0,
    "priority": 0,
    "record_type": "A",
    "ttl": 0,
    "value": "192.168.100.1",
    "weight": 0
  },
  {
    "_id": "6913a4964a990741124a6d97",
    "enabled": true,
    "key": "testhost2.local",
    "port": 0,
    "priority": 0,
    "record_type": "A",
    "ttl": 0,
    "value": "192.168.100.2",
    "weight": 0
  },
  {
    "_id": "6913a4964a990741124a6d98",
    "enabled": true,
    "key": "testhost3.local",
    "port": 0,
    "priority": 0,
    "record_type": "A",
    "ttl": 0,
    "value": "192.168.100.3",
    "weight": 0
  }
]`

	// Single DNS record for create/get tests.
	singleDNSRecord = `{
    "_id": "6913a4964a990741124a6d94",
    "enabled": true,
    "key": "testhost1.local",
    "port": 0,
    "priority": 0,
    "record_type": "A",
    "ttl": 0,
    "value": "192.168.100.1",
    "weight": 0
  }`

	// Real response from GET /v1/sites/{site}/devices with 2 devices.
	listDevicesSuccess = `{
  "count": 2,
  "data": [
    {
      "features": ["switching", "accessPoint"],
      "id": "6204b587-7215-235b-d068-f96ca12eab52",
      "interfaces": ["ports", "radios"],
      "ipAddress": "10.94.26.13",
      "macAddress": "aa:bb:cc:99:ea:6b",
      "model": "UDR7",
      "name": "Device-1",
      "state": "ONLINE"
    },
    {
      "features": ["switching"],
      "id": "0cd24618-8745-b626-b3c3-57692a02433e",
      "interfaces": ["ports"],
      "ipAddress": "10.166.169.226",
      "macAddress": "aa:bb:cc:6f:6d:73",
      "model": "USW Flex Mini",
      "name": "Device-2",
      "state": "ONLINE"
    }
  ],
  "limit": 25,
  "offset": 0,
  "totalCount": 2
}`

	// Real response from GET /v1/sites/{site}/devices/{id} with full device details.
	singleDeviceSuccess = `{
  "configurationId": "c212be130585ee93",
  "features": {
    "accessPoint": {},
    "switching": {}
  },
  "firmwareUpdatable": false,
  "firmwareVersion": "4.3.9",
  "id": "6204b587-7215-235b-d068-f96ca12eab52",
  "interfaces": {
    "ports": [
      {
        "connector": "RJ45",
        "idx": 1,
        "maxSpeedMbps": 2500,
        "poe": {
          "enabled": true,
          "standard": "802.3af",
          "state": "UP",
          "type": 1
        },
        "speedMbps": 1000,
        "state": "UP"
      },
      {
        "connector": "RJ45",
        "idx": 2,
        "maxSpeedMbps": 2500,
        "speedMbps": 2500,
        "state": "UP"
      }
    ],
    "radios": [
      {
        "channel": 6,
        "channelWidthMHz": 20,
        "frequencyGHz": 2.4,
        "wlanStandard": "802.11be"
      },
      {
        "channel": 40,
        "channelWidthMHz": 80,
        "frequencyGHz": 5,
        "wlanStandard": "802.11be"
      }
    ]
  },
  "ipAddress": "10.94.26.13",
  "macAddress": "aa:bb:cc:99:ea:6b",
  "model": "UDR7",
  "name": "Device-1",
  "provisionedAt": "2025-11-03T21:41:04Z",
  "state": "ONLINE",
  "supported": true
}`

	// Real response from GET /v1/sites/{site}/clients with 3 clients (truncated from 15).
	listClientsSuccess = `{
  "count": 3,
  "data": [
    {
      "access": {"type": "DEFAULT"},
      "connectedAt": "2025-10-19T10:09:31Z",
      "id": "7fe038e8-946b-fa53-7335-6c00bee84657",
      "ipAddress": "10.222.189.242",
      "macAddress": "aa:bb:cc:14:01:56",
      "name": "client-1",
      "type": "WIRED",
      "uplinkDeviceId": "6204b587-7215-235b-d068-f96ca12eab52"
    },
    {
      "access": {"type": "DEFAULT"},
      "connectedAt": "2025-10-19T10:10:24Z",
      "id": "17f9729f-a6d9-63da-7185-579a4bd70979",
      "ipAddress": "10.103.206.70",
      "macAddress": "aa:bb:cc:9c:58:6f",
      "name": "client-2",
      "type": "WIRELESS",
      "uplinkDeviceId": "6204b587-7215-235b-d068-f96ca12eab52"
    },
    {
      "access": {"type": "DEFAULT"},
      "connectedAt": "2025-10-24T21:15:12Z",
      "id": "d0fde4ea-6ed0-a42b-ae3c-e848132e56b4",
      "ipAddress": "10.157.45.243",
      "macAddress": "aa:bb:cc:10:a8:87",
      "name": "client-3",
      "type": "WIRELESS",
      "uplinkDeviceId": "6204b587-7215-235b-d068-f96ca12eab52"
    }
  ],
  "limit": 25,
  "offset": 0,
  "totalCount": 3
}`

	// Real response from GET /v1/sites/{site}/clients/{id} with single client.
	singleClientSuccess = `{
  "access": {"type": "DEFAULT"},
  "connectedAt": "2025-10-19T10:09:31Z",
  "id": "7fe038e8-946b-fa53-7335-6c00bee84657",
  "ipAddress": "10.222.189.242",
  "macAddress": "aa:bb:cc:14:01:56",
  "name": "client-1",
  "type": "WIRED",
  "uplinkDeviceId": "6204b587-7215-235b-d068-f96ca12eab52"
}`

	// Real response from GET /v2/api/site/{site}/dashboard.
	dashboardSuccess = `{
  "dashboard_meta": {
    "end_timestamp": 1762895866647,
    "layout": "wireless",
    "start_timestamp": 1762812000000,
    "widgets": [
      "most_active_clients",
      "most_active_aps",
      "wifi_activity",
      "wifi_channels",
      "wifi_client_experience",
      "wifi_tx_retries",
      "admin_activity",
      "device_client_count",
      "server_ip"
    ]
  },
  "most_active_aps": {
    "total_bytes": 0
  },
  "most_active_clients": {
    "total_bytes": 0
  },
  "wifi_channels": {
    "radio_channels": [
      {
        "available_channels": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11],
        "radio": "ng"
      },
      {
        "available_channels": [36, 40, 44, 48, 52, 56, 60, 64, 100, 104, 108, 112, 116, 120, 124, 128, 132, 136, 140, 144, 149, 153, 157, 161, 165],
        "radio": "na"
      }
    ]
  }
}`

	// Mock firewall policy for CREATE/UPDATE tests.
	singleFirewallPolicy = `{
    "_id": "507f1f77bcf86cd799439011",
    "action": "ALLOW",
    "enabled": true,
    "name": "test-policy-1"
  }`

	// Mock traffic rule for CREATE/UPDATE tests.
	singleTrafficRule = `{
    "_id": "507f1f77bcf86cd799439012",
    "enabled": true,
    "description": "test-rule-1",
    "matching_target": "INTERNET"
  }`

	// Mock hotspot vouchers response.
	listVouchersSuccess = `{
    "count": 2,
    "data": [
      {
        "code": "12345-67890",
        "id": "507f1f77bcf86cd799439013",
        "status": "VALID"
      },
      {
        "code": "98765-43210",
        "id": "507f1f77bcf86cd799439014",
        "status": "VALID"
      }
    ],
    "limit": 100,
    "offset": 0,
    "totalCount": 2
  }`

	singleVoucher = `{
    "code": "12345-67890",
    "id": "507f1f77bcf86cd799439013",
    "status": "VALID"
  }`

	// Empty responses.
	emptyDNSRecords       = `[]`
	emptyFirewallPolicies = `[]`
	emptyTrafficRules     = `[]`
	emptyVouchers         = `{"count": 0, "data": [], "limit": 100, "offset": 0, "totalCount": 0}`

	// Common error responses.
	unauthorizedError = `{"error": "unauthorized", "message": "Invalid API key"}`
	notFoundError     = `{"error": "not_found", "message": "Resource not found"}`
	badRequestError   = `{"error": "bad_request", "message": "Invalid request"}`
	rateLimitError    = `{"error": "rate_limit", "message": "Rate limit exceeded"}`
	serverError       = `{"error": "server_error", "message": "Internal server error"}`

	// Test constants.
	testAPIKey       = "test-api-key"
	testSiteInternal = "default"
	testRecordID     = "6913a4964a990741124a6d94"
	testHostKey      = "testhost1.local"
	testHostValue    = "192.168.100.1"
	testModelUDR7    = "UDR7"
	testTypeWired    = "WIRED"
)

var testSiteID = types.UUID{0x88, 0xf7, 0xaf, 0x54, 0x98, 0xf8, 0x30, 0x6a, 0xa1, 0xc7, 0xc9, 0x34, 0x97, 0x22, 0xb1, 0xf6}

func TestNew(t *testing.T) {
	t.Parallel()

	client, err := New("https://test.local", testAPIKey)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if client == nil {
		t.Fatal("New() returned nil client")
	}
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
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("Expected client, got nil")
			}
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
			mockResponse:   listSitesSuccess,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *SitesResponse) {
				t.Helper()
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.Count != 1 {
					t.Errorf("Count = %d, want 1", resp.Count)
				}
				if resp.TotalCount != 1 {
					t.Errorf("TotalCount = %d, want 1", resp.TotalCount)
				}
				if len(resp.Data) != 1 {
					t.Fatalf("len(Data) = %d, want 1", len(resp.Data))
				}
				site := resp.Data[0]
				if site.Id != testSiteID {
					t.Errorf("Site ID = %s, want %s", site.Id, testSiteID)
				}
				if site.InternalReference != testSiteInternal {
					t.Errorf("Site InternalReference = %s, want %s", site.InternalReference, testSiteInternal)
				}
				if site.Name != "Default" {
					t.Errorf("Site Name = %s, want Default", site.Name)
				}
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "rate limit",
			mockResponse:   rateLimitError,
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   serverError,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/proxy/network/integration/v1/sites" {
					t.Errorf("Request path = %s, want /proxy/network/integration/v1/sites", r.URL.Path)
				}
				if r.Header.Get("X-Api-Key") != testAPIKey {
					t.Error("X-Api-Key header not set")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.ListSites(context.Background(), nil)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

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
			mockResponse:   listDNSRecordsSuccess,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp []DNSRecord) {
				t.Helper()
				if len(resp) != 3 {
					t.Fatalf("len(resp) = %d, want 3", len(resp))
				}
				// Check first record
				if resp[0].Key != testHostKey {
					t.Errorf("Key = %s, want %s", resp[0].Key, testHostKey)
				}
				if resp[0].Value != testHostValue {
					t.Errorf("Value = %s, want %s", resp[0].Value, testHostValue)
				}
				if resp[0].RecordType != "A" {
					t.Errorf("RecordType = %s, want A", resp[0].RecordType)
				}
				if !resp[0].Enabled {
					t.Error("Enabled = false, want true")
				}
			},
		},
		{
			name:           "empty list",
			mockResponse:   emptyDNSRecords,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp []DNSRecord) {
				t.Helper()
				if len(resp) != 0 {
					t.Errorf("len(resp) = %d, want 0", len(resp))
				}
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   serverError,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/static-dns"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Header.Get("X-Api-Key") != testAPIKey {
					t.Error("X-Api-Key header not set")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.ListDNSRecords(context.Background(), testSiteInternal)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

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
			mockResponse:   singleDNSRecord,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *DNSRecord) {
				t.Helper()
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.Key != testHostKey {
					t.Errorf("Key = %s, want %s", resp.Key, testHostKey)
				}
				if resp.Value != testHostValue {
					t.Errorf("Value = %s, want %s", resp.Value, testHostValue)
				}
			},
		},
		{
			name:           "bad request",
			mockResponse:   badRequestError,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/static-dns"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodPost {
					t.Errorf("Method = %s, want POST", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			input := &DNSRecordInput{
				Key:        testHostKey,
				RecordType: DNSRecordInputRecordTypeA,
				Value:      testHostValue,
			}

			resp, err := client.CreateDNSRecord(context.Background(), testSiteInternal, input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestGetDNSRecordByID(t *testing.T) {
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
			mockResponse:   singleDNSRecord,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *DNSRecord) {
				t.Helper()
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.UnderscoreId != testRecordID {
					t.Errorf("_id = %s, want %s", resp.UnderscoreId, testRecordID)
				}
			},
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/static-dns/" + testRecordID
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.GetDNSRecordByID(context.Background(), testSiteInternal, testRecordID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

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
			mockResponse:   singleDNSRecord,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *DNSRecord) {
				t.Helper()
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.Key != testHostKey {
					t.Errorf("Key = %s, want %s", resp.Key, testHostKey)
				}
			},
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "bad request",
			mockResponse:   badRequestError,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/static-dns/" + testRecordID
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodPut {
					t.Errorf("Method = %s, want PUT", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			input := &DNSRecordInput{
				Key:        testHostKey,
				RecordType: DNSRecordInputRecordTypeA,
				Value:      testHostValue,
			}

			resp, err := client.UpdateDNSRecord(context.Background(), testSiteInternal, testRecordID, input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

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
			mockStatusCode: http.StatusNoContent,
			wantErr:        false,
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/static-dns/" + testRecordID
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodDelete {
					t.Errorf("Method = %s, want DELETE", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			err = client.DeleteDNSRecord(context.Background(), testSiteInternal, testRecordID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
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
			mockResponse:   listDevicesSuccess,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *DevicesResponse) {
				t.Helper()
				if resp.Count != 2 {
					t.Errorf("Count = %d, want 2", resp.Count)
				}
				if len(resp.Data) != 2 {
					t.Errorf("len(Data) = %d, want 2", len(resp.Data))
				}
				if resp.Data[0].Model != testModelUDR7 {
					t.Errorf("Model = %v, want %s", resp.Data[0].Model, testModelUDR7)
				}
				if resp.Data[1].Model != "USW Flex Mini" {
					t.Errorf("Model = %v, want USW Flex Mini", resp.Data[1].Model)
				}
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   serverError,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/devices"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.ListSiteDevices(context.Background(), testSiteID, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

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
			mockResponse:   singleDeviceSuccess,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *Device) {
				t.Helper()
				if resp.Model != testModelUDR7 {
					t.Errorf("Model = %v, want %s", resp.Model, testModelUDR7)
				}
				if resp.FirmwareVersion != "4.3.9" {
					t.Errorf("FirmwareVersion = %v, want 4.3.9", resp.FirmwareVersion)
				}
				if resp.Interfaces.Ports == nil || len(*resp.Interfaces.Ports) != 2 {
					t.Errorf("len(Ports) = %d, want 2", len(*resp.Interfaces.Ports))
				}
				if resp.Interfaces.Radios == nil || len(*resp.Interfaces.Radios) != 2 {
					t.Errorf("len(Radios) = %d, want 2", len(*resp.Interfaces.Radios))
				}
			},
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/devices/" + testDeviceID.String()
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.GetDeviceByID(context.Background(), testSiteID, testDeviceID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

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
			mockResponse:   listClientsSuccess,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *ClientsResponse) {
				t.Helper()
				if resp.Count != 3 {
					t.Errorf("Count = %d, want 3", resp.Count)
				}
				if len(resp.Data) != 3 {
					t.Errorf("len(Data) = %d, want 3", len(resp.Data))
				}
				if string(resp.Data[0].Type) != testTypeWired {
					t.Errorf("Type = %v, want %s", resp.Data[0].Type, testTypeWired)
				}
				if string(resp.Data[1].Type) != "WIRELESS" {
					t.Errorf("Type = %v, want WIRELESS", resp.Data[1].Type)
				}
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   serverError,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/clients"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.ListSiteClients(context.Background(), testSiteID, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

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
			mockResponse:   singleClientSuccess,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *NetworkClient) {
				t.Helper()
				if string(resp.Type) != testTypeWired {
					t.Errorf("Type = %v, want %s", resp.Type, testTypeWired)
				}
				if resp.Name != "client-1" {
					t.Errorf("Name = %v, want client-1", resp.Name)
				}
				if resp.MacAddress != "aa:bb:cc:14:01:56" {
					t.Errorf("MacAddress = %v, want aa:bb:cc:14:01:56", resp.MacAddress)
				}
			},
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/clients/" + testClientID.String()
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.GetClientByID(context.Background(), testSiteID, testClientID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

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
			mockResponse:   dashboardSuccess,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *AggregatedDashboard) {
				t.Helper()
				if resp.DashboardMeta == nil {
					t.Fatal("DashboardMeta is nil")
				}
				if resp.DashboardMeta.Layout == nil || *resp.DashboardMeta.Layout != "wireless" {
					t.Errorf("Layout = %v, want wireless", resp.DashboardMeta.Layout)
				}
				if resp.WifiChannels == nil {
					t.Fatal("WifiChannels is nil")
				}
				if resp.WifiChannels.RadioChannels == nil || len(*resp.WifiChannels.RadioChannels) != 2 {
					t.Errorf("len(RadioChannels) = %d, want 2", len(*resp.WifiChannels.RadioChannels))
				}
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/aggregated-dashboard"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.GetAggregatedDashboard(context.Background(), testSiteInternal, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

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
			mockResponse:   emptyFirewallPolicies,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp []FirewallPolicy) {
				t.Helper()
				if len(resp) != 0 {
					t.Errorf("len(resp) = %d, want 0", len(resp))
				}
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   serverError,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/firewall-policies"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.ListFirewallPolicies(context.Background(), testSiteInternal)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

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
			mockResponse:   singleFirewallPolicy,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *FirewallPolicy) {
				t.Helper()
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.Name != "test-policy-1" {
					t.Errorf("Name = %s, want test-policy-1", resp.Name)
				}
			},
		},
		{
			name:           "bad request",
			mockResponse:   badRequestError,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/firewall-policies"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodPost {
					t.Errorf("Method = %s, want POST", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			input := &FirewallPolicyInput{
				Action:  FirewallPolicyInputActionALLOW,
				Enabled: true,
				Name:    "test-policy-1",
			}

			resp, err := client.CreateFirewallPolicy(context.Background(), testSiteInternal, input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestUpdateFirewallPolicy(t *testing.T) {
	t.Parallel()

	policyID := "507f1f77bcf86cd799439011"

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *FirewallPolicy)
	}{
		{
			name:           "success",
			mockResponse:   singleFirewallPolicy,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *FirewallPolicy) {
				t.Helper()
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.Name != "test-policy-1" {
					t.Errorf("Name = %s, want test-policy-1", resp.Name)
				}
			},
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/firewall-policies/" + policyID
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodPut {
					t.Errorf("Method = %s, want PUT", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			input := &FirewallPolicyInput{
				Action:  FirewallPolicyInputActionALLOW,
				Enabled: true,
				Name:    "test-policy-1",
			}

			resp, err := client.UpdateFirewallPolicy(context.Background(), testSiteInternal, policyID, input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestDeleteFirewallPolicy(t *testing.T) {
	t.Parallel()

	policyID := "507f1f77bcf86cd799439011"

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
		{
			name:           "success",
			mockResponse:   `{}`,
			mockStatusCode: http.StatusNoContent,
			wantErr:        false,
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/firewall-policies/" + policyID
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodDelete {
					t.Errorf("Method = %s, want DELETE", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			err = client.DeleteFirewallPolicy(context.Background(), testSiteInternal, policyID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
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
			mockResponse:   emptyTrafficRules,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp []TrafficRule) {
				t.Helper()
				if len(resp) != 0 {
					t.Errorf("len(resp) = %d, want 0", len(resp))
				}
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "server error",
			mockResponse:   serverError,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/trafficrules"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.ListTrafficRules(context.Background(), testSiteInternal)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

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
			mockResponse:   singleTrafficRule,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *TrafficRule) {
				t.Helper()
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.Description == nil || *resp.Description != "test-rule-1" {
					t.Errorf("Description = %v, want test-rule-1", resp.Description)
				}
			},
		},
		{
			name:           "bad request",
			mockResponse:   badRequestError,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/trafficrules"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodPost {
					t.Errorf("Method = %s, want POST", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			desc := "test-rule-1"
			input := &TrafficRuleInput{
				Enabled:        true,
				MatchingTarget: TrafficRuleInputMatchingTargetINTERNET,
				Description:    &desc,
			}

			resp, err := client.CreateTrafficRule(context.Background(), testSiteInternal, input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestUpdateTrafficRule(t *testing.T) {
	t.Parallel()

	ruleID := "507f1f77bcf86cd799439012"

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *TrafficRule)
	}{
		{
			name:           "success",
			mockResponse:   singleTrafficRule,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *TrafficRule) {
				t.Helper()
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.Description == nil || *resp.Description != "test-rule-1" {
					t.Errorf("Description = %v, want test-rule-1", resp.Description)
				}
			},
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/trafficrules/" + ruleID
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodPut {
					t.Errorf("Method = %s, want PUT", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			desc := "test-rule-1"
			input := &TrafficRuleInput{
				Enabled:        true,
				MatchingTarget: TrafficRuleInputMatchingTargetINTERNET,
				Description:    &desc,
			}

			resp, err := client.UpdateTrafficRule(context.Background(), testSiteInternal, ruleID, input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestDeleteTrafficRule(t *testing.T) {
	t.Parallel()

	ruleID := "507f1f77bcf86cd799439012"

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
		{
			name:           "success",
			mockResponse:   `{}`,
			mockStatusCode: http.StatusNoContent,
			wantErr:        false,
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/v2/api/site/" + testSiteInternal + "/trafficrules/" + ruleID
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodDelete {
					t.Errorf("Method = %s, want DELETE", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			err = client.DeleteTrafficRule(context.Background(), testSiteInternal, ruleID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
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
			mockResponse:   listVouchersSuccess,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HotspotVouchersResponse) {
				t.Helper()
				if resp.Count != 2 {
					t.Errorf("Count = %d, want 2", resp.Count)
				}
				if len(resp.Data) != 2 {
					t.Errorf("len(Data) = %d, want 2", len(resp.Data))
				}
			},
		},
		{
			name:           "success with empty list",
			mockResponse:   emptyVouchers,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HotspotVouchersResponse) {
				t.Helper()
				if resp.Count != 0 {
					t.Errorf("Count = %d, want 0", resp.Count)
				}
			},
		},
		{
			name:           "unauthorized",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/hotspot/vouchers"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.ListHotspotVouchers(context.Background(), testSiteID, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

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
			mockResponse:   listVouchersSuccess,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "bad request",
			mockResponse:   badRequestError,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/hotspot/vouchers"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodPost {
					t.Errorf("Method = %s, want POST", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			quota := 1
			input := &CreateVouchersRequest{
				Count: 1,
				Quota: &quota,
			}

			_, err = client.CreateHotspotVouchers(context.Background(), testSiteID, input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
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
			mockResponse:   singleVoucher,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HotspotVoucher) {
				t.Helper()
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.Code != "12345-67890" {
					t.Errorf("Code = %v, want 12345-67890", resp.Code)
				}
			},
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/hotspot/vouchers/" + testVoucherID.String()
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			resp, err := client.GetHotspotVoucher(context.Background(), testSiteID, testVoucherID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

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
			mockStatusCode: http.StatusNoContent,
			wantErr:        false,
		},
		{
			name:           "not found",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/proxy/network/integration/v1/sites/" + testSiteID.String() + "/hotspot/vouchers/" + testVoucherID.String()
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != http.MethodDelete {
					t.Errorf("Method = %s, want DELETE", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := New(server.URL, testAPIKey)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			err = client.DeleteHotspotVoucher(context.Background(), testSiteID, testVoucherID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
