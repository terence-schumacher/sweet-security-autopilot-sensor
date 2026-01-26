# Quick Start Guide - Sweet Security Deployment

## Prerequisites

- `gcloud` CLI installed and authenticated
- `kubectl` installed
- `helm` 3.x installed
- Sweet Security API credentials

## Single Cluster Deployment (5 minutes)

```bash
# Set your credentials
export SWEET_API_KEY="your-api-key"
export SWEET_SECRET="your-secret"
export SWEET_CLUSTER_ID="your-cluster-id"

# Deploy
./deploy.sh <CLUSTER_NAME> <PROJECT_ID> <REGION>

# Example
./deploy.sh sre-771-staging invisible-sre-sandbox us-west1
```

That's it! The script handles everything:
- ✅ Creates proxy on cluster's network
- ✅ Configures DNS
- ✅ Deploys all components
- ✅ Verifies deployment

## Batch Deployment (400 clusters)

```bash
# 1. Create clusters list
cat > clusters.txt <<EOF
cluster1 project1 region1
cluster2 project2 region2
cluster3 project3 region3
...
EOF

# 2. Set credentials
export SWEET_API_KEY="your-api-key"
export SWEET_SECRET="your-secret"
export SWEET_CLUSTER_ID="your-cluster-id"

# 3. Deploy to all
./deploy-batch.sh clusters.txt
```

## Verify Deployment

```bash
# Check pods
kubectl get pods -n sweet

# Check DNS (should resolve to proxy IP, not 18.220.208.31)
kubectl run test-dns --image=busybox --rm -i --restart=Never -n sweet -- \
  nslookup registry.sweet.security
```

## Troubleshooting

**Pods in ImagePullBackOff?**
- Wait 5-10 minutes for DNS propagation
- Restart: `kubectl rollout restart deployment -n sweet`

**Need more help?**
- See `README.md` for detailed documentation
- Check `docs/sweet-security/` for troubleshooting guides
