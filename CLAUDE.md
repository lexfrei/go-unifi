# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**go-unifi** is a collection of Pure Go client libraries for UniFi APIs, generated from OpenAPI specifications.

- **Language**: Go 1.25+
- **APIs**: Site Manager API (cloud) + Network API (local controller)
- **Code Generation**: oapi-codegen v2
- **Error Handling**: github.com/cockroachdb/errors
- **Architecture**: Multi-module with shared infrastructure
- **Tested Hardware**: UniFi Dream Router (UDR7), UniFi VMs with various OS/Network versions

## Core Principles

### 1. Type Safety is Mandatory

**CRITICAL**: Avoid `map[string]interface{}` and `interface{}` types at all costs.

- All API structures MUST be fully typed in `openapi.yaml`
- Use `$ref` to reference typed schemas instead of `additionalProperties: true`
- Each nested object should have its own schema definition
- Only use `additionalProperties` when the structure is genuinely dynamic

**Bad**:
```yaml
hardware:
  type: object
  additionalProperties: true
```

**Good**:
```yaml
hardware:
  $ref: '#/components/schemas/HardwareInfo'
```

### 2. OpenAPI-First Development

All API changes MUST start with `openapi.yaml` in the respective API module:

1. Update/add schemas in `api/{sitemanager|network}/openapi.yaml`
2. Run code generation (see commands below)
3. Test with real API
4. Commit both `openapi.yaml` and `generated.go` together

**Never** manually edit `generated.go` files - they are auto-generated.

### 3. Real API Testing Required

- All new endpoints/schemas MUST be tested against real UniFi API
- Use environment variable `UNIFI_API_KEY` for authentication
- Test both success and error cases
- Verify type correctness with actual API responses

## Architecture

The repository follows a **multi-module architecture** with shared infrastructure:

```
.
├── api/                   # Public API clients (each is an independent module)
│   ├── sitemanager/       # Cloud-based Site Manager API
│   │   ├── openapi.yaml   # OpenAPI spec (source of truth)
│   │   ├── generated.go   # Generated client (DO NOT EDIT)
│   │   ├── client.go      # Hand-written wrapper
│   │   ├── interfaces.go  # Testable interfaces
│   │   └── .oapi-codegen.yaml
│   └── network/           # Local Network API
│       ├── openapi.yaml
│       ├── generated.go
│       ├── client.go
│       ├── interfaces.go
│       └── .oapi-codegen.yaml
├── internal/              # Shared infrastructure (not importable externally)
│   ├── httpclient/        # HTTP client with middleware support
│   ├── middleware/        # Composable middleware (auth, retry, rate limit, observability, TLS)
│   ├── ratelimit/         # Token bucket rate limiter
│   ├── retry/             # Exponential backoff retry logic
│   ├── response/          # Generic response handlers
│   └── testutil/          # Testing utilities
├── observability/         # Public Logger and MetricsRecorder interfaces
├── examples/              # Working examples for both APIs
│   ├── sitemanager/
│   ├── network/
│   ├── observability/     # Custom logging/metrics integration example
│   └── testing/           # Mocking examples
└── cmd/                   # Command-line tools
    └── test-reality/      # Validate types against real API responses
```

### Key Design Principles

1. **Middleware-based architecture**: All HTTP concerns (auth, retry, rate limiting, observability, TLS) are implemented as composable middleware in `internal/middleware/`
2. **Separation of concerns**: Generated code in `generated.go`, hand-written wrapper in `client.go`, testable interfaces in `interfaces.go`
3. **Testability**: All API clients expose interfaces (`SiteManagerAPIClient`, `NetworkAPIClient`) for easy mocking
4. **Dual rate limiting**: Site Manager uses separate limiters for v1 (10k/min) and EA (100/min) endpoints

## Code Generation

### Configuration

Each API module has `.oapi-codegen.yaml`:

```yaml
package: sitemanager  # or network
generate:
  client: true
  models: true
  embedded-spec: true
output: generated.go
output-options:
  skip-prune: true
```

### Regenerating Code

**Using go generate (recommended):**

```bash
# Regenerate specific API
cd api/sitemanager && go generate
cd api/network && go generate

# Regenerate all
go generate ./...
```

**Direct invocation:**

```bash
cd api/sitemanager && oapi-codegen -config .oapi-codegen.yaml openapi.yaml
cd api/network && oapi-codegen -config .oapi-codegen.yaml openapi.yaml
```

### After Generation

Always verify:
```bash
# Must compile
go build ./...

# Run tests
go test ./...

# Check type safety (should be minimal)
grep -c 'map\[string\]interface{}' api/sitemanager/generated.go
grep -c 'map\[string\]interface{}' api/network/generated.go

# Test with real API
UNIFI_API_KEY=key go run examples/sitemanager/list_hosts/main.go
UNIFI_BASE_URL=https://unifi.local UNIFI_API_KEY=key go run examples/network/list_sites/main.go
```

## OpenAPI Schema Guidelines

### Naming Conventions

- **Schemas**: PascalCase (e.g., `HardwareInfo`, `DeviceFeatures`)
- **Properties**: camelCase matching JSON (e.g., `ipAddress`, `firmwareVersion`)
- **Operations**: camelCase with action prefix (e.g., `listHosts`, `getHostById`)

### Schema Organization

Place schemas in logical order:
1. Main response/request types (e.g., `HostsResponse`, `HostResponse`)
2. Data models (e.g., `Host`, `Device`, `Site`)
3. Nested complex types (e.g., `HardwareInfo`, `Controller`)
4. Utility types (e.g., `ErrorResponse`, `SuccessResponse`)

### Required vs Optional Fields

- Use `required: [field1, field2]` for mandatory fields
- All other fields are optional (will be pointers in Go)
- Top-level identifiers (`id`, `hardwareId`, `type`) should be required

### Date/Time Fields

Always use:
```yaml
type: string
format: date-time
description: Time in RFC3339 format when...
```

### UUIDs

Always use:
```yaml
type: string
format: uuid
description: Unique identifier...
```

## Client Wrappers (client.go)

Each API module has a hand-written client wrapper that:
1. Wraps the generated `ClientWithResponses`
2. Applies middleware (auth, rate limiting, retry, observability, TLS)
3. Provides high-level methods that handle response unwrapping
4. Implements the public interface for testability

### Site Manager Client

```go
// High-level constructor
client, err := sitemanager.New("api-key")

// Custom configuration
client, err := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
    APIKey:               "api-key",
    BaseURL:              "https://api.ui.com",
    V1RateLimitPerMinute: 10000,
    EARateLimitPerMinute: 100,
    MaxRetries:           3,
    RetryWaitTime:        time.Second,
    Timeout:              30 * time.Second,
    Logger:               customLogger,
    Metrics:              customMetrics,
})
```

### Network API Client

```go
// High-level constructor
client, err := network.New("https://unifi.local", "api-key")

// Custom configuration with self-signed cert support
client, err := network.NewWithConfig(&network.ClientConfig{
    BaseURL:            "https://unifi.local",
    APIKey:             "api-key",
    RateLimitPerMinute: 1000,
    MaxRetries:         3,
    RetryWaitTime:      time.Second,
    Timeout:            30 * time.Second,
    InsecureSkipVerify: true,  // For self-signed certs
    Logger:             customLogger,
    Metrics:            customMetrics,
})
```

### Middleware Architecture

All HTTP concerns are implemented as composable middleware:

- **Auth middleware**: Adds `X-API-KEY` header
- **Rate limit middleware**: Token bucket algorithm, path-aware (dual limiters for Site Manager)
- **Retry middleware**: Exponential backoff for network errors, 5xx, 429
- **Observability middleware**: Logging and metrics via pluggable interfaces
- **TLS middleware**: Self-signed certificate support for Network API

Middleware is applied in `internal/httpclient` using reverse order (first middleware is outermost).

### Error Handling

Always use `github.com/cockroachdb/errors`:
```go
// Good
return nil, errors.Wrap(err, "failed to list hosts")
return nil, errors.Wrapf(err, "failed to get host %s", id)

// Bad
return nil, fmt.Errorf("failed to list hosts: %w", err)
```

### Observability

Clients support pluggable logging and metrics via interfaces in `observability/`:

```go
type Logger interface {
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
}

type MetricsRecorder interface {
    RecordHTTPRequest(method, path string, statusCode int, duration time.Duration, err error)
    RecordRateLimitWait(duration time.Duration)
    RecordRetry(attempt int)
}
```

See `examples/observability/` for integration examples.

## Testing Standards

### Automated Testing

The codebase has comprehensive test coverage using table-driven tests:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# View coverage
go tool cover -html=coverage.out

# Run specific package tests
go test ./api/sitemanager/...
go test ./internal/middleware/...
```

### Testing with Interfaces

All API clients expose interfaces for easy mocking. See `examples/testing/` for complete examples:

**Using gomock:**
```bash
go generate ./...  # Generates mocks if configured
```

**Using testify/mock:**
```go
type MockNetworkClient struct {
    mock.Mock
}

func (m *MockNetworkClient) ListDNSRecords(ctx context.Context, site network.Site) ([]network.DNSRecord, error) {
    args := m.Called(ctx, site)
    return args.Get(0).([]network.DNSRecord), args.Error(1)
}
```

### Manual Testing with Real API

Create example programs in `examples/{api}/`:

```go
// examples/sitemanager/test_feature/main.go
package main

import (
    "context"
    "log"
    "os"
    "github.com/lexfrei/go-unifi/api/sitemanager"
)

func main() {
    apiKey := os.Getenv("UNIFI_API_KEY")
    if apiKey == "" {
        log.Fatal("UNIFI_API_KEY required")
    }

    client, err := sitemanager.New(apiKey)
    if err != nil {
        log.Fatal(err)
    }

    // Test your feature here
}
```

Run:
```bash
# Site Manager API
UNIFI_API_KEY=your-key go run examples/sitemanager/test_feature/main.go

# Network API
UNIFI_BASE_URL=https://unifi.local UNIFI_API_KEY=your-key go run examples/network/test_feature/main.go
```

### Validation Tool

Use `cmd/test-reality` to validate types against real API responses:

```bash
go run github.com/lexfrei/go-unifi/cmd/test-reality@latest -api-key your-key
```

### Verification Checklist

Before committing API changes:
- [ ] OpenAPI spec updated in correct module
- [ ] Code regenerated with `go generate`
- [ ] All tests pass (`go test ./...`)
- [ ] Compiles without errors (`go build ./...`)
- [ ] Linters pass (`golangci-lint run ./...`)
- [ ] Tested with real API
- [ ] Example program created/updated
- [ ] Type safety verified (minimal `interface{}`)
- [ ] Interface updated if new methods added

## Documentation Standards

### README.md

Keep synchronized with code:
- Update API coverage when adding endpoints
- Update examples when changing client API
- Document tested hardware/firmware versions

### Code Comments

Generated code has comments from OpenAPI `description` fields:
```yaml
properties:
  shortname:
    type: string
    description: Short model name (e.g., UDR7)
```

Becomes:
```go
// Shortname Short model name (e.g., UDR7)
Shortname *string `json:"shortname,omitempty"`
```

### OpenAPI Descriptions

Write clear, concise descriptions:
- **Good**: "Short model name (e.g., UDR7)"
- **Bad**: "The name"
- Include examples where helpful
- Explain units for numbers (seconds, bytes, percentage, etc.)

## Common Development Tasks

### Adding a New Endpoint

**Example: Adding endpoint to Site Manager API**

1. **Update OpenAPI spec** in `api/sitemanager/openapi.yaml`:
```yaml
paths:
  /v1/new-endpoint:
    get:
      summary: Brief description
      operationId: getNewEndpoint
      tags:
        - TagName
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/NewEndpointResponse'
```

2. **Add response schema**:
```yaml
components:
  schemas:
    NewEndpointResponse:
      allOf:
        - $ref: '#/components/schemas/SuccessResponse'
        - type: object
          properties:
            data:
              $ref: '#/components/schemas/NewData'
```

3. **Regenerate code**:
```bash
cd api/sitemanager && go generate
```

4. **Add wrapper method** in `api/sitemanager/client.go`:
```go
func (c *UnifiClient) GetNewEndpoint(ctx context.Context) (*NewEndpointResponse, error) {
    return response.Handle200(
        c.client.GetNewEndpointWithResponse(ctx),
        "failed to get new endpoint",
    )
}
```

5. **Update interface** in `api/sitemanager/interfaces.go`:
```go
type SiteManagerAPIClient interface {
    // ... existing methods
    GetNewEndpoint(ctx context.Context) (*NewEndpointResponse, error)
}
```

6. **Create example** in `examples/sitemanager/new_endpoint/main.go`

7. **Add test** in `api/sitemanager/client_test.go`

8. **Test with real API**

9. **Commit in logical chunks**:
```bash
git add api/sitemanager/openapi.yaml
git commit --signoff --message "feat(sitemanager): add NewEndpoint schema"

git add api/sitemanager/generated.go
git commit --signoff --message "feat(sitemanager): regenerate from updated spec"

git add api/sitemanager/client.go api/sitemanager/interfaces.go api/sitemanager/client_test.go
git commit --signoff --message "feat(sitemanager): add GetNewEndpoint wrapper method"

git add examples/sitemanager/new_endpoint/
git commit --signoff --message "feat(sitemanager): add example for GetNewEndpoint"
```

### Adding Type Definitions

When API returns complex nested structures:

1. **Capture real JSON response**:
```bash
UNIFI_API_KEY=key go run examples/endpoint/main.go -v > /tmp/response.json
```

2. **Analyze structure**, identify all fields and types

3. **Create schema in openapi.yaml**:
```yaml
components:
  schemas:
    NewComplexType:
      type: object
      properties:
        field1:
          type: string
        field2:
          type: integer
        nestedObject:
          $ref: '#/components/schemas/NestedType'
```

4. **Reference in parent**:
```yaml
parentField:
  $ref: '#/components/schemas/NewComplexType'
```

5. **Regenerate and verify**:
```bash
cd api/{sitemanager|network} && go generate
grep "type NewComplexType" generated.go
```

## Git Workflow

### Commit Messages

Follow semantic commit format with API scope:
```
feat(sitemanager): add support for SD-WAN status endpoint
feat(network): add support for DNS records management
fix(middleware): correct rate limiter token bucket size
fix(network): handle DELETE requests returning 200 instead of 204
refactor(sitemanager): replace map[string]interface{} with typed structures
docs(readme): update tested hardware list
test(middleware): add retry backoff test cases
```

### Commit Granularity

**CRITICAL: Commit after EACH logical block of work** - do NOT accumulate changes:

- ✅ Good: 4 commits for adding an endpoint (schema → generated → wrapper → example)
- ❌ Bad: 1 giant commit with all changes

Always use `--signoff`:
```bash
git commit --signoff --message "feat(network): add DNS records schema"
```

### What to Commit Together

**Separate commits for:**
- OpenAPI schema changes
- Generated code (regeneration)
- Client wrapper methods + interfaces + tests
- Examples
- Documentation updates
- Internal infrastructure changes

**Keep in sync:**
- `openapi.yaml` and `generated.go` should be committed close together (2 sequential commits)
- `client.go` and `interfaces.go` for new methods
- Tests with the feature they test

## Dependencies

### Production Dependencies

```
github.com/cockroachdb/errors       # Enhanced error handling
github.com/oapi-codegen/runtime     # OpenAPI runtime support
golang.org/x/time/rate              # Rate limiting
```

### Development Dependencies

```
github.com/getkin/kin-openapi       # OpenAPI validation
github.com/stretchr/testify         # Testing utilities
```

### Code Generation Tools

```
github.com/oapi-codegen/oapi-codegen/v2  # Generate code from OpenAPI specs
```

Install:
```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

### Dependency Policy

**Only add dependencies for:**
- Core functionality (rate limiting, error handling, OpenAPI runtime)
- Code generation tools
- Testing frameworks

**DO NOT add:**
- General utilities (implement yourself or use stdlib)
- Logging libraries (users should choose their own, we provide interfaces)
- HTTP frameworks (stdlib `net/http` only)
- Heavy frameworks or ORMs

## API Conventions

### Pagination

**Site Manager API** uses `pageSize` and `nextToken`:
```go
params := &sitemanager.ListHostsParams{
    PageSize:  sitemanager.PtrString("10"),
    NextToken: sitemanager.PtrString(token),
}
```

**Network API** pagination varies by endpoint (some use limit/offset, some don't paginate).

### Error Responses

All endpoints return standard error structure:
```yaml
ErrorResponse:
  type: object
  properties:
    code:
      type: string
    message:
      type: string
    httpStatusCode:
      type: integer
    traceId:
      type: string
```

### Success Responses

All successful responses include:
```yaml
SuccessResponse:
  type: object
  properties:
    httpStatusCode:
      type: integer
    traceId:
      type: string
```

## Performance and Implementation Notes

### Middleware Performance

- **Rate limiter**: Token bucket algorithm (`golang.org/x/time/rate`), no goroutine per request
- **Retries**: Exponential backoff with jitter prevents thundering herd
- **Middleware chaining**: Applied once at client creation, not per request
- **Context propagation**: All methods accept `context.Context` for cancellation and timeouts

### Memory Efficiency

- **Pointers for optional fields**: Zero allocations for omitted fields in API responses
- **No reflection in hot path**: All type conversions are compile-time safe
- **Reusable HTTP client**: Single `*http.Client` instance with connection pooling

### Response Handling

Generic response handlers in `internal/response/`:
- `Handle200[T]`: For 200 OK responses
- `Handle204`: For 204 No Content responses
- Consistent error wrapping across all endpoints

## Security

- **API keys**: Always use environment variables, never hardcode
- **Sensitive data**: Never commit real API responses to git
- **Examples**: Use placeholder values in documentation
- **Logging**: Never log API keys or tokens

## Troubleshooting

### Code generation fails

```bash
# Verify oapi-codegen is installed
which oapi-codegen
oapi-codegen --version

# Re-install if needed
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# Verify OpenAPI spec is valid
cd api/sitemanager && oapi-codegen -config .oapi-codegen.yaml openapi.yaml
```

### Types not matching API response

1. Capture real response:
```bash
UNIFI_API_KEY=key go run examples/sitemanager/endpoint/main.go > /tmp/response.json
```

2. Compare with OpenAPI schema in `api/{module}/openapi.yaml`
3. Update schema to match reality (API is source of truth)
4. Regenerate: `cd api/{module} && go generate`
5. Test again with real API

### Rate limiting issues

**Site Manager:**
- v1 endpoints: 10,000 req/min
- EA endpoints: 100 req/min

```go
client, _ := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
    V1RateLimitPerMinute: 10000,
    EARateLimitPerMinute: 100,
})
```

**Network API:**
```go
client, _ := network.NewWithConfig(&network.ClientConfig{
    RateLimitPerMinute: 1000,
})
```

### Self-signed certificate issues (Network API)

```go
client, _ := network.NewWithConfig(&network.ClientConfig{
    InsecureSkipVerify: true,  // For self-signed certs
})
```

### Test failures

```bash
# Run specific test with verbose output
go test -v -run TestSpecificFunction ./api/sitemanager/...

# Check for race conditions
go test -race ./...

# Update test fixtures if API changed
# Edit testdata/fixtures.go with new responses
```

## Useful Commands

```bash
# Build everything
go build ./...

# Run all tests
go test ./...

# Run tests with coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out

# Run linters (MUST pass before commit)
golangci-lint run ./...

# Regenerate all code
go generate ./...

# Regenerate specific API
cd api/sitemanager && go generate
cd api/network && go generate

# Check type safety (should be minimal)
grep -c 'map\[string\]interface{}' api/sitemanager/generated.go
grep -c 'map\[string\]interface{}' api/network/generated.go

# Find all exported types in an API
grep -n '^type [A-Z]' api/sitemanager/generated.go | head -20

# Test with real Site Manager API
UNIFI_API_KEY=your-key go run examples/sitemanager/list_hosts/main.go

# Test with real Network API
UNIFI_BASE_URL=https://unifi.local UNIFI_API_KEY=your-key go run examples/network/list_sites/main.go

# Build all examples
for dir in examples/*/*/; do echo "Building $dir" && go build -o /dev/null "$dir" || break; done

# Validate types against real API
go run cmd/test-reality/main.go -api-key your-key
```

## References

### Official Documentation

- [UniFi Site Manager API](https://developer.ui.com/site-manager-api/gettingstarted) - Official cloud API docs
- [UniFi Network API](https://developer.ui.com/network-api/unifi-network-api) - Official local controller API docs
- [OpenAPI 3.0 Specification](https://spec.openapis.org/oas/v3.0.3) - OpenAPI spec standard

### Tools and Libraries

- [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) - OpenAPI to Go code generator
- [cockroachdb/errors](https://github.com/cockroachdb/errors) - Enhanced error handling
- [golangci-lint](https://golangci-lint.run/) - Go linters aggregator

### Project Documentation

- [README.md](README.md) - Project overview and quick start
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [TESTING.md](TESTING.md) - Testing guidelines
- [RELEASING.md](RELEASING.md) - Release process

## Import Paths

```go
// Public API clients
import "github.com/lexfrei/go-unifi/api/sitemanager"
import "github.com/lexfrei/go-unifi/api/network"

// Observability interfaces (for custom logging/metrics)
import "github.com/lexfrei/go-unifi/observability"

// Internal packages - DO NOT import from external code
// These are for internal use only
```

## Linter Configuration

See `.golangci.yaml` for full configuration. Key settings:

- **Disabled linters**: `depguard`, `exhaustruct`, `gochecknoglobals`, `canonicalheader` (UniFi uses X-API-KEY)
- **Function length**: Max 60 lines, 60 statements
- **Complexity**: Max 15 (gocyclo, cyclop)
- **Examples and cmd excluded**: Linting is relaxed for example programs

**CRITICAL**: ALL linting errors MUST be fixed before pushing. There are NO "minor" linting issues.

