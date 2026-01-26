#!/bin/bash
# Manual NAT detection script
# Usage: ./find-nat.sh <PROJECT_ID> <REGION>

PROJECT_ID=${1:-"invisible-client-analytics"}
REGION=${2:-"us-west1"}

echo "=== Manual NAT Detection Script ==="
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo ""

ROUTER_NAME="sweet-nat-router-${REGION}"

echo "1. Checking if router exists..."
if gcloud compute routers describe "$ROUTER_NAME" --region="$REGION" --project="$PROJECT_ID" &>/dev/null; then
    echo "✓ Router exists: $ROUTER_NAME"
else
    echo "✗ Router not found: $ROUTER_NAME"
    exit 1
fi

echo ""
echo "2. Getting router details..."
gcloud compute routers describe "$ROUTER_NAME" --region="$REGION" --project="$PROJECT_ID" --format="yaml"

echo ""
echo "3. Trying different NAT listing approaches..."

echo ""
echo "3a. Standard NAT list command:"
gcloud compute routers nats list --router="$ROUTER_NAME" --region="$REGION" --project="$PROJECT_ID" 2>&1 || echo "Command failed"

echo ""
echo "3b. Region-wide NAT list:"
gcloud compute routers nats list --region="$REGION" --project="$PROJECT_ID" 2>&1 || echo "Command failed"

echo ""
echo "3c. All NATs in project:"
gcloud compute routers nats list --project="$PROJECT_ID" 2>&1 || echo "Command failed"

echo ""
echo "4. Alternative: Check via router description..."
NATS_IN_ROUTER=$(gcloud compute routers describe "$ROUTER_NAME" --region="$REGION" --project="$PROJECT_ID" --format="value(nats[].name)" 2>/dev/null)
if [ -n "$NATS_IN_ROUTER" ]; then
    echo "✓ Found NATs in router description: $NATS_IN_ROUTER"

    for NAT in $NATS_IN_ROUTER; do
        echo ""
        echo "NAT Details: $NAT"
        gcloud compute routers nats describe "$NAT" --router="$ROUTER_NAME" --region="$REGION" --project="$PROJECT_ID"
    done
else
    echo "✗ No NATs found in router description"
fi

echo ""
echo "=== Detection Complete ==="