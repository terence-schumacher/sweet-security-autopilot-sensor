# Deployment Guide for GKE Autopilot

This guide walks through deploying the Autopilot Security Sensor (APSS) to your `sre-onboarding` GKE Autopilot cluster.

## Prerequisites

1. **GKE Autopilot cluster** running (verified: `sre-onboarding` in `invisible-sre-sandbox`)
2. **cert-manager** installed (for webhook TLS certificates)
3. **kubectl** configured to access the cluster
4. **Helm 3.x** installed

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/invisible-tech/autopilot-security-sensor.git
cd autopilot-security-sensor

# 2. Connect to the cluster
gcloud container clusters get-credentials sre-onboarding \
  --region us-west1 \
  --project invisible-sre-sandbox

# 3. Install cert-manager (if not already installed)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml

# 4. Deploy APSS
helm upgrade --install apss ./deploy/helm \
  --namespace apss-system \
  --create-namespace

# 5. Verify deployment
kubectl get pods -n apss-system
```

## What Gets Deployed

| Component | Type | Purpose |
|-----------|------|---------|
| `apss-controller` | Deployment (2 replicas) | Aggregates events, runs detection rules, generates alerts |
| `apss-webhook` | Deployment (2 replicas) | MutatingWebhook that injects sidecar into pods |
| Sidecar Agent | Injected into pods | Monitors processes, network, files within each pod |

## Configuration

### Enable Sweet Security Integration

```bash
helm upgrade apss ./deploy/helm \
  --namespace apss-system \
  --set sweetSecurity.enabled=true \
  --set sweetSecurity.apiEndpoint="https://api.sweet.security" \
  --set sweetSecurity.apiKeySecret.name=sweet-api-key \
  --set sweetSecurity.apiKeySecret.key=api-key
```

First create the secret:
```bash
kubectl create secret generic sweet-api-key \
  --from-literal=api-key=YOUR_API_KEY \
  -n apss-system
```

### Exclude Namespaces from Injection

By default, system namespaces are excluded. To exclude additional namespaces:

```bash
helm upgrade apss ./deploy/helm \
  --namespace apss-system \
  --set webhook.excludeNamespaces='{kube-system,kube-public,apss-system,your-namespace}'
```

Or label a namespace:
```bash
kubectl label namespace your-namespace apss.invisible.tech/inject=false
```

### Exclude Specific Pods

Add annotation to pod/deployment:
```yaml
metadata:
  annotations:
    apss.invisible.tech/inject: "false"
```

## Verifying It Works

### Check Controller is Running
```bash
kubectl logs -f deployment/apss-controller -n apss-system
```

### Check Webhook is Injecting Sidecars
```bash
# Create a test pod
kubectl run test-pod --image=nginx --restart=Never

# Check if sidecar was injected
kubectl get pod test-pod -o jsonpath='{.spec.containers[*].name}'
# Should show: nginx apss-agent
```

### View Security Events
```bash
# Check agent logs
kubectl logs test-pod -c apss-agent

# Query controller API
kubectl port-forward svc/apss-controller 8080:8080 -n apss-system &
curl http://localhost:8080/api/v1/alerts
```

### View Metrics
```bash
kubectl port-forward svc/apss-controller 8080:8080 -n apss-system &
curl http://localhost:8080/metrics
```

## Detection Rules

APSS includes these built-in detection rules:

| Rule ID | Name | Severity | MITRE ID |
|---------|------|----------|----------|
| APSS-001 | Reverse Shell Detection | CRITICAL | T1059.004 |
| APSS-002 | Cryptominer Detection | CRITICAL | T1496 |
| APSS-003 | Sensitive File Modification | HIGH | T1546 |
| APSS-004 | Shell Spawn Detection | MEDIUM | T1059 |
| APSS-005 | External Database Connection | MEDIUM | T1048 |

## Autopilot Limitations

Due to GKE Autopilot restrictions, APSS cannot:
- Load eBPF programs at the kernel level
- Use privileged containers
- Access host namespaces directly

APSS works around these by:
- Using `shareProcessNamespace` to see sibling container processes
- Monitoring `/proc` from within the pod's namespace
- Using inotify for file monitoring (limited to container filesystem)

## Troubleshooting

### Sidecar Not Injected
1. Check webhook logs: `kubectl logs -l app.kubernetes.io/component=webhook -n apss-system`
2. Verify namespace isn't excluded
3. Check MutatingWebhookConfiguration: `kubectl get mutatingwebhookconfigurations`

### No Events in Controller
1. Check agent logs: `kubectl logs <pod> -c apss-agent`
2. Verify controller service is reachable
3. Check network policies

### High Resource Usage
Reduce scan intervals in values.yaml:
```yaml
agent:
  monitoring:
    procScanIntervalSeconds: 10  # Default: 5
    netScanIntervalSeconds: 30   # Default: 10
```

## Uninstalling

```bash
helm uninstall apss -n apss-system
kubectl delete namespace apss-system
```
