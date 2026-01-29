# Sweet Security Deployment for GKE Autopilot

This directory contains the deployment automation for Sweet Security on GKE Autopilot clusters.

## Quick Start

### Single Cluster

```bash
cd deploy/sweet-security

# Set credentials
export SWEET_API_KEY="your-api-key"
export SWEET_SECRET="your-secret"
export SWEET_CLUSTER_ID="your-cluster-id"

# Deploy
./deploy.sh <CLUSTER_NAME> <PROJECT_ID> <REGION>
```

### Multiple Clusters (400 clusters)

```bash
cd deploy/sweet-security

# Set credentials
export SWEET_API_KEY="your-api-key"
export SWEET_SECRET="your-secret"
export SWEET_CLUSTER_ID="your-cluster-id"

# Create clusters.txt file
cat > clusters.txt <<EOF
cluster1 project1 region1
cluster2 project2 region2
...
EOF

# Deploy to all
./deploy-batch.sh clusters.txt
```

## What Gets Deployed

1. **DNS Proxy** - GCE VM instance forwarding traffic to Sweet Security
2. **Cloud DNS** - Private DNS zone with records for all Sweet Security endpoints
3. **Sweet Operator** - Main operator component (Helm chart)
4. **Sweet Scanner** - Sensor/scanner component (Helm chart)
5. **Frontier Informer** - Cluster metadata collector (Kubernetes deployment)

## Directory Structure

```
deploy/sweet-security/
├── README.md                    # Detailed deployment guide
├── deploy.sh                    # Main deployment script (single cluster)
├── deploy-batch.sh              # Batch deployment script (multiple clusters)
├── clusters.txt.example         # Example batch deployment config
├── configs/                     # Configuration files
│   ├── cluster-config.example.yaml
│   ├── terraform.tfvars.example
│   ├── gcp-proxy.tf
│   └── gke-autopilot.tf.example
├── scripts/                     # Helper scripts
│   ├── deploy-proxy.sh
│   └── proxy-startup-script.sh
└── manifests/                   # Kubernetes manifests
    ├── frontier-manual.yaml
    └── gke-autopilot-proxy.yaml
```

## Documentation

- **[Deployment Guide](deploy/sweet-security/README.md)** - Complete deployment documentation
- **[Deployment Checklist](docs/sweet-security/DEPLOYMENT_CHECKLIST.md)** - Step-by-step checklist
- **[GCP Deployment Guide](docs/sweet-security/GCP_DEPLOYMENT.md)** - GCP-specific details
- **[Troubleshooting](docs/sweet-security/PROXY_NETWORK_ISSUE.md)** - Common issues and solutions

## Prerequisites

- Google Cloud SDK (`gcloud`) installed and authenticated
- `kubectl` installed
- Helm 3.x installed
- Sweet Security API credentials (API key, secret, cluster ID)
- Appropriate GCP permissions (Compute Admin, DNS Admin, Kubernetes Engine Admin)

## Verification

After deployment, verify components:

```bash
# Check all pods are running
kubectl get pods -n sweet

# Check DNS resolution
kubectl run test-dns --image=busybox --rm -i --restart=Never -n sweet -- \
  nslookup registry.sweet.security
# Should resolve to proxy IP (10.0.x.x), not 18.220.208.31

# Check in GCP Console
# Navigate to: Kubernetes Engine > Workloads > sweet namespace
```

## Support

For detailed documentation, see:
- `deploy/sweet-security/README.md` - Complete deployment guide
- `docs/sweet-security/` - Additional documentation
