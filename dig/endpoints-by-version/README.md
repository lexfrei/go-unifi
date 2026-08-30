# UniFi Network API Versions

## API Version Summary

| Version | Path Pattern | Site Param | Auth | Status |
|---------|-------------|------------|------|--------|
| Integration v1 | `/integration/v1/sites/{siteId}/...` | UUID | X-API-KEY | Official, documented |
| Internal v2 | `/v2/api/site/{site}/...` or `/api/site/{siteName}/...` | string (e.g., "default") | X-API-KEY | Internal, undocumented |
| Legacy REST | `/api/s/{site}/rest/...` | string | Cookie session | Deprecated |
| Legacy Stat | `/api/s/{site}/stat/...` | string | Cookie session | Deprecated |
| Legacy Cmd | `/api/s/{site}/cmd/...` | string | Cookie session | Deprecated |

## Integration v1 (Official)

Source: `dig/network-integration-api-9.5.21.json`

Official, documented API endpoints:
- `/integration/v1/info` - Application info
- `/integration/v1/sites` - List sites
- `/integration/v1/sites/{siteId}/devices` - List devices
- `/integration/v1/sites/{siteId}/devices/{deviceId}` - Device details
- `/integration/v1/sites/{siteId}/devices/{deviceId}/statistics/latest` - Device stats
- `/integration/v1/sites/{siteId}/devices/{deviceId}/actions` - Device actions
- `/integration/v1/sites/{siteId}/devices/{deviceId}/interfaces/ports/{portIdx}/actions` - Port actions
- `/integration/v1/sites/{siteId}/clients` - List clients
- `/integration/v1/sites/{siteId}/clients/{clientId}` - Client details
- `/integration/v1/sites/{siteId}/clients/{clientId}/actions` - Client actions
- `/integration/v1/sites/{siteId}/hotspot/vouchers` - List/Create/Delete vouchers
- `/integration/v1/sites/{siteId}/hotspot/vouchers/{voucherId}` - Get/Delete voucher

## Internal v2 (Target for Implementation)

Source: `dig/all-api-site-endpoints.txt` (349 endpoints)

Path format: `/api/site/{siteName}/...` which maps to `/v2/api/site/{site}/...`

These are the endpoints we want to cover. See `v2-api.txt` for full categorized list.

## Legacy REST/Stat/Cmd

Old API format, cookie-based auth. NOT a priority.

Path formats:
- `/api/s/{site}/rest/{resource}` - CRUD operations
- `/api/s/{site}/stat/{resource}` - Statistics
- `/api/s/{site}/cmd/{command}` - Commands/actions

These may still exist on the controller but are being replaced by v2.

## Notes

- The `all-api-site-endpoints.txt` file contains only v2 endpoints (no legacy)
- Integration v1 is implemented from official OpenAPI spec
- Internal v2 requires reverse engineering via SSH/browser dev tools
