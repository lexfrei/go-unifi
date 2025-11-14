package sitemanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexfrei/go-unifi/api/sitemanager/testdata"
	"github.com/lexfrei/go-unifi/internal/testutil"
)

// Test constants.
const (
	testAPIKey    = "test-api-key"
	testToken     = "test-token"
	testNextToken = "ba8e384e-3308-4236-b344-7357657351ca" //nolint:gosec // Test pagination token, not a real credential
	testHostID    = "900A6F00301100000000074A6BA90000000007A3387E0000000063EC9853:123456789"
)

func TestNew(t *testing.T) {
	t.Parallel()

	client, err := New(testAPIKey)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NotNil(t, client.client)
}

func TestNewWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *ClientConfig
		wantErr bool
	}{
		{
			name: "minimal config",
			config: &ClientConfig{
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "custom rate limits",
			config: &ClientConfig{
				APIKey:               "test-key",
				V1RateLimitPerMinute: 5000,
				EARateLimitPerMinute: 50,
			},
			wantErr: false,
		},
		{
			name: "custom retry settings",
			config: &ClientConfig{
				APIKey:        "test-key",
				MaxRetries:    5,
				RetryWaitTime: 2 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "custom base URL",
			config: &ClientConfig{
				APIKey:  "test-key",
				BaseURL: "https://custom.api.com",
			},
			wantErr: false,
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
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
			assert.NotNil(t, client.client)
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
			mockResponse:   testdata.LoadFixture(t, "hosts/list_success_ucore.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HostsResponse) {
				t.Helper()
				require.NotNil(t, resp)
				assert.Len(t, resp.Data, 1)
				assert.Equal(t, HostType("ucore"), resp.Data[0].Type)
				require.NotNil(t, resp.NextToken)
				assert.Equal(t, testNextToken, *resp.NextToken)
			},
		},
		{
			name:           "success - console type",
			mockResponse:   testdata.LoadFixture(t, "hosts/list_success_console.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HostsResponse) {
				t.Helper()
				require.NotNil(t, resp)
				assert.Len(t, resp.Data, 1)
				assert.Equal(t, Console, resp.Data[0].Type)
				require.NotNil(t, resp.Data[0].ReportedState)
				require.NotNil(t, resp.Data[0].ReportedState.Hostname)
				assert.Equal(t, "example-console", *resp.Data[0].ReportedState.Hostname)
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
		{
			name:           "bad gateway",
			mockResponse:   testdata.LoadFixture(t, "errors/bad_gateway.json"),
			mockStatusCode: http.StatusBadGateway,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := testutil.NewMockServer(t, "/v1/hosts", testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  testAPIKey,
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			resp, err := client.ListHosts(context.Background(), nil)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
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
			mockResponse:   testdata.LoadFixture(t, "hosts/get_ucore.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HostResponse) {
				t.Helper()
				require.NotNil(t, resp)
				assert.Equal(t, HostType("ucore"), resp.Data.Type)
				require.NotNil(t, resp.Data.IpAddress)
				assert.Equal(t, "220.130.137.169", *resp.Data.IpAddress)
			},
		},
		{
			name:           "success - network-server",
			hostID:         "test-host-id",
			mockResponse:   testdata.LoadFixture(t, "hosts/get_network_server.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *HostResponse) {
				t.Helper()
				require.NotNil(t, resp)
				assert.Equal(t, NetworkServer, resp.Data.Type)
				assert.NotNil(t, resp.Data.ReportedState)
			},
		},
		{
			name:           "not found",
			hostID:         "invalid-id",
			mockResponse:   testdata.LoadFixture(t, "errors/not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "server error",
			hostID:         "test-host-id",
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/v1/hosts/" + tt.hostID
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  testAPIKey,
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			resp, err := client.GetHostByID(context.Background(), tt.hostID)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestRetryLogic(t *testing.T) {
	t.Parallel()

	attempts := 0
	successResponse := testdata.LoadFixture(t, "hosts/list_success_ucore.json")
	errorResponse := testdata.LoadFixture(t, "errors/server_error.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail first 2 attempts
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errorResponse))
		} else {
			// Succeed on 3rd attempt
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(successResponse))
		}
	}))
	defer server.Close()

	client, err := NewWithConfig(&ClientConfig{
		APIKey:        testAPIKey,
		BaseURL:       server.URL,
		MaxRetries:    3,
		RetryWaitTime: 10 * time.Millisecond,
	})
	require.NoError(t, err)

	_, err = client.ListHosts(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()

	successResponse := testdata.LoadFixture(t, "hosts/list_success_ucore.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Delay to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(successResponse))
	}))
	defer server.Close()

	client, err := NewWithConfig(&ClientConfig{
		APIKey:  testAPIKey,
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.ListHosts(ctx, nil)
	require.Error(t, err)
}

func TestPaginationParams(t *testing.T) {
	t.Parallel()

	successResponse := testdata.LoadFixture(t, "hosts/list_success_ucore.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		pageSize := r.URL.Query().Get("pageSize")
		nextToken := r.URL.Query().Get("nextToken")

		assert.Equal(t, "10", pageSize)
		assert.Equal(t, testToken, nextToken)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(successResponse))
	}))
	defer server.Close()

	client, err := NewWithConfig(&ClientConfig{
		APIKey:  testAPIKey,
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	pageSize := "10"
	nextToken := testToken
	params := &ListHostsParams{
		PageSize:  &pageSize,
		NextToken: &nextToken,
	}

	_, err = client.ListHosts(context.Background(), params)
	require.NoError(t, err)
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
				require.NotNil(t, resp)
				assert.Len(t, resp.Data, 1)
				require.NotNil(t, resp.Data[0].SiteId)
				assert.Equal(t, "661de833b6b2463f0c20b319", *resp.Data[0].SiteId)
				require.NotNil(t, resp.Data[0].Meta)
				require.NotNil(t, resp.Data[0].Meta.Name)
				assert.Equal(t, "default", *resp.Data[0].Meta.Name)
				assert.NotNil(t, resp.Data[0].Statistics)
				require.NotNil(t, resp.NextToken)
				assert.Equal(t, testNextToken, *resp.NextToken)
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
		{
			name:           "bad gateway",
			mockResponse:   testdata.LoadFixture(t, "errors/bad_gateway.json"),
			mockStatusCode: http.StatusBadGateway,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := testutil.NewMockServer(t, "/v1/sites", testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  testAPIKey,
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			resp, err := client.ListSites(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
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
			mockResponse:   testdata.LoadFixture(t, "devices/list_success.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *DevicesResponse) {
				t.Helper()
				require.NotNil(t, resp)
				assert.Len(t, resp.Data, 1)
				require.NotNil(t, resp.Data[0].HostId)
				assert.Equal(t, testHostID, *resp.Data[0].HostId)
				require.NotNil(t, resp.Data[0].Devices)
				assert.Len(t, *resp.Data[0].Devices, 2)

				// Check first device (USW Flex Mini)
				device := (*resp.Data[0].Devices)[0]
				require.NotNil(t, device.Model)
				assert.Equal(t, "USW Flex Mini", *device.Model)
				require.NotNil(t, device.Status)
				assert.Equal(t, "online", *device.Status)

				// Check second device (UDR7)
				device2 := (*resp.Data[0].Devices)[1]
				require.NotNil(t, device2.Model)
				assert.Equal(t, "UDR7", *device2.Model)
				require.NotNil(t, device2.IsConsole)
				assert.True(t, *device2.IsConsole)

				require.NotNil(t, resp.NextToken)
				assert.Equal(t, testNextToken, *resp.NextToken)
			},
		},
		{
			name:           "parameter invalid",
			mockResponse:   testdata.LoadFixture(t, "errors/parameter_invalid.json"),
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
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
		{
			name:           "bad gateway",
			mockResponse:   testdata.LoadFixture(t, "errors/bad_gateway.json"),
			mockStatusCode: http.StatusBadGateway,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := testutil.NewMockServer(t, "/v1/devices", testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  testAPIKey,
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			resp, err := client.ListDevices(context.Background(), nil)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
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
			mockResponse:   testdata.LoadFixture(t, "metrics/get_isp_metrics.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *ISPMetricsResponse) {
				t.Helper()
				require.NotNil(t, resp)
				assert.Len(t, resp.Data, 1)
				metric := resp.Data[0]
				require.NotNil(t, metric.MetricType)
				assert.Equal(t, string(N5m), *metric.MetricType)
				require.NotNil(t, metric.Periods)
				assert.NotEmpty(t, *metric.Periods)
				period := (*metric.Periods)[0]
				assert.NotNil(t, period.Data)
			},
		},
		{
			name:           "parameter invalid",
			metricType:     "5m",
			mockResponse:   testdata.LoadFixture(t, "errors/parameter_invalid.json"),
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			metricType:     "5m",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "rate limit",
			metricType:     "5m",
			mockResponse:   testdata.LoadFixture(t, "errors/rate_limit.json"),
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name:           "server error",
			metricType:     "5m",
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// For EA endpoints, we need to check path prefix instead of exact match
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.True(t, strings.HasPrefix(r.URL.Path, "/ea/isp-metrics/"))
				assert.Equal(t, testAPIKey, r.Header.Get("X-Api-Key"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  testAPIKey,
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			duration := GetISPMetricsParamsDuration("24h")
			params := &GetISPMetricsParams{
				Duration: &duration,
			}

			resp, err := client.GetISPMetrics(context.Background(), tt.metricType, params)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestGetSDWANConfigStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		configID       string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *SDWANConfigStatusResponse)
	}{
		{
			name:           "success",
			configID:       "test-config-id",
			mockResponse:   testdata.LoadFixture(t, "sdwan/config_status.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *SDWANConfigStatusResponse) {
				t.Helper()
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data.Fingerprint)
				assert.Equal(t, "85d521a1b3c8992f", *resp.Data.Fingerprint)
				require.NotNil(t, resp.Data.Hubs)
				assert.Len(t, *resp.Data.Hubs, 1)
				require.NotNil(t, resp.Data.Spokes)
				assert.Len(t, *resp.Data.Spokes, 2)
			},
		},
		{
			name:           "not found",
			configID:       "non-existent-id",
			mockResponse:   testdata.LoadFixture(t, "sdwan/config_status_not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			configID:       "test-config-id",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "rate limit",
			configID:       "test-config-id",
			mockResponse:   testdata.LoadFixture(t, "errors/rate_limit.json"),
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name:           "server error",
			configID:       "test-config-id",
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/ea/sd-wan-configs/" + tt.configID + "/status"
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  testAPIKey,
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			resp, err := client.GetSDWANConfigStatus(context.Background(), tt.configID)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestGetSDWANConfigByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		configID       string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *SDWANConfigResponse)
	}{
		{
			name:           "success",
			configID:       "b344034f-2636-478c-8c7a-e3350f8ed37a",
			mockResponse:   testdata.LoadFixture(t, "sdwan/get_config_by_id.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *SDWANConfigResponse) {
				t.Helper()
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data.Id)
				assert.Equal(t, "b344034f-2636-478c-8c7a-e3350f8ed37a", *resp.Data.Id)
				require.NotNil(t, resp.Data.Name)
				assert.Equal(t, "RS test", *resp.Data.Name)
				require.NotNil(t, resp.Data.Hubs)
				assert.Len(t, *resp.Data.Hubs, 1)
				require.NotNil(t, resp.Data.Spokes)
				assert.Len(t, *resp.Data.Spokes, 2)
			},
		},
		{
			name:           "not found",
			configID:       "non-existent-id",
			mockResponse:   testdata.LoadFixture(t, "sdwan/config_status_not_found.json"),
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			configID:       "b344034f-2636-478c-8c7a-e3350f8ed37a",
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "rate limit",
			configID:       "b344034f-2636-478c-8c7a-e3350f8ed37a",
			mockResponse:   testdata.LoadFixture(t, "errors/rate_limit.json"),
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name:           "server error",
			configID:       "b344034f-2636-478c-8c7a-e3350f8ed37a",
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/ea/sd-wan-configs/" + tt.configID
			server := testutil.NewMockServer(t, expectedPath, testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  testAPIKey,
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			resp, err := client.GetSDWANConfigByID(context.Background(), tt.configID)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestListSDWANConfigs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *SDWANConfigsResponse)
	}{
		{
			name:           "success",
			mockResponse:   testdata.LoadFixture(t, "sdwan/list_configs.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *SDWANConfigsResponse) {
				t.Helper()
				require.NotNil(t, resp)
				assert.Len(t, resp.Data, 1)
				config := resp.Data[0]
				require.NotNil(t, config.Id)
				assert.Equal(t, "9304163b-680d-4de8-a7a0-7617e328911d", *config.Id)
				require.NotNil(t, config.Name)
				assert.Equal(t, "SD-WAN test", *config.Name)
				require.NotNil(t, config.Type)
				assert.Equal(t, SDWANConfigType("sdwan-hbsp"), *config.Type)
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
		{
			name:           "bad gateway",
			mockResponse:   testdata.LoadFixture(t, "errors/bad_gateway.json"),
			mockStatusCode: http.StatusBadGateway,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := testutil.NewMockServer(t, "/ea/sd-wan-configs", testAPIKey, tt.mockResponse, tt.mockStatusCode)
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  testAPIKey,
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			resp, err := client.ListSDWANConfigs(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestQueryISPMetrics(t *testing.T) {
	t.Parallel()

	testSiteID := "661900ae6aec8f548d49fd54"
	testQuery := ISPMetricsQuery{
		Sites: &[]ISPMetricsQuerySiteItem{
			{
				SiteId: testSiteID,
				HostId: testHostID,
			},
		},
	}

	tests := []struct {
		name           string
		metricType     string
		query          ISPMetricsQuery
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		checkResponse  func(t *testing.T, resp *ISPMetricsQueryResponse)
	}{
		{
			name:           "success",
			metricType:     "5m",
			query:          testQuery,
			mockResponse:   testdata.LoadFixture(t, "metrics/query_isp_metrics_success.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *ISPMetricsQueryResponse) {
				t.Helper()
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data.Metrics)
				assert.Len(t, *resp.Data.Metrics, 1)
				metric := (*resp.Data.Metrics)[0]
				require.NotNil(t, metric.MetricType)
				assert.Equal(t, string(N5m), *metric.MetricType)
				require.NotNil(t, metric.SiteId)
				assert.Equal(t, testSiteID, *metric.SiteId)
			},
		},
		{
			name:           "partial success",
			metricType:     "5m",
			query:          testQuery,
			mockResponse:   testdata.LoadFixture(t, "metrics/query_isp_metrics_partial_success.json"),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *ISPMetricsQueryResponse) {
				t.Helper()
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data.Status)
				assert.Equal(t, ISPMetricsQueryResponseDataStatus("partialSuccess"), *resp.Data.Status)
				assert.NotNil(t, resp.Data.Message)
			},
		},
		{
			name:           "invalid parameter",
			metricType:     "5m",
			query:          testQuery,
			mockResponse:   testdata.LoadFixture(t, "errors/invalid_parameter.json"),
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			metricType:     "5m",
			query:          testQuery,
			mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "rate limit",
			metricType:     "5m",
			query:          testQuery,
			mockResponse:   testdata.LoadFixture(t, "errors/rate_limit.json"),
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name:           "server error",
			metricType:     "5m",
			query:          testQuery,
			mockResponse:   testdata.LoadFixture(t, "errors/server_error.json"),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name:           "bad gateway",
			metricType:     "5m",
			query:          testQuery,
			mockResponse:   testdata.LoadFixture(t, "errors/bad_gateway.json"),
			mockStatusCode: http.StatusBadGateway,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedPath := "/ea/isp-metrics/" + tt.metricType + "/query"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, testAPIKey, r.Header.Get("X-Api-Key"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewWithConfig(&ClientConfig{
				APIKey:  testAPIKey,
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			resp, err := client.QueryISPMetrics(context.Background(), tt.metricType, tt.query)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

// Edge case tests.

func TestContextTimeout(t *testing.T) {
	t.Parallel()

	server := testutil.NewMockServerWithHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewWithConfig(&ClientConfig{
		APIKey:  testAPIKey,
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = client.ListHosts(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestInvalidJSON(t *testing.T) {
	t.Parallel()

	server := testutil.NewMockServer(t, "/v1/hosts", testAPIKey,
		"{invalid json", http.StatusOK)
	defer server.Close()

	client, err := NewWithConfig(&ClientConfig{
		APIKey:  testAPIKey,
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	_, err = client.ListHosts(context.Background(), nil)
	require.Error(t, err)
}

func TestNetworkError(t *testing.T) {
	t.Parallel()

	// Use invalid URL to trigger connection error
	client, err := NewWithConfig(&ClientConfig{
		APIKey:  testAPIKey,
		BaseURL: "http://localhost:1",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = client.ListHosts(ctx, nil)
	require.Error(t, err)
}
