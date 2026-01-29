# Sweet Security Deployment Verification Report
**Cluster:** sre-onboarding  
**Project:** invisible-sre-sandbox  
**Date:** 2026-01-10

## Expected Components

Based on the tutorial steps, the following three components should be deployed:

1. **sweet-sensor** (scanner) - DaemonSet or Deployment for scanning
2. **sweet-operator** - Deployment managing Sweet Security components
3. **sweet-frontier-informer** - Deployment collecting cluster metadata

## Current Status

### ✅ sweet-operator
- **Status:** Deployed (but ImagePullBackOff)
- **Type:** Deployment
- **Namespace:** sweet
- **Helm Release:** sweet-operator (revision 1)
- **Pods:** 1 pod (sweet-operator-cf6d97487-vbvjw) - ImagePullBackOff
- **Images:**
  - `registry.sweet.security/operator:af88a2f6bc4664a8fd68b4f8e25362`
  - `registry.sweet.security/bitbox:f5783c545dd3192b9fdfe0c77708d25cae4d632f`

### ✅ sweet-frontier-informer
- **Status:** Deployed (but ImagePullBackOff)
- **Type:** Deployment
- **Namespace:** sweet
- **Pods:** 1 pod (sweet-frontier-informer-7bfff9d89b-bs4zv) - ImagePullBackOff
- **Images:**
  - `registry.sweet.security/informer:f594ddf99e46d9f69344e96c3f737347934bff03`
  - `registry.sweet.security/bitbox:2fceb477fb90ff3febbed42fe39a9af0ca71caa8`

### ❌ sweet-sensor (scanner)
- **Status:** NOT FOUND
- **Expected:** DaemonSet or Deployment named `sweet-scanner`
- **Action Required:** Scanner component not deployed

## Issues Identified

### 1. Image Pull Failures (CRITICAL)
**Problem:** All pods are in `ImagePullBackOff` state
- Cannot pull images from `registry.sweet.security`
- Error: `dial tcp 18.220.208.31:443: i/o timeout`
- DNS resolves `registry.sweet.security` → `18.220.208.31` (direct, not through proxy)

**Root Cause:** 
- DNS wildcard `*.sweet.security` → `10.0.30.53` (proxy) exists
- But `registry.sweet.security` doesn't match wildcard pattern
- Need explicit A record for `registry.sweet.security` → proxy IP

**Fix Applied:**
```bash
gcloud dns record-sets create registry.sweet.security. \
  --zone=sweet-security-zone \
  --type=A \
  --rrdatas=10.0.30.53 \
  --ttl=300 \
  --project=invisible-sre-sandbox
```

### 2. Missing Scanner Component
**Problem:** `sweet-scanner` (sensor) not found
- No Helm release for scanner
- No Deployment or DaemonSet
- May need to install separately or configure operator to deploy it

**Action Required:**
- Check if scanner should be deployed via operator
- Or install scanner Helm chart separately:
  ```bash
  helm install sweet-scanner oci://registry.sweet.security/helm/scannerchart \
    --namespace sweet \
    --set sweet.apiKey=<API_KEY> \
    --set sweet.secret=<SECRET> \
    --set sweet.clusterId=<CLUSTER_ID>
  ```

## Verification Checklist

- [x] sweet-operator Deployment exists
- [x] sweet-frontier-informer Deployment exists  
- [ ] sweet-scanner/sensor exists (NOT FOUND - needs to be deployed)
- [ ] All pods in Running state (currently ImagePullBackOff)
- [ ] Images can be pulled successfully (blocked by DNS/proxy)
- [x] DNS zone configured correctly
- [x] DNS record for registry.sweet.security created
- [ ] DNS propagation complete (may take 5-10 minutes)
- [ ] Components visible in GCP Console

## Next Steps

1. **Fix DNS for registry.sweet.security** ✅ (Done)
2. **Wait for DNS propagation** (5-10 minutes) - DNS changes can take time to propagate
3. **Restart pods to retry image pulls:**
   ```bash
   kubectl rollout restart deployment/sweet-operator -n sweet
   kubectl rollout restart deployment/sweet-frontier-informer -n sweet
   ```
4. **Deploy sweet-scanner** (REQUIRED - missing component):
   ```bash
   helm install sweet-scanner oci://registry.sweet.security/helm/scannerchart \
     --namespace sweet \
     --set sweet.apiKey=a161bee0-85a8-41cc-a139-c620175f8908 \
     --set sweet.secret=25206a83-c8bc-4bf2-9be4-662711ad2bb6 \
     --set sweet.clusterId=cdb157c8-9d13-5ba8-b136-ba013f31de21
   ```
5. **Verify all components in GCP Console**
6. **Verify DNS resolution from pods:**
   ```bash
   kubectl run test-dns --image=busybox --rm -i --restart=Never -n sweet -- \
     nslookup registry.sweet.security
   # Should resolve to 10.0.30.53 (proxy IP), not 18.220.208.31
   ```

## Proxy Configuration Notes

The proxy is configured at:
- **IP:** 10.0.30.53
- **DNS Zone:** sweet-security-zone
- **Wildcard:** *.sweet.security → 10.0.30.53
- **Registry:** registry.sweet.security → 10.0.30.53 (now fixed)

For 400 clusters, you'll need:
- Per-project proxy instances OR
- Shared proxy with proper DNS configuration per cluster
- Ensure each cluster's DNS zone has records for:
  - `*.sweet.security` → proxy IP
  - `registry.sweet.security` → proxy IP
  - `control.sweet.security` → proxy IP
  - `logger.sweet.security` → proxy IP
  - `receiver.sweet.security` → proxy IP
  - `vincent.sweet.security` → proxy IP
  - `api.sweet.security` → proxy IP
