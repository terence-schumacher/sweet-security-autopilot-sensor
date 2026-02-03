# External Integrations

**Analysis Date:** 2026-02-03

## APIs & External Services

**Sweet Security Platform:**
- Sweet Security API - Security events and alerts aggregation
  - SDK/Client: `pkg/sweetsecurity/client.go`
  - Auth: `SWEET_SECURITY_API_KEY` environment variable
  - Endpoints: `/api/v1/events`, `/api/v1/alerts`, `/api/v1/events/batch`, `/health`

**Kubernetes API:**
- Kubernetes API Server - Pod injection and cluster monitoring
  - SDK/Client: k8s.io/api, k8s.io/apimachinery
  - Auth: Service account tokens and RBAC

## Data Storage

**Databases:**
- None - System operates as stateless event forwarder

**File Storage:**
- Local filesystem monitoring only
- Watched paths: `/etc/passwd`, `/etc/shadow`, `/etc/sudoers`, `/root/.ssh`

**Caching:**
- In-memory event buffering
  - Event buffer: 100,000 events
  - Alert buffer: 10,000 alerts

## Authentication & Identity

**Auth Provider:**
- Kubernetes RBAC - Service account based authentication
  - Implementation: Service accounts with ClusterRole bindings
  - Files: `deploy/helm/templates/rbac.yaml`

**Sweet Security API:**
- Bearer token authentication
  - Header: `Authorization: Bearer {api_key}`

## Monitoring & Observability

**Error Tracking:**
- Structured logging via logrus
  - Implementation: `github.com/sirupsen/logrus`

**Logs:**
- JSON structured logging to stdout/stderr
- Log levels configurable per component

**Metrics:**
- Prometheus metrics exposure
  - Client: `github.com/prometheus/client_golang`
  - Endpoint: `/metrics` on port 8080
  - ServiceMonitor: `deploy/helm/templates/`

## CI/CD & Deployment

**Hosting:**
- Google Container Registry (gcr.io/invisible-sre-sandbox)
- Google Kubernetes Engine (GKE Autopilot)

**CI Pipeline:**
- Manual builds via Makefile
- Docker multi-stage builds for optimized images

**Container Images:**
- Base: `gcr.io/distroless/static-debian12:nonroot`
- Images: apss-agent, apss-controller, apss-webhook

## Environment Configuration

**Required env vars:**
- `SWEET_SECURITY_API_KEY` - API authentication token
- `SWEET_SECURITY_ENDPOINT` - API base URL (default: https://api.sweet.security)
- `CONTROLLER_ENDPOINT` - Internal service communication
- `TLS_CERT_FILE`, `TLS_KEY_FILE` - Webhook TLS certificates

**Secrets location:**
- Kubernetes secrets via Helm values
- Secret reference: `sweetSecurity.apiKeySecret`

## Webhooks & Callbacks

**Incoming:**
- Kubernetes mutating admission webhook - Pod sidecar injection
  - Endpoint: `/mutate` on port 8443
  - TLS: Required with valid certificates

**Outgoing:**
- Sweet Security API webhooks - Event and alert forwarding
  - HTTP POST to configured API endpoints
  - Retry logic with exponential backoff

**Kubernetes Events:**
- File system events via fsnotify
- Process monitoring via /proc filesystem
- Network connection monitoring

## Alert Integrations

**Slack:**
- Webhook integration available (disabled by default)
  - Config: `controller.alerting.slack.webhookUrl`

**PagerDuty:**
- Events API integration available (disabled by default)
  - Config: `controller.alerting.pagerduty.routingKey`

**Pub/Sub:**
- Google Cloud Pub/Sub for event streaming (disabled by default)
  - Config: `controller.alerting.pubsub.topicId`

---

*Integration audit: 2026-02-03*