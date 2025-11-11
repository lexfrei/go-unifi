# test-reality - Type Safety Validator

Tool for testing go-unifi against real UniFi API to find type mismatches, `any` usage, and other type safety issues.

## What it does

- Makes real API calls to all endpoints
- Deserializes responses into our generated types
- Finds fields typed as `any`/`interface{}`
- Detects potential type mismatches
- Reports empty/nil optional fields
- Validates both v1 and EA endpoints

## Usage

```bash
# Build
go build ./cmd/test-reality

# Run with API key from environment
UNIFI_API_KEY=your-key ./test-reality

# Run with API key flag
./test-reality -api-key your-key

# Verbose mode (shows full JSON samples)
./test-reality -verbose
```

## Output Example

```
🧪 Testing go-unifi against reality...
============================================================

📊 Test Summary
============================================================

✅ ListHosts (v1) (HTTP 200, 123ms)

⚠️  ListSites (v1) (HTTP 200, 89ms)
   ⚠️  Fields typed as 'any': 2
      - Site.Meta.AdditionalProperties
      - Site.UserData
   ⚠️  Type issues: 1
      - Site.Statistics.TotalDevices should be int, not *int

❌ GetISPMetrics (EA) (HTTP 500, 234ms)
   Error: failed to deserialize response

============================================================
⚠️  Found 3 potential type issues

Recommendations:
  1. Replace 'any' with concrete types in OpenAPI spec
  2. Add oneOf/anyOf schemas for polymorphic fields
  3. Review optional fields - some might be required
```

## Development Workflow

1. Run test-reality against production API
2. Review reported issues
3. Update `api/sitemanager/openapi.yaml`
4. Regenerate code: `cd api/sitemanager && go generate`
5. Run test-reality again to verify fixes
6. Commit changes

## Common Issues Found

### `any` fields

Fields typed as `interface{}` or `any` indicate missing schema definition:

```yaml
# Bad
additionalProperties: {}

# Good
additionalProperties:
  type: object
  properties:
    key:
      type: string
```

### Optional vs Required

Pointer fields that are always populated might not need to be optional:

```yaml
# If always populated, make required
properties:
  name:
    type: string  # not nullable
required:
  - name
```

### Polymorphic Types

Use `oneOf`/`anyOf` for fields with multiple possible types:

```yaml
data:
  oneOf:
    - $ref: '#/components/schemas/TypeA'
    - $ref: '#/components/schemas/TypeB'
```
