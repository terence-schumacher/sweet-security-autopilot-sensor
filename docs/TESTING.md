# Testing

## Running tests

From the repo root:

```bash
go test ./...
```

With coverage and race detector (as in `make test`):

```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # optional: view coverage
```

## Test layout

| Package | What is tested |
|---------|----------------|
| **internal/config** | `GetEnv`, `GetEnvDuration`, default configs (agent, controller, webhook) |
| **internal/version** | Version is non-empty |
| **internal/types** | JSON round-trip for `SecurityEvent` and `Alert` |
| **internal/detection** | Rules engine: `NewEngine`, `Evaluate` for APSS-001–005 (reverse shell, cryptominer, file modify, shell spawn, external DB), no-match and alert fields |
| **internal/controller** | `New`, `IngestEvent`, `GetAgents`, `GetAlerts`, buffer-full behavior |
| **internal/server** | HTTP handlers: `/health`, `POST /api/v1/events`, `GET /api/v1/agents`, `GET /api/v1/alerts`, method/JSON error cases |
| **internal/webhook** | `ShouldSkipInjection` (excluded ns, already injected, annotation, hostNetwork), `CreateSidecarPatches`, `ProcessAdmissionReview` (non-Pod, Pod inject, no request, invalid JSON) |
| **pkg/collector** | `New`, default buffer size, `EventChannel`, `GetStats`, `SendEvent` (with mock HTTP server; skips if bind not allowed) |
| **pkg/sweetsecurity** | `NewClient`, default timeout, `SendAlert`/`SendEvent`/`HealthCheck` success and error cases (mock server; skips if bind not allowed), not-configured and non-OK response |

Tests that start an HTTP server (`pkg/collector` and `pkg/sweetsecurity`) skip when binding a port is not allowed (e.g. in a restricted environment).

## Verifying locally

Run the full suite and confirm all packages pass:

```bash
make test
```

Check that coverage is at least 85%:

```bash
make test-coverage
```

Or without race/coverage:

```bash
go test ./...
go test ./internal/... ./pkg/... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```
