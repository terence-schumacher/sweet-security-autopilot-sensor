# Testing Patterns

**Analysis Date:** 2026-02-03

## Test Framework

**Runner:**
- Go's built-in testing package
- Config: Standard Go test discovery

**Assertion Library:**
- Built-in Go testing with manual assertions
- Custom error checking patterns

**Run Commands:**
```bash
go test -v -race -coverprofile=coverage.out ./...     # Run all tests with race detection
make test                                             # Makefile wrapper for above
make test-coverage                                    # Run with coverage check (>=85%)
go tool cover -func=coverage.out                     # View coverage report
```

## Test File Organization

**Location:**
- Co-located with source files in same package

**Naming:**
- `*_test.go` suffix (Go standard)
- Same package name as source code

**Structure:**
```
internal/config/
├── config.go
└── config_test.go

pkg/monitor/
├── monitor.go
└── monitor_test.go
```

## Test Structure

**Suite Organization:**
```go
func TestFunctionName(t *testing.T) {
	// Basic test structure
}

func TestFunctionName_Scenario(t *testing.T) {
	// Scenario-specific tests
}
```

**Patterns:**
- Subtests using `t.Run()` for different scenarios:
```go
func TestGetEnv(t *testing.T) {
	t.Run("returns default when unset", func(t *testing.T) {
		// test logic
	})
	t.Run("returns value when set", func(t *testing.T) {
		// test logic
	})
}
```

- Direct assertions with `t.Errorf()` and `t.Fatalf()`
- Setup/teardown using `defer` for cleanup

## Mocking

**Framework:** No dedicated mocking framework - using test doubles

**Patterns:**
```go
// Interface-based testing (implied from config tests)
cfg := &AgentConfig{
	ControllerEndpoint: "localhost:8080",
	WatchPaths:         []string{}, // empty to avoid real file system
}
```

**What to Mock:**
- External dependencies (file paths, network endpoints)
- Environment variables using `os.Setenv`/`os.Unsetenv`

**What NOT to Mock:**
- Pure functions and data structures
- In-memory operations

## Fixtures and Factories

**Test Data:**
```go
// Inline test data construction
pod := corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "app"},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{{Name: "app", Image: "app:latest"}},
	},
}

// Structured test event data
ev := SecurityEvent{
	ID:           "ev-1",
	AgentID:      "agent-1",
	Type:         "process_start",
	Severity:     "HIGH",
	Process: &ProcessEventData{
		PID:     1234,
		PPID:    1,
		Name:    "bash",
		Cmdline: []string{"bash", "-i"},
	},
}
```

**Location:**
- Inline in test functions rather than separate fixture files

## Coverage

**Requirements:** >= 85% (enforced in Makefile)

**View Coverage:**
```bash
make test-coverage                    # Check if meets 85% threshold
go tool cover -html=coverage.out     # HTML coverage report
```

## Test Types

**Unit Tests:**
- Testing individual functions in isolation
- Configuration parsing and validation
- JSON marshaling/unmarshaling
- Error handling paths

**Integration Tests:**
- End-to-end admission review processing
- Component initialization with real configurations
- Shutdown/lifecycle testing

**E2E Tests:**
- Not observed in current test files

## Common Patterns

**Async Testing:**
```go
func TestMonitor_Shutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = m.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}
```

**Error Testing:**
```go
func TestProcessAdmissionReview_InvalidJSON(t *testing.T) {
	_, err := ProcessAdmissionReview([]byte("not json"), cfg, log)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
```

**Environment Variable Testing:**
```go
func TestGetEnv(t *testing.T) {
	t.Run("trims space", func(t *testing.T) {
		os.Setenv("APSS_TEST_GETENV_TRIM", "  trimmed  ")
		defer os.Unsetenv("APSS_TEST_GETENV_TRIM")
		got := GetEnv("APSS_TEST_GETENV_TRIM", "default")
		if got != "trimmed" {
			t.Errorf("GetEnv(trim) = %q, want %q", got, "trimmed")
		}
	})
}
```

**JSON Round-trip Testing:**
```go
func TestSecurityEvent_JSONRoundTrip(t *testing.T) {
	ev := SecurityEvent{/* ... */}
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got SecurityEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	// Assertions on round-trip equality
}
```

**Test Structure Standards:**
- Use `t.Fatalf()` for setup errors that prevent test continuation
- Use `t.Errorf()` for assertion failures that allow test continuation
- Descriptive error messages with actual vs expected values
- Clean up resources using `defer` statements

---

*Testing analysis: 2026-02-03*