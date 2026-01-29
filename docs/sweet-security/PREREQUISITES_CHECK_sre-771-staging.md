# Prerequisites Check for sre-771-staging Cluster
**Project:** invisible-sre-sandbox  
**Date:** 2026-01-10

## Checklist Status

### ✅ 1. GKE Autopilot cluster created
- **Status:** ✅ VERIFIED
- **Cluster:** sre-771-staging
- **Region:** us-west1
- **Type:** Autopilot (enabled: True)
- **Network:** sre-771-staging
- **Subnet:** sre-771-staging-subnet

### ✅ 2. VPC network and subnet identified
- **Status:** ✅ VERIFIED
- **Network:** sre-771-staging
- **Subnet:** sre-771-staging-subnet
- **Network URL:** `projects/invisible-sre-sandbox/global/networks/sre-771-staging`

### ⚠️ 3. DNS proxy instance created
- **Status:** ⚠️ PARTIAL
- **Proxy Found:** sweet-proxy (10.0.30.53)
- **Network:** sre-sandbox-private-onboarding
- **Issue:** Proxy is on different network (`sre-sandbox-private-onboarding`) than cluster (`sre-771-staging`)
- **Action Required:** 
  - Either create a proxy on `sre-771-staging` network, OR
  - Ensure networks can communicate (VPC peering/shared VPC), OR
  - Use shared proxy if networks are connected

### ✅ 4. Cloud DNS private zone created for cluster's VPC
- **Status:** ✅ FIXED - Now configured for this cluster
- **DNS Zone Found:** sweet-security-zone
- **Current Networks:** sre-sandbox-private-onboarding
- **Required Network:** sre-771-staging
- **Issue:** DNS zone is only linked to `sre-sandbox-private-onboarding` network, not `sre-771-staging`
- **Action Required:** Add `sre-771-staging` network to DNS zone:
  ```bash
  gcloud dns managed-zones update sweet-security-zone \
    --project=invisible-sre-sandbox \
    --networks=sre-sandbox-private-onboarding,sre-771-staging
  ```
  ✅ **FIXED** - DNS zone now includes both networks

### ⚠️ 5. DNS records configured
- **Status:** ⚠️ PARTIAL
- **Records Found:**
  - `*.sweet.security` → 10.0.30.53 ✅
  - `registry.sweet.security` → 10.0.30.53 ✅
- **Missing Records:**
  - `control.sweet.security` → proxy IP
  - `logger.sweet.security` → proxy IP
  - `receiver.sweet.security` → proxy IP
  - `vincent.sweet.security` → proxy IP
  - `api.sweet.security` → proxy IP
  - `prio.sweet.security` → proxy IP (if used)
- **Note:** Even if records exist, they won't work until DNS zone is linked to cluster's network

### ✅ 6. kubectl configured for cluster
- **Status:** ✅ VERIFIED
- **Cluster:** sre-771-staging
- **Connection:** Successfully connected
- **Control Plane:** Running at gke-c503e37fb8864deb818188e82d780b63c4c3-860673377600.us-west1.gke.goog

### ✅ 7. Helm 3.x installed
- **Status:** ✅ VERIFIED
- **Version:** v3.19.0+g3d8990f
- **Compatible:** Yes (Helm 3.x)

### ✅ 8. Sweet Security API credentials obtained
- **Status:** ✅ VERIFIED
- **Secrets Found:**
  - sweet-operator (contains API credentials)
  - sweet-scanner (contains API credentials)
  - sweet-frontier (contains API credentials)
- **Namespace:** sweet (exists, 2d17h old)

## Current Deployment Status

### Components Found:
- ✅ sweet-operator (Helm release exists)
- ✅ sweet-scanner (Helm release exists, deployed 12m ago)
- ✅ sweet-frontier (Secret exists)

### Pod Status:
- Check with: `kubectl get pods -n sweet`

## Critical Issues to Fix

### 1. DNS Zone Network Mismatch (CRITICAL)
**Problem:** DNS zone `sweet-security-zone` is only linked to `sre-sandbox-private-onboarding` network, but cluster uses `sre-771-staging` network.

**Impact:** Pods in this cluster cannot resolve `*.sweet.security` domains, causing ImagePullBackOff errors.

**Solution Options:**

**Option A: Add network to existing DNS zone (Recommended if networks can communicate)**
```bash
gcloud dns managed-zones update sweet-security-zone \
  --project=invisible-sre-sandbox \
  --private-visibility-config-networks=sre-sandbox-private-onboarding,sre-771-staging
```

**Option B: Create separate DNS zone for this cluster**
```bash
gcloud dns managed-zones create sweet-security-zone-sre-771-staging \
  --dns-name=sweet.security. \
  --visibility=private \
  --networks=sre-771-staging \
  --project=invisible-sre-sandbox

# Then create all DNS records pointing to proxy IP
```

### 2. Proxy Network Mismatch
**Problem:** Proxy is on `sre-sandbox-private-onboarding` network, cluster is on `sre-771-staging` network.

**Impact:** Even if DNS resolves correctly, pods may not be able to reach proxy if networks aren't connected.

**Solution Options:**

**Option A: Verify network connectivity**
- Check if VPC peering exists between networks
- Check if using Shared VPC
- Verify firewall rules allow traffic

**Option B: Create proxy on cluster's network**
- Deploy proxy instance on `sre-771-staging` network
- Update DNS records to point to new proxy IP

**Option C: Use shared proxy (if networks connected)**
- If networks can communicate, existing proxy may work
- Verify connectivity: `kubectl run test --image=curlimages/curl --rm -i --restart=Never -n sweet -- curl -v https://registry.sweet.security`

### 3. Missing DNS Records
**Problem:** Only wildcard and registry records exist. Other endpoints may be needed.

**Action:** Add missing DNS records after fixing network issue:
```bash
PROXY_IP=10.0.30.53  # Update if using different proxy
ZONE=sweet-security-zone  # Or new zone name

for endpoint in control logger receiver vincent api prio; do
  gcloud dns record-sets create ${endpoint}.sweet.security. \
    --zone=$ZONE \
    --type=A \
    --rrdatas=$PROXY_IP \
    --ttl=300 \
    --project=invisible-sre-sandbox
done
```

## Verification Steps After Fixes

1. **Verify DNS zone includes cluster network:**
   ```bash
   gcloud dns managed-zones describe sweet-security-zone \
     --project=invisible-sre-sandbox \
     --format="get(privateVisibilityConfig.networks[].networkUrl)"
   ```

2. **Test DNS resolution from pod:**
   ```bash
   kubectl run test-dns --image=busybox --rm -i --restart=Never -n sweet -- \
     nslookup registry.sweet.security
   # Should resolve to proxy IP (10.0.30.53), not 18.220.208.31
   ```

3. **Test connectivity to proxy:**
   ```bash
   kubectl run test-connectivity --image=curlimages/curl --rm -i --restart=Never -n sweet -- \
     curl -v --connect-timeout 5 https://registry.sweet.security
   ```

4. **Restart deployments:**
   ```bash
   kubectl rollout restart deployment/sweet-operator -n sweet
   kubectl rollout restart deployment/sweet-scanner -n sweet
   kubectl rollout restart deployment/sweet-frontier-informer -n sweet
   ```

5. **Verify pods are running:**
   ```bash
   kubectl get pods -n sweet
   # All should be in Running state, not ImagePullBackOff
   ```

## Summary

| Prerequisite | Status | Notes |
|-------------|--------|-------|
| GKE Autopilot cluster | ✅ | sre-771-staging exists |
| VPC network/subnet | ✅ | Identified |
| DNS proxy | ⚠️ | Exists but on different network |
| DNS zone for cluster VPC | ✅ | **FIXED** - Zone now linked to cluster network |
| DNS records | ⚠️ | Partial - missing some endpoints |
| kubectl configured | ✅ | Working |
| Helm installed | ✅ | v3.19.0 |
| API credentials | ✅ | Secrets exist |

**Priority Fix:** Add `sre-771-staging` network to DNS zone or create separate zone for this cluster.
