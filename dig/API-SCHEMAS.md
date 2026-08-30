# UniFi Network API - Internal Schemas

This document describes the internal data structures found in UniFi Network 9.5.21.

## Data Sources

1. **MongoDB Collections** (`ace` database on port 27117)
2. **Java Class Introspection** (strings from `/usr/lib/unifi/lib/unifi`)
3. **OpenAPI Spec** (`/usr/lib/unifi/webapps/ROOT/api-docs/integration.json`)

## Static DNS Records

### MongoDB Collection: `static_dns`

```json
{
  "_id": "ObjectId",
  "site_id": "string",
  "enabled": "boolean",
  "record_type": "string (A|AAAA|CNAME|MX|NS|SRV|TXT)",
  "key": "string (hostname/domain)",
  "value": "string (IP/target)",
  "priority": "number (for MX/SRV)",
  "ttl": "number (seconds, 0 = default)",
  "weight": "number (for SRV)",
  "port": "number (for SRV)"
}
```

### Supported Record Types
- `A` - IPv4 address
- `AAAA` - IPv6 address
- `CNAME` - Canonical name
- `MX` - Mail exchange (uses priority)
- `NS` - Name server
- `SRV` - Service record (uses priority, weight, port)
- `TXT` - Text record

### API Endpoint: `/api/site/{siteName}/static-dns`

Methods: GET, POST, PUT, DELETE

## Network Configuration

### MongoDB Collection: `networkconf`

```json
{
  "_id": "ObjectId",
  "site_id": "string",
  "name": "string",
  "purpose": "string (wan|corporate|guest|remote-user-vpn|site-vpn|vlan-only)",
  "enabled": "boolean",

  // LAN settings
  "ip_subnet": "string (CIDR)",
  "domain_name": "string",
  "vlan_enabled": "boolean",
  "vlan": "number",
  "networkgroup": "string (LAN)",
  "firewall_zone_id": "string",

  // DHCP settings
  "dhcpd_enabled": "boolean",
  "dhcpd_start": "string (IP)",
  "dhcpd_stop": "string (IP)",
  "dhcpd_leasetime": "number (seconds)",
  "dhcpd_dns_enabled": "boolean",
  "dhcpd_dns_1": "string",
  "dhcpd_dns_2": "string",
  "dhcpd_dns_3": "string",
  "dhcpd_gateway_enabled": "boolean",

  // IPv6 settings
  "ipv6_enabled": "boolean",
  "ipv6_interface_type": "string (none|static|pd)",
  "ipv6_ra_enabled": "boolean",
  "dhcpdv6_enabled": "boolean",

  // WAN settings (purpose=wan)
  "wan_type": "string (dhcp|static|pppoe|disabled)",
  "wan_networkgroup": "string (WAN|WAN2)",
  "wan_dns_preference": "string (auto|manual)",
  "wan_dns1": "string",
  "wan_dns2": "string",
  "wan_load_balance_type": "string (weighted|failover-only)",
  "wan_load_balance_weight": "number",
  "wan_failover_priority": "number",

  // Other
  "is_nat": "boolean",
  "internet_access_enabled": "boolean",
  "network_isolation_enabled": "boolean",
  "igmp_snooping": "boolean",
  "upnp_lan_enabled": "boolean"
}
```

## Firewall Policy

### MongoDB Collection: `firewall_policy`

```json
{
  "_id": "ObjectId",
  "site_id": "string",
  "short_id": "number",
  "name": "string",
  "description": "string",
  "enabled": "boolean",
  "action": "string (ALLOW|BLOCK|DROP)",
  "index": "number (rule order)",

  "source": {
    "_class": "string (SOURCE_ANY|SOURCE_CLIENT|SOURCE_ZONE|SOURCE_IP|SOURCE_PORT)",
    "zone_id": "string",
    "matching_target": "string (ANY|CLIENT|IP|PORT)",
    "client_macs": ["string"],
    "ip_addresses": ["string"],
    "ports": ["string"],
    "port_matching_type": "string (ANY|SPECIFIC)",
    "match_opposite_ports": "boolean"
  },

  "destination": {
    "_class": "string (DESTINATION_ANY|DESTINATION_ZONE|DESTINATION_IP|DESTINATION_PORT)",
    "zone_id": "string",
    "matching_target": "string (ANY|IP|PORT)",
    "ip_addresses": ["string"],
    "ports": ["string"],
    "port_matching_type": "string (ANY|SPECIFIC)",
    "match_opposite_ports": "boolean"
  },

  "ip_version": "string (BOTH|IPV4|IPV6)",
  "protocol": "string (all|tcp|udp|tcp_udp|icmp|protocol_number)",
  "icmp_typename": "string",
  "icmp_v6_typename": "string",
  "connection_state_type": "string (ALL|NEW|ESTABLISHED|RELATED|INVALID)",
  "connection_states": ["string"],
  "match_ip_sec": "boolean",
  "logging": "boolean",
  "create_allow_respond": "boolean",

  "schedule": {
    "mode": "string (ALWAYS|CUSTOM)",
    "repeat_on_days": ["string (MON|TUE|WED|THU|FRI|SAT|SUN)"],
    "time_range_start": "string (HH:MM)",
    "time_range_end": "string (HH:MM)",
    "all_day": "boolean"
  }
}
```

## Device

### MongoDB Collection: `device`

```json
{
  "_id": "ObjectId",
  "site_id": "string",
  "mac": "string",
  "ip": "string",
  "model": "string (e.g., USMINI, UDR7)",
  "shortname": "string",
  "type": "string (usw|uap|ugw|udm)",
  "version": "string (firmware)",
  "adopted": "boolean",
  "adoption_completed": "boolean",
  "adopted_at": "number (timestamp)",
  "connected_at": "number (timestamp)",
  "disconnected_at": "number (timestamp)",
  "provisioned_at": "number (timestamp)",
  "inform_ip": "string",
  "inform_url": "string",
  "cfgversion": "string (config hash)",

  "config_network": {
    "type": "string (dhcp|static)",
    "ip": "string"
  },

  "port_table": [{
    "port_idx": "number",
    "media": "string (GE|SFP|SFP+)",
    "speed": "number (Mbps)",
    "speed_caps": "number (bitmask)",
    "port_poe": "boolean",
    "last_connection": {
      "mac": "string",
      "connected_at": "number"
    }
  }],

  "ethernet_table": [{
    "name": "string (eth0)",
    "mac": "string",
    "num_port": "number",
    "other_macs": ["string"]
  }],

  "last_uplink": {
    "type": "string (wire|wireless)",
    "uplink_mac": "string",
    "uplink_device_name": "string",
    "uplink_remote_port": "number",
    "port_idx": "number"
  },

  "switch_caps": {
    "feature_caps": "number",
    "vlan_caps": "number",
    "max_mirror_sessions": "number",
    "max_aggregate_sessions": "number"
  },

  "serial": "string",
  "board_rev": "number",
  "manufacturer_id": "number",
  "sysid": "number",
  "satisfaction": "number (-1 = N/A)",
  "anomalies": "number (-1 = N/A)",
  "unsupported": "boolean",
  "model_in_eol": "boolean",
  "model_in_lts": "boolean",
  "has_fan": "boolean",
  "has_temperature": "boolean"
}
```

## Port Forward

### MongoDB Collection: `portforward`

```json
{
  "_id": "ObjectId",
  "site_id": "string",
  "name": "string",
  "enabled": "boolean",
  "src": "string (any|specific IP/network)",
  "dst_port": "string (port or range)",
  "fwd": "string (destination IP)",
  "fwd_port": "string (destination port)",
  "proto": "string (tcp|udp|tcp_udp)",
  "pfwd_interface": "string (wan|wan2|both)",
  "destination_ip": "string (WAN IP or any)",
  "log": "boolean"
}
```

## WLAN Configuration

### MongoDB Collection: `wlanconf`

```json
{
  "_id": "ObjectId",
  "site_id": "string",
  "name": "string (SSID)",
  "enabled": "boolean",
  "security": "string (open|wpapsk|wpaeap)",
  "wpa_mode": "string (wpa2|wpa3|wpa2wpa3)",
  "wpa_enc": "string (ccmp|gcmp)",
  "x_passphrase": "string (encrypted)",
  "hide_ssid": "boolean",
  "wlan_band": "string (2g|5g|both)",
  "wlan_bands": ["string (2g|5g|6g)"],
  "networkconf_id": "string (network reference)",
  "usergroup_id": "string",
  "ap_group_ids": ["string"],

  // Rate limiting
  "minrate_ng_enabled": "boolean",
  "minrate_ng_data_rate_kbps": "number",

  // Attributes
  "attr_no_delete": "boolean",
  "attr_no_edit": "boolean",
  "attr_hidden_id": "string"
}
```

## User/Client

### MongoDB Collection: `user`

```json
{
  "_id": "ObjectId",
  "site_id": "string",
  "mac": "string",
  "hostname": "string",
  "oui": "string (vendor prefix)",
  "is_guest": "boolean",
  "is_wired": "boolean",
  "first_seen": "number (timestamp)",
  "last_seen": "number (timestamp)",
  "disconnect_timestamp": "number",
  "last_ip": "string",
  "last_ipv6": "string",

  // Fingerprinting
  "fingerprint_engine_version": "string",
  "fingerprint_source": "number",
  "confidence": "number",
  "dev_cat": "number (device category)",
  "dev_family": "number",
  "dev_vendor": "number",
  "dev_id": "number",
  "os_name": "string",

  // Connection info
  "usergroup_id": "string",
  "wlanconf_id": "string",
  "last_connection_network_id": "string",
  "last_connection_network_name": "string",
  "last_uplink_mac": "string",
  "last_uplink_name": "string",
  "last_radio": "string (na|ng|6e)"
}
```

## Voucher (Hotspot)

### MongoDB Collection: `voucher`

```json
{
  "_id": "ObjectId",
  "site_id": "string",
  "code": "string (10-digit)",
  "create_time": "number (timestamp)",
  "duration": "number (minutes)",
  "note": "string",
  "quota": "number (0 = unlimited, >0 = multi-use)",
  "used": "number (times used)",
  "qos_overwrite": "boolean",
  "qos_usage_quota": "number (bytes)",
  "qos_rate_max_up": "number (kbps)",
  "qos_rate_max_down": "number (kbps)",
  "for_hotspot": "boolean",
  "admin_name": "string (creator)",
  "status": "string (VALID_ONE|VALID_MULTI|USED_SINGLE)"
}
```

## Settings

### MongoDB Collection: `setting`

Various settings with different schemas based on `key`:

```json
{
  "_id": "ObjectId",
  "site_id": "string",
  "key": "string (mgmt|guest_access|connectivity|etc)",
  "hostname": "string (for mgmt)",
  // ... other fields vary by setting type
}
```

## API Response Format

### Internal API (`/api/site/{site}/...`)

```json
{
  "meta": {
    "rc": "ok|error",
    "msg": "string (error message if rc=error)"
  },
  "data": [...]
}
```

### Integration API (`/integration/v1/...`)

```json
{
  "offset": 0,
  "limit": 25,
  "count": 10,
  "totalCount": 100,
  "data": [...]
}
```

Or for single items:
```json
{
  "id": "uuid",
  ...
}
```

## Authentication

### Internal API
- Cookie-based session (TOKEN cookie)
- Created via `/api/login` with username/password

### Integration API
- Header: `X-API-KEY: <api-key>`
- API keys managed in UniFi UI → Settings → Integrations

## Rate Limits

- Internal API: No documented limits
- Integration API: Varies by endpoint (typically 1000 req/min)
