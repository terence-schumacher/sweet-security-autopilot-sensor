package detection

import (
	"testing"
	"time"

	"github.com/invisible-tech/autopilot-security-sensor/internal/types"
)

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("NewEngine() returned nil")
	}
	rules := e.Rules()
	if len(rules) < 5 {
		t.Errorf("expected at least 5 rules, got %d", len(rules))
	}
}

func TestEngine_Evaluate_NoMatch(t *testing.T) {
	e := NewEngine()
	ev := &types.SecurityEvent{
		ID: "ev-1", Type: "process_start", Severity: "INFO",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "default",
		Process: &types.ProcessEventData{PID: 1, Name: "sleep", Cmdline: []string{"sleep", "1"}},
	}
	alerts := e.Evaluate(ev)
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts for benign process, got %d", len(alerts))
	}
}

func TestEngine_Evaluate_APSS001_ReverseShell(t *testing.T) {
	e := NewEngine()
	ev := &types.SecurityEvent{
		ID: "ev-1", Type: "network_connect", Severity: "HIGH",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "default",
		Network: &types.NetworkEventData{
			Protocol: "tcp", DstIP: "1.2.3.4", DstPort: 4444,
			State: "ESTABLISHED", IsExternal: true, IsSuspiciousPort: true,
		},
	}
	alerts := e.Evaluate(ev)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert (APSS-001), got %d", len(alerts))
	}
	if alerts[0].RuleID != "APSS-001" || alerts[0].Severity != "CRITICAL" {
		t.Errorf("alert: RuleID=%q Severity=%q", alerts[0].RuleID, alerts[0].Severity)
	}
}

func TestEngine_Evaluate_APSS002_Cryptominer(t *testing.T) {
	e := NewEngine()
	ev := &types.SecurityEvent{
		ID: "ev-1", Type: "process_start", Severity: "HIGH",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "default",
		Process: &types.ProcessEventData{
			PID: 100, Name: "xmrig",
			SuspiciousIndicators: []string{"possible_cryptominer"},
		},
	}
	alerts := e.Evaluate(ev)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert (APSS-002), got %d", len(alerts))
	}
	if alerts[0].RuleID != "APSS-002" {
		t.Errorf("alert RuleID = %q", alerts[0].RuleID)
	}
}

func TestEngine_Evaluate_APSS003_SensitiveFile(t *testing.T) {
	e := NewEngine()
	ev := &types.SecurityEvent{
		ID: "ev-1", Type: "file_modify", Severity: "HIGH",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "default",
		File: &types.FileEventData{Path: "/etc/passwd", Operation: "modify"},
	}
	alerts := e.Evaluate(ev)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert (APSS-003), got %d", len(alerts))
	}
	if alerts[0].RuleID != "APSS-003" {
		t.Errorf("alert RuleID = %q", alerts[0].RuleID)
	}
}

func TestEngine_Evaluate_APSS003_NonCriticalPath(t *testing.T) {
	e := NewEngine()
	ev := &types.SecurityEvent{
		ID: "ev-1", Type: "file_modify", Severity: "MEDIUM",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "default",
		File: &types.FileEventData{Path: "/tmp/foo", Operation: "modify"},
	}
	alerts := e.Evaluate(ev)
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts for /tmp/foo, got %d", len(alerts))
	}
}

func TestEngine_Evaluate_APSS004_ShellSpawn(t *testing.T) {
	e := NewEngine()
	ev := &types.SecurityEvent{
		ID: "ev-1", Type: "process_start", Severity: "MEDIUM",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "default",
		Process: &types.ProcessEventData{
			PID: 200, Name: "bash",
			SuspiciousIndicators: []string{"shell_spawn"},
		},
	}
	alerts := e.Evaluate(ev)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert (APSS-004), got %d", len(alerts))
	}
	if alerts[0].RuleID != "APSS-004" {
		t.Errorf("alert RuleID = %q", alerts[0].RuleID)
	}
}

func TestEngine_Evaluate_APSS005_ExternalDB(t *testing.T) {
	e := NewEngine()
	ev := &types.SecurityEvent{
		ID: "ev-1", Type: "network_connect", Severity: "MEDIUM",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "default",
		Network: &types.NetworkEventData{
			Protocol: "tcp", DstIP: "8.8.8.8", DstPort: 3306,
			State: "ESTABLISHED", IsExternal: true, IsSuspiciousPort: false,
		},
	}
	alerts := e.Evaluate(ev)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert (APSS-005), got %d", len(alerts))
	}
	if alerts[0].RuleID != "APSS-005" {
		t.Errorf("alert RuleID = %q", alerts[0].RuleID)
	}
}

func TestEngine_Evaluate_AlertFields(t *testing.T) {
	e := NewEngine()
	ev := &types.SecurityEvent{
		ID: "ev-99", Type: "process_start", Severity: "CRITICAL",
		Timestamp: time.Now(), PodName: "my-pod", PodNamespace: "prod",
		Process: &types.ProcessEventData{SuspiciousIndicators: []string{"possible_cryptominer"}},
	}
	alerts := e.Evaluate(ev)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	a := alerts[0]
	if a.EventIDs[0] != "ev-99" || a.PodName != "my-pod" || a.PodNS != "prod" {
		t.Errorf("alert context: EventIDs=%v PodName=%q PodNS=%q", a.EventIDs, a.PodName, a.PodNS)
	}
	if len(a.Actions) == 0 {
		t.Error("alert should have recommended actions")
	}
}
