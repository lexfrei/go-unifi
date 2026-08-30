# UniFi Network API v2 - Implementation Guide

**Purpose**: Self-contained guide for implementing UniFi Network API v2 endpoints.
Multiple agents can use this guide to work in parallel without conflicts.

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [File Locations](#file-locations)
3. [API Versions & Categories](#api-versions--categories)
4. [SSH Access & Discovery](#ssh-access--discovery)
5. [TDD Methodology](#tdd-methodology)
6. [Safety Rules](#safety-rules)
7. [Progress Tracking](#progress-tracking)
8. [Parallel Work Guidelines](#parallel-work-guidelines)
9. [Pre-Implementation Tasks](#pre-implementation-tasks)

---

## Project Overview

**Goal**: Complete CRUD coverage for UniFi Network API v2 endpoints with:

- TDD approach (tests first)
- Detailed OpenAPI specification (fully typed, no `additionalProperties: true`)
- No destructive operations on production resources

**Device**: UDR7 (UniFi Dream Router 7)
**Network Version**: 9.5.21
**IP**: 172.16.0.1

---

## File Locations

### Source of Truth (NOT in git)

```text
dig/                                    # .gitignored - contains sensitive data
├── .api-key                            # API key for testing
├── all-api-site-endpoints.txt          # All 349 discovered endpoints
├── ENDPOINTS-ROADMAP.md                # Prioritized implementation list
├── FINDINGS.md                         # Architecture notes
├── API-SCHEMAS.md                      # MongoDB schemas documentation
├── mongodb-collection-schemas.json     # All collection field names
├── network-integration-api-9.5.21.json # Official OpenAPI spec from device
├── api-responses/                      # Captured API responses
│   ├── api-wlanconf.json               # WLAN config (2 SSIDs)
│   ├── api-networkconf.json            # Network config (4 networks)
│   ├── api-portforward.json            # Port forwards (3 rules)
│   ├── api-setting.json                # Various settings
│   ├── firewall-policies.json          # Firewall policies
│   ├── static-dns.json                 # DNS records
│   ├── topology.json                   # Network topology
│   ├── device.json                     # Devices list
│   ├── clients-active.json             # Active clients
│   └── ...
├── capabilities/                       # Controller capabilities
├── scopes/                             # API permission scopes
└── nginx-http/                         # Nginx proxy configs
```

### Code Files (in git)

```text
api/network/
├── openapi.yaml                        # OpenAPI specification (edit this)
├── .oapi-codegen.yaml                  # Code generator config
├── generated.go                        # Auto-generated (DO NOT EDIT)
├── client.go                           # Client wrapper methods
├── interfaces.go                       # Public interfaces for mocking
├── client_test.go                      # Unit tests
└── testdata/
    ├── fixtures.go                     # Fixture loader
    └── {resource}/                     # Fixtures per resource
        ├── list_success.json
        ├── single_success.json
        ├── create_request.json
        ├── create_response.json
        ├── error_not_found.json
        └── error_validation.json

examples/network/
├── list_sites/main.go
├── list_devices/main.go
├── list_dns_records/main.go
└── {new_resource}/main.go              # Add example for each new endpoint
```

### Key Config Files

```text
.golangci.yaml                          # Linter config
CLAUDE.md                               # Project conventions
api/network/.oapi-codegen.yaml          # oapi-codegen config
```

---

## API Versions & Categories

### API Version Summary

| Version | Path Pattern | Site Param | Auth | Status |
|---------|-------------|------------|------|--------|
| Integration v1 | `/integration/v1/sites/{siteId}/...` | UUID | X-API-KEY | Official, documented |
| Internal v2 | `/api/site/{siteName}/...` | string (e.g., "default") | X-API-KEY | Internal, undocumented |
| Legacy REST | `/api/s/{site}/rest/...` | string | Cookie session | Deprecated |
| Legacy Stat | `/api/s/{site}/stat/...` | string | Cookie session | Deprecated |
| Legacy Cmd | `/api/s/{site}/cmd/...` | string | Cookie session | Deprecated |

### Integration v1 (Official)

Source: `dig/network-integration-api-9.5.21.json` (OpenAPI spec from device)

Official, documented API with limited endpoints:

- Sites, Devices, Clients, Hotspot Vouchers
- Uses UUID siteId
- See `dig/endpoints-by-version/integration-v1.txt`

### Internal v2 (Target for Implementation)

Source: `dig/all-api-site-endpoints.txt` (349 endpoints)

The file contains all discovered internal v2 endpoints in format `/api/site/{siteName}/...`

**Characteristics:**

- Path: `/api/site/{siteName}/{resource}` (equivalent to `/v2/api/site/{site}/...`)
- Site param: string like "default" (NOT UUID!)
- Auth: `X-API-KEY` header
- Response: Direct JSON (not wrapped in `{"meta": ..., "data": ...}`)
- See `dig/endpoints-by-version/internal-v2.txt` for categorized list

### Legacy REST/Stat/Cmd (NOT a Priority)

Old API format with cookie-based auth. These may still exist but are being deprecated:

- `/api/s/{site}/rest/{resource}` - CRUD operations
- `/api/s/{site}/stat/{resource}` - Statistics
- `/api/s/{site}/cmd/{command}` - Commands/actions

### Already Implemented (v2)

| Resource | Path | Methods |
|----------|------|---------|
| Static DNS | `/v2/api/site/{site}/static-dns` | CRUD |
| Firewall Policies | `/v2/api/site/{site}/firewall-policies` | CRUD |
| Traffic Rules | `/v2/api/site/{site}/trafficrules` | CRUD |
| Topology | `/v2/api/site/{site}/topology` | GET |
| Devices | `/v2/api/site/{site}/device` | GET |
| Active Clients | `/v2/api/site/{site}/clients/active` | GET |
| Dashboard | `/v2/api/site/{site}/aggregated-dashboard` | GET |

### Priority v2 Endpoints to Implement

Based on `dig/ENDPOINTS-ROADMAP.md`:

**Priority 1 - Core Networking:**

- `/lan/enriched-configuration` - Full LAN config (GET)
- `/wan/enriched-configuration` - Full WAN config (GET)
- `/wlan/enriched-configuration` - Full WLAN config (GET)
- `/nat` - NAT rules (CRUD)
- `/nat/{id}` - Single NAT rule

**Priority 2 - Firewall Zones:**

- `/firewall/zone` - Zones list (GET, POST)
- `/firewall/zone/{zoneId}` - Single zone (GET, PUT, DELETE)
- `/firewall/zone-matrix` - Zone matrix (GET)

**Priority 3 - VPN:**

- `/wireguard/users` - WireGuard users
- `/wireguard/{networkId}/users` - Per-network users
- `/vpn/connections` - Active VPN connections
- `/teleport/token` - Teleport tokens

**Priority 4 - Monitoring:**

- `/isp/status` - ISP status
- `/isp/health` - ISP health
- `/speedtest` - Run speedtest
- `/traffic-rate` - Current traffic

---

## SSH Access & Discovery

### Connection

```bash
ssh root@172.16.0.1
```

### API Key Location

```bash
# On device:
cat /data/unifi-core/config/.api-key 2>/dev/null

# Or in dig/:
cat dig/.api-key
```

### Discover Endpoint

```bash
API_KEY=$(cat dig/.api-key)
BASE="https://172.16.0.1/proxy/network"

# Test endpoint
curl -sk "$BASE/v2/api/site/default/{resource}" \
  -H "X-API-KEY: $API_KEY" \
  -H "Accept: application/json" | jq .
```

### Capture Response for Fixture

```bash
# From local machine:
API_KEY=$(cat dig/.api-key)
curl -sk "https://172.16.0.1/proxy/network/v2/api/site/default/{resource}" \
  -H "X-API-KEY: $API_KEY" | jq . > dig/api-responses/{resource}.json
```

### MongoDB Access (on device)

```bash
ssh root@172.16.0.1

# Connect to MongoDB
mongo --port 27117

# List databases
show dbs

# Use UniFi database
use ace

# List collections
show collections

# Explore collection schema
db.{collection}.find().limit(1).pretty()

# Get all unique field names
var keys = {};
db.{collection}.find().forEach(function(doc) {
  Object.keys(doc).forEach(function(key) {
    keys[key] = typeof doc[key];
  });
});
printjson(keys);
```

### Nginx Routes (on device)

```bash
cat /data/unifi-core/config/http/site-local-ip.conf
# Shows how paths are proxied to backend services
```

---

## TDD Methodology

### Workflow for Each New Endpoint

#### Step 1: Capture Real Data

```bash
# GET list
curl -sk "https://172.16.0.1/proxy/network/v2/api/site/default/{resource}" \
  -H "X-API-KEY: $(cat dig/.api-key)" | jq . > dig/api-responses/{resource}.json

# Copy to testdata with sanitization
cp dig/api-responses/{resource}.json api/network/testdata/{resource}/list_success.json
# Edit to remove sensitive data (passwords, keys)
```

#### Step 2: Write OpenAPI Schema

Edit `api/network/openapi.yaml`:

```yaml
paths:
  /v2/api/site/{site}/{resource}:
    get:
      operationId: list{Resource}
      tags: [{Resource}]
      parameters:
        - $ref: '#/components/parameters/site'
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/{Resource}'

components:
  schemas:
    {Resource}:
      type: object
      required:
        - _id
      properties:
        _id:
          type: string
          description: Unique identifier
        # ... all other fields with proper types
        # NO additionalProperties: true!
```

#### Step 3: Generate Code

```bash
cd api/network && go generate
```

#### Step 4: Write Tests First

Add to `api/network/client_test.go`:

```go
func TestList{Resource}(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name           string
        mockResponse   string
        mockStatusCode int
        wantErr        bool
        wantErrMsg     string
        check          func(*testing.T, []{Resource})
    }{
        {
            name:           "success",
            mockResponse:   testdata.LoadFixture(t, "{resource}/list_success.json"),
            mockStatusCode: http.StatusOK,
            check: func(t *testing.T, items []{Resource}) {
                require.NotEmpty(t, items)
            },
        },
        {
            name:           "unauthorized",
            mockResponse:   testdata.LoadFixture(t, "errors/unauthorized.json"),
            mockStatusCode: http.StatusUnauthorized,
            wantErr:        true,
            wantErrMsg:     "unauthorized",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            server := testutil.NewMockServer(t,
                "/v2/api/site/default/{resource}",
                testAPIKey,
                tt.mockResponse,
                tt.mockStatusCode)
            defer server.Close()

            client, err := NewWithConfig(&ClientConfig{
                APIKey:  testAPIKey,
                BaseURL: server.URL,
            })
            require.NoError(t, err)

            result, err := client.List{Resource}(context.Background(), "default")

            if tt.wantErr {
                require.Error(t, err)
                if tt.wantErrMsg != "" {
                    assert.Contains(t, err.Error(), tt.wantErrMsg)
                }
                return
            }

            require.NoError(t, err)
            if tt.check != nil {
                tt.check(t, result)
            }
        })
    }
}
```

#### Step 5: Run Tests (Should Fail)

```bash
go test ./api/network/... -run TestList{Resource}
```

#### Step 6: Implement Client Method

Add to `api/network/client.go`:

```go
func (c *UnifiClient) List{Resource}(ctx context.Context, site Site) ([]{Resource}, error) {
    resp, err := c.client.List{Resource}WithResponse(ctx, site)
    if err != nil {
        return nil, errors.Wrap(err, "failed to list {resource}")
    }

    if resp.StatusCode() != http.StatusOK {
        return nil, response.HandleError(resp.HTTPResponse, resp.Body)
    }

    if resp.JSON200 == nil {
        return nil, errors.New("unexpected nil response")
    }

    return *resp.JSON200, nil
}
```

#### Step 7: Update Interface

Add to `api/network/interfaces.go`:

```go
type NetworkAPIClient interface {
    // ... existing methods
    List{Resource}(ctx context.Context, site Site) ([]{Resource}, error)
}
```

#### Step 8: Run Tests (Should Pass)

```bash
go test ./api/network/... -run TestList{Resource}
```

#### Step 9: Run All Tests + Linter

```bash
go test ./...
golangci-lint run ./...
```

#### Step 10: Create Example

Create `examples/network/list_{resource}/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lexfrei/go-unifi/api/network"
)

func main() {
    baseURL := os.Getenv("UNIFI_BASE_URL")
    apiKey := os.Getenv("UNIFI_API_KEY")

    if baseURL == "" || apiKey == "" {
        log.Fatal("UNIFI_BASE_URL and UNIFI_API_KEY required")
    }

    client, err := network.NewWithConfig(&network.ClientConfig{
        BaseURL:            baseURL,
        APIKey:             apiKey,
        InsecureSkipVerify: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    items, err := client.List{Resource}(context.Background(), "default")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d {resource}(s)\n", len(items))
    for _, item := range items {
        fmt.Printf("  - %s\n", item.Id)
    }
}
```

#### Step 11: Test Against Real API

```bash
UNIFI_BASE_URL=https://172.16.0.1 \
UNIFI_API_KEY=$(cat dig/.api-key) \
go run examples/network/list_{resource}/main.go
```

---

## Safety Rules

### ALWAYS OK (Safe Operations)

- All GET requests
- Creating test resources that don't affect production traffic (e.g., test DNS record, test firewall rule with no matching traffic)
- Deleting resources that were specifically created for testing (you created it → you can delete it)
- Reading configurations

**Key principle**: If you CREATE something for testing purposes, DELETING it is perfectly fine and expected. This is normal TDD workflow.

### CAUTION REQUIRED

- PUT/PATCH to modify existing resources that were NOT created by you
- Test on non-critical resources first
- Document what was changed

### FORBIDDEN Without Explicit User Confirmation

- DELETE on pre-existing production resources (not created by you)
- Modifying WAN configuration
- Disabling WLANs that serve management traffic
- Deleting networks with active clients
- Restarting devices
- Any changes that could interrupt network connectivity

### Implementation Pattern for Destructive Operations

```go
// For DELETE operations
type DeleteOptions struct {
    // Force must be true to actually delete
    Force bool
    // DryRun returns what would be deleted without action
    DryRun bool
}

func (c *Client) Delete{Resource}(
    ctx context.Context,
    site Site,
    id string,
    opts DeleteOptions,
) error {
    if !opts.Force && !opts.DryRun {
        return errors.New("destructive operation: set Force=true or DryRun=true")
    }

    if opts.DryRun {
        // Just validate the resource exists
        _, err := c.Get{Resource}(ctx, site, id)
        return err
    }

    // Actual deletion
    resp, err := c.client.Delete{Resource}WithResponse(ctx, site, id)
    // ...
}
```

---

## Methodical CRUD Testing

### Principles

1. **Delete only what you create** - never delete pre-existing resources
2. **Create safe test data** - resources that don't affect production traffic
3. **Test in order**: Create → Read → Update → Delete
4. **Clean up** - always delete test resources after testing

### Safe CRUD Candidates

Resources safe for CRUD testing (create won't break anything):

| Resource | Safe Test Data | Why Safe |
|----------|---------------|----------|
| **NAT rules** | Port forward to 65432 (unused port) | No traffic matches |
| **Traffic routes** | Route to 10.255.255.0/24 (unused) | No traffic matches |
| **Content filtering** | Block `test-crud-domain.invalid` | Domain doesn't exist |
| **Firewall policies** | Rule matching `10.255.255.0/24` | No traffic matches |
| **Static DNS** | `test-crud.local` → `127.0.0.1` | Internal only |
| **Traffic rules** | Rule for unused IP range | No traffic matches |
| **WireGuard users** | Test user (if WG network exists) | Can delete after |
| **RADIUS users** | Test user `crud-test-user` | Can delete after |

### CRUD Testing Workflow

```bash
# 1. Capture existing resources (baseline)
curl -sk "$BASE/v2/api/site/default/{resource}" \
  -H "X-API-KEY: $API_KEY" | jq . > /tmp/before.json

# 2. Create test resource
curl -sk "$BASE/v2/api/site/default/{resource}" \
  -H "X-API-KEY: $API_KEY" \
  -H "Content-Type: application/json" \
  -X POST -d @test-create.json | jq .

# 3. Verify created (save ID)
CREATED_ID=$(curl -sk "$BASE/v2/api/site/default/{resource}" \
  -H "X-API-KEY: $API_KEY" | jq -r '.[-1]._id')

# 4. Update created resource
curl -sk "$BASE/v2/api/site/default/{resource}/$CREATED_ID" \
  -H "X-API-KEY: $API_KEY" \
  -H "Content-Type: application/json" \
  -X PUT -d @test-update.json | jq .

# 5. Delete created resource
curl -sk "$BASE/v2/api/site/default/{resource}/$CREATED_ID" \
  -H "X-API-KEY: $API_KEY" \
  -X DELETE

# 6. Verify deleted (should match baseline)
curl -sk "$BASE/v2/api/site/default/{resource}" \
  -H "X-API-KEY: $API_KEY" | jq . > /tmp/after.json
diff /tmp/before.json /tmp/after.json
```

### Test Data Examples

**NAT Rule (safe - unused port):**

```json
{
  "name": "CRUD Test - Delete Me",
  "enabled": false,
  "pfwd_interface": "wan",
  "src": "any",
  "dst_port": "65432",
  "fwd": "192.168.1.1",
  "fwd_port": "65432",
  "proto": "tcp"
}
```

**Traffic Route (safe - unused network):**

```json
{
  "name": "CRUD Test Route - Delete Me",
  "enabled": false,
  "network": "10.255.255.0/24",
  "interface": "wan",
  "distance": 1
}
```

**Static DNS (safe - internal domain):**

```json
{
  "key": "test-crud.local",
  "value": "127.0.0.1",
  "record_type": "A"
}
```

### Marking CRUD Complete

After successful CRUD test, update `dig/ENDPOINTS-ROADMAP.md`:

```markdown
- [x] `/nat` - ✅ **CRUD** (tested 2025-11-26)
```

Only mark as **CRUD** when all four operations verified:

- ✅ POST (Create) - returns created resource
- ✅ GET (Read) - returns resource by ID
- ✅ PUT (Update) - modifies resource
- ✅ DELETE - removes resource

---

## Progress Tracking

### Tracking File

Update `dig/ENDPOINTS-ROADMAP.md` when completing endpoints:

```markdown
### Priority 1: Core Networking
- [x] `/static-dns` - CRUD complete (PR #XX)
- [x] `/firewall-policies` - CRUD complete (PR #XX)
- [ ] `/nat` - In progress (@agent-name)
- [ ] `/firewall/zone` - Not started
```

### Commit Convention

```text
feat(network): add {resource} CRUD endpoints

- Add OpenAPI schema for {Resource}
- Implement List/Get/Create/Update/Delete methods
- Add unit tests with fixtures
- Add example program

Co-Authored-By: Claude <noreply@anthropic.com>
```

---

## Parallel Work Guidelines

### Safe for Parallel Work

Multiple agents can work simultaneously on:

- Different endpoint groups (e.g., one on NAT, another on Zones)
- Different files (e.g., one on tests, another on examples)
- Documentation vs. implementation

### Requires Coordination

- Modifications to `openapi.yaml` - one agent at a time
- Regenerating `generated.go` - after OpenAPI changes
- Updates to `client.go` interface

### Claiming Work

Before starting, agent should:

1. Check `dig/ENDPOINTS-ROADMAP.md` for unclaimed work
2. Mark endpoint as "In progress (@agent-name)"
3. Complete the full TDD cycle
4. Mark as complete with PR reference

---

## Pre-Implementation Tasks

### Task 0: Mass API Scanning

Before implementing individual endpoints, scan all simple GET endpoints to capture real responses:

**Script location**: `dig/scan-endpoints.sh`

**Output directory**: `dig/api-responses/scan/`

**Usage**:

```bash
cd dig && ./scan-endpoints.sh
```

**Results**:

- Individual JSON responses in `dig/api-responses/scan/{endpoint}.json`
- Status log in `dig/api-responses/scan/status.log`
- Summary report showing working vs failing endpoints

### Task 1: Categorize Endpoints ✅ DONE

Endpoints have been categorized in `dig/endpoints-by-version/`:

```text
dig/endpoints-by-version/
├── README.md            # API versions overview
├── integration-v1.txt   # Official Integration API v1 endpoints
└── internal-v2.txt      # Internal v2 endpoints (349 total, categorized by priority)
```

The `all-api-site-endpoints.txt` contains ONLY Internal v2 endpoints (no legacy).
Legacy REST/Stat/Cmd APIs use different path patterns (`/api/s/{site}/...`) and are NOT in scope.

### Task 2: Audit Existing Schemas

Check `api/network/openapi.yaml` for `additionalProperties: true`:

```bash
grep -n "additionalProperties" api/network/openapi.yaml
```

These need to be replaced with explicit type definitions based on:

1. Real API responses in `dig/api-responses/`
2. MongoDB schemas in `dig/API-SCHEMAS.md`

### Task 3: Capture Missing API Responses

From `dig/ENDPOINTS-ROADMAP.md`, capture responses for Priority 1 endpoints:

```bash
API_KEY=$(cat dig/.api-key)
BASE="https://172.16.0.1/proxy/network/v2/api/site/default"

# NAT
curl -sk "$BASE/nat" -H "X-API-KEY: $API_KEY" | jq . > dig/api-responses/nat.json

# Firewall zones
curl -sk "$BASE/firewall/zone" -H "X-API-KEY: $API_KEY" | jq . > dig/api-responses/firewall-zone.json

# Enriched configs
curl -sk "$BASE/lan/enriched-configuration" -H "X-API-KEY: $API_KEY" | jq . > dig/api-responses/lan-enriched.json
curl -sk "$BASE/wan/enriched-configuration" -H "X-API-KEY: $API_KEY" | jq . > dig/api-responses/wan-enriched.json
curl -sk "$BASE/wlan/enriched-configuration" -H "X-API-KEY: $API_KEY" | jq . > dig/api-responses/wlan-enriched.json
```

---

## Quick Reference

### Commands

```bash
# Generate code from OpenAPI
cd api/network && go generate

# Run tests
go test ./api/network/...

# Run specific test
go test ./api/network/... -run TestListNAT

# Run linter
golangci-lint run ./...

# Test against real API
UNIFI_BASE_URL=https://172.16.0.1 \
UNIFI_API_KEY=$(cat dig/.api-key) \
go run examples/network/{example}/main.go
```

### File Templates

Fixture: `api/network/testdata/{resource}/list_success.json`
Test: See TDD Step 4
Client method: See TDD Step 6
Example: See TDD Step 10

---

## Summary

1. All sensitive data in `dig/` (gitignored)
2. TDD: tests first, then implementation
3. No destructive operations without explicit confirmation
4. Track progress in `dig/ENDPOINTS-ROADMAP.md`
5. Pre-work: categorize endpoints, audit schemas, capture responses

---

## CRITICAL REMINDERS

### Keep Roadmap Up-to-Date

**ALWAYS update `dig/ENDPOINTS-ROADMAP.md` after implementing endpoints!**

- Mark completed endpoints with `[x]` and `✅ **IMPLEMENTED**`
- Update the "Implementation Status" section with accurate counts
- Remove `- **NEW**` tags from older entries after a while
- Add `- **TODO:**` notes for endpoints requiring special handling (query params, POST bodies)

### Read This Guide First

**ALWAYS read this guide (`dig/IMPLEMENTATION-GUIDE.md`) before starting work!**

- Check what's already implemented
- See current priorities
- Follow established patterns
- Avoid duplicate work
