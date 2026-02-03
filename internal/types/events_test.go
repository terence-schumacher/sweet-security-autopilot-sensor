package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSecurityEvent_JSONRoundTrip(t *testing.T) {
	ev := SecurityEvent{
		ID:           "ev-1",
		AgentID:      "agent-1",
		Type:         "process_start",
		Severity:     "HIGH",
		Timestamp:    time.Now(),
		PodName:      "my-pod",
		PodNamespace: "default",
		Process: &ProcessEventData{
			PID:                  1234,
			PPID:                 1,
			Name:                 "bash",
			Cmdline:              []string{"bash", "-i"},
			SuspiciousIndicators: []string{"shell_spawn"},
		},
	}
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got SecurityEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.ID != ev.ID || got.AgentID != ev.AgentID || got.Type != ev.Type {
		t.Errorf("round trip: got ID=%q AgentID=%q Type=%q", got.ID, got.AgentID, got.Type)
	}
	if got.Process == nil || got.Process.PID != 1234 || got.Process.Name != "bash" {
		t.Errorf("round trip Process: got %+v", got.Process)
	}
}

func TestAlert_JSONRoundTrip(t *testing.T) {
	a := Alert{
		ID:          "alert-1",
		Timestamp:   time.Now(),
		Severity:    "CRITICAL",
		RuleID:      "APSS-001",
		RuleName:    "Reverse Shell",
		Description: "Test",
		EventIDs:    []string{"ev-1"},
		PodName:     "p",
		PodNS:       "default",
		Actions:     []string{"Investigate"},
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Alert
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.RuleID != a.RuleID || got.Severity != a.Severity {
		t.Errorf("round trip: got RuleID=%q Severity=%q", got.RuleID, got.Severity)
	}
}
