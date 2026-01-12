#!/bin/bash
# Batch Deployment Script for Multiple Clusters
# 
# Usage:
#   1. Create a clusters.txt file with one cluster per line:
#      CLUSTER_NAME PROJECT_ID REGION
#
#   2. Set credentials as environment variables:
#      export SWEET_API_KEY="your-api-key"
#      export SWEET_SECRET="your-secret"
#      export SWEET_CLUSTER_ID="your-cluster-id"
#
#   3. Run: ./deploy-batch.sh [clusters.txt]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_SCRIPT="${SCRIPT_DIR}/deploy.sh"
CLUSTERS_FILE=${1:-"${SCRIPT_DIR}/clusters.txt"}"

# Source .env file if it exists
[ -f "${SCRIPT_DIR}/.env" ] && source "${SCRIPT_DIR}/.env"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if clusters file exists
if [ ! -f "$CLUSTERS_FILE" ]; then
    log_error "Clusters file not found: $CLUSTERS_FILE"
    log_info "Create a file with one cluster per line:"
    log_info "  CLUSTER_NAME PROJECT_ID REGION"
    exit 1
fi

# Check credentials
if [ -z "$SWEET_API_KEY" ] || [ -z "$SWEET_SECRET" ] || [ -z "$SWEET_CLUSTER_ID" ]; then
    log_error "Missing credentials. Set environment variables:"
    log_error "  SWEET_API_KEY"
    log_error "  SWEET_SECRET"
    log_error "  SWEET_CLUSTER_ID"
    exit 1
fi

# Count clusters
TOTAL=$(wc -l < "$CLUSTERS_FILE" | tr -d ' ')
log_info "Found $TOTAL cluster(s) to deploy"
log_info "Starting batch deployment..."

SUCCESS=0
FAILED=0
FAILED_CLUSTERS=()

# Process each cluster
LINE_NUM=0
while IFS= read -r line || [ -n "$line" ]; do
    LINE_NUM=$((LINE_NUM + 1))
    
    # Skip empty lines and comments
    [[ -z "$line" || "$line" =~ ^# ]] && continue
    
    # Parse line
    read -r cluster project region <<< "$line"
    
    if [ -z "$cluster" ] || [ -z "$project" ] || [ -z "$region" ]; then
        log_warn "Skipping invalid line $LINE_NUM: $line"
        continue
    fi
    
    log_info ""
    log_info "=========================================="
    log_info "Deploying to cluster $LINE_NUM/$TOTAL: $cluster"
    log_info "=========================================="
    
    if "$DEPLOY_SCRIPT" "$cluster" "$project" "$region"; then
        SUCCESS=$((SUCCESS + 1))
        log_info "✓ Successfully deployed to $cluster"
    else
        FAILED=$((FAILED + 1))
        FAILED_CLUSTERS+=("$cluster")
        log_error "✗ Failed to deploy to $cluster"
    fi
    
    # Small delay between deployments
    sleep 5
    
done < "$CLUSTERS_FILE"

# Summary
log_info ""
log_info "=========================================="
log_info "Batch Deployment Summary"
log_info "=========================================="
log_info "Total clusters: $TOTAL"
log_info "Successful: $SUCCESS"
log_info "Failed: $FAILED"

if [ $FAILED -gt 0 ]; then
    log_error ""
    log_error "Failed clusters:"
    for cluster in "${FAILED_CLUSTERS[@]}"; do
        log_error "  - $cluster"
    done
    log_error ""
    log_error "Review logs above for each failed cluster"
    exit 1
else
    log_info ""
    log_info "All clusters deployed successfully! ✓"
    exit 0
fi
