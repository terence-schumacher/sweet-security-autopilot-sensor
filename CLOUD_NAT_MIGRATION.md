# Cloud NAT Migration Guide

## Overview

The deployment has been migrated from proxy VM instances to **Cloud NAT** for a more secure solution.

## What Changed

### Before (Proxy VM Approach)
- ❌ Individual GCE VM instances per cluster/network
- ❌ Manual iptables configuration for NAT
- ❌ DNS configuration required (private DNS zones)
- ❌ Proxy IP management
- ❌ Higher operational overhead

### After (Cloud NAT Approach)
- ✅ Managed NAT service (no VMs to manage)
- ✅ Automatic scaling
- ✅ Shared NAT gateway per region
- ✅ No DNS configuration needed
- ✅ Direct access to external IPs
- ✅ Lower cost and operational overhead

## Benefits

1. **Scalability**: One NAT gateway can serve multiple clusters in the same region
2. **Cost**: No VM instances to run and maintain
3. **Simplicity**: No DNS configuration or proxy IP management
4. **Reliability**: Managed service with automatic failover
5. **Performance**: Lower latency, no proxy hop

## How It Works

Cloud NAT provides outbound NAT for private GKE clusters:

1. **Cloud Router**: Created per region to manage NAT
2. **Cloud NAT Gateway**: Attached to router, handles all outbound traffic
3. **Automatic NAT**: All private IPs in the subnet get NAT'd automatically
4. **Direct Access**: Clusters can directly access Sweet Security endpoints (18.220.208.31)

## Deployment

The deployment script now automatically creates Cloud NAT:

```bash
./deploy.sh <CLUSTER_NAME> <PROJECT_ID> <REGION>
```

This will:
1. Detect the cluster's network and subnet
2. Create/verify Cloud Router exists
3. Create/verify Cloud NAT gateway exists
4. Deploy Sweet Security components

## Manual Cloud NAT Setup

If you need to set up Cloud NAT manually:

```bash
# 1. Create Cloud Router
gcloud compute routers create sweet-nat-router-REGION \
    --network=NETWORK_NAME \
    --region=REGION \
    --project=PROJECT_ID

# 2. Create Cloud NAT (Private NAT with auto-allocated external IPs)
gcloud compute routers nats create sweet-nat-REGION \
    --router=sweet-nat-router-REGION \
    --region=REGION \
    --nat-all-subnet-ip-ranges \
    --auto-allocate-nat-external-ips \
    --enable-logging \
    --project=PROJECT_ID
```

## Migration from Proxy VMs

If you have existing proxy VMs, you can:

1. **Deploy Cloud NAT** (new deployments use it automatically)
2. **Keep existing proxies** (they'll continue to work)
3. **Gradually migrate** clusters to Cloud NAT
4. **Remove proxy VMs** once all clusters are migrated

## Verification

Check Cloud NAT status:

```bash
# List NAT gateways
gcloud compute routers nats list \
    --router=sweet-nat-router-REGION \
    --region=REGION \
    --project=PROJECT_ID

# Check router status
gcloud compute routers describe sweet-nat-router-REGION \
    --region=REGION \
    --project=PROJECT_ID
```

## Troubleshooting

### NAT Not Working

1. **Check router exists**:
   ```bash
   gcloud compute routers list --project=PROJECT_ID
   ```

2. **Check NAT configuration**:
   ```bash
   gcloud compute routers nats describe sweet-nat-REGION \
       --router=sweet-nat-router-REGION \
       --region=REGION \
       --project=PROJECT_ID
   ```

3. **Verify subnet is included**:
   - NAT should have `--nat-all-subnet-ip-ranges` or specific subnet ranges

### Connectivity Issues

1. **Test from a pod**:
   ```bash
   kubectl run test-curl --image=curlimages/curl --rm -i --restart=Never \
       -- curl -v https://control.sweet.security
   ```

2. **Check firewall rules**:
   - Ensure egress is allowed (default for private clusters)

3. **Check NAT logs** (if enabled):
   ```bash
   gcloud logging read "resource.type=cloud_nat" --limit=50
   ```

## Cost Comparison

### Proxy VM Approach
- **Per cluster**: ~$30-50/month (e2-standard-2 VM)
- **400 clusters**: ~$12,000-20,000/month
- **Plus**: DNS zone costs, management overhead

### Cloud NAT Approach
- **Per region**: ~$45/month (NAT gateway)
- **400 clusters (10 regions)**: ~$450/month
- **Savings**: ~95% cost reduction

## Notes

- Cloud NAT is **region-specific** - one NAT per region
- Multiple clusters in the same region share the same NAT
- NAT automatically handles all outbound traffic
- No DNS configuration needed - direct IP access works
- Sweet Security endpoints are accessible via their public IPs (18.220.208.31)
