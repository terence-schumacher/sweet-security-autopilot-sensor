# Sweet Security Deployment - Project Summary

## Project Structure

This project has been organized for repeatable deployment across 400+ GKE Autopilot clusters.

### Main Components

1. **Deployment Scripts** (`deploy/sweet-security/`)
   - `deploy.sh` - Single cluster deployment
   - `deploy-batch.sh` - Batch deployment for multiple clusters
   - `scripts/deploy-proxy.sh` - Proxy deployment automation

2. **Configuration Files** (`deploy/sweet-security/configs/`)
   - `cluster-config.example.yaml` - Cluster configuration template
   - `terraform.tfvars.example` - Terraform variables template
   - `gcp-proxy.tf` - Terraform configuration for proxy

3. **Kubernetes Manifests** (`deploy/sweet-security/manifests/`)
   - `frontier-manual.yaml` - Autopilot-compatible frontier deployment
   - `gke-autopilot-proxy.yaml` - Alternative proxy deployment (Kubernetes)

4. **Documentation** (`docs/sweet-security/`)
   - Deployment guides and troubleshooting

## Deployment Process

### Automated Deployment (Recommended)

The `deploy.sh` script automates the entire process:

1. ✅ Detects cluster network/subnet
2. ✅ Deploys DNS proxy on cluster's network
3. ✅ Creates/updates Cloud DNS zone
4. ✅ Creates all required DNS records
5. ✅ Connects to cluster
6. ✅ Installs Sweet Operator (Helm)
7. ✅ Installs Sweet Scanner (Helm)
8. ✅ Deploys Frontier Informer (Kubernetes)
9. ✅ Verifies DNS resolution
10. ✅ Restarts deployments

### Manual Deployment

If you need to deploy manually, follow the steps in:
- `docs/sweet-security/DEPLOYMENT_CHECKLIST.md`

## Key Features

### Network Isolation Support
- Each cluster can have its own proxy on its network
- Script automatically detects and deploys to correct network
- No manual network configuration needed

### DNS Automation
- Automatically creates DNS zone if needed
- Adds cluster network to existing zone
- Creates all required DNS records pointing to proxy

### Autopilot Compatibility
- Frontier informer deployed without privileged access
- All components work within Autopilot constraints
- Sensor component handled separately (if needed)

### Batch Deployment
- Deploy to hundreds of clusters with single command
- Progress tracking and error reporting
- Continues on failure (logs which clusters failed)

## Usage Examples

### Single Cluster
```bash
export SWEET_API_KEY="key"
export SWEET_SECRET="secret"
export SWEET_CLUSTER_ID="cluster-id"

./deploy.sh sre-771-staging invisible-sre-sandbox us-west1
```

### Batch Deployment
```bash
# Create clusters.txt
cat > clusters.txt <<EOF
cluster1 project1 region1
cluster2 project2 region2
EOF

# Deploy all
export SWEET_API_KEY="key"
export SWEET_SECRET="secret"
export SWEET_CLUSTER_ID="cluster-id"

./deploy-batch.sh clusters.txt
```

## Verification Checklist

After deployment, verify:

- [ ] Proxy instance created and running
- [ ] DNS zone includes cluster network
- [ ] DNS records point to proxy IP
- [ ] DNS resolves correctly from pods
- [ ] sweet-operator pod running
- [ ] sweet-scanner pod running
- [ ] sweet-frontier-informer pod running
- [ ] All pods can pull images successfully
- [ ] Components visible in GCP Console

## Troubleshooting

Common issues and solutions documented in:
- `docs/sweet-security/PROXY_NETWORK_ISSUE.md`
- `docs/sweet-security/PREREQUISITES_CHECK_*.md`
- `deploy/sweet-security/README.md`

## Next Steps for 400 Clusters

1. **Prepare cluster list:**
   ```bash
   # Export cluster list from your inventory
   # Format: CLUSTER_NAME PROJECT_ID REGION
   ```

2. **Set up credentials:**
   ```bash
   export SWEET_API_KEY="..."
   export SWEET_SECRET="..."
   export SWEET_CLUSTER_ID="..."  # May be same for all or per-cluster
   ```

3. **Test on a few clusters first:**
   ```bash
   # Test on 2-3 clusters
   ./deploy-batch.sh test-clusters.txt
   ```

4. **Deploy to all:**
   ```bash
   ./deploy-batch.sh all-clusters.txt
   ```

5. **Monitor and verify:**
   - Check deployment logs
   - Verify pods are running
   - Check GCP Console

## Files Organization

```
autopilot-security-sensor/
├── README_SWEET_SECURITY.md          # Main entry point
├── deploy/
│   └── sweet-security/               # Deployment automation
│       ├── deploy.sh                 # Single cluster deployment
│       ├── deploy-batch.sh           # Batch deployment
│       ├── README.md                 # Detailed guide
│       ├── configs/                  # Configuration templates
│       ├── scripts/                  # Helper scripts
│       └── manifests/                # Kubernetes manifests
└── docs/
    └── sweet-security/               # Documentation
        ├── DEPLOYMENT_CHECKLIST.md
        ├── GCP_DEPLOYMENT.md
        └── PROXY_NETWORK_ISSUE.md
```
