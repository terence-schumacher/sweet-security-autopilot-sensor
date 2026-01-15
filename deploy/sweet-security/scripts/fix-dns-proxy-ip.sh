#!/bin/bash
# Fix DNS Proxy IP Issue for Sweet Security
# This script updates DNS records to point to the correct proxy IP for a cluster
#
# Usage: ./fix-dns-proxy-ip.sh <CLUSTER_NAME> <PROJECT_ID> [REGION]
#
# Example:
#   ./fix-dns-proxy-ip.sh client-analytics-production invisible-client-analytics us-west1

set -e

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

# Parse arguments
CLUSTER_NAME=${1:-}
PROJECT_ID=${2:-}
REGION=${3:-us-west1}

if [ -z "$CLUSTER_NAME" ] || [ -z "$PROJECT_ID" ]; then
    log_error "Usage: $0 <CLUSTER_NAME> <PROJECT_ID> [REGION]"
    log_error "Example: $0 client-analytics-production invisible-client-analytics us-west1"
    exit 1
fi

log_info "Fixing DNS proxy IP for cluster: $CLUSTER_NAME"
log_info "Project: $PROJECT_ID, Region: $REGION"
echo ""

# Step 1: Get cluster subnet to determine zone
log_info "Step 1: Getting cluster network information..."
SUBNET=$(gcloud container clusters describe $CLUSTER_NAME \
    --region=$REGION \
    --project=$PROJECT_ID \
    --format="get(subnetwork)" 2>/dev/null || echo "")

if [ -z "$SUBNET" ]; then
    log_error "Failed to get cluster subnet information"
    exit 1
fi

# Get subnet region and zone
if [[ "$SUBNET" == *"/regions/"* ]]; then
    SUBNET_REGION=$(echo "$SUBNET" | sed 's|.*/regions/\([^/]*\)/.*|\1|')
else
    SUBNET_REGION=$(gcloud compute networks subnets describe "$SUBNET" \
        --project=$PROJECT_ID \
        --format="get(region)" 2>/dev/null | sed 's|.*/regions/||')
fi

ZONE="${SUBNET_REGION}-a"
PROXY_NAME="sweet-proxy-${CLUSTER_NAME}"

log_info "Zone: $ZONE"
log_info "Proxy name: $PROXY_NAME"

# Step 2: Get proxy IP
log_info "Step 2: Getting proxy IP..."
PROXY_IP=$(gcloud compute instances describe "${PROXY_NAME}" \
    --zone="$ZONE" \
    --project=$PROJECT_ID \
    --format="get(networkInterfaces[0].networkIP)" 2>/dev/null || echo "")

if [ -z "$PROXY_IP" ]; then
    log_error "Failed to get proxy IP. Proxy may not exist: $PROXY_NAME"
    log_error "Zone: $ZONE"
    exit 1
fi

log_info "Proxy IP: $PROXY_IP"

# Step 3: Check proxy status
log_info "Step 3: Checking proxy status..."
PROXY_STATUS=$(gcloud compute instances describe "${PROXY_NAME}" \
    --zone="$ZONE" \
    --project=$PROJECT_ID \
    --format="get(status)" 2>/dev/null || echo "")

if [ "$PROXY_STATUS" != "RUNNING" ]; then
    log_warn "Proxy status is: $PROXY_STATUS (expected RUNNING)"
fi

# Step 4: Check DNS zone
log_info "Step 4: Checking DNS zone..."
DNS_ZONE="sweet-security-zone"
if ! gcloud dns managed-zones describe $DNS_ZONE --project=$PROJECT_ID &>/dev/null; then
    log_error "DNS zone '$DNS_ZONE' does not exist in project $PROJECT_ID"
    exit 1
fi

log_info "DNS zone exists: $DNS_ZONE"

# Step 5: Get current DNS records
log_info "Step 5: Checking current DNS records..."
CURRENT_IP=$(gcloud dns record-sets describe registry.sweet.security. \
    --type=A \
    --zone=$DNS_ZONE \
    --project=$PROJECT_ID \
    --format="get(rrdatas[0])" 2>/dev/null || echo "")

if [ -z "$CURRENT_IP" ]; then
    log_error "DNS record for registry.sweet.security not found"
    exit 1
fi

log_info "Current DNS IP: $CURRENT_IP"
log_info "Target proxy IP: $PROXY_IP"

if [ "$CURRENT_IP" == "$PROXY_IP" ]; then
    log_info "DNS already points to correct proxy IP. Checking other records..."
else
    log_warn "DNS IP mismatch detected! Will update all records."
fi

# Step 6: Update DNS records
log_info "Step 6: Updating DNS records to proxy IP: $PROXY_IP"

ENDPOINTS=("registry.sweet.security" "*.sweet.security" "api.sweet.security" \
           "control.sweet.security" "logger.sweet.security" "prio.sweet.security" \
           "receiver.sweet.security" "vincent.sweet.security")

UPDATED=0
FAILED=0

for endpoint in "${ENDPOINTS[@]}"; do
    # Add trailing dot for DNS record name
    if [[ "$endpoint" != *. ]]; then
        endpoint="${endpoint}."
    fi
    
    if gcloud dns record-sets update "$endpoint" \
        --zone=$DNS_ZONE \
        --type=A \
        --rrdatas=$PROXY_IP \
        --ttl=300 \
        --project=$PROJECT_ID &>/dev/null; then
        log_info "✓ Updated: $endpoint → $PROXY_IP"
        ((UPDATED++))
    else
        log_warn "✗ Failed to update: $endpoint"
        ((FAILED++))
    fi
done

echo ""
log_info "DNS update summary: $UPDATED updated, $FAILED failed"

if [ $FAILED -gt 0 ]; then
    log_warn "Some DNS records failed to update. Please check manually."
fi

# Step 7: Wait for DNS propagation
log_info "Step 7: Waiting 10 seconds for DNS propagation..."
sleep 10

# Step 8: Connect to cluster and restart deployments
log_info "Step 8: Connecting to cluster and restarting deployments..."
if gcloud container clusters get-credentials $CLUSTER_NAME \
    --region=$REGION \
    --project=$PROJECT_ID &>/dev/null; then
    
    # Check if namespace exists
    if kubectl get namespace sweet &>/dev/null; then
        log_info "Restarting deployments in 'sweet' namespace..."
        
        # Restart each deployment if it exists
        for deployment in sweet-operator sweet-scanner sweet-frontier-informer; do
            if kubectl get deployment $deployment -n sweet &>/dev/null; then
                if kubectl rollout restart deployment/$deployment -n sweet &>/dev/null; then
                    log_info "✓ Restarted: $deployment"
                else
                    log_warn "✗ Failed to restart: $deployment"
                fi
            else
                log_warn "Deployment $deployment not found, skipping"
            fi
        done
        
        echo ""
        log_info "Waiting 20 seconds for pods to restart..."
        sleep 20
        
        log_info "Current pod status:"
        kubectl get pods -n sweet 2>/dev/null || log_warn "Could not get pod status"
    else
        log_warn "Namespace 'sweet' does not exist. Deployments may not be installed yet."
    fi
else
    log_warn "Failed to connect to cluster. Please restart deployments manually:"
    log_warn "  kubectl rollout restart deployment -n sweet"
fi

# Step 9: Verify DNS resolution
log_info "Step 9: Verifying DNS resolution..."
if kubectl get namespace sweet &>/dev/null; then
    DNS_RESULT=$(kubectl run test-dns-verify-$(date +%s) \
        --image=busybox \
        --rm -i --restart=Never \
        -n sweet \
        -- nslookup registry.sweet.security 2>&1 | grep -A 2 "Address:" | tail -1 | awk '{print $2}' || echo "")
    
    if [ "$DNS_RESULT" == "$PROXY_IP" ]; then
        log_info "✓ DNS verification: registry.sweet.security → $PROXY_IP"
    else
        log_warn "✗ DNS verification failed: resolved to '$DNS_RESULT', expected '$PROXY_IP'"
        log_warn "  DNS may need more time to propagate (up to 5 minutes)"
    fi
fi

# Final summary
echo ""
log_info "=========================================="
log_info "Fix Summary"
log_info "=========================================="
log_info "Cluster: $CLUSTER_NAME"
log_info "Project: $PROJECT_ID"
log_info "Region: $REGION"
log_info "Proxy IP: $PROXY_IP"
log_info "DNS Zone: $DNS_ZONE"
log_info "Records Updated: $UPDATED"
echo ""
log_info "Next steps:"
log_info "  1. Wait 5-10 minutes for full DNS propagation"
log_info "  2. Check pod status: kubectl get pods -n sweet"
log_info "  3. If pods still in ImagePullBackOff, restart again:"
log_info "     kubectl rollout restart deployment -n sweet"
log_info "=========================================="
