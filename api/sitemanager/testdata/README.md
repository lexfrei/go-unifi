# Test Fixtures for Site Manager API

This directory contains real JSON responses from UniFi Site Manager API, extracted from `client_test.go` for reusability and maintainability.

## Structure

```
testdata/
├── devices/          # Device-related responses
│   └── list_success.json
├── errors/           # Error responses (4xx, 5xx)
│   ├── bad_gateway.json
│   ├── invalid_parameter.json
│   ├── not_found.json
│   ├── parameter_invalid.json
│   ├── rate_limit.json
│   ├── server_error.json
│   └── unauthorized.json
├── hosts/            # Host-related responses (controllers, consoles)
│   ├── get_network_server.json
│   ├── get_ucore.json
│   ├── list_success_console.json
│   └── list_success_ucore.json
├── metrics/          # ISP metrics responses
│   ├── get_isp_metrics.json
│   ├── query_isp_metrics_partial_success.json
│   └── query_isp_metrics_success.json
├── sdwan/            # SD-WAN configuration responses
│   ├── config_status_not_found.json
│   ├── config_status.json
│   ├── get_config_by_id.json
│   └── list_configs.json
└── sites/            # Site-related responses
    └── list_success.json
```

## Usage

Use the `LoadFixture` or `LoadFixtureJSON` functions from `fixtures.go`:

```go
import "github.com/lexfrei/go-unifi/api/sitemanager/testdata"

func TestSomething(t *testing.T) {
    // Load as string
    jsonStr := testdata.LoadFixture(t, "hosts/list_success_ucore.json")

    // Load and unmarshal into struct
    var resp HostsResponse
    testdata.LoadFixtureJSON(t, "hosts/list_success_ucore.json", &resp)
}
```

## Data Source

All fixtures are real API responses from:
- **Hardware**: UniFi Dream Router (UDR7), UniFi Console, UniFi Network Server
- **Firmware**: UniFi OS 4.3.87
- **API Version**: v1 (UniFi Site Manager API + Early Access endpoints)

Data has been anonymized where needed (MAC addresses, IPs, hostnames use test values).

## Fixture Details

### Hosts
- `list_success_ucore.json` - List response with UCore (Dream Router) host
- `list_success_console.json` - List response with Console host type
- `get_ucore.json` - Single UCore host details
- `get_network_server.json` - Single Network Server host details

### Metrics
- `get_isp_metrics.json` - ISP metrics for single host
- `query_isp_metrics_success.json` - Metrics query with full results
- `query_isp_metrics_partial_success.json` - Partial results scenario

### SD-WAN
- `list_configs.json` - List of SD-WAN configurations
- `get_config_by_id.json` - Single configuration details
- `config_status.json` - Configuration status response
- `config_status_not_found.json` - Status for non-existent config

### Errors
Standard error responses covering common API errors:
- `unauthorized.json` (401) - Invalid/missing API key
- `not_found.json` (404) - Resource not found
- `invalid_parameter.json` / `parameter_invalid.json` (400) - Bad request
- `rate_limit.json` (429) - Rate limit exceeded
- `server_error.json` (500) - Internal server error
- `bad_gateway.json` (502) - Gateway/proxy error

## Maintenance

When adding new fixtures:
1. Capture real API response from UniFi controller
2. Place in appropriate subdirectory
3. Use descriptive filename (snake_case)
4. Pretty-print JSON with 2-space indentation
5. Validate with `jq . file.json`
6. Anonymize sensitive data (IPs, MACs, hostnames)
