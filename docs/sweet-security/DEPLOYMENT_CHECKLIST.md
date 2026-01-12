# Sweet Security Deployment Checklist for 400 Clusters

## Prerequisites Per Cluster

- [ ] GKE Autopilot cluster created
- [ ] VPC network and subnet identified
- [ ] DNS proxy instance created (or shared proxy configured)
- [ ] Cloud DNS private zone created for cluster's VPC
- [ ] DNS records configured (see DNS Configuration section)
- [ ] kubectl configured for cluster
- [ ] Helm 3.x installed
- [ ] Sweet Security API credentials obtained

## Step-by-Step Deployment

### 1. Create DNS Proxy (if not using shared proxy)
```bash
# Use terraform or gcloud to create proxy instance
# See: gcp-proxy.tf for Terraform configuration
```

### 2. Configure Cloud DNS
```bash
# Create private DNS zone
gcloud dns managed-zones create sweet-security-zone \
  --dns-name=sweet.security. \
  --visibility=private \
  --networks=<VPC_NETWORK> \
  --project=<PROJECT_ID>

# Get proxy IP
PROXY_IP=$(gcloud compute instances describe sweet-proxy \
  --zone=<ZONE> \
  --project=<PROJECT_ID> \
  --format="get(networkInterfaces[0].networkIP)")

# Create DNS records
gcloud dns record-sets create *.sweet.security. \
  --zone=sweet-security-zone \
  --type=A \
  --rrdatas=$PROXY_IP \
  --ttl=300 \
  --project=<PROJECT_ID>

gcloud dns record-sets create registry.sweet.security. \
  --zone=sweet-security-zone \
  --type=A \
  --rrdatas=$PROXY_IP \
  --ttl=300 \
  --project=<PROJECT_ID>

gcloud dns record-sets create control.sweet.security. \
  --zone=sweet-security-zone \
  --type=A \
  --rrdatas=$PROXY_IP \
  --ttl=300 \
  --project=<PROJECT_ID>

gcloud dns record-sets create logger.sweet.security. \
  --zone=sweet-security-zone \
  --type=A \
  --rrdatas=$PROXY_IP \
  --ttl=300 \
  --project=<PROJECT_ID>

gcloud dns record-sets create receiver.sweet.security. \
  --zone=sweet-security-zone \
  --type=A \
  --rrdatas=$PROXY_IP \
  --ttl=300 \
  --project=<PROJECT_ID>

gcloud dns record-sets create vincent.sweet.security. \
  --zone=sweet-security-zone \
  --type=A \
  --rrdatas=$PROXY_IP \
  --ttl=300 \
  --project=<PROJECT_ID>

gcloud dns record-sets create api.sweet.security. \
  --zone=sweet-security-zone \
  --type=A \
  --rrdatas=$PROXY_IP \
  --ttl=300 \
  --project=<PROJECT_ID>
```

### 3. Connect to Cluster
```bash
gcloud container clusters get-credentials <CLUSTER_NAME> \
  --region=<REGION> \
  --project=<PROJECT_ID>
```

### 4. Create Namespace
```bash
kubectl create namespace sweet
```

### 5. Install Sweet Operator
```bash
helm install sweet-operator oci://registry.sweet.security/helm/operatorchart \
  --namespace sweet \
  --set sweet.apiKey=<API_KEY> \
  --set sweet.secret=<SECRET> \
  --set sweet.clusterId=<CLUSTER_ID> \
  --set frontier.extraValues.serviceAccount.create=true \
  --set frontier.extraValues.priorityClass.enabled=true \
  --set frontier.extraValues.priorityClass.value=1000
```

### 6. Install Sweet Scanner (Sensor)
```bash
helm install sweet-scanner oci://registry.sweet.security/helm/scannerchart \
  --namespace sweet \
  --set sweet.apiKey=<API_KEY> \
  --set sweet.secret=<SECRET> \
  --set sweet.clusterId=<CLUSTER_ID>
```

### 7. Deploy Frontier Informer (Manual - Autopilot Compatible)
```bash
# The operator will try to deploy frontier but will fail due to Autopilot restrictions
# Deploy manually using the frontier-manual.yaml
kubectl apply -f frontier-manual.yaml
```

### 8. Wait for DNS Propagation
```bash
# Wait 5-10 minutes for DNS to propagate
# Verify DNS resolution:
kubectl run test-dns --image=busybox --rm -i --restart=Never -n sweet -- \
  nslookup registry.sweet.security
# Should resolve to proxy IP, not 18.220.208.31
```

### 9. Restart Deployments (if needed)
```bash
kubectl rollout restart deployment/sweet-operator -n sweet
kubectl rollout restart deployment/sweet-frontier-informer -n sweet
kubectl rollout restart daemonset/sweet-scanner -n sweet  # if DaemonSet
```

### 10. Verify Deployment
```bash
# Check all pods are running
kubectl get pods -n sweet

# Expected output:
# sweet-operator-xxx             2/2     Running
# sweet-frontier-informer-xxx     2/2     Running  
# sweet-scanner-xxx               1/1     Running (or multiple if DaemonSet)

# Check Helm releases
helm list -n sweet

# Verify in GCP Console
# Navigate to: Kubernetes Engine > Workloads > sweet namespace
```

## Verification Commands

```bash
# Check all components exist
kubectl get deployment,daemonset -n sweet

# Check pod status
kubectl get pods -n sweet -o wide

# Check DNS resolution from pod
kubectl run test-dns --image=busybox --rm -i --restart=Never -n sweet -- \
  nslookup registry.sweet.security

# Check image pull success
kubectl describe pod -n sweet <POD_NAME> | grep -A 5 "Events:"

# Check metrics (if operator is running)
kubectl port-forward -n sweet deployment/sweet-operator 8080:8080 &
curl http://localhost:8080/metrics
```

## Common Issues & Solutions

### Issue: ImagePullBackOff
**Cause:** DNS not resolving through proxy or DNS not propagated
**Solution:**
1. Verify DNS records exist in Cloud DNS
2. Wait 5-10 minutes for propagation
3. Verify DNS zone is linked to correct VPC network
4. Restart pods after DNS propagation

### Issue: Frontier Installation Fails
**Cause:** Autopilot blocks privileged containers
**Solution:** Use manual deployment (frontier-manual.yaml) instead of operator-managed

### Issue: Components Not Visible in GCP Console
**Cause:** May take a few minutes to appear, or namespace filter issue
**Solution:**
1. Check namespace filter in GCP Console
2. Verify kubectl shows resources
3. Refresh GCP Console

## Automation Script Template

```bash
#!/bin/bash
# deploy-sweet-security.sh

set -e

CLUSTER_NAME=$1
PROJECT_ID=$2
REGION=$3
API_KEY=$4
SECRET=$5
CLUSTER_ID=$6
VPC_NETWORK=$7
PROXY_IP=$8

# Connect to cluster
gcloud container clusters get-credentials $CLUSTER_NAME \
  --region=$REGION \
  --project=$PROJECT_ID

# Create namespace
kubectl create namespace sweet --dry-run=client -o yaml | kubectl apply -f -

# Install operator
helm upgrade --install sweet-operator oci://registry.sweet.security/helm/operatorchart \
  --namespace sweet \
  --set sweet.apiKey=$API_KEY \
  --set sweet.secret=$SECRET \
  --set sweet.clusterId=$CLUSTER_ID \
  --set frontier.extraValues.serviceAccount.create=true

# Install scanner
helm upgrade --install sweet-scanner oci://registry.sweet.security/helm/scannerchart \
  --namespace sweet \
  --set sweet.apiKey=$API_KEY \
  --set sweet.secret=$SECRET \
  --set sweet.clusterId=$CLUSTER_ID

# Deploy frontier manually
kubectl apply -f frontier-manual.yaml

echo "Deployment complete. Wait 5-10 minutes for DNS propagation, then restart pods."
```

## Per-Project Proxy Configuration

For 400 clusters, consider:

1. **Shared Proxy Approach:**
   - One proxy instance per GCP project
   - All clusters in project use same proxy
   - Single DNS zone per project

2. **Per-Cluster Proxy:**
   - One proxy per cluster
   - More isolated but more resources
   - Use Terraform to automate

3. **Regional Proxy:**
   - One proxy per region
   - Shared across multiple clusters
   - Requires careful network configuration

## DNS Records Required Per Cluster/Project

All these should point to the proxy IP:
- `*.sweet.security` (wildcard)
- `registry.sweet.security`
- `control.sweet.security`
- `logger.sweet.security`
- `receiver.sweet.security`
- `vincent.sweet.security`
- `api.sweet.security`
- `prio.sweet.security` (if used)
