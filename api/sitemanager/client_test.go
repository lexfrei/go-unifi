package sitemanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Mock responses based on actual UniFi API responses
const (
	// Original examples from user (ucore)
	listHostsSuccess = `{
  "data": [
    {
      "id": "70A7419783ED0000000006797F060000000006C719490000000062ABD4EA:1261206302",
      "hardwareId": "e5bf13cd-98a7-5a96-9463-0d65d78cd3a4",
      "type": "ucore",
      "ipAddress": "203.0.113.1",
      "owner": true,
      "isBlocked": false,
      "lastConnectionStateChange": "2024-04-16T02:52:54.193Z",
      "userData": {
        "apps": ["users"],
        "consoleGroupMembers": [
          {
            "mac": "AABBCCDDEEFF",
            "role": "UNADOPTED",
            "roleAttributes": {
              "applications": {
                "network": {
                  "owned": false,
                  "required": true,
                  "supported": true
                }
              },
              "candidateRoles": ["PRIMARY"],
              "connectedState": "CONNECTED",
              "connectedStateLastChanged": "2024-04-16T02:52:54.193Z"
            },
            "sysId": 59925
          }
        ],
        "controllers": ["network", "protect", "access"],
        "email": "example@ui.com",
        "fullName": "UniFi User",
        "localId": "d4eb483d-98a7-438b-abe1-f46628dff73f",
        "role": "owner",
        "status": "ACTIVE"
      },
      "reportedState": null
    }
  ],
  "httpStatusCode": 200,
  "traceId": "a7dc15e0eb4527142d7823515b15f87d",
  "nextToken": "ba8e384e-3308-4236-b344-7357657351ca"
}`

	// Example from test-reality (console type)
	listHostsSuccessConsole = `{
  "data": [
    {
      "id": "942A6FCE26520000000008A62C8000000000091C92E70000000067801E31:392959371",
      "hardwareId": "0d5e371f-5c89-50ab-a732-f2d68bc2acd0",
      "type": "console",
      "ipAddress": "203.0.113.2",
      "owner": true,
      "isBlocked": false,
      "lastConnectionStateChange": "2024-04-10T21:08:21.567Z",
      "registrationTime": "2024-04-10T21:11:53Z",
      "latestBackupTime": "2024-11-05T18:40:36Z",
      "userData": {
        "apps": ["users"],
        "email": "user@example.com",
        "fullName": "Example User",
        "role": "owner",
        "status": "ACTIVE"
      },
      "reportedState": {
        "hostname": "example-console",
        "hardware": {
          "shortname": "UDR7",
          "firmwareVersion": "4.3.9"
        },
        "version": "4.3.87"
      }
    }
  ],
  "httpStatusCode": 200,
  "traceId": "a7dc15e0eb4527142d7823515b15f87d",
  "nextToken": ""
}`

	getHostUcoreSuccess = `{
  "data": {
    "id": "70A7419783ED0000000006797F060000000006C719490000000062ABD4EA:1261206302",
    "hardwareId": "e5bf13cd-98a7-5a96-9463-0d65d78cd3a4",
    "type": "ucore",
    "ipAddress": "220.130.137.169",
    "owner": true,
    "isBlocked": false,
    "lastConnectionStateChange": "2024-04-16T02:52:54.193Z",
    "userData": {
      "apps": ["users"],
      "email": "example@ui.com",
      "fullName": "UniFi User",
      "role": "owner",
      "status": "ACTIVE"
    },
    "reportedState": null
  },
  "httpStatusCode": 200,
  "traceId": "a7dc15e0eb4527142d7823515b15f87d"
}`

	getHostNetworkServerSuccess = `{
  "data": {
    "id": "1d9cf3ee-0c0f-466e-933c-9af829f09b50",
    "hardwareId": "b000b21e-0000-1111-add9-c1eed3897602",
    "type": "network-server",
    "ipAddress": "192.168.1.124",
    "owner": true,
    "isBlocked": false,
    "registrationTime": "2024-02-07T10:25:21.981Z",
    "lastConnectionStateChange": "2024-06-10T07:52:23.382Z",
    "userData": null,
    "reportedState": {
      "controller_uuid": "b000b21e-0000-1111-add9-c1eed3897602",
      "deviceId": "1d9cf3ee-0c0f-466e-933c-9af829f09b50",
      "hardware_id": "b000b21e-0000-1111-add9-c1eed3897602",
      "host_type": 0,
      "hostname": "example-domain.ui.com",
      "inform_port": 8080,
      "ipAddrs": ["192.168.1.124"],
      "mgmt_port": 8443,
      "name": "Self-Hosted Site",
      "override_inform_host": false,
      "release_channel": "release",
      "state": "connected",
      "version": "8.3.11"
    }
  },
  "httpStatusCode": 200,
  "traceId": "a7dc15e0eb4527142d7823515b15f87d"
}`

	unauthorizedError = `{
  "code": "unauthorized",
  "httpStatusCode": 401,
  "message": "unauthorized",
  "traceId": "a7dc15e0eb4527142d7823515b15f87d"
}`

	notFoundError = `{
  "code": "NOT_FOUND",
  "httpStatusCode": 404,
  "message": "thing not found: 942A6F00301100000000074A6BA90000000007A3387E0000000063EC9853:714694",
  "traceId": "a7dc15e0eb4527142d7823515b15f87d"
}`

	rateLimitError = `{
  "code": "rate_limit",
  "httpStatusCode": 429,
  "message": "rate limit exceeded, retry after 5.372786998s",
  "traceId": "a7dc15e0eb4527142d7823515b15f87d"
}`

	serverError = `{
  "code": "server_error",
  "httpStatusCode": 500,
  "message": "failed to list hosts",
  "traceId": "a7dc15e0eb4527142d7823515b15f87d"
}`

	badGatewayError = `{
  "code": "bad_gateway",
  "httpStatusCode": 502,
  "message": "bad gateway",
  "traceId": "a7dc15e0eb4527142d7823515b15f87d"
}`

	parameterInvalidError = `{
  "code": "parameter_invalid",
  "httpStatusCode": 400,
  "message": "invalid time format: 2024-04-24",
  "traceId": "a7dc15e0eb4527142d7823515b15f87d"
}`

	listDevicesSuccess = `{
  "data": [
    {
      "hostId": "900A6F00301100000000074A6BA90000000007A3387E0000000063EC9853:123456789",
      "hostName": "unifi.example.com",
      "devices": [
        {
          "id": "AABBCCDDEEFF",
          "mac": "AABBCCDDEEFF",
          "name": "USW Flex Mini",
          "model": "USW Flex Mini",
          "shortname": "USMINI",
          "ip": "172.16.10.34",
          "productLine": "network",
          "status": "online",
          "version": "2.1.6",
          "firmwareStatus": "upToDate",
          "updateAvailable": null,
          "isConsole": false,
          "isManaged": true,
          "startupTime": "2024-06-19T13:41:43Z",
          "adoptionTime": "2024-04-22T19:02:31Z",
          "note": null,
          "uidb": {
            "guid": null,
            "iconId": "7a060fbb-8a8b-471c-9f57-da477ff3e04e",
            "id": "b13610ef-7e73-4a14-985a-53bb75a62401",
            "images": {
              "default": "e3fbf220c145f4301929576039d52d19",
              "nopadding": "4608b98cfbc13adfb538490b117db81a",
              "topology": "96c7cd1cb0ba8f00200d7a771136d05f"
            }
          }
        },
        {
          "id": "112233445566",
          "mac": "112233445566",
          "name": "UniFi Console",
          "model": "UDR7",
          "shortname": "UDR7",
          "ip": "192.168.1.1",
          "productLine": "network",
          "status": "online",
          "version": "4.3.9",
          "firmwareStatus": "upToDate",
          "updateAvailable": null,
          "isConsole": true,
          "isManaged": true,
          "startupTime": "2024-10-19T10:07:38Z",
          "adoptionTime": "2024-08-22T18:34:28Z",
          "note": null,
          "uidb": {
            "guid": "54167124-c67e-1b94-7b83-d6cf5607422e",
            "iconId": "de18d135-06a1-5654-f233-e8c843840c28",
            "id": "e06fdc4f-c73f-2124-775b-6ab68813fae6",
            "images": {
              "default": "3288444f002474b27c076cfc801d3d0c",
              "nopadding": "88cb62422549d0014c077c510ae98b19",
              "topology": "a11d0f6ce02119c83f9cca6e2fe361aa"
            }
          }
        }
      ],
      "updatedAt": "2025-06-17T02:45:58Z"
    }
  ],
  "httpStatusCode": 200,
  "traceId": "a7dc15e0eb4527142d7823515b15f87d",
  "nextToken": "ba8e384e-3308-4236-b344-7357657351ca"
}`

	getISPMetricsSuccess = `{
  "data": [
    {
      "metricType": "5m",
      "periods": [
        {
          "data": {
            "wan": {
              "avgLatency": 14,
              "download_kbps": 1000000,
              "downtime": 0,
              "ispAsn": "12345",
              "ispName": "Example ISP",
              "maxLatency": 15,
              "packetLoss": 0,
              "upload_kbps": 1000000,
              "uptime": 100
            }
          },
          "metricTime": "2024-11-10T18:55:00Z",
          "version": "9.4.19"
        }
      ]
    }
  ],
  "httpStatusCode": 200,
  "traceId": "a7dc15e0eb4527142d7823515b15f87d"
}`

	listSitesSuccess = `{
  "data": [
    {
      "siteId": "661de833b6b2463f0c20b319",
      "hostId": "900A6F00301100000000074A6BA90000000007A3387E0000000063EC9853:123456789",
      "meta": {
        "desc": "Default",
        "gatewayMac": "aa:bb:cc:dd:ee:ff",
        "name": "default",
        "timezone": "Asia/Taipei"
      },
      "statistics": {
        "counts": {
          "criticalNotification": 0,
          "gatewayDevice": 1,
          "guestClient": 0,
          "lanConfiguration": 1,
          "offlineDevice": 0,
          "offlineGatewayDevice": 0,
          "offlineWifiDevice": 0,
          "offlineWiredDevice": 0,
          "pendingUpdateDevice": 0,
          "totalDevice": 1,
          "wanConfiguration": 2,
          "wifiClient": 0,
          "wifiConfiguration": 0,
          "wifiDevice": 0,
          "wiredClient": 0,
          "wiredDevice": 0
        },
        "gateway": {
          "hardwareId": "e5bf13cd-98a7-5a96-9463-0d65d78cd3a4",
          "inspectionState": "off",
          "ipsMode": "disabled",
          "ipsSignature": {
            "rulesCount": 53031,
            "type": "ET"
          },
          "shortname": "UDMPRO"
        },
        "internetIssues": [],
        "ispInfo": {
          "name": "Chunghwa Telecom",
          "organization": "Data Communication Business Group"
        },
        "percentages": {
          "wanUptime": 100
        }
      },
      "permission": "admin",
      "isOwner": true
    }
  ],
  "httpStatusCode": 200,
  "traceId": "a7dc15e0eb4527142d7823515b15f87d",
  "nextToken": "ba8e384e-3308-4236-b344-7357657351ca"
}`
)

func TestNew(t *testing.T) {
	t.Parallel()

	client, err := New("test-api-key")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if client == nil {
		t.Fatal("New() returned nil client")
	}

	if client.client == nil {
		t.Error("client.client is nil")
	}

	if client.httpClient == nil {
		t.Error("client.httpClient is nil")
	}

	if client.v1RateLimiter == nil {
		t.Error("client.v1RateLimiter is nil")
	}

	if client.eaRateLimiter == nil {
		t.Error("client.eaRateLimiter is nil")
	}

	// Check defaults
	if client.maxRetries != DefaultMaxRetries {
		t.Errorf("maxRetries = %d, want %d", client.maxRetries, DefaultMaxRetries)
	}

	if client.retryWait != DefaultRetryWaitTime {
		t.Errorf("retryWait = %v, want %v", client.retryWait, DefaultRetryWaitTime)
	}
}

func TestNewWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *ClientConfig
		wantErr     bool
		checkFields func(t *testing.T, client *UnifiClient)
	}{
		{
			name: "minimal config",
			config: &ClientConfig{
				APIKey: "test-key",
			},
			wantErr: false,
			checkFields: func(t *testing.T, client *UnifiClient) {
				if client.maxRetries != DefaultMaxRetries {
					t.Errorf("maxRetries = %d, want %d", client.maxRetries, DefaultMaxRetries)
				}
			},
		},
		{
			name: "custom rate limits",
			config: &ClientConfig{
				APIKey:               "test-key",
				V1RateLimitPerMinute: 5000,
				EARateLimitPerMinute: 50,
			},
			wantErr: false,
			checkFields: func(t *testing.T, client *UnifiClient) {
				// Rate limiters are created, just check they exist
				if client.v1RateLimiter == nil {
					t.Error("v1RateLimiter is nil")
				}
				if client.eaRateLimiter == nil {
					t.Error("eaRateLimiter is nil")
				}
			},
		},
		{
			name: "custom retry settings",
			config: &ClientConfig{
				APIKey:        "test-key",
				MaxRetries:    5,
				RetryWaitTime: 2 * time.Second,
			},
			wantErr: false,
			checkFields: func(t *testing.T, client *UnifiClient) {
				if client.maxRetries != 5 {
					t.Errorf("maxRetries = %d, want 5", client.maxRetries)
				}
				if client.retryWait != 2*time.Second {
					t.Errorf("retryWait = %v, want 2s", client.retryWait)
				}
			},
		},
		{
			name: "custom base URL",
			config: &ClientConfig{
				APIKey:  "test-key",
				BaseURL: "https://custom.api.com",
			},
			wantErr: false,
			checkFields: func(t *testing.T, client *UnifiClient) {
				// Just verify client was created
				if client.client == nil {
					t.Error("client is nil")
				}
			},
		},
		{
			name: "empty API key",
			config: &ClientConfig{
				APIKey: "",
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
					t.Error("NewWithConfig() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewWithConfig() failed: %v", err)
			}

			if client == nil {
				t.Fatal("NewWithConfig() returned nil client")
			}

			if tt.checkFields != nil {
				tt.checkFields(t, client)
			}
		})
	}
}

func TestListHosts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *HostsResponse)
	}{
		{
			name:           "success - ucore type",
			mockResponse:   listHostsSuccess,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HostsResponse) {
				if resp == nil {
					t.Fatal("response is nil")
				}
				if len(resp.Data) != 1 {
					t.Errorf("len(Data) = %d, want 1", len(resp.Data))
				}
				if resp.Data[0].Type != HostType("ucore") {
					t.Errorf("Host type = %v, want ucore", resp.Data[0].Type)
				}
				if resp.NextToken == nil || *resp.NextToken != "ba8e384e-3308-4236-b344-7357657351ca" {
					t.Error("NextToken not set correctly")
				}
			},
		},
		{
			name:           "success - console type",
			mockResponse:   listHostsSuccessConsole,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HostsResponse) {
				if resp == nil {
					t.Fatal("response is nil")
				}
				if len(resp.Data) != 1 {
					t.Errorf("len(Data) = %d, want 1", len(resp.Data))
				}
				if resp.Data[0].Type != Console {
					t.Errorf("Host type = %v, want console", resp.Data[0].Type)
				}
				if resp.Data[0].ReportedState == nil {
					t.Fatal("ReportedState is nil for console")
				}
				if resp.Data[0].ReportedState.Hostname == nil || *resp.Data[0].ReportedState.Hostname != "example-console" {
					t.Error("Hostname not set correctly")
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
		{
			name:           "bad gateway",
			mockResponse:   badGatewayError,
			mockStatusCode: http.StatusBadGateway,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.URL.Path != "/v1/hosts" {
					t.Errorf("Request path = %s, want /v1/hosts", r.URL.Path)
				}
				if r.Header.Get("X-API-KEY") != "test-api-key" {
					t.Error("X-API-KEY header not set")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatalf("NewWithConfig failed: %v", err)
			}

			resp, err := client.ListHosts(context.Background(), nil)

			if tt.wantErr {
				if err == nil {
					t.Error("ListHosts() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ListHosts() unexpected error: %v", err)
				return
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestGetHostByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		hostID         string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *HostResponse)
	}{
		{
			name:           "success - ucore",
			hostID:         "test-host-id",
			mockResponse:   getHostUcoreSuccess,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HostResponse) {
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.Data.Type != HostType("ucore") {
					t.Errorf("Host type = %v, want ucore", resp.Data.Type)
				}
				if resp.Data.IpAddress == nil || *resp.Data.IpAddress != "220.130.137.169" {
					t.Error("IP address not set correctly")
				}
			},
		},
		{
			name:           "success - network-server",
			hostID:         "test-host-id",
			mockResponse:   getHostNetworkServerSuccess,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HostResponse) {
				if resp == nil {
					t.Fatal("response is nil")
				}
				if resp.Data.Type != NetworkServer {
					t.Errorf("Host type = %v, want network-server", resp.Data.Type)
				}
				if resp.Data.ReportedState == nil {
					t.Error("ReportedState is nil for network-server")
				}
			},
		},
		{
			name:           "not found",
			hostID:         "invalid-id",
			mockResponse:   notFoundError,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "server error",
			hostID:         "test-host-id",
			mockResponse:   serverError,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				expectedPath := "/v1/hosts/" + tt.hostID
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Header.Get("X-API-KEY") != "test-api-key" {
					t.Error("X-API-KEY header not set")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatalf("NewWithConfig failed: %v", err)
			}

			resp, err := client.GetHostByID(context.Background(), tt.hostID)

			if tt.wantErr {
				if err == nil {
					t.Error("GetHostByID() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetHostByID() unexpected error: %v", err)
				return
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestRetryLogic(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail first 2 attempts
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(serverError))
		} else {
			// Succeed on 3rd attempt
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(listHostsSuccess))
		}
	}))
	defer server.Close()

	client, err := NewWithConfig(&ClientConfig{
		APIKey:        "test-api-key",
		BaseURL:       server.URL,
		MaxRetries:    3,
		RetryWaitTime: 10 * time.Millisecond, // Short wait for tests
	})
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}

	_, err = client.ListHosts(context.Background(), nil)
	if err != nil {
		t.Errorf("ListHosts() failed after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(listHostsSuccess))
	}))
	defer server.Close()

	client, err := NewWithConfig(&ClientConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.ListHosts(ctx, nil)
	if err == nil {
		t.Error("Expected error from cancelled context, got nil")
	}
}

func TestPaginationParams(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		pageSize := r.URL.Query().Get("pageSize")
		nextToken := r.URL.Query().Get("nextToken")

		if pageSize != "10" {
			t.Errorf("pageSize = %s, want 10", pageSize)
		}
		if nextToken != "test-token" {
			t.Errorf("nextToken = %s, want test-token", nextToken)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(listHostsSuccess))
	}))
	defer server.Close()

	client, err := NewWithConfig(&ClientConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}

	pageSize := "10"
	nextToken := "test-token"
	params := &ListHostsParams{
		PageSize:  &pageSize,
		NextToken: &nextToken,
	}

	_, err = client.ListHosts(context.Background(), params)
	if err != nil {
		t.Errorf("ListHosts() failed: %v", err)
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
				if resp == nil {
					t.Fatal("response is nil")
				}
				if len(resp.Data) != 1 {
					t.Errorf("len(Data) = %d, want 1", len(resp.Data))
				}
				if resp.Data[0].SiteId == nil || *resp.Data[0].SiteId != "661de833b6b2463f0c20b319" {
					t.Error("SiteId not set correctly")
				}
				if resp.Data[0].Meta == nil || resp.Data[0].Meta.Name == nil || *resp.Data[0].Meta.Name != "default" {
					t.Error("Site name not set correctly")
				}
				if resp.Data[0].Statistics == nil {
					t.Error("Statistics is nil")
				}
				if resp.NextToken == nil || *resp.NextToken != "ba8e384e-3308-4236-b344-7357657351ca" {
					t.Error("NextToken not set correctly")
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
		{
			name:           "bad gateway",
			mockResponse:   badGatewayError,
			mockStatusCode: http.StatusBadGateway,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.URL.Path != "/v1/sites" {
					t.Errorf("Request path = %s, want /v1/sites", r.URL.Path)
				}
				if r.Header.Get("X-API-KEY") != "test-api-key" {
					t.Error("X-API-KEY header not set")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatalf("NewWithConfig failed: %v", err)
			}

			resp, err := client.ListSites(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("ListSites() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ListSites() unexpected error: %v", err)
				return
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestListDevices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *DevicesResponse)
	}{
		{
			name:           "success",
			mockResponse:   listDevicesSuccess,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *DevicesResponse) {
				if resp == nil {
					t.Fatal("response is nil")
				}
				if len(resp.Data) != 1 {
					t.Errorf("len(Data) = %d, want 1", len(resp.Data))
				}
				if resp.Data[0].HostId == nil || *resp.Data[0].HostId != "900A6F00301100000000074A6BA90000000007A3387E0000000063EC9853:123456789" {
					t.Error("HostId not set correctly")
				}
				if resp.Data[0].Devices == nil || len(*resp.Data[0].Devices) != 2 {
					t.Errorf("len(Devices) = %d, want 2", len(*resp.Data[0].Devices))
				}
				// Check first device (USW Flex Mini)
				device := (*resp.Data[0].Devices)[0]
				if device.Model == nil || *device.Model != "USW Flex Mini" {
					t.Error("First device model not set correctly")
				}
				if device.Status == nil || *device.Status != "online" {
					t.Error("First device status not set correctly")
				}
				// Check second device (UDR7)
				device2 := (*resp.Data[0].Devices)[1]
				if device2.Model == nil || *device2.Model != "UDR7" {
					t.Error("Second device model not set correctly")
				}
				if device2.IsConsole == nil || !*device2.IsConsole {
					t.Error("Second device should be console")
				}
				if resp.NextToken == nil || *resp.NextToken != "ba8e384e-3308-4236-b344-7357657351ca" {
					t.Error("NextToken not set correctly")
				}
			},
		},
		{
			name:           "parameter invalid",
			mockResponse:   parameterInvalidError,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
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
		{
			name:           "bad gateway",
			mockResponse:   badGatewayError,
			mockStatusCode: http.StatusBadGateway,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.URL.Path != "/v1/devices" {
					t.Errorf("Request path = %s, want /v1/devices", r.URL.Path)
				}
				if r.Header.Get("X-API-KEY") != "test-api-key" {
					t.Error("X-API-KEY header not set")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatalf("NewWithConfig failed: %v", err)
			}

			resp, err := client.ListDevices(context.Background(), nil)

			if tt.wantErr {
				if err == nil {
					t.Error("ListDevices() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ListDevices() unexpected error: %v", err)
				return
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestGetISPMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		metricType     GetISPMetricsParamsType
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *ISPMetricsResponse)
	}{
		{
			name:           "success",
			metricType:     "5m",
			mockResponse:   getISPMetricsSuccess,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *ISPMetricsResponse) {
				if resp == nil {
					t.Fatal("response is nil")
				}
				if len(resp.Data) != 1 {
					t.Errorf("len(Data) = %d, want 1", len(resp.Data))
				}
				metric := resp.Data[0]
				if metric.MetricType == nil || *metric.MetricType != "5m" {
					t.Error("MetricType not set correctly")
				}
				if metric.Periods == nil || len(*metric.Periods) == 0 {
					t.Fatal("Periods is nil or empty")
				}
				period := (*metric.Periods)[0]
				if period.Data == nil {
					t.Fatal("Period data is nil")
				}
			},
		},
		{
			name:           "parameter invalid",
			metricType:     "5m",
			mockResponse:   parameterInvalidError,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			metricType:     "5m",
			mockResponse:   unauthorizedError,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "rate limit",
			metricType:     "5m",
			mockResponse:   rateLimitError,
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name:           "server error",
			metricType:     "5m",
			mockResponse:   serverError,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request - EA endpoint
				if !strings.HasPrefix(r.URL.Path, "/ea/isp-metrics/") {
					t.Errorf("Request path = %s, want /ea/isp-metrics/*", r.URL.Path)
				}
				if r.Header.Get("X-API-KEY") != "test-api-key" {
					t.Error("X-API-KEY header not set")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatalf("NewWithConfig failed: %v", err)
			}

			duration := GetISPMetricsParamsDuration("24h")
			params := &GetISPMetricsParams{
				Duration: &duration,
			}

			resp, err := client.GetISPMetrics(context.Background(), tt.metricType, params)

			if tt.wantErr {
				if err == nil {
					t.Error("GetISPMetrics() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetISPMetrics() unexpected error: %v", err)
				return
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}
