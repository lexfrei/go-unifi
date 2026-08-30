#!/bin/bash
# Mass scan of UniFi Network API v2 GET endpoints
# Captures responses for analysis and OpenAPI schema generation
#
# Usage: cd dig && ./scan-endpoints.sh
#
# Output:
#   dig/api-responses/scan/{endpoint}.json - individual responses
#   dig/api-responses/scan/status.log      - HTTP status codes
#   dig/api-responses/scan/summary.txt     - final summary

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_KEY_FILE="$SCRIPT_DIR/.api-key"
OUTPUT_DIR="$SCRIPT_DIR/api-responses/scan"
STATUS_LOG="$OUTPUT_DIR/status.log"
SUMMARY_FILE="$OUTPUT_DIR/summary.txt"

# Configuration
BASE_URL="https://172.16.0.1/proxy/network"
SITE="default"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check API key
if [[ ! -f "$API_KEY_FILE" ]]; then
    echo -e "${RED}Error: API key file not found at $API_KEY_FILE${NC}"
    exit 1
fi

API_KEY=$(cat "$API_KEY_FILE")

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Initialize logs
echo "# Scan started at $(date -Iseconds)" > "$STATUS_LOG"
echo "" >> "$STATUS_LOG"

# Simple GET endpoints (no path parameters required)
# Extracted from all-api-site-endpoints.txt
ENDPOINTS=(
    # Core device/client endpoints
    "device"
    "device/wireless-links"
    "clients/active"
    "clients/history"
    "clients/metadata"
    "clients/traffic-control"
    "topology"

    # Network configuration
    "lan/enriched-configuration"
    "lan/defaults"
    "lan/mdns"
    "wan/enriched-configuration"
    "wan/defaults"
    "wan/ddns"
    "wan/networkgroups"
    "wan/load-balancing/configuration"
    "wan/load-balancing/status"
    "wan/provider-capabilities"
    "wan/provider-capabilities/legacy"
    "wan/magic/configuration"
    "wan/magic/subscription"
    "wlan/enriched-configuration"
    "wlan/defaults"
    "wlan-capabilities"

    # Firewall
    "firewall-policies"
    "firewall-policies/defaults"
    "firewall-rules/combined-traffic-firewall-rules"
    "firewall-rules/defaults"
    "firewall/zone"
    "firewall/zone-matrix"
    "firewall/zone/defaults"
    "firewall-app-blocks"

    # NAT & Port forwarding
    "nat"

    # Traffic & Routing
    "trafficrules"
    "trafficroutes"
    "traffic"
    "traffic-rate"
    "traffic-flow-latest-statistics"
    "traffic-flows"
    "traffic-flows/filter-data"
    "qos-rules"

    # DNS
    "static-dns"
    "static-dns/devices"

    # VPN
    "vpn/connections"
    "vpn/client-connections"
    "vpn/openvpn/configuration"
    "vpn/openvpn/certificates"
    "vpn/l2tp/defaults"
    "wireguard/users"
    "wireguard/users/existing-subnets"
    "teleport/token"
    "teleport/invitation-history"

    # WiFi
    "apgroups"
    "wifi-stats/aps"
    "wifi-stats/radios"
    "wifi-stats/channelization"
    "wifi-stats/details"
    "wifi-connectivity"
    "wifiman"

    # Monitoring & Status
    "isp/status"
    "isp/health"
    "isp/health/compact"
    "speedtest"
    "speedtest/latest"
    "speedtest/latest-per-wan"
    "speedtest/csv"
    "aggregated-dashboard"
    "dashboard"
    "network_status"
    "score"

    # System logs
    "system-log/all"
    "system-log/count"
    "system-log/critical"
    "system-log/threats"
    "system-log/admin-access"
    "system-log/admin-activity"
    "system-log/client-alert"
    "system-log/device-alert"
    "system-log/threat-alert"
    "system-log/update-alert"
    "system-log/vpn-alert"
    "system-log/next-ai-alert"
    "system-log/system-critical-alert"
    "system-log/ap-logs"
    "system-log/ap-logs/display-options/aps"
    "system-log/display-options/admins"
    "system-log/threat/display-options/clients"
    "system-log/triggers"
    "system-log/triggers/display-options/hosts"
    "system-log/network-ai/logs"
    "system-log/setting"
    "system-log/setting/defaults"
    "system-log/remote-settings"

    # Alerts & Notifications
    "alert"
    "warnings"
    "notifications"
    "ips_alerts"

    # Settings
    "settings/mgmt"
    "settings/connectivity/defaults"
    "settings/doh/defaults"
    "settings/doh/available-server-names"
    "settings/element_adopt/defaults"
    "settings/global_nat/defaults"
    "settings/global_switch/defaults"
    "settings/ips/advanced-filtering-auto-values"
    "settings/ips/advanced-filtering-defaults"
    "settings/ips/available-categories"
    "settings/netflow/defaults"
    "settings/ntp/defaults"
    "settings/roaming_assistant/defaults"
    "settings/shortcuts"
    "settings/teleport/defaults"
    "settings/traffic_flow/defaults"
    "settings/usg/defaults"
    "settings/wifiai/defaults"

    # SSL Inspection
    "ssl-inspection/setting"
    "ssl-inspection/setting/defaults"
    "ssl-inspection/applications"
    "ssl-inspection/categories"
    "ssl-inspection/certificates"
    "ssl-inspection/certificates/active"
    "ssl-inspection/file-extensions"
    "ssl-inspection/profiles"
    "ssl-inspection/profiles/defaults"
    "ssl-inspection/search-engines"

    # Content Filtering
    "content-filtering"
    "content-filtering/categories"

    # DPI
    "dpi"
    "dpi/latest-client-stats"

    # RADIUS
    "radius/profiles"
    "radius/users"

    # Routing protocols
    "bgp/config"
    "bgp/config/all"
    "ospf/router"
    "ospf/neighbors"

    # DHCP
    "active-leases"

    # Ports & PoE
    "poe-info"
    "ports/mac-tables"
    "ports/port-anomalies"
    "ports/system-logs"
    "port-profiles/defaults"

    # ACL
    "acl-rules"

    # Misc
    "alias"
    "features"
    "described-features"
    "fingerprint/assets"
    "fingerprint_override"
    "missing_fingerprint"
    "vendor-ids"
    "country-traffic"
    "app-traffic-rate"
    "excluded-ips"
    "floorplan"
    "gateway/engine/features"
    "gateway/engine/logs"
    "gateway/engine/most-active-networks"
    "gateway/engine/utilization"
    "global/config/network"
    "hotspot/clients"
    "hotspot/info"
    "insights/filtering/overview"
    "insights/filtering/watchlist"
    "loop-detection/info"
    "magicsitetositevpn/configs"
    "mclag-groups"
    "network-members-group"
    "network-members-groups"
    "network/port-suggest"
    "network/suggest"
    "object-oriented-network-config"
    "object-oriented-network-configs"
    "search"
    "shadowmode/info"
    "shadowmode/status"
    "smart-subnet"
    "stacking"
    "uid/radius-server"
    "uid/vpn-server"
    "uid/wlan"
    "ulp/users_groups"
    "utilization/last_days"
    "utilization/time_range"
    "visual-programming/virtual-network"
    "wan-slas"
    "next-ai/logs"
)

# Counters
total=${#ENDPOINTS[@]}
success=0
failed=0
not_found=0
unauthorized=0

echo -e "${BLUE}Starting scan of $total endpoints...${NC}"
echo ""

for i in "${!ENDPOINTS[@]}"; do
    endpoint="${ENDPOINTS[$i]}"
    # Convert slashes to dashes for filename
    filename=$(echo "$endpoint" | tr '/' '-')
    output_file="$OUTPUT_DIR/${filename}.json"

    # Progress indicator
    progress=$((i + 1))
    printf "\r[%3d/%3d] Scanning: %-60s" "$progress" "$total" "$endpoint"

    # Make the request
    http_code=$(curl --silent --insecure \
        --output "$output_file" \
        --write-out "%{http_code}" \
        --max-time 30 \
        "$BASE_URL/v2/api/site/$SITE/$endpoint" \
        -H "X-API-KEY: $API_KEY" \
        -H "Accept: application/json")

    # Log the result
    echo "$http_code $endpoint" >> "$STATUS_LOG"

    # Count by status
    case "$http_code" in
        200)
            ((success++))
            ;;
        401|403)
            ((unauthorized++))
            ;;
        404)
            ((not_found++))
            ;;
        *)
            ((failed++))
            ;;
    esac
done

echo ""
echo ""
echo -e "${BLUE}Scan complete!${NC}"
echo ""

# Generate summary
cat > "$SUMMARY_FILE" << EOF
# UniFi Network API v2 Scan Summary
# Generated: $(date -Iseconds)
# Base URL: $BASE_URL
# Site: $SITE

## Results

Total endpoints scanned: $total
- Success (200):        $success
- Not Found (404):      $not_found
- Unauthorized (401/3): $unauthorized
- Other errors:         $failed

## Working Endpoints (200)

EOF

grep "^200 " "$STATUS_LOG" | awk '{print $2}' >> "$SUMMARY_FILE"

cat >> "$SUMMARY_FILE" << EOF

## Not Found (404)

EOF

grep "^404 " "$STATUS_LOG" | awk '{print $2}' >> "$SUMMARY_FILE"

cat >> "$SUMMARY_FILE" << EOF

## Unauthorized (401/403)

EOF

grep -E "^(401|403) " "$STATUS_LOG" | awk '{print $2}' >> "$SUMMARY_FILE" 2>/dev/null || echo "(none)" >> "$SUMMARY_FILE"

cat >> "$SUMMARY_FILE" << EOF

## Other Errors

EOF

grep -vE "^(200|401|403|404|#) " "$STATUS_LOG" | grep -v "^$" >> "$SUMMARY_FILE" 2>/dev/null || echo "(none)" >> "$SUMMARY_FILE"

# Print summary
echo -e "${GREEN}Results:${NC}"
echo "  Success (200):        $success"
echo "  Not Found (404):      $not_found"
echo "  Unauthorized (401/3): $unauthorized"
echo "  Other errors:         $failed"
echo ""
echo -e "Output directory: ${YELLOW}$OUTPUT_DIR${NC}"
echo -e "Status log:       ${YELLOW}$STATUS_LOG${NC}"
echo -e "Summary:          ${YELLOW}$SUMMARY_FILE${NC}"
