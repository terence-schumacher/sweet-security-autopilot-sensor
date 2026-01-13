#!/bin/bash
# Deploy Sweet Security Proxy for a GKE Cluster
# Usage: ./deploy-proxy.sh <CLUSTER_NAME> <PROJECT_ID> <REGION>

set -e

CLUSTER_NAME=${1:-sre-771-staging}
PROJECT_ID=${2:-invisible-sre-sandbox}
REGION=${3:-us-west1}

echo "Deploying proxy for cluster: $CLUSTER_NAME in project: $PROJECT_ID"

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

# Get subnet region and zone
SUBNET_REGION=$(gcloud compute networks subnets describe $SUBNET \
  --project=$PROJECT_ID \
  --format="get(region)" 2>/dev/null | sed 's|.*/regions/||')

ZONE="${SUBNET_REGION}-a"  # Use first zone in region
PROXY_NAME="sweet-proxy-${CLUSTER_NAME}"

echo "Zone: $ZONE"
echo "Proxy name: $PROXY_NAME"

# Check if proxy already exists
if gcloud compute instances describe $PROXY_NAME --zone=$ZONE --project=$PROJECT_ID &>/dev/null; then
  echo "Proxy $PROXY_NAME already exists. Getting IP..."
  PROXY_IP=$(gcloud compute instances describe $PROXY_NAME \
    --zone=$ZONE \
    --project=$PROJECT_ID \
    --format="get(networkInterfaces[0].networkIP)")
  echo "Existing proxy IP: $PROXY_IP"
else
  echo "Creating proxy instance..."
  
  # Create proxy instance
  gcloud compute instances create $PROXY_NAME \
    --zone=$ZONE \
    --project=$PROJECT_ID \
    --machine-type=e2-standard-2 \
    --network=$NETWORK \
    --subnet=$SUBNET \
    --image-family=ubuntu-2204-lts \
    --image-project=ubuntu-os-cloud \
    --boot-disk-size=20GB \
    --boot-disk-type=pd-balanced \
    --tags=sweet-proxy \
    --metadata-from-file=startup-script=${BASH_SOURCE%/*}/proxy-startup-script.sh \
    --no-service-account \
    --no-scopes \
    --quiet

  # Wait for instance to be ready
  echo "Waiting for instance to be ready..."
  sleep 30

  # Get proxy IP
  PROXY_IP=$(gcloud compute instances describe $PROXY_NAME \
    --zone=$ZONE \
    --project=$PROJECT_ID \
    --format="get(networkInterfaces[0].networkIP)")
  
  echo "Proxy created with IP: $PROXY_IP"
fi

# Create firewall rules if they don't exist
echo "Creating firewall rules..."

# Ingress TCP
gcloud compute firewall-rules create sweet-proxy-${CLUSTER_NAME}-ingress-tcp \
  --network=$NETWORK \
  --project=$PROJECT_ID \
  --allow=tcp:443 \
  --source-ranges=0.0.0.0/0 \
  --target-tags=sweet-proxy \
  --description="Allow TCP 443 to Sweet Security proxy" \
  --quiet 2>/dev/null || echo "Firewall rule already exists"

# Ingress UDP
gcloud compute firewall-rules create sweet-proxy-${CLUSTER_NAME}-ingress-udp \
  --network=$NETWORK \
  --project=$PROJECT_ID \
  --allow=udp:443 \
  --source-ranges=0.0.0.0/0 \
  --target-tags=sweet-proxy \
  --description="Allow UDP 443 to Sweet Security proxy" \
  --quiet 2>/dev/null || echo "Firewall rule already exists"

# Egress
gcloud compute firewall-rules create sweet-proxy-${CLUSTER_NAME}-egress \
  --network=$NETWORK \
  --project=$PROJECT_ID \
  --direction=EGRESS \
  --allow=all \
  --destination-ranges=18.220.208.31/32 \
  --target-tags=sweet-proxy \
  --description="Allow egress to Sweet Security" \
  --quiet 2>/dev/null || echo "Firewall rule already exists"

echo ""
echo "✅ Proxy deployment complete!"
echo "Proxy IP: $PROXY_IP"
echo ""
echo "Next steps:"
echo "1. Update DNS records to point to $PROXY_IP"
echo "2. Wait for DNS propagation (5-10 minutes)"
echo "3. Restart pods: kubectl rollout restart deployment -n sweet"
