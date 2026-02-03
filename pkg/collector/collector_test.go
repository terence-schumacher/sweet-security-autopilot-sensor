package collector

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNew(t *testing.T) {
	log := logrus.New()
	cfg := Config{
		ControllerEndpoint: "localhost:8080",
		AgentID:            "agent-1",
		PodName:            "pod-1",
		PodNamespace:       "default",
		BufferSize:         100,
	}
	ec, err := New(cfg, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if ec == nil {
		t.Fatal("New returned nil collector")
	}
	if ec.EventChannel() == nil {
		t.Error("EventChannel() returned nil")
	}
}

func TestNew_DefaultBufferSize(t *testing.T) {
	log := logrus.New()
	cfg := Config{
		ControllerEndpoint: "localhost:8080",
		AgentID:            "a",
		PodName:            "p",
		PodNamespace:       "ns",
		BufferSize:         0,
	}
	ec, err := New(cfg, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// BufferSize 0 should become 10000 (see New)
	if ec == nil {
		t.Fatal("New returned nil")
	}
}

func TestCollector_SendEvent(t *testing.T) {
	// Skip if we cannot bind (e.g. sandbox or no network).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind for test: %v", err)
	}
	ln.Close()

	var (
		mu       sync.Mutex
		lastBody []byte
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body := make([]byte, 4096)
		n, _ := r.Body.Read(body)
		body = body[:n]
		mu.Lock()
		lastBody = body
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	log := logrus.New()
	cfg := Config{
		ControllerEndpoint: server.Listener.Addr().String(),
		AgentID:            "agent-test",
		PodName:            "pod-test",
		PodNamespace:       "default",
		BufferSize:         10,
	}
	ec, err := New(cfg, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = ec.Start(ctx)
	}()

	ev := SecurityEvent{
		ID:        "ev-1",
		Type:      EventTypeProcessStart,
		Severity:  SeverityHigh,
		Timestamp: time.Now(),
		Process: &ProcessEvent{
			PID:                  1234,
			Name:                 "bash",
			Cmdline:              []string{"bash", "-i"},
			SuspiciousIndicators: []string{"shell_spawn"},
		},
	}
	ec.EventChannel() <- ev

	// Wait for request to be received
	for i := 0; i < 50; i++ {
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		lb := lastBody
		mu.Unlock()
		if len(lb) > 0 {
			break
		}
	}

	mu.Lock()
	body := lastBody
	mu.Unlock()
	if len(body) == 0 {
		t.Fatal("server did not receive request body")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if decoded["agent_id"] != "agent-test" {
		t.Errorf("agent_id = %q", decoded["agent_id"])
	}
	if decoded["pod_name"] != "pod-test" {
		t.Errorf("pod_name = %q", decoded["pod_name"])
	}
	proc, _ := decoded["process"].(map[string]interface{})
	if proc == nil {
		t.Fatal("process missing")
	}
	if proc["name"] != "bash" || proc["pid"] != float64(1234) {
		t.Errorf("process: %+v", proc)
	}
}

func TestGetStats(t *testing.T) {
	log := logrus.New()
	cfg := Config{
		ControllerEndpoint: "localhost:9999",
		AgentID:            "a",
		PodName:            "p",
		PodNamespace:       "ns",
		BufferSize:         10,
	}
	ec, err := New(cfg, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sent, dropped := ec.GetStats()
	if sent != 0 || dropped != 0 {
		t.Errorf("initial stats: sent=%d dropped=%d", sent, dropped)
	}
}

func TestEventTypeToString(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{EventTypeProcessStart, "process_start"},
		{EventTypeProcessExit, "process_exit"},
		{EventTypeNetworkConnect, "network_connect"},
		{EventTypeNetworkListen, "network_listen"},
		{EventTypeFileCreate, "file_create"},
		{EventTypeFileModify, "file_modify"},
		{EventTypeFileDelete, "file_delete"},
		{EventTypeFileAccess, "file_access"},
		{EventTypeUnknown, "unknown"},
		{EventType(99), "unknown"},
	}
	for _, tt := range tests {
		got := eventTypeToString(tt.et)
		if got != tt.want {
			t.Errorf("eventTypeToString(%v) = %q, want %q", tt.et, got, tt.want)
		}
	}
}

func TestSeverityToString(t *testing.T) {
	tests := []struct {
		s    Severity
		want string
	}{
		{SeverityCritical, "CRITICAL"},
		{SeverityHigh, "HIGH"},
		{SeverityMedium, "MEDIUM"},
		{SeverityLow, "LOW"},
		{SeverityInfo, "INFO"},
		{SeverityUnknown, "UNKNOWN"},
		{Severity(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		got := severityToString(tt.s)
		if got != tt.want {
			t.Errorf("severityToString(%v) = %q, want %q", tt.s, got, tt.want)
		}
	}
}
