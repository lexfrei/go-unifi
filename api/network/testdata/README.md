# Test Fixtures for Network API

This directory contains real JSON responses from UniFi Network API, extracted from `client_test.go` for reusability and maintainability.

## Structure

```
testdata/
├── clients/          # Client-related responses
│   ├── list_success.json
│   └── single_client.json
├── dashboard/        # Dashboard data responses
│   └── aggregated.json
├── devices/          # Device-related responses
│   ├── list_success.json
│   └── single_device.json
├── dns/              # DNS record responses
│   ├── empty_list.json
│   ├── list_success.json
│   └── single_record.json
├── errors/           # Error responses (4xx, 5xx)
│   ├── bad_request.json
│   ├── not_found.json
│   ├── rate_limit.json
│   ├── server_error.json
│   └── unauthorized.json
├── firewall/         # Firewall policy responses
│   ├── empty_list.json
│   └── single_policy.json
├── hotspot/          # Hotspot voucher responses
│   ├── empty_list.json
│   ├── list_vouchers_success.json
│   └── single_voucher.json
├── sites/            # Site-related responses
│   └── list_success.json
└── traffic/          # Traffic rule responses
    ├── empty_list.json
    └── single_rule.json
```

## Usage

Use the `LoadFixture` or `LoadFixtureJSON` functions from `fixtures.go`:

```go
import "github.com/lexfrei/go-unifi/api/network/testdata"

func TestSomething(t *testing.T) {
    // Load as string
    jsonStr := testdata.LoadFixture(t, "sites/list_success.json")

    // Load and unmarshal into struct
    var resp SitesResponse
    testdata.LoadFixtureJSON(t, "sites/list_success.json", &resp)
}
```

## Data Source

All fixtures are real API responses from:
- **Hardware**: UniFi Dream Router (UDR7)
- **Firmware**: UniFi OS 4.3.87
- **API Version**: v1 and v2 (UniFi Site Manager API)

Data has been anonymized where needed (MAC addresses, IPs use test ranges).

## Maintenance

When adding new fixtures:
1. Capture real API response
2. Place in appropriate subdirectory
3. Use descriptive filename (snake_case)
4. Pretty-print JSON with 2-space indentation
5. Validate with `jq . file.json`
