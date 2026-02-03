package netpolicy

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/pkg/collector"
)

func TestNew(t *testing.T) {
	log := logrus.New()
	ch := make(chan collector.SecurityEvent, 1)
	nm := New(Config{
		ScanInterval:    time.Second,
		SuspiciousPorts: []int{4444},
		EventChan:       ch,
	}, log)
	if nm == nil {
		t.Fatal("New returned nil")
	}
}

func TestNetworkMonitor_parseState(t *testing.T) {
	log := logrus.New()
	nm := New(Config{
		ScanInterval: time.Second,
		EventChan:    make(chan collector.SecurityEvent, 1),
	}, log)
	tests := []struct {
		in   string
		want string
	}{
		{"01", "ESTABLISHED"},
		{"0A", "LISTEN"},
		{"06", "TIME_WAIT"},
		{"FF", "UNKNOWN"},
	}
	for _, tt := range tests {
		got := nm.parseState(tt.in)
		if got != tt.want {
			t.Errorf("parseState(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNetworkMonitor_connectionKey(t *testing.T) {
	log := logrus.New()
	nm := New(Config{ScanInterval: time.Second, EventChan: make(chan collector.SecurityEvent, 1)}, log)
	conn := &Connection{
		Protocol:   "tcp",
		LocalIP:    net.IPv4(127, 0, 0, 1),
		LocalPort:  8080,
		RemoteIP:   net.IPv4(1, 2, 3, 4),
		RemotePort: 4444,
		State:      "ESTABLISHED",
	}
	key := nm.connectionKey(conn)
	if key == "" {
		t.Error("connectionKey returned empty")
	}
	if len(key) < 10 {
		t.Errorf("connectionKey too short: %q", key)
	}
}

func TestNetworkMonitor_isPrivateIP(t *testing.T) {
	log := logrus.New()
	nm := New(Config{ScanInterval: time.Second, EventChan: make(chan collector.SecurityEvent, 1)}, log)
	if !nm.isPrivateIP(net.IPv4(127, 0, 0, 1)) {
		t.Error("127.0.0.1 should be private")
	}
	if !nm.isPrivateIP(net.IPv4(10, 0, 0, 1)) {
		t.Error("10.0.0.1 should be private")
	}
	if !nm.isPrivateIP(net.IPv4(192, 168, 1, 1)) {
		t.Error("192.168.1.1 should be private")
	}
	if nm.isPrivateIP(net.IPv4(8, 8, 8, 8)) {
		t.Error("8.8.8.8 should not be private")
	}
	if !nm.isPrivateIP(nil) {
		t.Error("nil IP should be treated as private")
	}
}

func TestNetworkMonitor_isPotentialReverseShell(t *testing.T) {
	log := logrus.New()
	nm := New(Config{ScanInterval: time.Second, EventChan: make(chan collector.SecurityEvent, 1)}, log)
	conn := &Connection{RemotePort: 4444, LocalPort: 80}
	if !nm.isPotentialReverseShell(conn) {
		t.Error("port 4444 should be reverse shell")
	}
	conn.RemotePort = 80
	conn.LocalPort = 1337
	if !nm.isPotentialReverseShell(conn) {
		t.Error("port 1337 should be reverse shell")
	}
	conn.RemotePort = 80
	conn.LocalPort = 80
	if nm.isPotentialReverseShell(conn) {
		t.Error("port 80 should not be reverse shell")
	}
}

func TestNetworkMonitor_analyzeConnection(t *testing.T) {
	log := logrus.New()
	ch := make(chan collector.SecurityEvent, 10)
	nm := New(Config{
		ScanInterval:    time.Second,
		SuspiciousPorts: []int{4444},
		EventChan:       ch,
	}, log)
	conn := &Connection{
		Protocol:   "tcp",
		LocalIP:    net.IPv4(127, 0, 0, 1),
		LocalPort:  5000,
		RemoteIP:   net.IPv4(8, 8, 8, 8),
		RemotePort: 4444,
		State:      "ESTABLISHED",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	nm.analyzeConnection(ctx, conn)
	select {
	case <-ch:
		// received event
	default:
		t.Error("expected one event from analyzeConnection")
	}
}
