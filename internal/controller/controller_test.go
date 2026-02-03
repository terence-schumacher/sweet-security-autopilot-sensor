package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/internal/config"
	"github.com/invisible-tech/autopilot-security-sensor/internal/types"
)

func TestNew(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{
		EventBufferSize: 10,
		AlertBufferSize: 10,
	}
	c := New(cfg, log)
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestController_IngestEvent_GetAgents(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{
		EventBufferSize: 100,
		AlertBufferSize: 100,
	}
	c := New(cfg, log)
	ctx := context.Background()

	agents := c.GetAgents()
	if len(agents) != 0 {
		t.Errorf("initial GetAgents: want 0, got %d", len(agents))
	}

	ev := &types.SecurityEvent{
		ID: "ev-1", AgentID: "agent-1", Type: "process_start", Severity: "INFO",
		Timestamp: time.Now(), PodName: "pod-1", PodNamespace: "default",
	}
	err := c.IngestEvent(ctx, ev)
	if err != nil {
		t.Fatalf("IngestEvent: %v", err)
	}

	agents = c.GetAgents()
	if len(agents) != 1 {
		t.Fatalf("after IngestEvent: want 1 agent, got %d", len(agents))
	}
	if agents[0].ID != "agent-1" || agents[0].PodName != "pod-1" || agents[0].EventCount != 1 {
		t.Errorf("agent: ID=%q PodName=%q EventCount=%d", agents[0].ID, agents[0].PodName, agents[0].EventCount)
	}

	// Same agent again: count should increment
	err = c.IngestEvent(ctx, ev)
	if err != nil {
		t.Fatalf("IngestEvent second: %v", err)
	}
	agents = c.GetAgents()
	if len(agents) != 1 || agents[0].EventCount != 2 {
		t.Errorf("after second event: want EventCount=2, got %d", agents[0].EventCount)
	}
}

func TestController_IngestEvent_BufferFull(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{
		EventBufferSize: 1,
		AlertBufferSize: 10,
	}
	c := New(cfg, log)
	ctx := context.Background()

	ev := &types.SecurityEvent{ID: "ev-1", AgentID: "a1", PodName: "p", PodNamespace: "ns"}
	if err := c.IngestEvent(ctx, ev); err != nil {
		t.Fatalf("first IngestEvent: %v", err)
	}
	// Second event: buffer size 1, so first is still in buffer (nothing consuming)
	// We need to fill the buffer. With size 1, one event is in; second send blocks or fails?
	// IngestEvent does: update agents, then select { case c.eventBuffer <- event }. So if buffer is full (1),
	// the second send will block. So we can't test "buffer full" without a consumer. Alternatively use a tiny buffer
	// and start the consumer that drains slowly - complex. Simpler: test with buffer size 0? No, make(chan x, 0) would block on first send. So buffer size 1: first IngestEvent succeeds, second blocks forever. So we need to either not test buffer full, or use a goroutine that consumes. Actually the doc says "Returns error if buffer is full" - so it's a non-blocking send. Let me check the code.
	// IngestEvent: select { case c.eventBuffer <- event: return nil; default: return fmt.Errorf("event buffer full") }
	// So it's non-blocking. With buffer size 1, first event goes in, second event hits default and returns error. But nothing is reading from eventBuffer, so the first event stays there. So second IngestEvent will get "event buffer full". Good.
	ev2 := &types.SecurityEvent{ID: "ev-2", AgentID: "a2", PodName: "p2", PodNamespace: "ns"}
	err := c.IngestEvent(ctx, ev2)
	if err == nil {
		t.Error("expected error when buffer full, got nil")
	}
}

func TestController_GetAlerts_Empty(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{EventBufferSize: 10, AlertBufferSize: 10}
	c := New(cfg, log)
	alerts := c.GetAlerts(100)
	if len(alerts) != 0 {
		t.Errorf("GetAlerts: want 0, got %d", len(alerts))
	}
}

func TestController_Start_EventToAlertFlow(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{
		EventBufferSize: 100,
		AlertBufferSize: 100,
	}
	c := New(cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx)

	// Ingest event that triggers APSS-002 (cryptominer)
	ev := &types.SecurityEvent{
		ID: "ev-1", AgentID: "agent-1", Type: "process_start", Severity: "CRITICAL",
		Timestamp: time.Now(), PodName: "pod-1", PodNamespace: "default",
		Process: &types.ProcessEventData{
			PID: 100, Name: "xmrig",
			SuspiciousIndicators: []string{"possible_cryptominer"},
		},
	}
	if err := c.IngestEvent(ctx, ev); err != nil {
		t.Fatalf("IngestEvent: %v", err)
	}

	// Wait for processEvents and processAlerts to run
	time.Sleep(150 * time.Millisecond)

	alerts := c.GetAlerts(10)
	if len(alerts) < 1 {
		t.Errorf("expected at least 1 alert from cryptominer event, got %d", len(alerts))
	}
	found := false
	for _, a := range alerts {
		if a.RuleID == "APSS-002" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected APSS-002 alert")
	}
}

func TestController_GetAlerts_Limit(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{EventBufferSize: 10, AlertBufferSize: 10}
	c := New(cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx)

	// Ingest two events that trigger alerts
	for i := 0; i < 2; i++ {
		ev := &types.SecurityEvent{
			ID: fmt.Sprintf("ev-%d", i), AgentID: "a", Type: "process_start", Severity: "HIGH",
			Timestamp: time.Now(), PodName: "p", PodNamespace: "ns",
			Process: &types.ProcessEventData{SuspiciousIndicators: []string{"shell_spawn"}},
		}
		_ = c.IngestEvent(ctx, ev)
	}
	time.Sleep(150 * time.Millisecond)

	// GetAlerts(1) should return only 1
	alerts := c.GetAlerts(1)
	if len(alerts) != 1 {
		t.Errorf("GetAlerts(1): want 1, got %d", len(alerts))
	}

	// GetAlerts(0) - limit <= 0 means return all
	alerts0 := c.GetAlerts(0)
	if len(alerts0) < 2 {
		t.Errorf("GetAlerts(0): want at least 2, got %d", len(alerts0))
	}

	// GetAlerts(999) - limit > n returns n
	alerts999 := c.GetAlerts(999)
	if len(alerts999) != len(alerts0) {
		t.Errorf("GetAlerts(999): got %d, want %d", len(alerts999), len(alerts0))
	}
}

func TestController_SendHighSeverityEvent_NoClient(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{EventBufferSize: 10, AlertBufferSize: 10}
	c := New(cfg, log)
	ctx := context.Background()
	ev := &types.SecurityEvent{
		ID: "ev-1", AgentID: "a", Type: "process_start", Severity: "CRITICAL",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "ns",
	}
	c.SendHighSeverityEvent(ctx, ev) // no panic when client is nil
}

func TestController_SweetSecurity(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{EventBufferSize: 10, AlertBufferSize: 10}
	c := New(cfg, log)
	client := c.SweetSecurity()
	if client != nil {
		t.Error("SweetSecurity should be nil when not configured")
	}
}
