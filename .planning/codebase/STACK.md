# Technology Stack

**Analysis Date:** 2026-02-03

## Languages

**Primary:**
- Go 1.22 - All core components, services, and packages

**Secondary:**
- Shell/Bash - Build scripts and deployment automation
- YAML - Configuration and Kubernetes manifests

## Runtime

**Environment:**
- Go 1.22

**Package Manager:**
- Go modules (go mod)
- Lockfile: `go.sum` present

## Frameworks

**Core:**
- Standard library HTTP server - Web services and API endpoints
- Kubernetes client-go - Kubernetes API interactions

**Testing:**
- Go standard testing - Unit tests with coverage tracking
- Requirements: >= 85% coverage enforced

**Build/Dev:**
- Make - Build orchestration via `Makefile`
- Docker multi-stage builds - Container image creation
- golangci-lint - Code linting
- goimports - Code formatting

## Key Dependencies

**Critical:**
- github.com/sirupsen/logrus v1.9.3 - Structured logging throughout application
- k8s.io/api v0.29.2 - Kubernetes API types and resources
- k8s.io/apimachinery v0.29.2 - Kubernetes metadata and runtime objects

**Infrastructure:**
- github.com/fsnotify/fsnotify v1.7.0 - File system event monitoring
- github.com/prometheus/client_golang v1.19.0 - Metrics collection and exposure

## Configuration

**Environment:**
- Environment variables for runtime configuration
- Key configs: `SWEET_SECURITY_API_KEY`, `CONTROLLER_ENDPOINT`, `TLS_CERT_FILE`
- Defaults provided via `internal/config/config.go`

**Build:**
- `Makefile` - Build targets and Docker image creation
- `go.mod` - Dependency management
- Multi-stage Dockerfiles in `build/` directory

## Platform Requirements

**Development:**
- Go 1.22+
- Make
- Docker (for containerization)
- kubectl (for deployment)
- golangci-lint (for linting)

**Production:**
- Kubernetes cluster (GKE Autopilot compatible)
- Container registry access (gcr.io/invisible-sre-sandbox)
- Helm 3 (for deployment)

---

*Stack analysis: 2026-02-03*