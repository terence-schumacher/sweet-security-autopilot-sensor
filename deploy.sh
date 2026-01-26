#!/bin/bash
# Sweet Security Deployment Script for GKE Autopilot Clusters
# This script automates the deployment of Sweet Security to a GKE Autopilot cluster
#
# Usage: ./deploy.sh <CLUSTER_NAME> <PROJECT_ID> <REGION> [API_KEY] [SECRET] [CLUSTER_ID]
#   Or set environment variables: SWEET_API_KEY, SWEET_SECRET, SWEET_CLUSTER_ID
#   Or create .env file in this directory with credentials
#
# Example:
#   ./deploy.sh sre-771-staging invisible-sre-sandbox us-west1

set -e

# Source .env file if it exists
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[ -f "${SCRIPT_DIR}/.env" ] && source "${SCRIPT_DIR}/.env"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIGS_DIR="${SCRIPT_DIR}/configs"
SCRIPTS_DIR="${SCRIPT_DIR}/scripts"
MANIFESTS_DIR="${SCRIPT_DIR}/manifests"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing=0
    
    if ! command -v gcloud &> /dev/null; then
        log_error "gcloud CLI not found. Please install Google Cloud SDK."
        missing=1
    fi
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found. Please install kubectl."
        missing=1
    fi
    
    if ! command -v helm &> /dev/null; then
        log_error "helm not found. Please install Helm 3.x."
        missing=1
    fi
    
    if [ $missing -eq 1 ]; then
        exit 1
    fi
    
    log_info "All prerequisites met ✓"
}

# Parse arguments
CLUSTER_NAME=${1:-}
PROJECT_ID=${2:-}
REGION=${3:-}
API_KEY=${4:-${SWEET_API_KEY:-}}
SECRET=${5:-${SWEET_SECRET:-}}
CLUSTER_ID=${6:-${SWEET_CLUSTER_ID:-}}

if [ -z "$CLUSTER_NAME" ] || [ -z "$PROJECT_ID" ] || [ -z "$REGION" ]; then
    log_error "Usage: $0 <CLUSTER_NAME> <PROJECT_ID> <REGION> [API_KEY] [SECRET] [CLUSTER_ID]"
    log_error "   Or set: SWEET_API_KEY, SWEET_SECRET, SWEET_CLUSTER_ID"
    exit 1
fi

if [ -z "$API_KEY" ] || [ -z "$SECRET" ] || [ -z "$CLUSTER_ID" ]; then
    log_error "Missing required credentials. Provide API_KEY, SECRET, and CLUSTER_ID"
    log_error "Either as arguments or environment variables (SWEET_API_KEY, SWEET_SECRET, SWEET_CLUSTER_ID)"
    exit 1
fi

log_info "Deploying Sweet Security to cluster: $CLUSTER_NAME"
log_info "Project: $PROJECT_ID, Region: $REGION"

check_prerequisites

# Step 1: Get cluster network information
log_info "Step 1: Getting cluster network information..."
NETWORK=$(gcloud container clusters describe $CLUSTER_NAME \
    --region=$REGION \
    --project=$PROJECT_ID \
    --format="get(network)" 2>/dev/null || echo "")

SUBNET=$(gcloud container clusters describe $CLUSTER_NAME \
    --region=$REGION \
    --project=$PROJECT_ID \
    --format="get(subnetwork)" 2>/dev/null || echo "")

if [ -z "$NETWORK" ] || [ -z "$SUBNET" ]; then
    log_error "Failed to get cluster network information"
    exit 1
fi

log_info "Network: $NETWORK"
log_info "Subnet: $SUBNET"

# Step 2: Deploy Cloud NAT
log_info "Step 2: Deploying Cloud NAT..."
"${SCRIPTS_DIR}/deploy-nat.sh" "$CLUSTER_NAME" "$PROJECT_ID" "$REGION"

# Get subnet region for verification
if [[ "$SUBNET" == *"/regions/"* ]]; then
    SUBNET_REGION=$(echo "$SUBNET" | sed 's|.*/regions/\([^/]*\)/.*|\1|')
else
    SUBNET_REGION=$(gcloud compute networks subnets describe "$SUBNET" \
        --project=$PROJECT_ID \
        --format="get(region)" 2>/dev/null | sed 's|.*/regions/||')
fi

# Verify Cloud NAT deployment
log_info "Verifying Cloud NAT deployment..."
ROUTER_NAME="sweet-nat-router-${SUBNET_REGION}"

# Get the actual NAT name that was deployed/found
DEPLOYED_NAT_NAME=$(gcloud compute routers nats list \
    --router="$ROUTER_NAME" \
    --region="$SUBNET_REGION" \
    --project="$PROJECT_ID" \
    --format="value(name)" 2>/dev/null | grep "sweet-nat" | head -1 || echo "")

if [ -z "$DEPLOYED_NAT_NAME" ]; then
    DEPLOYED_NAT_NAME="sweet-nat-${SUBNET_REGION}"
fi

if gcloud compute routers nats describe "$DEPLOYED_NAT_NAME" \
    --router="$ROUTER_NAME" \
    --region="$SUBNET_REGION" \
    --project="$PROJECT_ID" &>/dev/null; then
    log_info "Cloud NAT verified: $DEPLOYED_NAT_NAME ✓"
else
    log_warn "Cloud NAT verification failed. Continuing anyway..."
fi

# Update NAT_NAME variable for later use
NAT_NAME="$DEPLOYED_NAT_NAME"

# Step 3: Connect to cluster
log_info "Step 3: Connecting to cluster..."
gcloud container clusters get-credentials $CLUSTER_NAME \
    --region=$REGION \
    --project=$PROJECT_ID

# Step 4: Create namespace
log_info "Step 4: Creating namespace..."
kubectl create namespace sweet --dry-run=client -o yaml | kubectl apply -f -

# Step 5: Install Sweet Operator
log_info "Step 5: Installing Sweet Operator..."
helm upgrade --install sweet-operator oci://registry.sweet.security/helm/operatorchart \
    --namespace sweet \
    --set sweet.apiKey=$API_KEY \
    --set sweet.secret=$SECRET \
    --set sweet.clusterId=$CLUSTER_ID \
    --set frontier.extraValues.serviceAccount.create=true \
    --set frontier.extraValues.priorityClass.enabled=true \
    --set frontier.extraValues.priorityClass.value=1000 \
    --wait --timeout=5m || log_warn "Operator installation may need manual intervention"

# Step 6: Install Sweet Scanner
log_info "Step 6: Installing Sweet Scanner..."
helm upgrade --install sweet-scanner oci://registry.sweet.security/helm/scannerchart \
    --namespace sweet \
    --set sweet.apiKey=$API_KEY \
    --set sweet.secret=$SECRET \
    --set sweet.clusterId=$CLUSTER_ID \
    --wait --timeout=5m || log_warn "Scanner installation may need manual intervention"

# Step 7: Deploy Frontier Informer (Autopilot-compatible)
log_info "Step 7: Deploying Frontier Informer..."
# Update manifest with credentials
TEMP_MANIFEST=$(mktemp)

# Base64 encode (handle both macOS and Linux)
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    API_KEY_B64=$(echo -n "$API_KEY" | base64)
    SECRET_B64=$(echo -n "$SECRET" | base64)
else
    # Linux
    API_KEY_B64=$(echo -n "$API_KEY" | base64 -w 0)
    SECRET_B64=$(echo -n "$SECRET" | base64 -w 0)
fi

sed -e "s|SWEET_CLUSTER_ID_PLACEHOLDER|$CLUSTER_ID|g" \
    -e "s|SWEET_API_KEY_PLACEHOLDER|$API_KEY_B64|g" \
    -e "s|SWEET_SECRET_PLACEHOLDER|$SECRET_B64|g" \
    "${MANIFESTS_DIR}/frontier-manual.yaml" > "$TEMP_MANIFEST"
kubectl apply -f "$TEMP_MANIFEST"
rm -f "$TEMP_MANIFEST"

# Step 8: Verify deployment
log_info "Step 8: Verifying deployment..."

# Verify Cloud NAT status
log_info "Checking Cloud NAT status..."
if gcloud compute routers describe "$ROUTER_NAME" \
    --region="$SUBNET_REGION" \
    --project="$PROJECT_ID" &>/dev/null; then
    log_info "Cloud Router exists: $ROUTER_NAME ✓"
    
    if gcloud compute routers nats list \
        --router="$ROUTER_NAME" \
        --region="$SUBNET_REGION" \
        --project="$PROJECT_ID" \
        --format="get(name)" 2>/dev/null | grep -q "$NAT_NAME"; then
        log_info "Cloud NAT gateway exists: $NAT_NAME ✓"
    else
        log_warn "Cloud NAT gateway not found. Check deployment."
    fi
else
    log_warn "Cloud Router not found. Check deployment."
fi

# Test connectivity from cluster (if kubectl is available)
log_info "Testing connectivity to Sweet Security..."
if kubectl get namespace sweet &>/dev/null; then
    log_info "Running connectivity test..."
    if kubectl run test-connectivity-$(date +%s) \
        --image=curlimages/curl \
        --rm -i --restart=Never \
        -n sweet \
        -- curl -s --connect-timeout 5 -o /dev/null -w "%{http_code}" https://control.sweet.security 2>/dev/null | grep -q "200\|301\|302\|401\|403"; then
        log_info "Connectivity test passed ✓"
    else
        log_warn "Connectivity test failed. Check Cloud NAT configuration."
        log_warn "You can test manually: kubectl run test-curl --image=curlimages/curl --rm -i --restart=Never -- curl -v https://control.sweet.security"
    fi
else
    log_info "Skipping connectivity test (namespace not ready yet)"
fi

log_info "Cloud NAT allows direct access to Sweet Security endpoints."
log_info "No DNS configuration needed - clusters can access external IPs directly."

# Final status
log_info ""
log_info "=========================================="
log_info "Deployment Summary"
log_info "=========================================="
log_info "Cluster: $CLUSTER_NAME"
log_info "Project: $PROJECT_ID"
log_info "Region: $REGION"
log_info "Network: $NETWORK"
log_info ""
log_info "Components deployed:"
log_info "  - Cloud NAT Gateway: $NAT_NAME (region: $SUBNET_REGION)"
log_info "  - Cloud Router: $ROUTER_NAME"
log_info "  - sweet-operator"
log_info "  - sweet-scanner"
log_info "  - sweet-frontier-informer"
log_info ""
log_info "Next steps:"
log_info "  1. Check pod status: kubectl get pods -n sweet"
log_info "  2. Verify in GCP Console: Kubernetes Engine > Workloads"
log_info "  3. Verify Cloud NAT: gcloud compute routers nats list --router=$ROUTER_NAME --region=$SUBNET_REGION --project=$PROJECT_ID"
log_info "  4. Test connectivity: kubectl run test-curl --image=curlimages/curl --rm -i --restart=Never -- curl -v https://control.sweet.security"
log_info ""
log_info "Troubleshooting:"
log_info "  - Check NAT logs: gcloud logging read \"resource.type=cloud_nat\" --limit=50 --project=$PROJECT_ID"
log_info "  - Verify router: gcloud compute routers describe $ROUTER_NAME --region=$SUBNET_REGION --project=$PROJECT_ID"
log_info ""
log_info "Note: No DNS configuration needed - Cloud NAT handles outbound traffic"
log_info "=========================================="
