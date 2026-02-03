# Codebase Concerns

**Analysis Date:** 2026-02-03

## Tech Debt

**Event Channel Backpressure Issues:**
- Issue: Multiple components drop events when channels are full without retry mechanisms
- Files: `pkg/fileintegrity/fileintegrity.go:249`, `pkg/netpolicy/netpolicy.go:339`, `pkg/procmon/procmon.go:300`, `internal/controller/controller.go:249`
- Impact: Critical security events may be silently lost during high load
- Fix approach: Implement buffering, retry mechanisms, or circuit breakers for event delivery

**Hard-coded Configuration Values:**
- Issue: Buffer sizes, timeouts, and thresholds are embedded in code rather than configurable
- Files: `internal/config/config.go:118-119` (EventBufferSize: 100000, AlertBufferSize: 10000)
- Impact: Cannot tune performance without code changes, difficult to optimize for different environments
- Fix approach: Move all fixed values to configuration with environment variable overrides

**Limited Error Recovery:**
- Issue: Failed network scans and file operations only log debug messages, no retry or degradation strategy
- Files: `pkg/netpolicy/netpolicy.go:110-122`, `pkg/fileintegrity/fileintegrity.go:72-96`
- Impact: Monitoring gaps when /proc filesystem is temporarily unavailable or paths cannot be watched
- Fix approach: Implement retry logic with exponential backoff for critical operations

## Known Bugs

**Non-blocking Event Buffer Returns Error:**
- Symptoms: Controller.IngestEvent returns "event buffer full" error but continues operation
- Files: `internal/controller/controller.go:135`
- Trigger: When event buffer reaches capacity (100,000 events)
- Workaround: Error is logged but operation continues

**File Watcher Memory Leak Potential:**
- Symptoms: File watchers added but not removed when directories are deleted
- Files: `pkg/fileintegrity/fileintegrity.go:252-257`
- Trigger: Creating and deleting directories rapidly
- Workaround: None implemented

## Security Considerations

**Environment Variable Exposure:**
- Risk: Sensitive credentials stored in plain text environment variables
- Files: `internal/config/config.go:114-124` (SWEET_SECURITY_API_KEY)
- Current mitigation: None
- Recommendations: Use Kubernetes secrets, secret management solutions, or vault integration

**Privileged Container Requirements:**
- Risk: Sidecar containers require access to host /proc filesystem
- Files: `pkg/procmon/procmon.go:100`, `pkg/netpolicy/netpolicy.go:110-122`
- Current mitigation: None documented
- Recommendations: Implement least-privilege principle, document required capabilities

**Hardcoded Suspicious Patterns:**
- Risk: Attack patterns easily bypassed by knowing detection rules
- Files: `internal/config/config.go:96-109` (defaultSuspiciousProcesses, defaultSuspiciousPorts)
- Current mitigation: None
- Recommendations: Move to configurable rules, implement ML-based detection

## Performance Bottlenecks

**Recursive File Hashing on Startup:**
- Problem: All watched paths are hashed synchronously during initialization
- Files: `pkg/fileintegrity/fileintegrity.go:60-64`, `pkg/fileintegrity/fileintegrity.go:87-88`
- Cause: No async processing for baseline creation
- Improvement path: Implement background hashing with progress tracking

**Process Scanning Frequency:**
- Problem: Full /proc scan every 5 seconds by default
- Files: `pkg/procmon/procmon.go:98-106`, `internal/config/config.go:80`
- Cause: Fixed interval regardless of system load
- Improvement path: Adaptive scanning based on system activity

**Memory Growth from Event Retention:**
- Problem: Alerts retained in memory with fixed retention count (10,000)
- Files: `internal/controller/controller.go:285-294`, `internal/config/config.go:121`
- Cause: No memory pressure awareness
- Improvement path: Implement memory-based limits with LRU eviction

## Fragile Areas

**Network Connection Parsing:**
- Files: `pkg/netpolicy/netpolicy.go:156-189`
- Why fragile: Direct /proc/net/* file parsing with hardcoded format assumptions
- Safe modification: Add format validation and error recovery
- Test coverage: Limited edge cases

**File System Event Handling:**
- Files: `pkg/fileintegrity/fileintegrity.go:163-258`
- Why fragile: Race conditions between file operations and watcher events
- Safe modification: Add synchronization and event deduplication
- Test coverage: Missing concurrent access tests

**TLS Certificate Management:**
- Files: `cmd/webhook/main.go:49`
- Why fragile: Fatal error on TLS cert load failure, no graceful degradation
- Safe modification: Add cert validation and rotation support
- Test coverage: No TLS error scenarios tested

## Scaling Limits

**Event Buffer Capacity:**
- Current capacity: 100,000 events in memory
- Limit: High-throughput environments will hit buffer limits
- Scaling path: Implement persistent queue or external message broker

**Agent Tracking:**
- Current capacity: All agents tracked in memory map
- Limit: Large clusters (1000+ nodes) may cause memory issues
- Scaling path: Add agent TTL and cleanup mechanisms

## Dependencies at Risk

**fsnotify Library:**
- Risk: Platform-specific file watching with kernel limitations
- Impact: File monitoring may fail on certain filesystems or kernel versions
- Migration plan: Add fallback polling mechanism

**Kubernetes API Dependencies:**
- Risk: Hard dependency on specific K8s API versions (v0.29.2)
- Impact: Breaks compatibility with newer/older clusters
- Migration plan: Use client-go compatibility matrix

## Missing Critical Features

**Audit Logging:**
- Problem: No audit trail for security events and admin actions
- Blocks: Compliance requirements and incident investigation
- Priority: High

**Event Deduplication:**
- Problem: Duplicate events generated for file system changes
- Blocks: Alert fatigue and accurate threat assessment
- Priority: Medium

**Health Check Monitoring:**
- Problem: No monitoring of component health beyond basic HTTP endpoints
- Blocks: Operational visibility into agent status
- Priority: Medium

## Test Coverage Gaps

**Concurrent Event Processing:**
- What's not tested: Multiple agents sending events simultaneously
- Files: `internal/controller/controller.go`, `pkg/collector/collector.go`
- Risk: Race conditions in agent tracking and event processing
- Priority: High

**Error Recovery Scenarios:**
- What's not tested: Behavior when /proc filesystem is unavailable
- Files: `pkg/procmon/procmon.go`, `pkg/netpolicy/netpolicy.go`
- Risk: Agent failure in constrained environments
- Priority: Medium

**TLS and Certificate Handling:**
- What's not tested: Invalid certificates, certificate rotation
- Files: `cmd/webhook/main.go`
- Risk: Webhook service failures in production
- Priority: Medium

---

*Concerns audit: 2026-02-03*