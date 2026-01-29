# Sweet Security Deployment - File Index

## Quick Navigation

- **[QUICK_START.md](QUICK_START.md)** - Get started in 5 minutes
- **[README.md](README.md)** - Complete deployment documentation
- **[DEPLOYMENT_SUMMARY.md](DEPLOYMENT_SUMMARY.md)** - Project overview

## Scripts

| File | Purpose | Usage |
|------|---------|-------|
| `deploy.sh` | Single cluster deployment | `./deploy.sh <cluster> <project> <region>` |
| `deploy-batch.sh` | Batch deployment | `./deploy-batch.sh clusters.txt` |
| `scripts/deploy-proxy.sh` | Proxy deployment only | `./scripts/deploy-proxy.sh <cluster> <project> <region>` |

## Configuration Files

| File | Purpose |
|------|---------|
| `configs/cluster-config.example.yaml` | Cluster configuration template |
| `configs/terraform.tfvars.example` | Terraform variables template |
| `configs/gcp-proxy.tf` | Terraform proxy configuration |
| `clusters.txt.example` | Batch deployment cluster list template |

## Kubernetes Manifests

| File | Purpose |
|------|---------|
| `manifests/frontier-manual.yaml` | Frontier informer (Autopilot-compatible) |
| `manifests/gke-autopilot-proxy.yaml` | Alternative proxy deployment (K8s) |

## Documentation

All documentation is in `../../docs/sweet-security/`:
- `DEPLOYMENT_CHECKLIST.md` - Step-by-step checklist
- `GCP_DEPLOYMENT.md` - GCP-specific details
- `PROXY_NETWORK_ISSUE.md` - Network troubleshooting
- `PREREQUISITES_CHECK_*.md` - Prerequisites verification

## Usage Examples

### Single Cluster
```bash
export SWEET_API_KEY="key"
export SWEET_SECRET="secret"
export SWEET_CLUSTER_ID="cluster-id"
./deploy.sh sre-771-staging invisible-sre-sandbox us-west1
```

### Batch (Makefile)
```bash
export SWEET_API_KEY="key"
export SWEET_SECRET="secret"
export SWEET_CLUSTER_ID="cluster-id"
make deploy-batch CLUSTERS_FILE=clusters.txt
```

### Batch (Script)
```bash
export SWEET_API_KEY="key"
export SWEET_SECRET="secret"
export SWEET_CLUSTER_ID="cluster-id"
./deploy-batch.sh clusters.txt
```

## Project Structure

```
deploy/sweet-security/
├── INDEX.md                      # This file
├── QUICK_START.md                # Quick start guide
├── README.md                      # Full documentation
├── DEPLOYMENT_SUMMARY.md         # Project summary
├── Makefile                      # Make targets
├── .gitignore                    # Git ignore rules
│
├── deploy.sh                     # Main deployment script
├── deploy-batch.sh               # Batch deployment script
├── clusters.txt.example          # Batch config template
│
├── configs/                      # Configuration files
│   ├── cluster-config.example.yaml
│   ├── terraform.tfvars.example
│   ├── gcp-proxy.tf
│   └── gke-autopilot.tf.example
│
├── scripts/                      # Helper scripts
│   ├── deploy-proxy.sh
│   └── proxy-startup-script.sh
│
└── manifests/                    # Kubernetes manifests
    ├── frontier-manual.yaml
    └── gke-autopilot-proxy.yaml
```
