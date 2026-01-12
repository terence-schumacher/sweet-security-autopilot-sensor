# Sweet Security Deployment - Project Structure

## Overview

This project has been organized and formalized for repeatable deployment across 400+ GKE Autopilot clusters. All deployment automation is centralized in `deploy/sweet-security/`.

## Directory Structure

```
autopilot-security-sensor/
├── README_SWEET_SECURITY.md          # Main entry point for Sweet Security deployment
│
├── deploy/
│   └── sweet-security/               # 🎯 MAIN DEPLOYMENT DIRECTORY
│       ├── deploy.sh                 # Single cluster deployment (automated)
│       ├── deploy-batch.sh           # Batch deployment for 400 clusters
│       ├── Makefile                  # Make targets for deployment
│       │
│       ├── QUICK_START.md            # 5-minute quick start guide
│       ├── README.md                 # Complete deployment documentation
│       ├── INDEX.md                  # File index and navigation
│       ├── DEPLOYMENT_SUMMARY.md     # Project summary
│       │
│       ├── configs/                   # Configuration templates
│       │   ├── cluster-config.example.yaml
│       │   ├── terraform.tfvars.example
│       │   ├── gcp-proxy.tf
│       │   └── gke-autopilot.tf.example
│       │
│       ├── scripts/                   # Helper scripts
│       │   ├── deploy-proxy.sh       # Proxy deployment automation
│       │   └── proxy-startup-script.sh
│       │
│       └── manifests/                 # Kubernetes manifests
│           ├── frontier-manual.yaml  # Frontier informer (Autopilot-compatible)
│           └── gke-autopilot-proxy.yaml
│
└── docs/
    └── sweet-security/                # Documentation
        ├── DEPLOYMENT_CHECKLIST.md
        ├── GCP_DEPLOYMENT.md
        ├── PROXY_NETWORK_ISSUE.md
        ├── PREREQUISITES_CHECK_*.md
        └── VERIFICATION_REPORT.md
```

## Key Files

### Deployment Scripts

1. **`deploy/sweet-security/deploy.sh`**
   - Main deployment script for single cluster
   - Automates all steps: proxy, DNS, Kubernetes components
   - Usage: `./deploy.sh <CLUSTER> <PROJECT> <REGION>`

2. **`deploy/sweet-security/deploy-batch.sh`**
   - Batch deployment for multiple clusters
   - Reads from `clusters.txt` file
   - Provides progress tracking and error reporting

3. **`deploy/sweet-security/scripts/deploy-proxy.sh`**
   - Standalone proxy deployment
   - Can be used independently if needed

### Configuration Files

1. **`configs/cluster-config.example.yaml`**
   - Template for cluster-specific configuration
   - Copy to `cluster-config.yaml` and customize

2. **`configs/terraform.tfvars.example`**
   - Terraform variables template
   - For Terraform-based proxy deployment

3. **`configs/gcp-proxy.tf`**
   - Terraform configuration for proxy
   - Can be used instead of script-based deployment

### Kubernetes Manifests

1. **`manifests/frontier-manual.yaml`**
   - Autopilot-compatible frontier informer
   - Uses placeholders that are replaced during deployment
   - No privileged access required

## Quick Start

### Single Cluster
```bash
cd deploy/sweet-security
export SWEET_API_KEY="key"
export SWEET_SECRET="secret"
export SWEET_CLUSTER_ID="cluster-id"
./deploy.sh sre-771-staging invisible-sre-sandbox us-west1
```

### 400 Clusters
```bash
cd deploy/sweet-security

# Create cluster list
cat > clusters.txt <<EOF
cluster1 project1 region1
cluster2 project2 region2
...
EOF

# Set credentials
export SWEET_API_KEY="key"
export SWEET_SECRET="secret"
export SWEET_CLUSTER_ID="cluster-id"

# Deploy all
./deploy-batch.sh clusters.txt
```

## What Gets Deployed

The deployment script automates:

1. ✅ **DNS Proxy** - GCE VM on cluster's network
2. ✅ **Cloud DNS Zone** - Private DNS zone for cluster's VPC
3. ✅ **DNS Records** - All Sweet Security endpoints → proxy IP
4. ✅ **Sweet Operator** - Helm chart deployment
5. ✅ **Sweet Scanner** - Helm chart deployment
6. ✅ **Frontier Informer** - Kubernetes deployment (Autopilot-compatible)

## Features

- **Network-Aware**: Automatically detects and deploys to correct network
- **DNS Automation**: Creates/updates DNS zone and records automatically
- **Autopilot-Compatible**: All components work within Autopilot constraints
- **Batch Support**: Deploy to hundreds of clusters with single command
- **Error Handling**: Continues on failure, reports which clusters failed
- **Verification**: Built-in DNS and connectivity checks

## Documentation

- **Quick Start**: `deploy/sweet-security/QUICK_START.md`
- **Full Guide**: `deploy/sweet-security/README.md`
- **Checklist**: `docs/sweet-security/DEPLOYMENT_CHECKLIST.md`
- **Troubleshooting**: `docs/sweet-security/PROXY_NETWORK_ISSUE.md`

## Cleanup

Temporary files have been removed:
- ✅ Removed `temp_proxy_plan.txt`
- ✅ Removed `Untitled-1.json`
- ✅ Removed Terraform state files (should be in .gitignore)
- ✅ Organized all files into proper directories

## Next Steps

1. **Test on a few clusters** to verify the process
2. **Prepare cluster list** for batch deployment
3. **Set up credentials** securely (consider using secret management)
4. **Run batch deployment** for all 400 clusters
5. **Monitor and verify** deployments

## Support

For issues:
1. Check `deploy/sweet-security/README.md` troubleshooting section
2. Review logs from deployment script
3. Check `docs/sweet-security/` for specific issues
