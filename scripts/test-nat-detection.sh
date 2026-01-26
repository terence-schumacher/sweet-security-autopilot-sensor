#!/bin/bash
# Test script to verify NAT detection logic
# Usage: ./test-nat-detection.sh <PROJECT_ID> <REGION>

set -e

PROJECT_ID=${1:-}
REGION=${2:-"us-west1"}

if [ -z "$PROJECT_ID" ]; then
    echo "Usage: $0 <PROJECT_ID> [REGION]"
    exit 1
fi

echo "Testing NAT detection in project: $PROJECT_ID, region: $REGION"

# Check for routers in the region
ROUTER_NAME="sweet-nat-router-${REGION}"
echo ""
echo "Checking for Cloud Router: $ROUTER_NAME"

if gcloud compute routers describe "$ROUTER_NAME" \
    --region="$REGION" \
    --project="$PROJECT_ID" &>/dev/null; then
    echo "✓ Cloud Router exists: $ROUTER_NAME"

    # List all NATs on this router using multiple methods
    echo ""
    echo "NATs on router $ROUTER_NAME (using list command):"
    EXISTING_NATS=""
    if gcloud compute routers nats list \
        --router="$ROUTER_NAME" \
        --region="$REGION" \
        --project="$PROJECT_ID" \
        --format="value(name)" &>/dev/null; then
        EXISTING_NATS=$(gcloud compute routers nats list \
            --router="$ROUTER_NAME" \
            --region="$REGION" \
            --project="$PROJECT_ID" \
            --format="value(name)" 2>/dev/null)
    fi

    # Also try router description method
    ROUTER_NATS=$(gcloud compute routers describe "$ROUTER_NAME" \
        --region="$REGION" \
        --project="$PROJECT_ID" \
        --format="value(nats[].name)" 2>/dev/null || echo "")

    echo "NATs via list command: $EXISTING_NATS"
    echo "NATs via router description: $ROUTER_NATS"

    # Use the method that works
    ALL_NATS="$EXISTING_NATS"
    if [ -z "$ALL_NATS" ] && [ -n "$ROUTER_NATS" ]; then
        ALL_NATS="$ROUTER_NATS"
    fi

    if [ -n "$ALL_NATS" ] && [ "$ALL_NATS" != "" ]; then
        for NAT in $ALL_NATS; do
            echo "  - $NAT"

            # Get NAT configuration
            NAT_CONFIG=$(gcloud compute routers nats describe "$NAT" \
                --router="$ROUTER_NAME" \
                --region="$REGION" \
                --project="$PROJECT_ID" \
                --format="get(sourceSubnetworkIpRangesToNat)" 2>/dev/null || echo "")

            echo "    Config: $NAT_CONFIG"

            # Check if it's a Sweet Security NAT
            if [[ "$NAT" == sweet-nat* ]]; then
                echo "    ✓ This is a Sweet Security NAT"

                if [ "$NAT_CONFIG" = "ALL_SUBNETWORKS_ALL_IP_RANGES" ]; then
                    echo "    ✓ Covers all subnets - COMPATIBLE"
                else
                    echo "    ? Specific subnet configuration - needs subnet check"
                fi
            fi
        done
    else
        echo "  No NATs found on router"
    fi
else
    echo "✗ Cloud Router not found: $ROUTER_NAME"
fi

echo ""
echo "Test complete. The deploy script should now detect existing sweet-nat NATs."