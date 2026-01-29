package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/pkg/collector"
	"github.com/invisible-tech/autopilot-security-sensor/pkg/fileintegrity"
	"github.com/invisible-tech/autopilot-security-sensor/pkg/netpolicy"
	"github.com/invisible-tech/autopilot-security-sensor/pkg/procmon"
)

// AgentConfig holds configuration for the monitoring agent
type AgentConfig struct {
	AgentID            string
	PodName            string
	PodNamespace       string
	NodeName           string
	ControllerEndpoint string

	// Monitoring intervals
	ProcScanInterval time.Duration
	NetScanInterval  time.Duration
	FileScanInterval time.Duration

	// Detection patterns
	WatchPaths          []string
	SuspiciousProcesses []string
	SuspiciousPorts     []int
}

// Monitor orchestrates all security monitoring components
type Monitor struct {
	cfg *AgentConfig
	log *logrus.Logger

	// Sub-monitors
	procMon *procmon.ProcessMonitor
	netMon  *netpolicy.NetworkMonitor
	fileMon *fileintegrity.FileMonitor

	// Event collector (sends to controller)
	collector *collector.EventCollector

	// Synchronization
	wg     sync.WaitGroup
	stopCh chan struct{}
}

// New creates a new Monitor instance
func New(cfg *AgentConfig, log *logrus.Logger) (*Monitor, error) {
	m := &Monitor{
		cfg:    cfg,
		log:    log,
		stopCh: make(chan struct{}),
	}

	// Initialize event collector
	var err error
	m.collector, err = collector.New(collector.Config{
		ControllerEndpoint: cfg.ControllerEndpoint,
		AgentID:            cfg.AgentID,
		PodName:            cfg.PodName,
		PodNamespace:       cfg.PodNamespace,
		BufferSize:         10000,
	}, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create collector: %w", err)
	}

	// Initialize process monitor
	m.procMon = procmon.New(procmon.Config{
		ScanInterval:        cfg.ProcScanInterval,
		SuspiciousProcesses: cfg.SuspiciousProcesses,
		EventChan:           m.collector.EventChannel(),
	}, log)

	// Initialize network monitor
	m.netMon = netpolicy.New(netpolicy.Config{
		ScanInterval:    cfg.NetScanInterval,
		SuspiciousPorts: cfg.SuspiciousPorts,
		EventChan:       m.collector.EventChannel(),
	}, log)

	// Initialize file integrity monitor
	m.fileMon, err = fileintegrity.New(fileintegrity.Config{
		WatchPaths: cfg.WatchPaths,
		EventChan:  m.collector.EventChannel(),
	}, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create file monitor: %w", err)
	}

	return m, nil
}

// Start begins all monitoring goroutines
func (m *Monitor) Start(ctx context.Context) error {
	m.log.Info("Starting security monitors")

	// Start collector first
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		if err := m.collector.Start(ctx); err != nil {
			m.log.WithError(err).Error("Collector error")
		}
	}()

	// Start process monitor
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.procMon.Start(ctx)
	}()

	// Start network monitor
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.netMon.Start(ctx)
	}()

	// Start file integrity monitor
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.fileMon.Start(ctx)
	}()

	m.log.Info("All monitors started")

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

// Shutdown gracefully stops all monitors
func (m *Monitor) Shutdown(ctx context.Context) error {
	m.log.Info("Shutting down monitors")

	close(m.stopCh)

	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.log.Info("All monitors stopped")
	case <-ctx.Done():
		m.log.Warn("Shutdown timeout, some monitors may not have stopped cleanly")
	}

	return nil
}
