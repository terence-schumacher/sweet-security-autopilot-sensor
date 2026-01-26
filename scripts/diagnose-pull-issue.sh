#!/bin/bash
# Diagnostic script for image pull issues
# Usage: ./diagnose-pull-issue.sh <CLUSTER_NAME> <PROJECT_ID> <REGION>

set -e

CLUSTER_NAME=${1:-client-analytics-production}
PROJECT_ID=${2:-invisible-client-analytics}
REGION=${3:-us-west1}

echo "=========================================="
echo "Diagnosing Image Pull Issue"
echo "=========================================="
echo "Cluster: $CLUSTER_NAME"
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo ""

# Connect to cluster
echo "1. Connecting to cluster..."
gcloud container clusters get-credentials $CLUSTER_NAME \
  --region=$REGION \
  --project=$PROJECT_ID 2>&1 || {
  echo "ERROR: Failed to connect to cluster"
  exit 1
}

# Check namespace
echo ""
echo "2. Checking namespace..."
kubectl get namespace sweet 2>&1 || echo "WARNING: Namespace 'sweet' does not exist"

# Check pod status
echo ""
echo "3. Checking pod status..."
kubectl get pods -n sweet 2>&1 || echo "WARNING: No pods found in 'sweet' namespace"

# Check pod events
echo ""
echo "4. Checking pod events for image pull errors..."
kubectl get events -n sweet --sort-by='.lastTimestamp' | tail -20 || echo "No events found"

# Check specific pod errors
echo ""
echo "5. Checking sweet-operator pod details..."
OPERATOR_POD=$(kubectl get pods -n sweet -l app.kubernetes.io/name=sweet-operator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$OPERATOR_POD" ]; then
  echo "Operator pod: $OPERATOR_POD"
  kubectl describe pod $OPERATOR_POD -n sweet | grep -A 10 "Events:" || echo "No events"
  kubectl describe pod $OPERATOR_POD -n sweet | grep -A 5 "Image:" || echo "No image info"
else
  echo "No operator pod found"
fi

# Check DNS zone
echo ""
echo "6. Checking DNS zone configuration..."
DNS_ZONE="sweet-security-zone"
if gcloud dns managed-zones describe $DNS_ZONE --project=$PROJECT_ID &>/dev/null; then
  echo "DNS zone exists: $DNS_ZONE"
  echo "Networks in zone:"
  gcloud dns managed-zones describe $DNS_ZONE --project=$PROJECT_ID \
    --format="get(privateVisibilityConfig.networks[].networkUrl)" || echo "No networks"
  
  echo ""
  echo "DNS records for registry.sweet.security:"
  gcloud dns record-sets describe registry.sweet.security. \
    --type=A \
    --zone=$DNS_ZONE \
    --project=$PROJECT_ID 2>&1 || echo "ERROR: DNS record not found"
else
  echo "ERROR: DNS zone '$DNS_ZONE' does not exist"
fi

# Check proxy
echo ""
echo "7. Checking proxy deployment..."
SUBNET=$(gcloud container clusters describe $CLUSTER_NAME \
  --region=$REGION \
  --project=$PROJECT_ID \
  --format="get(subnetwork)" 2>/dev/null || echo "")

if [ -n "$SUBNET" ]; then
  if [[ "$SUBNET" == *"/regions/"* ]]; then
    SUBNET_REGION=$(echo "$SUBNET" | sed 's|.*/regions/\([^/]*\)/.*|\1|')
  else
    SUBNET_REGION=$(gcloud compute networks subnets describe "$SUBNET" \
      --project=$PROJECT_ID \
      --format="get(region)" 2>/dev/null | sed 's|.*/regions/||')
  fi
  ZONE="${SUBNET_REGION}-a"
  PROXY_NAME="sweet-proxy-${CLUSTER_NAME}"
  
  if gcloud compute instances describe $PROXY_NAME --zone=$ZONE --project=$PROJECT_ID &>/dev/null; then
    PROXY_IP=$(gcloud compute instances describe $PROXY_NAME \
      --zone=$ZONE \
      --project=$PROJECT_ID \
      --format="get(networkInterfaces[0].networkIP)")
    echo "Proxy exists: $PROXY_NAME"
    echo "Proxy IP: $PROXY_IP"
    echo "Proxy status:"
    gcloud compute instances describe $PROXY_NAME \
      --zone=$ZONE \
      --project=$PROJECT_ID \
      --format="get(status)" || echo "Could not get status"
  else
    echo "ERROR: Proxy '$PROXY_NAME' does not exist in zone $ZONE"
  fi
else
  echo "ERROR: Could not determine subnet"
fi

# Test DNS resolution from cluster
echo ""
echo "8. Testing DNS resolution from cluster..."
kubectl run test-dns-$(date +%s) --image=busybox --rm -i --restart=Never -n sweet -- \
  nslookup registry.sweet.security 2>&1 || echo "ERROR: DNS test failed"

echo ""
echo "=========================================="
echo "Diagnosis Complete"
echo "=========================================="
