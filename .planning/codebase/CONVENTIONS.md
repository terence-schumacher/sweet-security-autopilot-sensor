# Coding Conventions

**Analysis Date:** 2026-02-03

## Naming Patterns

**Files:**
- `main.go` for entry points in `cmd/` directories
- `*_test.go` for test files (Go standard)
- CamelCase for Go files: `config.go`, `events.go`, `admission.go`
- Descriptive names: `fileintegrity.go`, `procmon.go`, `netpolicy.go`

**Functions:**
- Exported functions use PascalCase: `GetEnv()`, `ProcessAdmissionReview()`
- Private functions use camelCase: `processRequest()`, `defaultWatchPaths()`
- Constructor pattern: `New()` for package constructors
- Default pattern: `DefaultAgentConfig()`, `DefaultWebhookConfig()`

**Variables:**
- camelCase for local variables: `shutdownCtx`, `monCfg`
- PascalCase for exported struct fields: `AgentID`, `PodName`, `ControllerEndpoint`
- Abbreviations in caps when part of name: `HTTPAddr`, `TLSCertFile`

**Types:**
- PascalCase for all types: `SecurityEvent`, `ProcessEventData`, `AgentConfig`
- Suffix for grouped types: `*Config` structs, `*EventData` types
- Interface naming not observed (no interfaces in analyzed code)

## Code Style

**Formatting:**
- `go fmt` used (referenced in Makefile)
- `goimports -w .` for import organization (referenced in Makefile)

**Linting:**
- `golangci-lint run ./...` (referenced in Makefile)
- No configuration file found, likely using defaults

## Import Organization

**Order:**
1. Standard library imports (grouped together)
2. Third-party imports (grouped together)
3. Internal imports (grouped together)

**Example from `internal/webhook/handler.go`:**
```go
import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/invisible-tech/autopilot-security-sensor/internal/config"
)
```

**Path Aliases:**
- No path aliases used - full import paths throughout

## Error Handling

**Patterns:**
- Standard Go error handling with explicit checks
- Wrap errors with context: `fmt.Errorf("decode admission review: %w", err)`
- Log errors with structured logging before returning
- Return early on error conditions

**Example from `internal/webhook/handler.go`:**
```go
if err := json.Unmarshal(body, &review); err != nil {
    return nil, fmt.Errorf("decode admission review: %w", err)
}
```

## Logging

**Framework:** `github.com/sirupsen/logrus`

**Patterns:**
- JSON formatter for structured output: `log.SetFormatter(&logrus.JSONFormatter{})`
- Info level default: `log.SetLevel(logrus.InfoLevel)`
- Structured fields: `log.WithFields(logrus.Fields{"pod": pod.Name, "namespace": req.Namespace})`
- Context-aware logging with relevant metadata

**Examples:**
```go
log.WithError(err).Error("Failed to create monitor")
log.WithField("signal", sig.String()).Info("Received shutdown signal")
```

## Comments

**When to Comment:**
- Package-level documentation for all packages
- Exported functions and types documented
- Complex business logic explained

**JSDoc/TSDoc:**
- Go doc comments used: `// ProcessAdmissionReview decodes the admission review request...`

## Function Design

**Size:** Functions tend to be focused and single-purpose (50-100 lines typical)

**Parameters:**
- Configuration structs passed by reference
- Context passed as first parameter when used
- Logger passed explicitly rather than global

**Return Values:**
- Error as last return value (Go idiom)
- Pointer returns for optional data structures

## Module Design

**Exports:**
- Minimal public API - only necessary types and functions exported
- Configuration types fully exported with public fields

**Barrel Files:**
- Not applicable in Go - no barrel export pattern

## Structure Patterns

**Configuration:**
- Centralized config package (`internal/config`)
- Default constructors with environment override pattern
- Struct-based configuration with sensible defaults

**Error Types:**
- Standard error interface usage
- No custom error types observed

**Testing:**
- Table-driven tests not heavily used
- Focused unit tests per function
- Test helper functions when needed

---

*Convention analysis: 2026-02-03*