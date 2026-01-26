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
echo "Checking for existing NATs on router $ROUTER_NAME..."

# Try to list NATs - if command fails, we'll catch it
EXISTING_NATS=""
if gcloud compute routers nats list \
    --router="$ROUTER_NAME" \
    --region="$SUBNET_REGION" \
    --project="$PROJECT_ID" \
    --format="value(name)" &>/dev/null; then
    EXISTING_NATS=$(gcloud compute routers nats list \
        --router="$ROUTER_NAME" \
        --region="$SUBNET_REGION" \
        --project="$PROJECT_ID" \
        --format="value(name)" 2>/dev/null)
fi

echo "Found NATs: $EXISTING_NATS"

NAT_NAME="sweet-nat-${SUBNET_REGION}"
FOUND_EXISTING_NAT=""

if [ -n "$EXISTING_NATS" ] && [ "$EXISTING_NATS" != "" ]; then
    # Check each existing NAT to see if one uses ALL_SUBNETWORKS_ALL_IP_RANGES
    # Also check for existing sweet-nat NATs with different naming patterns
    echo "Checking existing NATs on router..."
    for EXISTING_NAT in $EXISTING_NATS; do
        EXISTING_NAT_CONFIG=$(gcloud compute routers nats describe "$EXISTING_NAT" \
            --router="$ROUTER_NAME" \
            --region="$SUBNET_REGION" \
            --project="$PROJECT_ID" \
            --format="get(sourceSubnetworkIpRangesToNat)" 2>/dev/null || echo "")

        echo "  NAT: $EXISTING_NAT, Config: $EXISTING_NAT_CONFIG"

        # Check if it's a sweet-nat with any naming pattern and has proper configuration
        if [[ "$EXISTING_NAT" == sweet-nat* ]]; then
            if [ "$EXISTING_NAT_CONFIG" = "ALL_SUBNETWORKS_ALL_IP_RANGES" ]; then
                echo "✓ Found existing Sweet Security NAT with all subnets: $EXISTING_NAT"
                echo "Using existing NAT gateway (covers all subnets including $SUBNET_NAME)"
                NAT_NAME="$EXISTING_NAT"
                FOUND_EXISTING_NAT="$EXISTING_NAT"
                break
            elif [[ "$EXISTING_NAT_CONFIG" == *"$SUBNET_NAME"* ]]; then
                echo "✓ Found existing Sweet Security NAT for subnet: $EXISTING_NAT"
                echo "Using existing NAT gateway (covers subnet $SUBNET_NAME)"
                NAT_NAME="$EXISTING_NAT"
                FOUND_EXISTING_NAT="$EXISTING_NAT"
                break
            else
                # Check if this NAT's subnets include our target subnet
                EXISTING_SUBNETS=$(gcloud compute routers nats describe "$EXISTING_NAT" \
                    --router="$ROUTER_NAME" \
                    --region="$SUBNET_REGION" \
                    --project="$PROJECT_ID" \
                    --format="value(subnetworks[].name)" 2>/dev/null || echo "")

                if [[ "$EXISTING_SUBNETS" == *"$SUBNET_NAME"* ]]; then
                    echo "✓ Found existing Sweet Security NAT with compatible subnets: $EXISTING_NAT"
                    echo "Using existing NAT gateway (includes subnet $SUBNET_NAME)"
                    NAT_NAME="$EXISTING_NAT"
                    FOUND_EXISTING_NAT="$EXISTING_NAT"
                    break
                else
                    echo "  Sweet Security NAT found but doesn't cover our subnet: $EXISTING_NAT"
                fi
            fi
        elif [ "$EXISTING_NAT_CONFIG" = "ALL_SUBNETWORKS_ALL_IP_RANGES" ]; then
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
    # No NATs found in list command, but let's try a different approach
    # Try to describe the router directly and look for NATs in the output
    echo "No NATs found via list command, checking router description..."

    ROUTER_NATS=$(gcloud compute routers describe "$ROUTER_NAME" \
        --region="$SUBNET_REGION" \
        --project="$PROJECT_ID" \
        --format="value(nats[].name)" 2>/dev/null || echo "")

    if [ -n "$ROUTER_NATS" ] && [ "$ROUTER_NATS" != "" ]; then
        echo "Found NATs via router description: $ROUTER_NATS"
        # Use the first NAT found
        FIRST_NAT=$(echo "$ROUTER_NATS" | head -1 | tr -d ' ')
        if [ -n "$FIRST_NAT" ]; then
            echo "✓ Found existing NAT on router: $FIRST_NAT"
            echo "Using existing NAT gateway instead of creating new one"
            NAT_NAME="$FIRST_NAT"
        fi
    else
        # Truly no existing NAT, create one with specific subnet IP ranges
        echo "No existing NATs found. Creating Cloud NAT: $NAT_NAME (Private NAT for VMs) with specific subnet"

        # But first, try to detect if there's a permission issue or if the error indicates an existing NAT
        if gcloud compute routers nats create "$NAT_NAME" \
            --router="$ROUTER_NAME" \
            --region="$SUBNET_REGION" \
            --nat-custom-subnet-ip-ranges="${SUBNET_NAME}:ALL" \
            --auto-allocate-nat-external-ips \
            --endpoint-types=ENDPOINT_TYPE_VM \
            --enable-logging \
            --project="$PROJECT_ID" \
            --quiet 2>&1; then
            echo "✅ Successfully created NAT: $NAT_NAME"
        else
            echo "❌ Failed to create NAT. This usually means:"
            echo "   1. A NAT with ALL_SUBNETWORKS_ALL_IP_RANGES already exists"
            echo "   2. Insufficient permissions"
            echo "   3. Resource conflict"
            echo ""
            echo "Checking for existing NATs again using alternative method..."

            # Try listing all NATs in the region and filter by router
            ALL_REGION_NATS=$(gcloud compute routers nats list \
                --region="$SUBNET_REGION" \
                --project="$PROJECT_ID" \
                --format="csv[no-heading](name,router)" 2>/dev/null || echo "")

            if [ -n "$ALL_REGION_NATS" ]; then
                echo "All NATs in region $SUBNET_REGION:"
                echo "$ALL_REGION_NATS"

                # Filter for our router
                MATCHING_NAT=$(echo "$ALL_REGION_NATS" | grep "$ROUTER_NAME" | cut -d',' -f1 | head -1)
                if [ -n "$MATCHING_NAT" ]; then
                    echo "✓ Found existing NAT for our router: $MATCHING_NAT"
                    NAT_NAME="$MATCHING_NAT"
                else
                    echo "❌ No NAT found for router $ROUTER_NAME"
                    echo "Manual intervention may be required"
                    exit 1
                fi
            else
                echo "❌ Could not list NATs in region"
                echo "Manual intervention may be required"
                exit 1
            fi
        fi
    fi
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
