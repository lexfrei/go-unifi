#!/bin/bash
# Scan UniFi Network API v2 endpoints that require parameters
# - GET endpoints with query parameters (wifi-stats, traffic, etc.)
# - POST endpoints (system-log, dpi, etc.)
#
# Usage: cd dig && ./scan-parametrized.sh
#
# Output:
#   dig/api-responses/parametrized/{endpoint}.json - individual responses
#   dig/api-responses/parametrized/status.log      - HTTP status codes

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_KEY_FILE="$SCRIPT_DIR/.api-key"
OUTPUT_DIR="$SCRIPT_DIR/api-responses/parametrized"
STATUS_LOG="$OUTPUT_DIR/status.log"

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
    echo "Create the file with your UniFi API key"
    exit 1
fi

API_KEY=$(cat "$API_KEY_FILE")

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Initialize logs
echo "# Parametrized scan started at $(date -Iseconds)" > "$STATUS_LOG"
echo "" >> "$STATUS_LOG"

# Timestamps: last 24 hours in milliseconds
END_MS=$(($(date +%s) * 1000))
START_MS=$((END_MS - 86400000))

echo -e "${BLUE}Time range: $(date -r $((START_MS / 1000))) to $(date -r $((END_MS / 1000)))${NC}"
echo ""

# Helper function for GET requests with query params
scan_get() {
    local endpoint="$1"
    local params="$2"
    local filename="$3"
    local output_file="$OUTPUT_DIR/${filename}.json"

    printf "  GET  %-50s " "$endpoint"

    http_code=$(curl --silent --insecure \
        --output "$output_file" \
        --write-out "%{http_code}" \
        --max-time 30 \
        "$BASE_URL/v2/api/site/$SITE/${endpoint}${params}" \
        -H "X-API-KEY: $API_KEY" \
        -H "Accept: application/json")

    echo "$http_code GET $endpoint$params" >> "$STATUS_LOG"

    if [[ "$http_code" == "200" ]]; then
        echo -e "${GREEN}$http_code${NC}"
        return 0
    else
        echo -e "${RED}$http_code${NC}"
        return 1
    fi
}

# Helper function for POST requests
scan_post() {
    local endpoint="$1"
    local body="$2"
    local filename="$3"
    local output_file="$OUTPUT_DIR/${filename}.json"

    printf "  POST %-50s " "$endpoint"

    http_code=$(curl --silent --insecure \
        --output "$output_file" \
        --write-out "%{http_code}" \
        --max-time 30 \
        -X POST \
        "$BASE_URL/v2/api/site/$SITE/$endpoint" \
        -H "X-API-KEY: $API_KEY" \
        -H "Accept: application/json" \
        -H "Content-Type: application/json" \
        -d "$body")

    echo "$http_code POST $endpoint" >> "$STATUS_LOG"

    if [[ "$http_code" == "200" ]]; then
        echo -e "${GREEN}$http_code${NC}"
        return 0
    else
        echo -e "${RED}$http_code${NC}"
        return 1
    fi
}

# Counters
success=0
failed=0

echo -e "${BLUE}=== Category A: GET with Query Parameters ===${NC}"
echo ""

echo -e "${YELLOW}A1. WiFi Stats (interval, start, end)${NC}"
for ep in aps radios channelization details; do
    if scan_get "wifi-stats/$ep" "?interval=hourly&start=$START_MS&end=$END_MS" "wifi-stats-$ep"; then
        ((success++))
    else
        ((failed++))
    fi
done
echo ""

echo -e "${YELLOW}A2. Traffic Stats (start, end)${NC}"
for ep in traffic traffic-rate traffic-flow-latest-statistics; do
    if scan_get "$ep" "?start=$START_MS&end=$END_MS" "$ep"; then
        ((success++))
    else
        ((failed++))
    fi
done
echo ""

echo -e "${YELLOW}A3. Other GET with params${NC}"
scan_get "country-traffic" "?start=$START_MS&end=$END_MS" "country-traffic" && ((success++)) || ((failed++))
scan_get "utilization/time_range" "?start=$START_MS&end=$END_MS" "utilization-time_range" && ((success++)) || ((failed++))
scan_get "poe-info" "" "poe-info" && ((success++)) || ((failed++))
echo ""

echo -e "${BLUE}=== Category B: POST Endpoints ===${NC}"
echo ""

echo -e "${YELLOW}B1. System Logs (POST with pagination)${NC}"
SYSTEM_LOG_BODY='{"page_number":0,"page_size":10}'
for ep in all count critical threats admin-access admin-activity client-alert device-alert threat-alert update-alert vpn-alert next-ai-alert system-critical-alert; do
    if scan_post "system-log/$ep" "$SYSTEM_LOG_BODY" "system-log-$ep"; then
        ((success++))
    else
        ((failed++))
    fi
done
echo ""

echo -e "${YELLOW}B2. DPI & Traffic Flows${NC}"
scan_post "dpi" '{}' "dpi" && ((success++)) || ((failed++))
scan_post "traffic-flows" '{}' "traffic-flows" && ((success++)) || ((failed++))
scan_post "clients/metadata" '{}' "clients-metadata" && ((success++)) || ((failed++))
echo ""

echo -e "${YELLOW}B3. VPN Config${NC}"
scan_post "vpn/openvpn/configuration" '{}' "vpn-openvpn-configuration" && ((success++)) || ((failed++))
scan_post "vpn/openvpn/certificates" '{}' "vpn-openvpn-certificates" && ((success++)) || ((failed++))
echo ""

echo -e "${YELLOW}B4. Ports & Misc${NC}"
scan_post "ports/mac-tables" '{}' "ports-mac-tables" && ((success++)) || ((failed++))
scan_post "ports/system-logs" '{}' "ports-system-logs" && ((success++)) || ((failed++))
scan_post "alias" '{}' "alias" && ((success++)) || ((failed++))
scan_post "fingerprint/assets" '{}' "fingerprint-assets" && ((success++)) || ((failed++))
scan_post "ssl-inspection/applications" '{}' "ssl-inspection-applications" && ((success++)) || ((failed++))
echo ""

# Summary
total=$((success + failed))
echo -e "${BLUE}=== Summary ===${NC}"
echo "Total endpoints scanned: $total"
echo -e "  Success: ${GREEN}$success${NC}"
echo -e "  Failed:  ${RED}$failed${NC}"
echo ""
echo -e "Output directory: ${YELLOW}$OUTPUT_DIR${NC}"
echo -e "Status log:       ${YELLOW}$STATUS_LOG${NC}"
