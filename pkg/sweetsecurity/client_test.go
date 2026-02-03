package sweetsecurity

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewClient(t *testing.T) {
	log := logrus.New()
	cfg := Config{
		APIEndpoint: "https://api.example.com",
		APIKey:      "test-key",
		Timeout:     5 * time.Second,
	}
	c := NewClient(cfg, log)
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	log := logrus.New()
	cfg := Config{APIEndpoint: "https://api.example.com", APIKey: "key"}
	c := NewClient(cfg, log)
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func canListen(t *testing.T) bool {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind for test: %v", err)
		return false
	}
	ln.Close()
	return true
}

func TestClient_SendAlert_Success(t *testing.T) {
	if !canListen(t) {
		return
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/alerts" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer my-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logrus.New()
	c := NewClient(Config{
		APIEndpoint: server.URL,
		APIKey:      "my-key",
		Timeout:     5 * time.Second,
	}, log)

	ctx := context.Background()
	alert := &Alert{
		ID: "alert-1", Severity: "CRITICAL", RuleID: "APSS-001",
		RuleName: "Test", Description: "Test alert",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "ns",
	}
	err := c.SendAlert(ctx, alert)
	if err != nil {
		t.Errorf("SendAlert: %v", err)
	}
}

func TestClient_SendAlert_NotConfigured(t *testing.T) {
	log := logrus.New()
	c := NewClient(Config{APIEndpoint: "", APIKey: ""}, log)
	ctx := context.Background()
	err := c.SendAlert(ctx, &Alert{ID: "a"})
	if err == nil {
		t.Error("expected error when not configured")
	}
}

func TestClient_SendEvent_NotConfigured(t *testing.T) {
	log := logrus.New()
	c := NewClient(Config{}, log)
	err := c.SendEvent(context.Background(), &Event{ID: "e1"})
	if err == nil {
		t.Error("expected error when not configured")
	}
}

func TestClient_SendBatchEvents_NotConfigured(t *testing.T) {
	log := logrus.New()
	c := NewClient(Config{}, log)
	err := c.SendBatchEvents(context.Background(), []*Event{{ID: "e1"}})
	if err == nil {
		t.Error("expected error when not configured")
	}
}

func TestClient_SendEvent_Success(t *testing.T) {
	if !canListen(t) {
		return
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logrus.New()
	c := NewClient(Config{
		APIEndpoint: server.URL,
		APIKey:      "key",
		Timeout:     5 * time.Second,
	}, log)

	ctx := context.Background()
	ev := &Event{
		ID: "ev-1", Type: "process_start", Severity: "HIGH",
		Timestamp: time.Now(), PodName: "p", PodNamespace: "ns",
	}
	err := c.SendEvent(ctx, ev)
	if err != nil {
		t.Errorf("SendEvent: %v", err)
	}
}

func TestClient_HealthCheck_Success(t *testing.T) {
	if !canListen(t) {
		return
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logrus.New()
	c := NewClient(Config{
		APIEndpoint: server.URL,
		APIKey:      "key",
		Timeout:     5 * time.Second,
	}, log)

	ctx := context.Background()
	err := c.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck: %v", err)
	}
}

func TestClient_HealthCheck_NotConfigured(t *testing.T) {
	log := logrus.New()
	c := NewClient(Config{}, log)
	err := c.HealthCheck(context.Background())
	if err == nil {
		t.Error("expected error when not configured")
	}
}

func TestClient_HealthCheck_NonOK(t *testing.T) {
	if !canListen(t) {
		return
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	log := logrus.New()
	c := NewClient(Config{
		APIEndpoint: server.URL,
		APIKey:      "key",
		Timeout:     5 * time.Second,
	}, log)

	err := c.HealthCheck(context.Background())
	if err == nil {
		t.Error("expected error on 503")
	}
}

func TestClient_SendAlert_Non2xx(t *testing.T) {
	if !canListen(t) {
		return
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	log := logrus.New()
	c := NewClient(Config{
		APIEndpoint: server.URL,
		APIKey:      "key",
		Timeout:     5 * time.Second,
	}, log)

	err := c.SendAlert(context.Background(), &Alert{ID: "a", Severity: "HIGH", Timestamp: time.Now()})
	if err == nil {
		t.Error("expected error on 500")
	}
}

func TestClient_SendBatchEvents_Success(t *testing.T) {
	if !canListen(t) {
		return
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events/batch" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logrus.New()
	c := NewClient(Config{
		APIEndpoint: server.URL,
		APIKey:      "key",
		Timeout:     5 * time.Second,
	}, log)

	err := c.SendBatchEvents(context.Background(), []*Event{
		{ID: "e1", Type: "process_start", Severity: "HIGH", Timestamp: time.Now(), PodName: "p", PodNamespace: "ns"},
	})
	if err != nil {
		t.Errorf("SendBatchEvents: %v", err)
	}
}
