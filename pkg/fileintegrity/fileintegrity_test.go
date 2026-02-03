package fileintegrity

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/pkg/collector"
)

func TestNew_EmptyWatchPaths(t *testing.T) {
	log := logrus.New()
	ch := make(chan collector.SecurityEvent, 1)
	fm, err := New(Config{WatchPaths: []string{}, EventChan: ch}, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if fm == nil {
		t.Fatal("New returned nil")
	}
}

func TestFileMonitor_classifySeverity(t *testing.T) {
	log := logrus.New()
	ch := make(chan collector.SecurityEvent, 1)
	fm, err := New(Config{WatchPaths: []string{}, EventChan: ch}, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	tests := []struct {
		path string
		op   string
		def  collector.Severity
		want collector.Severity
	}{
		{"/etc/passwd", "modify", collector.SeverityMedium, collector.SeverityCritical},
		{"/etc/shadow", "modify", collector.SeverityMedium, collector.SeverityCritical},
		{"/etc/crontab", "modify", collector.SeverityMedium, collector.SeverityHigh},
		{"/tmp/foo.sh", "create", collector.SeverityLow, collector.SeverityMedium},
		{"/tmp/foo.txt", "create", collector.SeverityLow, collector.SeverityLow},
	}
	for _, tt := range tests {
		got := fm.classifySeverity(tt.path, tt.op, tt.def)
		if got != tt.want {
			t.Errorf("classifySeverity(%q, %q, %v) = %v, want %v", tt.path, tt.op, tt.def, got, tt.want)
		}
	}
}
