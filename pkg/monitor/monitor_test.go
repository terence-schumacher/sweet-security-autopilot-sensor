package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNew(t *testing.T) {
	log := logrus.New()
	cfg := &AgentConfig{
		AgentID:             "agent-1",
		PodName:             "pod-1",
		PodNamespace:        "default",
		ControllerEndpoint:  "localhost:8080",
		ProcScanInterval:    time.Second,
		NetScanInterval:     time.Second,
		FileScanInterval:    time.Second,
		WatchPaths:          []string{}, // empty so fileintegrity doesn't watch real paths
		SuspiciousProcesses: []string{"nc"},
		SuspiciousPorts:     []int{4444},
	}
	m, err := New(cfg, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if m == nil {
		t.Fatal("New returned nil")
	}
	if m.collector == nil || m.procMon == nil || m.netMon == nil || m.fileMon == nil {
		t.Error("monitor sub-components should be initialized")
	}
}

func TestMonitor_Shutdown(t *testing.T) {
	log := logrus.New()
	cfg := &AgentConfig{
		ControllerEndpoint: "localhost:8080",
		WatchPaths:         []string{},
	}
	m, err := New(cfg, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = m.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}
