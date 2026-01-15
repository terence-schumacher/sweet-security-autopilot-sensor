#!/bin/bash
# Deploy Cloud NAT for Sweet Security Traffic
# Usage: ./deploy-nat.sh <CLUSTER_NAME> <PROJECT_ID> <REGION>
#
# This creates a Cloud NAT gateway that allows private GKE clusters
# to access external services (Sweet Security) without needing proxy VMs.

set -e

CLUSTER_NAME=${1:-}
PROJECT_ID=${2:-}
REGION=${3:-}

if [ -z "$CLUSTER_NAME" ] || [ -z "$PROJECT_ID" ] || [ -z "$REGION" ]; then
    echo "Usage: $0 <CLUSTER_NAME> <PROJECT_ID> <REGION>"
    exit 1
fi

echo "Deploying Cloud NAT for cluster: $CLUSTER_NAME in project: $PROJECT_ID"

# Get cluster network and subnet
NETWORK=$(gcloud container clusters describe $CLUSTER_NAME \
  --region=$REGION \
  --project=$PROJECT_ID \
  --format="get(network)" 2>/dev/null || echo "")

SUBNET=$(gcloud container clusters describe $CLUSTER_NAME \
  --region=$REGION \
  --project=$PROJECT_ID \
  --format="get(subnetwork)" 2>/dev/null || echo "")

if [ -z "$NETWORK" ] || [ -z "$SUBNET" ]; then
  echo "Error: Could not get cluster network/subnet"
  exit 1
fi

echo "Network: $NETWORK"
echo "Subnet: $SUBNET"

# Get subnet region
if [[ "$SUBNET" == *"/regions/"* ]]; then
    # Extract region from full path
    SUBNET_REGION=$(echo "$SUBNET" | sed 's|.*/regions/\([^/]*\)/.*|\1|')
else
    # Get region from subnet description
    SUBNET_REGION=$(gcloud compute networks subnets describe "$SUBNET" \
        --project=$PROJECT_ID \
        --format="get(region)" 2>/dev/null | sed 's|.*/regions/||')
fi

# Extract subnet name (last part of path or just the name)
if [[ "$SUBNET" == *"/"* ]]; then
    SUBNET_NAME=$(echo "$SUBNET" | sed 's|.*/||')
else
    SUBNET_NAME="$SUBNET"
fi

echo "Subnet Region: $SUBNET_REGION"
echo "Subnet Name: $SUBNET_NAME"

# Create a Cloud Router if it doesn't exist
ROUTER_NAME="sweet-nat-router-${SUBNET_REGION}"
echo "Checking for Cloud Router: $ROUTER_NAME"

if ! gcloud compute routers describe "$ROUTER_NAME" \
    --region="$SUBNET_REGION" \
    --project="$PROJECT_ID" &>/dev/null; then
    echo "Creating Cloud Router: $ROUTER_NAME"
    gcloud compute routers create "$ROUTER_NAME" \
        --network="$NETWORK" \
        --region="$SUBNET_REGION" \
        --project="$PROJECT_ID" \
        --quiet
else
    echo "Cloud Router already exists: $ROUTER_NAME"
fi

# Check if any NAT already exists on this router
# Use the router's region (SUBNET_REGION) for listing NATs
EXISTING_NATS=$(gcloud compute routers nats list \
    --router="$ROUTER_NAME" \
    --region="$SUBNET_REGION" \
    --project="$PROJECT_ID" \
    --format="value(name)" 2>/dev/null || echo "")

NAT_NAME="sweet-nat-${SUBNET_REGION}"
FOUND_EXISTING_NAT=""

if [ -n "$EXISTING_NATS" ] && [ "$EXISTING_NATS" != "" ]; then
    # Check each existing NAT to see if one uses ALL_SUBNETWORKS_ALL_IP_RANGES
    echo "Checking existing NATs on router..."
    for EXISTING_NAT in $EXISTING_NATS; do
        EXISTING_NAT_CONFIG=$(gcloud compute routers nats describe "$EXISTING_NAT" \
            --router="$ROUTER_NAME" \
            --region="$SUBNET_REGION" \
            --project="$PROJECT_ID" \
            --format="get(sourceSubnetworkIpRangesToNat)" 2>/dev/null || echo "")
        
        echo "  NAT: $EXISTING_NAT, Config: $EXISTING_NAT_CONFIG"
        
        if [ "$EXISTING_NAT_CONFIG" = "ALL_SUBNETWORKS_ALL_IP_RANGES" ]; then
            echo "✓ Found existing NAT with all subnet ranges: $EXISTING_NAT"
            echo "Using existing NAT gateway (covers all subnets including $SUBNET_NAME)"
            NAT_NAME="$EXISTING_NAT"
            FOUND_EXISTING_NAT="$EXISTING_NAT"
            break
        fi
    done
    
    if [ -z "$FOUND_EXISTING_NAT" ]; then
        # No NAT with ALL_SUBNETWORKS_ALL_IP_RANGES found
        # Check if our specific NAT exists
        if gcloud compute routers nats describe "$NAT_NAME" \
            --router="$ROUTER_NAME" \
            --region="$SUBNET_REGION" \
            --project="$PROJECT_ID" &>/dev/null; then
            echo "Cloud NAT already exists: $NAT_NAME"
        else
            echo "Warning: Cannot create new NAT with custom ranges when a NAT with ALL_SUBNETWORKS_ALL_IP_RANGES exists."
            echo "Please use the existing NAT or remove it first."
            echo "Existing NATs on router:"
            for EXISTING_NAT in $EXISTING_NATS; do
                echo "  - $EXISTING_NAT"
            done
            exit 1
        fi
    fi
else
    # No existing NAT, create one with specific subnet IP ranges
    echo "No existing NATs found. Creating Cloud NAT: $NAT_NAME (Private NAT for VMs) with specific subnet"
    gcloud compute routers nats create "$NAT_NAME" \
        --router="$ROUTER_NAME" \
        --region="$SUBNET_REGION" \
        --nat-custom-subnet-ip-ranges="${SUBNET_NAME}:ALL" \
        --auto-allocate-nat-external-ips \
        --endpoint-types=ENDPOINT_TYPE_VM \
        --enable-logging \
        --project="$PROJECT_ID" \
        --quiet
fi

echo ""
echo "✅ Cloud NAT deployment complete!"
echo ""
echo "Cloud NAT Configuration:"
echo "  Router: $ROUTER_NAME"
echo "  NAT Gateway: $NAT_NAME"
echo "  Region: $SUBNET_REGION"
echo "  Network: $NETWORK"
echo ""
echo "Note: Cloud NAT allows private GKE clusters to access external services."
echo "No DNS configuration needed - clusters can directly access Sweet Security endpoints."
