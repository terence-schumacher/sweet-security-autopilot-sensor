package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/internal/config"
	"github.com/invisible-tech/autopilot-security-sensor/internal/controller"
	"github.com/invisible-tech/autopilot-security-sensor/internal/types"
)

func TestServer_Health(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{HTTPAddr: ":0", EventBufferSize: 10, AlertBufferSize: 10}
	ctrl := controller.New(cfg, log)
	srv := New(cfg, ctrl, log)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /health: status %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode health body: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("health status = %q", body["status"])
	}
	if body["version"] == "" {
		t.Error("health version should be set")
	}
}

func TestServer_Events_Post(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{HTTPAddr: ":0", EventBufferSize: 10, AlertBufferSize: 10}
	ctrl := controller.New(cfg, log)
	srv := New(cfg, ctrl, log)

	ev := types.SecurityEvent{
		ID: "ev-1", AgentID: "agent-1", Type: "process_start", Severity: "INFO",
		Timestamp: time.Now(), PodName: "pod-1", PodNamespace: "default",
	}
	body, _ := json.Marshal(ev)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleEvents(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("POST /api/v1/events: status %d", rec.Code)
	}

	agents := ctrl.GetAgents()
	if len(agents) != 1 || agents[0].ID != "agent-1" {
		t.Errorf("after POST events: agents = %+v", agents)
	}
}

func TestServer_Events_MethodNotAllowed(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{HTTPAddr: ":0", EventBufferSize: 10, AlertBufferSize: 10}
	ctrl := controller.New(cfg, log)
	srv := New(cfg, ctrl, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	rec := httptest.NewRecorder()
	srv.handleEvents(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/v1/events: status %d", rec.Code)
	}
}

func TestServer_Events_InvalidJSON(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{HTTPAddr: ":0", EventBufferSize: 10, AlertBufferSize: 10}
	ctrl := controller.New(cfg, log)
	srv := New(cfg, ctrl, log)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleEvents(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST invalid JSON: status %d", rec.Code)
	}
}

func TestServer_Agents(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{HTTPAddr: ":0", EventBufferSize: 10, AlertBufferSize: 10}
	ctrl := controller.New(cfg, log)
	srv := New(cfg, ctrl, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	srv.handleAgents(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/v1/agents: status %d", rec.Code)
	}
	var agents []*types.AgentInfo
	if err := json.NewDecoder(rec.Body).Decode(&agents); err != nil {
		t.Fatalf("decode agents: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("initial agents: want 0, got %d", len(agents))
	}
}

func TestServer_Alerts(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{HTTPAddr: ":0", EventBufferSize: 10, AlertBufferSize: 10}
	ctrl := controller.New(cfg, log)
	srv := New(cfg, ctrl, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	rec := httptest.NewRecorder()
	srv.handleAlerts(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/v1/alerts: status %d", rec.Code)
	}
	var alerts []*types.Alert
	if err := json.NewDecoder(rec.Body).Decode(&alerts); err != nil {
		t.Fatalf("decode alerts: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("initial alerts: want 0, got %d", len(alerts))
	}
}

func TestServer_Events_BufferFull(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{HTTPAddr: ":0", EventBufferSize: 1, AlertBufferSize: 10}
	ctrl := controller.New(cfg, log)
	// Do not call ctrl.Start() so no consumer drains the buffer

	// Fill the buffer (size 1) with one event
	ev := types.SecurityEvent{
		ID: "ev-1", AgentID: "a1", Type: "process_start", Severity: "INFO",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "ns",
	}
	_ = ctrl.IngestEvent(context.Background(), &ev)

	// Second event should get 503 (buffer full)
	ev2 := types.SecurityEvent{
		ID: "ev-2", AgentID: "a1", Type: "process_start", Severity: "INFO",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "ns",
	}
	body, _ := json.Marshal(ev2)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv := New(cfg, ctrl, log)
	srv.handleEvents(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when buffer full, got %d", rec.Code)
	}
}

func TestServer_Events_HighSeverityCallsSendHighSeverityEvent(t *testing.T) {
	log := logrus.New()
	cfg := config.ControllerConfig{HTTPAddr: ":0", EventBufferSize: 10, AlertBufferSize: 10}
	ctrl := controller.New(cfg, log)
	srv := New(cfg, ctrl, log)
	ev := types.SecurityEvent{
		ID: "ev-1", AgentID: "a1", Type: "process_start", Severity: "CRITICAL",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "ns",
		Process: &types.ProcessEventData{Name: "bash", PID: 1},
	}
	body, _ := json.Marshal(ev)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleEvents(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("POST CRITICAL event: status %d", rec.Code)
	}
}
