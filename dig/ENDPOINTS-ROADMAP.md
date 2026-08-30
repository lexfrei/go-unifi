# Network API Endpoints Roadmap

## Scan Results Summary (2025-11-25)

Automated scan of 184 endpoints:

- ✅ **129 working** (HTTP 200)
- ❌ 7 not found (HTTP 404)
- ⚠️ 35 POST-only (HTTP 405)
- ⚠️ 13 require query params (HTTP 400)

Full results: `dig/api-responses/scan/summary.txt`

---

## Implementation Status (Updated 2025-11-26)

**Total endpoints in OpenAPI spec:** 173+

- v2 API: ~145 endpoints
- Integration API v1: ~20 endpoints
- Legacy Cmd API: 1 endpoint

### Implementation Level Legend

- ✅ **CRUD** - Full Create/Read/Update/Delete tested
- ✅ **GET** - Read-only endpoint in OpenAPI, wrapper method exists
- [ ] - Not implemented or not verified

---

## Already Implemented

### Full CRUD (tested Create/Update/Delete)

- ✅ **CRUD** static-dns
- ✅ **CRUD** firewall-policies
- ✅ **CRUD** trafficrules
- ✅ **CRUD** hotspot/vouchers (Integration API)

### GET only (wrapper methods, not tested for mutations)

All other endpoints have OpenAPI schemas and GET wrapper methods, but Create/Update/Delete not tested:

- ✅ **GET** aggregated-dashboard, topology, device, clients/active, clients/history
- ✅ **GET** wifi-stats/aps, wifi-connectivity, wlan-capabilities
- ✅ **GET** firewall/zone, firewall/zone-matrix, firewall-rules, firewall-app-blocks
- ✅ **GET** isp/status, isp/health, isp/health/compact
- ✅ **GET** speedtest, speedtest/latest, speedtest/latest-per-wan
- ✅ **GET** vpn/connections, vpn/client-connections, vpn/l2tp/defaults
- ✅ **GET** trafficroutes, nat, active-leases
- ✅ **GET** alert, warnings, notifications, ips_alerts
- ✅ **GET** apgroups, radius/profiles, radius/users
- ✅ **GET** content-filtering, content-filtering/categories
- ✅ **GET** acl-rules, qos-rules
- ✅ **GET** features, teleport/token, teleport/invitation-history
- ✅ **GET** hotspot/info, hotspot/clients
- ✅ **GET** clients/traffic-control
- ✅ **GET** wireguard/users, wireguard/users/existing-subnets
- ✅ **GET** bgp/config, bgp/config/all, ospf/router, ospf/neighbors
- ✅ **GET** ssl-inspection/* (9 endpoints)
- ✅ **GET** system-log/* (17 POST endpoints for queries)
- ✅ **GET** settings/* (~15 endpoints)
- ✅ **GET** lan/wan/wlan enriched-configuration and defaults
- ✅ **GET** ~50 more low-priority endpoints

---

## Priority 1: Core Networking (High Value, Common Use)

### Device Management

- [x] `/device` - ✅ GET works, response captured
- [x] `/device/wireless-links` - ✅ GET works
- [ ] `/device/{mac}` - ❌ v2 API returns 404; use legacy `api/s/default/stat/device/{mac}`

### Network Configuration

- [x] `/lan/enriched-configuration` - ✅ GET
- [x] `/wan/enriched-configuration` - ✅ GET
- [x] `/wlan/enriched-configuration` - ✅ GET
- [x] `/lan/defaults` - ✅ GET
- [x] `/wan/defaults` - ✅ GET
- [x] `/wlan/defaults` - ✅ GET

### NAT Rules

- [x] `/nat` - ✅ GET (CRUD not tested)
- [ ] `/nat/{id}` - needs NAT rule ID

### Topology

- [x] `/topology` - ✅ GET works, response captured

---

## Priority 2: Monitoring & Stats

### Clients

- [x] `/clients/active` - ✅ GET works, response captured
- [x] `/clients/history` - ✅ GET
- [x] `/clients/traffic-control` - ✅ GET
- [x] `/clients/active/{clientMac}` - ✅ GET (e.g., `36:e9:55:9b:9a:a7`)
- [ ] `/clients/metadata` - ⚠️ 405 (POST only)

### ISP/WAN Status

- [x] `/isp/status` - ✅ GET
- [x] `/isp/health` - ✅ GET
- [x] `/isp/health/compact` - ✅ GET

### Speedtest

- [x] `/speedtest` - ✅ GET
- [x] `/speedtest/latest` - ✅ GET
- [x] `/speedtest/latest-per-wan` - ✅ GET
- [ ] `/speedtest/csv` - ⚠️ 406 (needs Accept header?)

### Traffic

- [ ] `/traffic` - ⚠️ 400 (needs query params) - **TODO: add query params**
- [ ] `/traffic-rate` - ⚠️ 400 (needs query params) - **TODO: add query params**
- [ ] `/traffic-flows` - ⚠️ 405 (POST only) - **TODO: POST endpoint**
- [x] `/traffic-flows/filter-data` - ✅ GET
- [ ] `/traffic-flow-latest-statistics` - ⚠️ 400 (needs query params) - **TODO: add query params**

---

## Priority 3: Security

### Firewall Zones

- [x] `/firewall/zone` - ✅ GET
- [x] `/firewall/zone-matrix` - ✅ GET
- [x] `/firewall/zone/defaults` - ✅ GET
- [ ] `/firewall/zone/{zoneId}` - ⚠️ 405 (PUT/DELETE only, no GET)

### Firewall Rules

- [x] `/firewall-policies` - ✅ **CRUD** (tested)
- [x] `/firewall-policies/defaults` - ✅ GET
- [x] `/firewall-rules/defaults` - ✅ GET
- [x] `/firewall-rules/combined-traffic-firewall-rules` - ✅ GET
- [x] `/firewall-app-blocks` - ✅ GET

### IPS/IDS

- [x] `/ips_alerts` - ✅ GET

### Content Filtering

- [x] `/content-filtering` - ✅ GET
- [x] `/content-filtering/categories` - ✅ GET
- [ ] `/content-filtering/{id}` - needs rule ID

### SSL Inspection

- [x] `/ssl-inspection/setting` - ✅ GET
- [x] `/ssl-inspection/setting/defaults` - ✅ GET
- [x] `/ssl-inspection/categories` - ✅ GET
- [x] `/ssl-inspection/certificates` - ✅ GET
- [x] `/ssl-inspection/certificates/active` - ✅ GET
- [x] `/ssl-inspection/profiles` - ✅ GET
- [x] `/ssl-inspection/profiles/defaults` - ✅ GET
- [x] `/ssl-inspection/file-extensions` - ✅ GET
- [x] `/ssl-inspection/search-engines` - ✅ GET
- [ ] `/ssl-inspection/applications` - ⚠️ 405 (POST only) - **TODO: POST endpoint**

---

## Priority 4: VPN

### WireGuard

- [x] `/wireguard/users` - ✅ GET
- [x] `/wireguard/users/existing-subnets` - ✅ GET
- [ ] `/wireguard/{networkId}/users` - needs network ID

### OpenVPN

- [ ] `/vpn/openvpn/configuration` - ⚠️ 405 (POST only?) - **TODO: POST endpoint**
- [ ] `/vpn/openvpn/certificates` - ⚠️ 405 (POST only?) - **TODO: POST endpoint**

### VPN Connections

- [x] `/vpn/connections` - ✅ GET
- [x] `/vpn/client-connections` - ✅ GET
- [x] `/vpn/l2tp/defaults` - ✅ GET

### Teleport

- [x] `/teleport/token` - ✅ GET
- [x] `/teleport/invitation-history` - ✅ GET
- [ ] `/teleport/client/{clientId}` - needs client ID

---

## Priority 5: WiFi Management

### AP Groups

- [x] `/apgroups` - ✅ GET
- [ ] `/apgroups/{id}` - needs group ID

### WiFi Stats

- [x] `/wifi-stats/aps` - ✅ GET (requires `?interval=hourly&start={ms}&end={ms}`)
- [ ] `/wifi-stats/radios` - requires `interval`, `start`, `end` query params - **TODO: add query params**
- [ ] `/wifi-stats/channelization` - requires `interval`, `start`, `end` query params - **TODO: add query params**
- [ ] `/wifi-stats/details` - requires `interval`, `start`, `end` query params - **TODO: add query params**

### Other WiFi

- [x] `/wifi-connectivity` - ✅ GET
- [x] `/wifiman` - ✅ GET
- [x] `/wlan-capabilities` - ✅ GET

---

## Priority 6: Advanced Routing

### BGP

- [x] `/bgp/config` - ✅ GET
- [x] `/bgp/config/all` - ✅ GET
- [ ] `/bgp/config/{deviceMac}` - needs device MAC

### OSPF

- [x] `/ospf/router` - ✅ GET
- [x] `/ospf/neighbors` - ✅ GET
- [ ] `/ospf/router/{id}` - needs router ID

### Traffic Routes

- [x] `/trafficroutes` - ✅ GET
- [ ] `/trafficroutes/{routeId}` - needs route ID
- [ ] `/trafficroutes/{routeId}/enable` - POST - **TODO: POST endpoint**
- [ ] `/trafficroutes/{routeId}/disable` - POST - **TODO: POST endpoint**

### QoS Rules

- [x] `/qos-rules` - ✅ GET
- [ ] `/qos-rules/{id}` - needs rule ID
- [ ] `/qos-rules/batch-reorder` - POST - **TODO: POST endpoint**

---

## Priority 7: System & Logs

### System Logs

- [x] `/system-log/all` - ✅ GET (POST with `{}` body)
- [x] `/system-log/count` - ✅ GET
- [x] `/system-log/critical` - ✅ GET
- [x] `/system-log/threats` - ✅ GET
- [x] `/system-log/admin-access` - ✅ GET
- [x] `/system-log/admin-activity` - ✅ GET
- [x] `/system-log/client-alert` - ✅ GET
- [x] `/system-log/device-alert` - ✅ GET
- [x] `/system-log/threat-alert` - ✅ GET
- [x] `/system-log/update-alert` - ✅ GET
- [x] `/system-log/vpn-alert` - ✅ GET
- [x] `/system-log/next-ai-alert` - ✅ GET
- [x] `/system-log/system-critical-alert` - ✅ GET
- [x] `/system-log/setting` - ✅ GET
- [x] `/system-log/setting/defaults` - ✅ GET
- [x] `/system-log/remote-settings` - ✅ GET
- [x] `/system-log/display-options/*` - ✅ GET (4 endpoints)

### Settings

- [ ] `/settings/mgmt` - ❌ 404 (use legacy API)
- [x] `/settings/connectivity/defaults` - ✅ GET
- [x] `/settings/doh/defaults` - ✅ GET
- [x] `/settings/doh/available-server-names` - ✅ GET
- [x] `/settings/*` - ✅ GET (~15 endpoints)

### Alerts

- [x] `/alert` - ✅ GET
- [x] `/warnings` - ✅ GET
- [x] `/notifications` - ✅ GET

---

## Priority 8: Specialized Features

### DHCP

- [x] `/active-leases` - ✅ GET
- [ ] `/active-leases/{networkId}` - needs network ID

### PoE

- [ ] `/poe-info` - ⚠️ 400 (needs query params?)

### Ports

- [ ] `/ports/mac-tables` - ⚠️ 405 (POST only) - **TODO: POST endpoint**
- [x] `/ports/port-anomalies` - ✅ GET
- [x] `/ports/system-logs` - ✅ GET (POST)
- [x] `/port-profiles/defaults` - ✅ GET

### RADIUS

- [x] `/radius/profiles` - ✅ GET
- [x] `/radius/users` - ✅ GET

### ACL

- [x] `/acl-rules` - ✅ GET

---

## Low Priority / Rarely Used

All implemented:

- [x] `/floorplan` - ✅ GET
- [x] `/features` - ✅ GET
- [x] `/described-features` - ✅ GET
- [x] `/vendor-ids` - ✅ GET
- [x] `/gateway/engine/*` - ✅ GET (4 endpoints)
- [x] `/hotspot/clients` - ✅ GET
- [x] `/hotspot/info` - ✅ GET
- [x] `/loop-detection/info` - ✅ GET
- [x] `/magicsitetositevpn/configs` - ✅ GET
- [x] `/mclag-groups` - ✅ GET
- [x] `/network-members-groups` - ✅ GET
- [x] `/network/port-suggest` - ✅ GET
- [x] `/network/suggest` - ✅ GET
- [x] `/object-oriented-network-configs` - ✅ GET
- [x] `/search` - ✅ GET
- [x] `/shadowmode/info` - ✅ GET
- [x] `/stacking` - ✅ GET
- [x] `/ulp/users_groups` - ✅ GET
- [x] `/utilization/last_days` - ✅ GET
- [x] `/wan-slas` - ✅ GET
- [x] `/next-ai/logs` - ✅ GET

Not found (404) - classified:

**"Local UCore only method"** (available via legacy API `api/s/default/rest/...`):

- `/settings/mgmt` → use `api/s/default/rest/setting/mgmt`
- `/shadowmode/status` → internal UCore only
- `/smart-subnet` → internal UCore only
- `/uid/radius-server` → internal UCore only
- `/uid/vpn-server` → internal UCore only
- `/uid/wlan` → internal UCore only

**"Supported device not found"** (requires specific hardware/subscription):

- `/wan/magic/subscription` → requires UniFi Magic WAN subscription

POST-only (405) - **tested and working:**

- `/system-log/*` - POST with `{}` body returns logs
- `/dpi` - use legacy API `api/s/default/stat/dpi` instead
- `/alias` - POST only
- `/fingerprint/*` - POST only
- `/ports/mac-tables` - POST only
- `/visual-programming/*` - POST only

---

## Implementation Strategy

### Remaining Work (Updated 2025-11-26)

**Endpoints requiring query parameters (HTTP 400):**

- [ ] `/traffic` - needs `start`, `end` params
- [ ] `/traffic-rate` - needs `start`, `end` params
- [ ] `/traffic-flow-latest-statistics` - needs `start`, `end` params
- [ ] `/wifi-stats/radios` - needs `interval`, `start`, `end` params
- [ ] `/wifi-stats/channelization` - needs `interval`, `start`, `end` params
- [ ] `/wifi-stats/details` - needs `interval`, `start`, `end` params
- [ ] `/poe-info` - needs query params (unknown)

**POST-only endpoints (HTTP 405):**

- [ ] `/traffic-flows` - POST endpoint for traffic flow queries
- [ ] `/ssl-inspection/applications` - POST endpoint
- [ ] `/vpn/openvpn/configuration` - POST endpoint
- [ ] `/vpn/openvpn/certificates` - POST endpoint
- [ ] `/ports/mac-tables` - POST endpoint
- [ ] `/qos-rules/batch-reorder` - POST endpoint
- [ ] `/trafficroutes/{routeId}/enable` - POST endpoint
- [ ] `/trafficroutes/{routeId}/disable` - POST endpoint

**Parametrized endpoints (need IDs):**

- [ ] `/device/{mac}` - ❌ 404 in v2, use legacy API
- [ ] `/nat/{id}` - needs NAT rule ID
- [ ] `/content-filtering/{id}` - needs rule ID
- [ ] `/firewall/zone/{zoneId}` - PUT/DELETE only
- [ ] `/wireguard/{networkId}/users` - needs network ID
- [ ] `/teleport/client/{clientId}` - needs client ID
- [ ] `/apgroups/{id}` - needs group ID
- [ ] `/bgp/config/{deviceMac}` - needs device MAC
- [ ] `/ospf/router/{id}` - needs router ID
- [ ] `/trafficroutes/{routeId}` - needs route ID
- [ ] `/qos-rules/{id}` - needs rule ID
- [ ] `/active-leases/{networkId}` - needs network ID

**Special cases:**

- [ ] `/speedtest/csv` - needs Accept header (HTTP 406)
- [ ] `/settings/mgmt` - ❌ 404, use legacy API `api/s/default/rest/setting/mgmt`

### Response Locations

All scan responses saved to: `dig/api-responses/scan/`

Key files:

- `firewall-zone.json` - 6 zones
- `active-leases.json` - DHCP leases with device info
- `isp-status.json` - large ISP monitoring data
- `lan-enriched-configuration.json` - full LAN config
- `wan-enriched-configuration.json` - full WAN config
- `wlan-enriched-configuration.json` - WiFi configuration

Parameterized endpoint responses: `dig/api-responses/scan/parameterized/`

- `client-by-mac.json` - single client details
- `system-log-all.json` - system logs (POST response)
- `wifi-stats-aps.json` - AP statistics with time range

### Legacy API Fallbacks

Some v2 endpoints return 404/405 but work via legacy API:

| v2 Endpoint | Legacy Alternative |
|-------------|-------------------|
| `/device/{mac}` (404) | `api/s/default/stat/device/{mac}` |
| `/dpi` (405) | `api/s/default/stat/dpi` |
| `/settings/mgmt` (404) | `api/s/default/rest/setting/mgmt` |
| All settings | `api/s/default/rest/setting` |
