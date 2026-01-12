# Sweet Security Deployment for GKE Autopilot

This directory contains scripts and configurations for deploying Sweet Security to GKE Autopilot clusters.

## Quick Start

### Prerequisites

1. **Google Cloud SDK** (`gcloud`) installed and authenticated
2. **kubectl** installed
3. **Helm 3.x** installed
4. **Sweet Security API credentials** (API key, secret, cluster ID)

### Single Cluster Deployment

```bash
# Set credentials as environment variables
export SWEET_API_KEY="your-api-key"
export SWEET_SECRET="your-secret"
export SWEET_CLUSTER_ID="your-cluster-id"

# Deploy
./deploy.sh <CLUSTER_NAME> <PROJECT_ID> <REGION>

# Example
./deploy.sh sre-771-staging invisible-sre-sandbox us-west1
```

### Batch Deployment (Multiple Clusters)

```bash
# Create a clusters.txt file with one cluster per line:
# Format: CLUSTER_NAME PROJECT_ID REGION
cat > clusters.txt <<EOF
sre-771-staging invisible-sre-sandbox us-west1
sre-771-production invisible-sre-sandbox us-west1
sre-onboarding invisible-sre-sandbox us-west1
EOF

# Deploy to all clusters
export SWEET_API_KEY="your-api-key"
export SWEET_SECRET="your-secret"
export SWEET_CLUSTER_ID="your-cluster-id"

while read -r cluster project region; do
  echo "Deploying to $cluster..."
  ./deploy.sh "$cluster" "$project" "$region" || echo "Failed: $cluster"
done < clusters.txt
```

## What Gets Deployed

The deployment script automates the following:

1. **DNS Proxy Instance** - GCE VM that forwards traffic to Sweet Security
2. **Cloud DNS Zone** - Private DNS zone for `*.sweet.security` domains
3. **DNS Records** - A records pointing to proxy for all Sweet Security endpoints
4. **Sweet Operator** - Helm chart deployment
5. **Sweet Scanner** - Helm chart deployment (sensor component)
6. **Frontier Informer** - Kubernetes deployment (Autopilot-compatible)

## Directory Structure

```
deploy/sweet-security/
├── README.md                    # This file
├── deploy.sh                     # Main deployment script
├── configs/                      # Configuration files
│   ├── cluster-config.example.yaml
│   ├── terraform.tfvars.example
│   ├── gcp-proxy.tf
│   └── gke-autopilot.tf.example
├── scripts/                      # Helper scripts
│   ├── deploy-proxy.sh          # Proxy deployment script
│   └── proxy-startup-script.sh  # Proxy iptables configuration
└── manifests/                    # Kubernetes manifests
    ├── frontier-manual.yaml     # Frontier informer (Autopilot-compatible)
    └── gke-autopilot-proxy.yaml # Alternative proxy deployment
```

## Configuration

### Using Configuration File

1. Copy the example config:
   ```bash
   cp configs/cluster-config.example.yaml configs/cluster-config.yaml
   ```

2. Edit `configs/cluster-config.yaml` with your values

3. The deployment script will automatically use this file if present

### Using Environment Variables

```bash
export SWEET_API_KEY="your-api-key"
export SWEET_SECRET="your-secret"
export SWEET_CLUSTER_ID="your-cluster-id"
```

### Using Command Line Arguments

```bash
./deploy.sh <CLUSTER_NAME> <PROJECT_ID> <REGION> <API_KEY> <SECRET> <CLUSTER_ID>
```

## Manual Steps (if needed)

### Deploy Proxy Only

```bash
./scripts/deploy-proxy.sh <CLUSTER_NAME> <PROJECT_ID> <REGION>
```

### Configure DNS Only

```bash
# After proxy is deployed, get the IP
PROXY_IP=$(gcloud compute instances describe sweet-proxy-<CLUSTER_NAME> \
  --zone=<REGION>-a \
  --project=<PROJECT_ID> \
  --format="get(networkInterfaces[0].networkIP)")

# Create/update DNS zone
gcloud dns managed-zones update sweet-security-zone \
  --project=<PROJECT_ID> \
  --networks=<NETWORK_NAME>

# Create DNS records
for endpoint in "*.sweet.security" "registry.sweet.security" "control.sweet.security" \
                "logger.sweet.security" "receiver.sweet.security" "vincent.sweet.security" \
                "api.sweet.security" "prio.sweet.security"; do
  gcloud dns record-sets create "${endpoint}." \
    --zone=sweet-security-zone \
    --type=A \
    --rrdatas=$PROXY_IP \
    --ttl=300 \
    --project=<PROJECT_ID>
done
```

### Deploy Kubernetes Components Only

```bash
# Connect to cluster
gcloud container clusters get-credentials <CLUSTER_NAME> \
  --region=<REGION> \
  --project=<PROJECT_ID>

# Install operator
helm upgrade --install sweet-operator oci://registry.sweet.security/helm/operatorchart \
  --namespace sweet \
  --set sweet.apiKey=<API_KEY> \
  --set sweet.secret=<SECRET> \
  --set sweet.clusterId=<CLUSTER_ID>

# Install scanner
helm upgrade --install sweet-scanner oci://registry.sweet.security/helm/scannerchart \
  --namespace sweet \
  --set sweet.apiKey=<API_KEY> \
  --set sweet.secret=<SECRET> \
  --set sweet.clusterId=<CLUSTER_ID>

# Deploy frontier
kubectl apply -f manifests/frontier-manual.yaml
```

## Troubleshooting

### Pods in ImagePullBackOff

1. **Check DNS resolution:**
   ```bash
   kubectl run test-dns --image=busybox --rm -i --restart=Never -n sweet -- \
     nslookup registry.sweet.security
   ```
   Should resolve to proxy IP (10.0.x.x), not 18.220.208.31

2. **Wait for DNS propagation** (5-10 minutes)

3. **Restart deployments:**
   ```bash
   kubectl rollout restart deployment -n sweet
   ```

### Proxy Not Reachable

1. **Verify proxy is running:**
   ```bash
   gcloud compute instances describe sweet-proxy-<CLUSTER_NAME> \
     --zone=<ZONE> \
     --project=<PROJECT_ID>
   ```

2. **Check firewall rules:**
   ```bash
   gcloud compute firewall-rules list --filter="name~sweet-proxy"
   ```

3. **Test connectivity from pod:**
   ```bash
   kubectl run test-connectivity --image=curlimages/curl --rm -i --restart=Never -n sweet -- \
     curl -v --connect-timeout 5 https://10.0.30.53:443
   ```

### DNS Zone Not Working

1. **Verify DNS zone includes cluster network:**
   ```bash
   gcloud dns managed-zones describe sweet-security-zone \
     --project=<PROJECT_ID> \
     --format="get(privateVisibilityConfig.networks[].networkUrl)"
   ```

2. **Add network to zone:**
   ```bash
   gcloud dns managed-zones update sweet-security-zone \
     --project=<PROJECT_ID> \
     --networks=<EXISTING_NETWORKS>,<NEW_NETWORK>
   ```

## Verification

After deployment, verify all components:

```bash
# Check pods
kubectl get pods -n sweet

# Check deployments
kubectl get deployments -n sweet

# Check Helm releases
helm list -n sweet

# Check DNS resolution
kubectl run test-dns --image=busybox --rm -i --restart=Never -n sweet -- \
  nslookup registry.sweet.security

# Check operator metrics
kubectl port-forward -n sweet deployment/sweet-operator 8080:8080 &
curl http://localhost:8080/metrics
```

## Cleanup

To remove Sweet Security from a cluster:

```bash
# Delete Kubernetes resources
helm uninstall sweet-operator -n sweet
helm uninstall sweet-scanner -n sweet
kubectl delete -f manifests/frontier-manual.yaml

# Delete namespace
kubectl delete namespace sweet

# Delete proxy (optional)
gcloud compute instances delete sweet-proxy-<CLUSTER_NAME> \
  --zone=<ZONE> \
  --project=<PROJECT_ID>

# Delete firewall rules (optional)
gcloud compute firewall-rules delete sweet-proxy-<CLUSTER_NAME>-ingress-tcp \
  --project=<PROJECT_ID>
gcloud compute firewall-rules delete sweet-proxy-<CLUSTER_NAME>-ingress-udp \
  --project=<PROJECT_ID>
gcloud compute firewall-rules delete sweet-proxy-<CLUSTER_NAME>-egress \
  --project=<PROJECT_ID>
```

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review logs: `kubectl logs -n sweet <pod-name>`
3. Check events: `kubectl get events -n sweet --sort-by='.lastTimestamp'`
