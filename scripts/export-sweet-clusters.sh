#!/bin/bash
# Export cluster list from Sweet Security API
# This script attempts to retrieve clusters via the Sweet Security API
# Usage: ./export-sweet-clusters.sh [API_KEY] [API_SECRET]

set -e

API_KEY=${1:-${SWEET_API_KEY:-}}
API_SECRET=${2:-${SWEET_SECRET:-}}
OUTPUT_FILE=${3:-"sweet-clusters.txt"}

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Sweet Security Cluster Export${NC}"
echo "================================"
echo ""

if [ -z "$API_KEY" ] || [ -z "$API_SECRET" ]; then
    echo -e "${YELLOW}Usage: $0 [API_KEY] [API_SECRET] [OUTPUT_FILE]${NC}"
    echo ""
    echo "Or set environment variables:"
    echo "  export SWEET_API_KEY='your-key'"
    echo "  export SWEET_SECRET='your-secret'"
    echo "  $0"
    echo ""
    exit 1
fi

echo "Attempting to retrieve clusters from Sweet Security API..."
echo ""

# Try different API endpoints
ENDPOINTS=(
    "https://api.sweet.security/v1/clusters"
    "https://api.sweet.security/v1/cloud-entities"
    "https://control.sweet.security/api/v1/clusters"
    "https://control.sweet.security/v1/clusters"
)

for endpoint in "${ENDPOINTS[@]}"; do
    echo "Trying: $endpoint"
    
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X GET \
        -H "Authorization: Bearer $API_KEY" \
        -H "X-Api-Key: $API_KEY" \
        -H "X-Api-Secret: $API_SECRET" \
        -H "Content-Type: application/json" \
        "$endpoint" 2>/dev/null || echo -e "\n000")
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "${GREEN}✓ Success!${NC}"
        echo "$BODY" | jq '.' > "$OUTPUT_FILE.json" 2>/dev/null || echo "$BODY" > "$OUTPUT_FILE.json"
        echo "Response saved to $OUTPUT_FILE.json"
        break
    else
        echo -e "${RED}✗ Failed (HTTP $HTTP_CODE)${NC}"
    fi
    echo ""
done

echo ""
echo -e "${YELLOW}Note:${NC} If API calls fail, you can:"
echo "1. Export from the dashboard manually"
echo "2. Use list-clusters.sh to get clusters from GCP directly"
echo "3. Check Sweet Security API documentation for the correct endpoint"
