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

# Step 2: Deploy proxy
log_info "Step 2: Deploying DNS proxy..."
"${SCRIPTS_DIR}/deploy-proxy.sh" "$CLUSTER_NAME" "$PROJECT_ID" "$REGION"

# Get the zone from subnet region (same logic as deploy-proxy.sh)
# Handle both full path (projects/.../regions/.../subnetworks/...) and subnet name
if [[ "$SUBNET" == *"/regions/"* ]]; then
    # Extract region from full path
    SUBNET_REGION=$(echo "$SUBNET" | sed 's|.*/regions/\([^/]*\)/.*|\1|')
else
    # Get region from subnet description
    SUBNET_REGION=$(gcloud compute networks subnets describe "$SUBNET" \
        --project=$PROJECT_ID \
        --format="get(region)" 2>/dev/null | sed 's|.*/regions/||')
fi

ZONE="${SUBNET_REGION}-a"
PROXY_INSTANCE_NAME="sweet-proxy-${CLUSTER_NAME}"

log_info "Retrieving proxy IP from zone: $ZONE"
PROXY_IP=$(gcloud compute instances describe "${PROXY_INSTANCE_NAME}" \
    --zone="$ZONE" \
    --project=$PROJECT_ID \
    --format="get(networkInterfaces[0].networkIP)" 2>/dev/null || echo "")

if [ -z "$PROXY_IP" ]; then
    log_error "Failed to get proxy IP. Check proxy deployment."
    log_error "Tried zone: $ZONE for instance: $PROXY_INSTANCE_NAME"
    exit 1
fi
log_info "Proxy IP: $PROXY_IP"

# Step 3: Configure DNS
log_info "Step 3: Configuring Cloud DNS..."

# Check if DNS zone exists, create if not
DNS_ZONE="sweet-security-zone"
if ! gcloud dns managed-zones describe $DNS_ZONE --project=$PROJECT_ID &>/dev/null; then
    log_info "Creating DNS zone: $DNS_ZONE"
    gcloud dns managed-zones create $DNS_ZONE \
        --dns-name=sweet.security. \
        --description="Sweet Security DNS zone" \
        --visibility=private \
        --networks=$NETWORK \
        --project=$PROJECT_ID \
        --quiet
else
    log_info "DNS zone exists, updating networks..."
    # Add network to existing zone
    EXISTING_NETWORKS=$(gcloud dns managed-zones describe $DNS_ZONE \
        --project=$PROJECT_ID \
        --format="get(privateVisibilityConfig.networks[].networkUrl)" | tr '\n' ',' | sed 's/,$//')
    
    if [[ ! "$EXISTING_NETWORKS" == *"$NETWORK"* ]]; then
        gcloud dns managed-zones update $DNS_ZONE \
            --project=$PROJECT_ID \
            --networks="${EXISTING_NETWORKS},${NETWORK}" \
            --quiet
    fi
fi

# Create DNS records
log_info "Creating DNS records..."
ENDPOINTS=("*.sweet.security" "registry.sweet.security" "control.sweet.security" \
           "logger.sweet.security" "receiver.sweet.security" "vincent.sweet.security" \
           "api.sweet.security" "prio.sweet.security")

for endpoint in "${ENDPOINTS[@]}"; do
    if ! gcloud dns record-sets describe "${endpoint}." \
        --type=A \
        --zone=$DNS_ZONE \
        --project=$PROJECT_ID &>/dev/null; then
        log_info "Creating DNS record: $endpoint"
        gcloud dns record-sets create "${endpoint}." \
            --zone=$DNS_ZONE \
            --type=A \
            --rrdatas=$PROXY_IP \
            --ttl=300 \
            --project=$PROJECT_ID \
            --quiet
    else
        log_info "Updating DNS record: $endpoint"
        gcloud dns record-sets update "${endpoint}." \
            --zone=$DNS_ZONE \
            --type=A \
            --rrdatas=$PROXY_IP \
            --ttl=300 \
            --project=$PROJECT_ID \
            --quiet
    fi
done

# Step 4: Connect to cluster
log_info "Step 4: Connecting to cluster..."
gcloud container clusters get-credentials $CLUSTER_NAME \
    --region=$REGION \
    --project=$PROJECT_ID

# Step 5: Create namespace
log_info "Step 5: Creating namespace..."
kubectl create namespace sweet --dry-run=client -o yaml | kubectl apply -f -

# Step 6: Install Sweet Operator
log_info "Step 6: Installing Sweet Operator..."
helm upgrade --install sweet-operator oci://registry.sweet.security/helm/operatorchart \
    --namespace sweet \
    --set sweet.apiKey=$API_KEY \
    --set sweet.secret=$SECRET \
    --set sweet.clusterId=$CLUSTER_ID \
    --set frontier.extraValues.serviceAccount.create=true \
    --set frontier.extraValues.priorityClass.enabled=true \
    --set frontier.extraValues.priorityClass.value=1000 \
    --wait --timeout=5m || log_warn "Operator installation may need manual intervention"

# Step 7: Install Sweet Scanner
log_info "Step 7: Installing Sweet Scanner..."
helm upgrade --install sweet-scanner oci://registry.sweet.security/helm/scannerchart \
    --namespace sweet \
    --set sweet.apiKey=$API_KEY \
    --set sweet.secret=$SECRET \
    --set sweet.clusterId=$CLUSTER_ID \
    --wait --timeout=5m || log_warn "Scanner installation may need manual intervention"

# Step 8: Deploy Frontier Informer (Autopilot-compatible)
log_info "Step 8: Deploying Frontier Informer..."
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

# Step 9: Wait for DNS propagation
log_info "Step 9: Waiting for DNS propagation (30 seconds)..."
sleep 30

# Step 10: Verify and restart if needed
log_info "Step 10: Verifying deployment..."
log_info "Checking DNS resolution..."
kubectl run test-dns-$(date +%s) --image=busybox --rm -i --restart=Never -n sweet -- \
    nslookup registry.sweet.security 2>&1 | grep -q "10.0" && \
    log_info "DNS resolution working ✓" || \
    log_warn "DNS may not be fully propagated yet"

log_info "Restarting deployments to pick up DNS changes..."
kubectl rollout restart deployment/sweet-operator -n sweet 2>/dev/null || true
kubectl rollout restart deployment/sweet-scanner -n sweet 2>/dev/null || true
kubectl rollout restart deployment/sweet-frontier-informer -n sweet 2>/dev/null || true

# Final status
log_info ""
log_info "=========================================="
log_info "Deployment Summary"
log_info "=========================================="
log_info "Cluster: $CLUSTER_NAME"
log_info "Project: $PROJECT_ID"
log_info "Region: $REGION"
log_info "Proxy IP: $PROXY_IP"
log_info "DNS Zone: $DNS_ZONE"
log_info ""
log_info "Components deployed:"
log_info "  - sweet-operator"
log_info "  - sweet-scanner"
log_info "  - sweet-frontier-informer"
log_info ""
log_info "Next steps:"
log_info "  1. Wait 5-10 minutes for DNS propagation"
log_info "  2. Check pod status: kubectl get pods -n sweet"
log_info "  3. Verify in GCP Console: Kubernetes Engine > Workloads"
log_info ""
log_info "If pods are in ImagePullBackOff, wait for DNS and restart:"
log_info "  kubectl rollout restart deployment -n sweet"
log_info "=========================================="
