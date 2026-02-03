# Codebase Structure

**Analysis Date:** 2026-02-03

## Directory Layout

```
sweet-security/
├── cmd/                    # Entry points for each component
│   ├── agent/              # Agent sidecar main
│   ├── controller/         # Controller service main
│   └── webhook/            # Admission webhook main
├── internal/               # Private application code
│   ├── config/             # Configuration management
│   ├── controller/         # Event processing logic
│   ├── detection/          # Security rule engine
│   ├── server/             # HTTP server and API handlers
│   ├── types/              # Shared type definitions
│   ├── version/            # Version information
│   └── webhook/            # Admission control logic
├── pkg/                    # Public library code
│   ├── api/                # API definitions (protobuf)
│   ├── collector/          # Event collection and forwarding
│   ├── fileintegrity/      # File integrity monitoring
│   ├── monitor/            # Monitor orchestration
│   ├── netpolicy/          # Network policy monitoring
│   ├── procmon/            # Process monitoring
│   └── sweetsecurity/      # External API client
├── build/                  # Dockerfiles for each component
├── deploy/                 # Kubernetes deployment manifests
│   ├── helm/               # Helm chart
│   └── sweet-security/     # Direct YAML manifests
├── bin/                    # Compiled binaries
├── configs/                # Example configurations
├── scripts/                # Deployment and utility scripts
└── docs/                   # Documentation
```

## Directory Purposes

**cmd/**
- Purpose: Application entry points following Go conventions
- Contains: main.go files for each deployable component
- Key files: `agent/main.go`, `controller/main.go`, `webhook/main.go`

**internal/**
- Purpose: Private application code not meant for external use
- Contains: Business logic, internal APIs, configuration
- Key files: `controller/controller.go`, `types/events.go`, `webhook/admission.go`

**pkg/**
- Purpose: Library code that could be imported by external projects
- Contains: Reusable monitoring components and clients
- Key files: `monitor/monitor.go`, `sweetsecurity/client.go`

**build/**
- Purpose: Container build definitions
- Contains: Dockerfiles for each component
- Key files: `Dockerfile.agent`, `Dockerfile.controller`, `Dockerfile.webhook`

**deploy/**
- Purpose: Kubernetes deployment configurations
- Contains: Helm charts and raw manifests
- Key files: `helm/Chart.yaml`, `sweet-security/manifests/`

## Key File Locations

**Entry Points:**
- `cmd/agent/main.go`: Agent sidecar initialization
- `cmd/controller/main.go`: Controller service startup
- `cmd/webhook/main.go`: Admission webhook server

**Configuration:**
- `internal/config/config.go`: Configuration structs and defaults
- `configs/`: Example configuration files

**Core Logic:**
- `internal/controller/controller.go`: Event processing pipeline
- `internal/detection/rules.go`: Security detection engine
- `pkg/monitor/monitor.go`: Agent monitoring orchestration

**Testing:**
- `*_test.go`: Co-located with source files
- Test coverage requirement: 85% minimum

## Naming Conventions

**Files:**
- Go files: `lowercase_with_underscores.go`
- Test files: `*_test.go` pattern
- Main entries: `main.go` in cmd subdirectories

**Directories:**
- Package names: `lowercase` matching directory name
- Multi-word: `lowercasewithoutunderscore` (e.g., `sweetsecurity`)

**Components:**
- Binary names: `apss-{component}` (e.g., `apss-agent`)
- Docker images: `gcr.io/invisible-sre-sandbox/apss-{component}`

## Where to Add New Code

**New Monitoring Feature:**
- Primary code: `pkg/{feature}/` for reusable components
- Integration: `pkg/monitor/monitor.go` to orchestrate
- Tests: `pkg/{feature}/{feature}_test.go`

**New Detection Rule:**
- Implementation: `internal/detection/rules.go`
- Types: Add to `internal/types/alerts.go` if needed

**New API Endpoint:**
- Handler: `internal/server/server.go`
- Types: `internal/types/` for request/response structs

**New Configuration:**
- Config struct: `internal/config/config.go`
- Example: `configs/` directory

**Utilities:**
- Shared helpers: `pkg/` if reusable, `internal/` if application-specific

## Special Directories

**bin/**
- Purpose: Compiled binary outputs
- Generated: Yes (by Makefile)
- Committed: No (.gitignore excluded)

**.planning/**
- Purpose: GSD codebase analysis documents
- Generated: Yes (by analysis commands)
- Committed: Varies by project

**deploy/sweet-security/**
- Purpose: Legacy deployment manifests
- Generated: No
- Committed: Yes

**coverage.out**
- Purpose: Test coverage reports
- Generated: Yes (by test commands)
- Committed: No

---

*Structure analysis: 2026-02-03*