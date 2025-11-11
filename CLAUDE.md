# CLAUDE.md - go-unifi Development Standards

This file provides project-specific guidance for Claude Code when working on the go-unifi library.

## Project Overview

**go-unifi** is a Pure Go client library for UniFi Site Manager API v1, generated from OpenAPI specification.

- **Language**: Go 1.21+
- **API Version**: UniFi Site Manager API v1 + Early Access
- **Code Generation**: oapi-codegen v2.5.1
- **Error Handling**: github.com/cockroachdb/errors
- **Tested Hardware**: UniFi Dream Router (UDR7) running UniFi OS 4.3.87

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

All API changes MUST start with `openapi.yaml`:

1. Update/add schemas in `openapi.yaml`
2. Run `make generate` or `$HOME/go/bin/oapi-codegen --config .oapi-codegen.yaml openapi.yaml > generated.go`
3. Test with real API
4. Commit both `openapi.yaml` and `generated.go` together

**Never** manually edit `generated.go` - it's auto-generated.

### 3. Real API Testing Required

- All new endpoints/schemas MUST be tested against real UniFi API
- Use environment variable `UNIFI_API_KEY` for authentication
- Test both success and error cases
- Verify type correctness with actual API responses

## File Structure

```
.
├── openapi.yaml           # OpenAPI 3.0 specification (source of truth)
├── generated.go           # Generated API client (DO NOT EDIT)
├── client.go              # Hand-written wrapper with rate limiting & retries
├── .oapi-codegen.yaml     # Code generation config
├── Makefile               # Build tasks
├── examples/              # Example programs
│   ├── list_hosts/        # List all hosts example
│   └── get_host/          # Get host by ID example
└── CLAUDE.md              # This file
```

## Code Generation

### Configuration (.oapi-codegen.yaml)

```yaml
package: unifi
generate:
  models: true
  client: true
output-options:
  skip-prune: true
```

### Regenerating Code

```bash
# Method 1 (recommended)
make generate

# Method 2 (direct)
$HOME/go/bin/oapi-codegen --config .oapi-codegen.yaml openapi.yaml > generated.go
```

### After Generation

Always verify:
```bash
go build .                                    # Must compile
go run examples/get_host/main.go              # Must work with real API
grep -c 'map\[string\]interface{}' generated.go  # Should be minimal
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

## Client Wrapper (client.go)

### Rate Limiting

- **v1 endpoints**: 10,000 requests/minute
- **EA endpoints**: 100 requests/minute
- Implemented using `golang.org/x/time/rate`

### Retry Logic

Automatically retries:
- Network errors (connection failures, timeouts)
- 5xx server errors
- 429 rate limit errors (respects `Retry-After` header)

Strategy:
- Exponential backoff
- Default: 3 retries, 1s wait time
- Configurable via `ClientConfig`

### Error Handling

Use `github.com/cockroachdb/errors`:
```go
// Good
return nil, errors.Wrap(err, "failed to list hosts")
return nil, errors.Wrapf(err, "failed to get host %s", id)

// Bad
return nil, fmt.Errorf("failed to list hosts: %w", err)
```

## Testing Standards

### Manual Testing

Create example programs in `examples/`:
```go
// examples/test_feature/main.go
package main

import (
    "context"
    "log"
    "os"
    "github.com/lexfrei/go-unifi"
)

func main() {
    apiKey := os.Getenv("UNIFI_API_KEY")
    if apiKey == "" {
        log.Fatal("UNIFI_API_KEY required")
    }

    client, err := unifi.NewUnifiClient(unifi.ClientConfig{
        APIKey: apiKey,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Test your feature here
}
```

Run:
```bash
UNIFI_API_KEY=your-key go run examples/test_feature/main.go
```

### Verification Checklist

Before committing API changes:
- [ ] OpenAPI spec updated
- [ ] Code regenerated
- [ ] Compiles without errors
- [ ] Tested with real API
- [ ] Example program created/updated
- [ ] Type safety verified (minimal `interface{}`)

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

## Common Tasks

### Adding a New Endpoint

1. **Add to openapi.yaml**:
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

3. **Regenerate**: `make generate`

4. **Add wrapper method in client.go**:
```go
func (c *UnifiClient) GetNewEndpoint(ctx context.Context) (*NewEndpointResponse, error) {
    resp, err := c.client.GetNewEndpointWithResponse(ctx)
    if err != nil {
        return nil, errors.Wrap(err, "failed to get new endpoint")
    }

    if resp.StatusCode() != http.StatusOK {
        return nil, errors.Newf("API error: status=%d", resp.StatusCode())
    }

    if resp.JSON200 == nil {
        return nil, errors.New("empty response from API")
    }

    return resp.JSON200, nil
}
```

5. **Create example**: `examples/new_endpoint/main.go`

6. **Test with real API**

7. **Commit together**: `git add openapi.yaml generated.go client.go examples/`

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
make generate
grep "type NewComplexType" generated.go
```

## Git Workflow

### Commit Messages

Follow semantic commit format:
```
feat(api): add support for network devices endpoint
fix(client): correct rate limiter token bucket size
refactor(api): replace map[string]interface{} with typed structures
docs(readme): update tested hardware list
```

### Commit Granularity

- **Commit after each logical change**
- Small focused commits preferred
- Always use `--signoff`
- Examples:
  - "Add OpenAPI schema for X" (just openapi.yaml)
  - "Regenerate code from updated spec" (generated.go)
  - "Add client wrapper for X endpoint" (client.go)
  - "Add example for X" (examples/)

### What to Commit Together

- OpenAPI + generated code: **Together** (they must be in sync)
- Client wrapper: **Separate** (hand-written code)
- Examples: **Separate** (can be independent)
- Tests: **With feature** they test

## Dependencies

### Core

- `golang.org/x/time/rate` - Rate limiting
- `github.com/cockroachdb/errors` - Enhanced errors
- `github.com/oapi-codegen/runtime` - OpenAPI runtime
- `github.com/oapi-codegen/oapi-codegen/v2` - Code generator (dev tool)

### Adding Dependencies

Only add dependencies for:
- Core functionality (rate limiting, error handling)
- Code generation (oapi-codegen)
- Testing (if needed)

**DO NOT add**:
- General utilities (implement yourself)
- Logging libraries (let users choose)
- HTTP frameworks (stdlib only)

## API Conventions

### Pagination

Use `pageSize` and `nextToken` pattern:
```go
params := &unifi.ListHostsParams{
    PageSize:  unifi.PtrString("10"),
    NextToken: unifi.PtrString(token),
}
```

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

## Performance Considerations

- **Rate limiter**: Token bucket algorithm, no goroutine per request
- **Retries**: Exponential backoff prevents thundering herd
- **Context**: All methods accept `context.Context` for cancellation
- **Pointers**: Optional fields use pointers (zero allocations for omitted fields)

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
$HOME/go/bin/oapi-codegen --version

# Re-install if needed
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1
```

### Types not matching API response

1. Capture real response: `go run example -v > /tmp/response.json`
2. Compare with OpenAPI schema
3. Update schema to match reality
4. Regenerate: `make generate`
5. Test again

### Rate limiting issues

Check `RateLimitPerMinute` in config:
```go
unifi.ClientConfig{
    APIKey: apiKey,
    RateLimitPerMinute: 10000,  // v1: 10000, EA: 100
}
```

## Useful Commands

```bash
# Validate OpenAPI spec
yamllint openapi.yaml

# Check generated types
go doc -all github.com/lexfrei/go-unifi | grep "type.*struct"

# Count map[string]interface{} usage
grep -c 'map\[string\]interface{}' generated.go

# Find all exported types
grep -n '^type [A-Z]' generated.go

# Test example
UNIFI_API_KEY=key go run examples/get_host/main.go

# Build all examples
for dir in examples/*/; do go build -o /dev/null "$dir"; done
```

## References

- [UniFi Site Manager API Docs](https://developer.ui.com/site-manager-api/gettingstarted)
- [oapi-codegen Documentation](https://github.com/oapi-codegen/oapi-codegen)
- [OpenAPI 3.0 Specification](https://spec.openapis.org/oas/v3.0.3)
- [Go Error Handling Best Practices](https://github.com/cockroachdb/errors)
