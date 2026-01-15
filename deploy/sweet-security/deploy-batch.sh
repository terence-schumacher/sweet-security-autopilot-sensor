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
#
# Environment Variables:
#   SKIP_DEPLOYED=true   - Skip clusters that already have Sweet Security deployed (default: true)
#   FORCE_REDEPLOY=true  - Force redeploy even on already deployed clusters (default: false)
#
# Examples:
#   ./deploy-batch.sh                           # Skip already deployed clusters
#   SKIP_DEPLOYED=false ./deploy-batch.sh       # Deploy to all clusters regardless of status
#   FORCE_REDEPLOY=true ./deploy-batch.sh       # Force redeploy to all clusters

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_SCRIPT="${SCRIPT_DIR}/deploy.sh"
CLUSTERS_FILE="${1:-"${SCRIPT_DIR}/all-clusters.txt"}"
SKIP_DEPLOYED="${SKIP_DEPLOYED:-true}"  # Skip already deployed clusters by default
FORCE_REDEPLOY="${FORCE_REDEPLOY:-false}"  # Set to true to force redeploy

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

# Check if Sweet Security is already deployed to a cluster
check_deployment_status() {
    local cluster="$1"
    local project="$2"
    local region="$3"

    log_info "Checking deployment status for $cluster..."

    # Connect to cluster
    if ! gcloud container clusters get-credentials "$cluster" \
        --region="$region" \
        --project="$project" \
        --quiet 2>/dev/null; then
        log_warn "Could not connect to cluster $cluster for status check"
        return 1
    fi

    # Check if sweet namespace exists
    if ! kubectl get namespace sweet &>/dev/null; then
        log_info "Sweet namespace not found - cluster not deployed"
        return 1
    fi

    # Check if Sweet Security components are deployed and ready
    local components=("sweet-operator" "sweet-scanner" "sweet-frontier-informer")
    local deployed_components=0

    for component in "${components[@]}"; do
        if kubectl get deployment "$component" -n sweet &>/dev/null; then
            local ready
            local desired
            ready=$(kubectl get deployment "$component" -n sweet -o jsonpath='{.status.readyReplicas}' 2>/dev/null) || ready="0"
            desired=$(kubectl get deployment "$component" -n sweet -o jsonpath='{.status.replicas}' 2>/dev/null) || desired="0"

            if [ "$ready" = "$desired" ] && [ "$ready" != "0" ]; then
                log_info "✓ $component is deployed and ready ($ready/$desired)"
                deployed_components=$((deployed_components + 1))
            else
                log_warn "△ $component exists but not ready ($ready/$desired)"
            fi
        else
            log_info "✗ $component not found"
        fi
    done

    if [ $deployed_components -eq ${#components[@]} ]; then
        log_info "Sweet Security is fully deployed and ready on $cluster"
        return 0
    elif [ $deployed_components -gt 0 ]; then
        log_warn "Sweet Security is partially deployed on $cluster ($deployed_components/${#components[@]} components ready)"
        return 2  # Partial deployment
    else
        log_info "Sweet Security is not deployed on $cluster"
        return 1
    fi
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
log_info "Found $TOTAL cluster\(s\) to deploy"
log_info "Starting batch deployment..."

SUCCESS=0
FAILED=0
SKIPPED=0
FAILED_CLUSTERS=()
SKIPPED_CLUSTERS=()

# Process each cluster
LINE_NUM=0
while IFS= read -r line || [ -n "${line}" ]; do
    LINE_NUM=$((LINE_NUM + 1))
    
    # Skip empty lines and comments
    [[ -z "${line}" || "${line}" =~ ^# ]] && continue
    
    # Parse line
    read -r cluster project region <<< "${line}"
    
    if [ -z "$cluster" ] || [ -z "$project" ] || [ -z "$region" ]; then
        log_warn "Skipping invalid line $LINE_NUM: $line"
        continue
    fi
    
    log_info ""
    log_info "=========================================="
    log_info "Processing cluster $LINE_NUM/$TOTAL: $cluster"
    log_info "=========================================="

    # Check deployment status first (disable set -e temporarily)
    set +e
    check_deployment_status "$cluster" "$project" "$region"
    deployment_status=$?
    set -e

    should_deploy=false

    case $deployment_status in
        0)  # Fully deployed
            if [ "$FORCE_REDEPLOY" = "true" ]; then
                log_warn "Sweet Security already deployed, but FORCE_REDEPLOY=true - proceeding with deployment"
                should_deploy=true
            elif [ "$SKIP_DEPLOYED" = "true" ]; then
                log_info "✓ Sweet Security already deployed - skipping"
                SKIPPED=$((SKIPPED + 1))
                SKIPPED_CLUSTERS+=("$cluster")
            else
                log_info "Sweet Security already deployed - proceeding with deployment (SKIP_DEPLOYED=false)"
                should_deploy=true
            fi
            ;;
        1)  # Not deployed
            log_info "Sweet Security not deployed - proceeding with deployment"
            should_deploy=true
            ;;
        2)  # Partially deployed
            log_warn "Sweet Security partially deployed - proceeding with deployment to fix"
            should_deploy=true
            ;;
        *)  # Connection/check failed
            log_warn "Could not check deployment status - proceeding with deployment"
            should_deploy=true
            ;;
    esac

    if [ "$should_deploy" = "true" ]; then
        log_info "Deploying to cluster: $cluster"
        # Disable set -e for deployment to handle failures gracefully
        set +e
        "$DEPLOY_SCRIPT" "$cluster" "$project" "$region"
        deploy_result=$?
        set -e

        if [ $deploy_result -eq 0 ]; then
            SUCCESS=$((SUCCESS + 1))
            log_info "✓ Successfully deployed to $cluster"
        else
            FAILED=$((FAILED + 1))
            FAILED_CLUSTERS+=("$cluster")
            log_error "✗ Failed to deploy to $cluster (exit code: $deploy_result)"
        fi
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
log_info "Skipped: $SKIPPED"

if [ $SKIPPED -gt 0 ]; then
    log_info ""
    log_info "Skipped clusters (already deployed):"
    for cluster in "${SKIPPED_CLUSTERS[@]}"; do
        log_info "  - $cluster"
    done
fi

if [ $FAILED -gt 0 ]; then
    log_error ""
    log_error "Failed clusters:"
    for cluster in "${FAILED_CLUSTERS[@]}"; do
        log_error "  - $cluster"
    done
    log_error ""
    log_error "Please Review logs above for each failed cluster"
    exit 1
else
    exit 0
fi
