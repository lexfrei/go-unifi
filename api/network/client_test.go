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

	// Real response from GET /v1/sites/{site}/devices (empty).
	listDevicesEmpty = `{
  "count": 0,
  "data": [],
  "limit": 25,
  "offset": 0,
  "totalCount": 0
}`

	// Real response from GET /v1/sites/{site}/clients (empty).
	listClientsEmpty = `{
  "count": 0,
  "data": [],
  "limit": 25,
  "offset": 0,
  "totalCount": 0
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

	// Empty responses.
	emptyDNSRecords       = `[]`
	emptyFirewallPolicies = `[]`
	emptyTrafficRules     = `[]`

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
			name:           "success with empty list",
			mockResponse:   listDevicesEmpty,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *DevicesResponse) {
				t.Helper()
				if resp.Count != 0 {
					t.Errorf("Count = %d, want 0", resp.Count)
				}
				if len(resp.Data) != 0 {
					t.Errorf("len(Data) = %d, want 0", len(resp.Data))
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

	testDeviceID := types.UUID{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00}

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
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

			_, err = client.GetDeviceByID(context.Background(), testSiteID, testDeviceID)

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
			name:           "success with empty list",
			mockResponse:   listClientsEmpty,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *ClientsResponse) {
				t.Helper()
				if resp.Count != 0 {
					t.Errorf("Count = %d, want 0", resp.Count)
				}
				if len(resp.Data) != 0 {
					t.Errorf("len(Data) = %d, want 0", len(resp.Data))
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

	testClientID := types.UUID{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99}

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
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

			_, err = client.GetClientByID(context.Background(), testSiteID, testClientID)

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
