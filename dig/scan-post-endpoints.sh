#!/bin/bash
# Scan 405 endpoints with POST method

BASE_URL="${UNIFI_BASE_URL:-https://192.168.1.1}"
API_KEY="${UNIFI_API_KEY}"
SITE="${UNIFI_SITE:-default}"

if [ -z "$API_KEY" ]; then
    echo "Error: UNIFI_API_KEY not set"
    exit 1
fi

OUTPUT_DIR="dig/api-responses/post-test"
mkdir -p "$OUTPUT_DIR"

# Endpoints that returned 405 (Method Not Allowed) on GET
ENDPOINTS=(
    "clients/metadata"
    "wan/provider-capabilities"
    "wan/provider-capabilities/legacy"
    "vpn/openvpn/configuration"
    "vpn/openvpn/certificates"
    "ports/mac-tables"
    "ports/system-logs"
    "dpi"
    "alias"
    "ssl-inspection/applications"
    "fingerprint/assets"
    "fingerprint_override"
    "missing_fingerprint"
    "app-traffic-rate"
    "insights/filtering/overview"
    "network-members-group"
    "object-oriented-network-config"
    "visual-programming/virtual-network"
)

echo "# POST endpoint scan started at $(date -Iseconds)" | tee "$OUTPUT_DIR/status.log"
echo "" | tee -a "$OUTPUT_DIR/status.log"

for endpoint in "${ENDPOINTS[@]}"; do
    safe_name=$(echo "$endpoint" | tr '/' '-')
    url="$BASE_URL/proxy/network/v2/api/site/$SITE/$endpoint"

    # Try POST with empty body first
    response=$(curl -s -k -w "\n%{http_code}" \
        -X POST \
        -H "X-API-KEY: $API_KEY" \
        -H "Content-Type: application/json" \
        -d '{}' \
        "$url")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    echo "$http_code $endpoint (empty body)" | tee -a "$OUTPUT_DIR/status.log"

    if [ "$http_code" = "200" ]; then
        echo "$body" > "$OUTPUT_DIR/$safe_name.json"
    fi

    # If still 400, try with pagination body
    if [ "$http_code" = "400" ]; then
        response=$(curl -s -k -w "\n%{http_code}" \
            -X POST \
            -H "X-API-KEY: $API_KEY" \
            -H "Content-Type: application/json" \
            -d '{"page":0,"pageSize":50}' \
            "$url")

        http_code=$(echo "$response" | tail -n1)
        body=$(echo "$response" | sed '$d')

        echo "$http_code $endpoint (with pagination)" | tee -a "$OUTPUT_DIR/status.log"

        if [ "$http_code" = "200" ]; then
            echo "$body" > "$OUTPUT_DIR/$safe_name.json"
        fi
    fi

    sleep 0.1
done

echo ""
echo "Done! Check $OUTPUT_DIR for results"
