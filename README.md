# Autopilot Security Sensor (APSS)

A GKE Autopilot-compatible runtime security monitoring solution that works within Autopilot's security constraints.

## The Challenge

GKE Autopilot prohibits:
- Privileged containers
- `hostPID`, `hostNetwork`, `hostIPC`
- Most `hostPath` mounts
- Arbitrary DaemonSets with elevated privileges
- `CAP_SYS_ADMIN`, `CAP_BPF`, `CAP_SYS_PTRACE`

Traditional runtime security sensors (Falco, Sweet Security, Sysdig, etc.) require these capabilities to load eBPF programs at the kernel level.

## Our Approach

Instead of fighting Autopilot's restrictions, we work within them using a **multi-layer monitoring architecture**:

### Layer 1: Sidecar Agent (Pod-Level Monitoring)
A lightweight sidecar injected via MutatingWebhook that monitors:
- **Process execution** within the container's PID namespace (via `/proc`)
- **Network connections** via `/proc/net/*` and socket monitoring
- **File integrity** via inotify on critical paths
- **Resource anomalies** (CPU/memory spikes indicating cryptominers, etc.)

### Layer 2: Kubernetes Audit Log Analysis
Real-time processing of K8s audit logs for:
- Suspicious API calls (secrets access, exec into pods, privilege escalation attempts)
- RBAC violations and anomalies
- Resource creation patterns (cryptomining deployments, reverse shells)

### Layer 3: Network Policy Monitoring
- Tracks NetworkPolicy changes
- Monitors egress patterns via DNS query analysis
- Detects C2 communication patterns

### Layer 4: Workload Intelligence
- Container image vulnerability correlation
- Behavioral baseline and anomaly detection
- Integration with GKE Security Posture

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         GKE Autopilot Cluster                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │   App Pod    │  │   App Pod    │  │   App Pod    │               │
│  │ ┌──────────┐ │  │ ┌──────────┐ │  │ ┌──────────┐ │               │
│  │ │   App    │ │  │ │   App    │ │  │ │   App    │ │               │
│  │ └──────────┘ │  │ └──────────┘ │  │ └──────────┘ │               │
│  │ ┌──────────┐ │  │ ┌──────────┐ │  │ ┌──────────┐ │               │
│  │ │ Sidecar  │ │  │ │ Sidecar  │ │  │ │ Sidecar  │ │               │
│  │ │  Agent   │ │  │ │  Agent   │ │  │ │  Agent   │ │               │
│  │ └────┬─────┘ │  │ └────┬─────┘ │  │ └────┬─────┘ │               │
│  └──────┼───────┘  └──────┼───────┘  └──────┼───────┘               │
│         │                 │                 │                        │
│         └────────────────┬┴─────────────────┘                        │
│                          │ gRPC streams                              │
│                          ▼                                           │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    APSS Controller                             │  │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐              │  │
│  │  │   Event     │ │   Audit     │ │  Anomaly    │              │  │
│  │  │ Aggregator  │ │   Log       │ │  Detection  │              │  │
│  │  │             │ │  Processor  │ │   Engine    │              │  │
│  │  └─────────────┘ └─────────────┘ └─────────────┘              │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                          │                                           │
│  ┌───────────────────────┴───────────────────────────────────────┐  │
│  │                 Mutating Webhook                               │  │
│  │            (Injects sidecar into pods)                         │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
                          │
                          ▼ Alerts/Events
              ┌───────────────────────┐
              │   External Systems    │
              │  - Sweet Security API │
              │  - Slack/PagerDuty    │
              │  - SIEM Integration   │
              │  - Pub/Sub            │
              └───────────────────────┘
```

## Detection Capabilities

### What We CAN Detect (Autopilot-Compatible)

| Category | Detection | Method |
|----------|-----------|--------|
| **Cryptomining** | High CPU/unusual processes | Proc monitoring + resource metrics |
| **Reverse Shells** | Suspicious network connections | Socket monitoring + DNS analysis |
| **Container Escape Attempts** | Syscall patterns, /proc writes | File integrity + proc monitoring |
| **Privilege Escalation** | K8s API abuse, RBAC anomalies | Audit log analysis |
| **Data Exfiltration** | Unusual egress, DNS tunneling | Network monitoring |
| **Malicious Images** | Known bad signatures | Image analysis integration |
| **Lateral Movement** | Service account abuse | Audit logs + network patterns |
| **Web Shells** | New listening processes | Socket + process monitoring |

### What We CANNOT Detect (Requires Kernel Access)

- Raw syscall interception (would need eBPF)
- Host-level rootkit detection
- Kernel module loading
- Full container breakout detection

## Installation

```bash
# Add Helm repo
helm repo add apss https://your-org.github.io/autopilot-security-sensor

# Install with default settings
helm install apss apss/autopilot-security-sensor \
  --namespace apss-system \
  --create-namespace

# Install with Sweet Security integration
helm install apss apss/autopilot-security-sensor \
  --namespace apss-system \
  --create-namespace \
  --set sweetSecurity.enabled=true \
  --set sweetSecurity.apiKey=$SWEET_API_KEY
```

## Configuration

See [docs/configuration.md](docs/configuration.md) for full options.

## Requirements

- GKE Autopilot cluster (1.27+)
- Workload Identity enabled
- Kubernetes audit logging enabled

## License

Apache 2.0
