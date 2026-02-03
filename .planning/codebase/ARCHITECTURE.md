# Architecture

**Analysis Date:** 2026-02-03

## Pattern Overview

**Overall:** Multi-Component Security Monitoring System

**Key Characteristics:**
- Agent-Controller-Webhook pattern for distributed security monitoring
- Event-driven architecture with buffered processing pipelines
- Kubernetes-native design with sidecar injection
- External integration with Sweet Security platform

## Layers

**Entry Layer:**
- Purpose: Accept external requests and inject monitoring capabilities
- Location: `cmd/`
- Contains: Main entry points for each component
- Depends on: Internal configuration and business logic
- Used by: Kubernetes cluster operators and security teams

**Business Logic Layer:**
- Purpose: Core security monitoring, detection, and alert processing
- Location: `internal/`
- Contains: Controller logic, detection engine, webhook admission control
- Depends on: Package layer for specialized monitoring
- Used by: Entry layer main functions

**Package Layer:**
- Purpose: Specialized monitoring components and external integrations
- Location: `pkg/`
- Contains: Process monitoring, network policy, file integrity, Sweet Security client
- Depends on: Standard library and external dependencies
- Used by: Business logic layer

## Data Flow

**Agent Flow:**

1. Agent (`cmd/agent/main.go`) starts and initializes monitor
2. Monitor orchestrates procmon, netpolicy, and fileintegrity scanners
3. Events collected via collector and sent to controller endpoint
4. Agent runs as sidecar in target pods

**Controller Flow:**

1. Controller (`cmd/controller/main.go`) starts HTTP server and event processor
2. Events received via `/api/v1/events` endpoint and queued
3. Detection engine evaluates events against security rules
4. Alerts generated and sent to Sweet Security platform

**Webhook Flow:**

1. Webhook (`cmd/webhook/main.go`) receives admission reviews
2. Determines if pod should receive security sidecar injection
3. Returns JSON patches to inject APSS agent container

**State Management:**
- In-memory buffering with configurable sizes
- Agent tracking with health checks and timeout cleanup
- Alert retention with configurable limits

## Key Abstractions

**SecurityEvent:**
- Purpose: Unified event structure for all security monitoring
- Examples: `internal/types/events.go`
- Pattern: Rich type with specialized payload structs

**Monitor:**
- Purpose: Orchestrates multiple security scanning components
- Examples: `pkg/monitor/monitor.go`
- Pattern: Composite pattern with lifecycle management

**Controller:**
- Purpose: Central event processing and detection coordination
- Examples: `internal/controller/controller.go`
- Pattern: Event-driven processor with multiple channels

**Detection Engine:**
- Purpose: Rule-based security event evaluation
- Examples: `internal/detection/rules.go`
- Pattern: Rule engine with configurable detection logic

## Entry Points

**Agent Entry Point:**
- Location: `cmd/agent/main.go`
- Triggers: Kubernetes pod startup (sidecar injection)
- Responsibilities: Initialize monitoring, graceful shutdown handling

**Controller Entry Point:**
- Location: `cmd/controller/main.go`
- Triggers: Kubernetes deployment startup
- Responsibilities: Start HTTP API, event processing, agent management

**Webhook Entry Point:**
- Location: `cmd/webhook/main.go`
- Triggers: Kubernetes admission controller calls
- Responsibilities: TLS server for mutating admission webhook

## Error Handling

**Strategy:** Structured logging with graceful degradation

**Patterns:**
- Context-based cancellation for clean shutdown
- Channel-based error propagation
- Sweet Security integration failures logged but not blocking

## Cross-Cutting Concerns

**Logging:** Structured JSON logging via logrus with contextual fields
**Validation:** JSON schema validation for API payloads and webhook requests
**Authentication:** Bearer token authentication for Sweet Security API

---

*Architecture analysis: 2026-02-03*