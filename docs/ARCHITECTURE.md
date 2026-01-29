# APSS Application Architecture

This document describes the productionized layout of the Autopilot Security Sensor (APSS) Go codebase: modular packages, shared configuration, and separation of features.

## Layout Overview

```
sweet-security/
├── cmd/                    # Entry points (thin mains)
│   ├── agent/              # Sidecar agent main
│   ├── controller/         # Controller main
│   └── webhook/            # Mutating webhook main
├── internal/               # Application-specific, not importable by external modules
│   ├── config/             # Shared config (env, defaults) for all components
│   ├── version/            # Single source of version (set via ldflags at build)
│   ├── types/              # Shared API types (events, alerts, agents)
│   ├── detection/          # Detection rules engine (evaluate events → alerts)
│   ├── controller/         # Core controller (event buffer, alert pipeline, Sweet Security)
│   ├── server/             # HTTP server and API handlers for controller
│   └── webhook/             # Admission webhook logic (patch generation)
├── pkg/                    # Reusable packages (agent monitoring stack)
│   ├── api/v1/             # Protobuf API definitions
│   ├── collector/          # Event collection and HTTP send to controller
│   ├── fileintegrity/       # File integrity monitoring (inotify)
│   ├── monitor/            # Agent orchestrator (proc + net + file)
│   ├── netpolicy/          # Network connection monitoring
│   ├── procmon/            # Process monitoring
│   └── sweetsecurity/      # Sweet Security API client
└── build/                  # Dockerfiles
```

## Design Principles

1. **Thin mains** – `cmd/*` only parse config, wire components, and run. No business logic.
2. **Shared config** – `internal/config` provides env-based config and defaults for agent, controller, and webhook.
3. **Single version** – `internal/version.Version` is set at build time via `-ldflags -X ...version.Version=$(VERSION)`.
4. **Shared types** – `internal/types` defines HTTP/API types (SecurityEvent, Alert, AgentInfo) used by the controller and server so event/alert handling is consistent.
5. **Feature separation** – Detection rules live in `internal/detection`; HTTP in `internal/server`; admission logic in `internal/webhook`; core event/alert pipeline in `internal/controller`.
6. **Reusable agent stack** – `pkg/collector`, `pkg/procmon`, `pkg/netpolicy`, `pkg/fileintegrity`, `pkg/monitor` stay independent of `internal/` so they can be imported by other projects if needed.

## Data Flow

- **Agent** (`cmd/agent`): Uses `config.DefaultAgentConfig()` → `monitor.New(monCfg)` → proc/net/file monitors send events to `collector` → collector POSTs to controller `/api/v1/events`.
- **Controller** (`cmd/controller`): Uses `config.DefaultControllerConfig()` → `controller.New(cfg)` → `server.New(cfg, ctrl)` → HTTP handler receives events → `ctrl.IngestEvent()` → detection engine evaluates → alerts logged and sent to Sweet Security; server exposes `/health`, `/api/v1/events`, `/api/v1/agents`, `/api/v1/alerts`, `/metrics`.
- **Webhook** (`cmd/webhook`): Uses `config.DefaultWebhookConfig()` → on `/mutate`, `webhook.ProcessAdmissionReview(body, cfg)` decodes request, calls `ShouldSkipInjection` and `CreateSidecarPatches`, returns admission response.

## Build

Version is injected at build time:

```bash
make build                    # VERSION=0.1.0 (default in Makefile)
VERSION=1.2.3 make build      # Custom version
```

All three binaries (`bin/apss-agent`, `bin/apss-controller`, `bin/apss-webhook`) are built with the same ldflags (including version).
