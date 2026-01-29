# Proxy Network Connectivity Issue

## Problem
- **Proxy Network:** `sre-sandbox-private-onboarding` (IP: 10.0.30.53)
- **Cluster Network:** `sre-771-staging`
- **Issue:** Networks are separate, pods cannot reach proxy (connection timeout)

## Solution Options

### Option 1: Create Proxy on Cluster's Network (Recommended for 400 clusters)
Deploy a proxy instance on each cluster's network.

**Quick Fix for sre-771-staging:**
```bash
# Update terraform.tfvars
network_name = "sre-771-staging"
subnet_name = "sre-771-staging-subnet"  # Get actual subnet name

# Apply terraform
cd autopilot-security-sensor
terraform apply
```

### Option 2: VPC Peering (If networks should communicate)
```bash
# Create peering from sre-771-staging to sre-sandbox-private-onboarding
gcloud compute networks peerings create sre-771-staging-to-sandbox \
  --network=sre-771-staging \
  --peer-network=sre-sandbox-private-onboarding \
  --project=invisible-sre-sandbox

# Create reverse peering
gcloud compute networks peerings create sandbox-to-sre-771-staging \
  --network=sre-sandbox-private-onboarding \
  --peer-network=sre-771-staging \
  --project=invisible-sre-sandbox
```

### Option 3: Shared Proxy with Proper Routing
If using a shared proxy, ensure:
- Firewall rules allow traffic from all cluster networks
- Routes are configured correctly
- Proxy is accessible from all networks

## Immediate Fix for sre-771-staging

1. **Get cluster subnet:**
```bash
gcloud container clusters describe sre-771-staging \
  --region=us-west1 \
  --project=invisible-sre-sandbox \
  --format="get(subnetwork)"
```

2. **Create proxy on cluster network:**
```bash
# Use terraform with updated network/subnet
# Or create manually with gcloud
```

3. **Update DNS records to point to new proxy IP**
