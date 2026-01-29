// Package controller provides the core event processing, detection, and
// alert pipeline for the APSS controller.
package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/internal/config"
	"github.com/invisible-tech/autopilot-security-sensor/internal/detection"
	"github.com/invisible-tech/autopilot-security-sensor/internal/types"
	"github.com/invisible-tech/autopilot-security-sensor/pkg/sweetsecurity"
)

// Prometheus metrics (registered once).
var (
	eventsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apss_events_received_total",
			Help: "Total security events received",
		},
		[]string{"type", "severity", "namespace"},
	)
	alertsGenerated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apss_alerts_generated_total",
			Help: "Total security alerts generated",
		},
		[]string{"rule", "severity"},
	)
	activeAgents = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "apss_active_agents",
			Help: "Number of active APSS agents",
		},
	)
)

func init() {
	prometheus.MustRegister(eventsReceived)
	prometheus.MustRegister(alertsGenerated)
	prometheus.MustRegister(activeAgents)
}

// Controller orchestrates event processing, detection, and alert handling.
type Controller struct {
	cfg      config.ControllerConfig
	log      *logrus.Logger
	engine   *detection.Engine
	agents   map[string]*types.AgentInfo
	agentsMu sync.RWMutex
	alerts   []*types.Alert
	alertsMu sync.RWMutex

	eventBuffer chan *types.SecurityEvent
	alertChan   chan *types.Alert

	sweetSecurity   *sweetsecurity.Client
	sweetSecurityMu sync.RWMutex
}

// New creates a new Controller with the given config and logger.
func New(cfg config.ControllerConfig, log *logrus.Logger) *Controller {
	c := &Controller{
		cfg:         cfg,
		log:         log,
		engine:      detection.NewEngine(),
		agents:      make(map[string]*types.AgentInfo),
		eventBuffer: make(chan *types.SecurityEvent, cfg.EventBufferSize),
		alertChan:   make(chan *types.Alert, cfg.AlertBufferSize),
	}
	c.initSweetSecurity()
	return c
}

func (c *Controller) initSweetSecurity() {
	if !c.cfg.SweetSecurityEnabled {
		return
	}
	client := sweetsecurity.NewClient(sweetsecurity.Config{
		APIEndpoint: c.cfg.SweetSecurityEndpoint,
		APIKey:      c.cfg.SweetSecurityAPIKey,
		Timeout:     c.cfg.SweetSecurityTimeout,
	}, c.log)
	c.sweetSecurityMu.Lock()
	c.sweetSecurity = client
	c.sweetSecurityMu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.HealthCheck(ctx); err != nil {
			c.log.WithError(err).Warn("Sweet Security health check failed, will retry on first alert")
		} else {
			c.log.Info("Sweet Security API connection verified")
		}
	}()
}

// Start begins event processing and agent health check goroutines.
// Caller must run the HTTP server separately.
func (c *Controller) Start(ctx context.Context) {
	go c.processEvents(ctx)
	go c.processAlerts(ctx)
	go c.checkAgentHealth(ctx)
}

// IngestEvent accepts an event from the HTTP API and queues it for processing.
// It also updates agent tracking. Returns error if buffer is full.
func (c *Controller) IngestEvent(ctx context.Context, event *types.SecurityEvent) error {
	c.agentsMu.Lock()
	if agent, ok := c.agents[event.AgentID]; ok {
		agent.LastSeen = time.Now()
		agent.EventCount++
	} else {
		c.agents[event.AgentID] = &types.AgentInfo{
			ID:           event.AgentID,
			PodName:      event.PodName,
			PodNamespace: event.PodNamespace,
			ConnectedAt:  time.Now(),
			LastSeen:     time.Now(),
			EventCount:   1,
		}
	}
	c.agentsMu.Unlock()

	select {
	case c.eventBuffer <- event:
		return nil
	default:
		return fmt.Errorf("event buffer full")
	}
}

// GetAgents returns a copy of connected agents.
func (c *Controller) GetAgents() []*types.AgentInfo {
	c.agentsMu.RLock()
	defer c.agentsMu.RUnlock()
	out := make([]*types.AgentInfo, 0, len(c.agents))
	for _, a := range c.agents {
		out = append(out, a)
	}
	return out
}

// GetAlerts returns the most recent alerts, up to limit.
func (c *Controller) GetAlerts(limit int) []*types.Alert {
	c.alertsMu.RLock()
	defer c.alertsMu.RUnlock()
	n := len(c.alerts)
	if limit <= 0 || limit > n {
		limit = n
	}
	start := n - limit
	if start < 0 {
		start = 0
	}
	out := make([]*types.Alert, limit)
	copy(out, c.alerts[start:])
	return out
}

// SweetSecurity returns the Sweet Security client if configured (for sending events from server).
func (c *Controller) SweetSecurity() *sweetsecurity.Client {
	c.sweetSecurityMu.RLock()
	defer c.sweetSecurityMu.RUnlock()
	return c.sweetSecurity
}

// SendHighSeverityEvent sends a high/critical event to Sweet Security if configured.
// Call from the HTTP handler after IngestEvent for HIGH/CRITICAL severity.
func (c *Controller) SendHighSeverityEvent(ctx context.Context, event *types.SecurityEvent) {
	c.sweetSecurityMu.RLock()
	client := c.sweetSecurity
	c.sweetSecurityMu.RUnlock()
	if client == nil {
		return
	}
	sweetEvent := &sweetsecurity.Event{
		ID:           event.ID,
		AgentID:      event.AgentID,
		Type:         event.Type,
		Severity:     event.Severity,
		Timestamp:    event.Timestamp,
		PodName:      event.PodName,
		PodNamespace: event.PodNamespace,
		Metadata:     make(map[string]interface{}),
	}
	if event.Process != nil {
		sweetEvent.Process = map[string]interface{}{
			"pid":                   event.Process.PID,
			"ppid":                  event.Process.PPID,
			"name":                  event.Process.Name,
			"cmdline":               event.Process.Cmdline,
			"suspicious_indicators": event.Process.SuspiciousIndicators,
		}
	}
	if event.Network != nil {
		sweetEvent.Network = map[string]interface{}{
			"protocol":           event.Network.Protocol,
			"dst_ip":             event.Network.DstIP,
			"dst_port":           event.Network.DstPort,
			"state":              event.Network.State,
			"is_external":        event.Network.IsExternal,
			"is_suspicious_port": event.Network.IsSuspiciousPort,
		}
	}
	if event.File != nil {
		sweetEvent.File = map[string]interface{}{
			"path":      event.File.Path,
			"operation": event.File.Operation,
			"old_hash":  event.File.OldHash,
			"new_hash":  event.File.NewHash,
		}
	}
	if event.Metadata != nil {
		for k, v := range event.Metadata {
			sweetEvent.Metadata[k] = v
		}
	}
	go func() {
		if err := client.SendEvent(ctx, sweetEvent); err != nil {
			c.log.WithError(err).WithField("event_id", event.ID).Debug("Failed to send event to Sweet Security")
		}
	}()
}

func (c *Controller) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-c.eventBuffer:
			c.evaluateEvent(event)
		}
	}
}

func (c *Controller) evaluateEvent(event *types.SecurityEvent) {
	eventsReceived.WithLabelValues(event.Type, event.Severity, event.PodNamespace).Inc()
	for _, alert := range c.engine.Evaluate(event) {
		select {
		case c.alertChan <- alert:
		default:
			c.log.Warn("Alert channel full, dropping alert")
		}
	}
}

func (c *Controller) processAlerts(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert := <-c.alertChan:
			c.alertsMu.Lock()
			c.alerts = append(c.alerts, alert)
			if len(c.alerts) > c.cfg.AlertRetentionCount {
				c.alerts = c.alerts[len(c.alerts)-c.cfg.AlertRetentionCount:]
			}
			c.alertsMu.Unlock()

			alertsGenerated.WithLabelValues(alert.RuleID, alert.Severity).Inc()
			c.log.WithFields(logrus.Fields{
				"alert_id": alert.ID, "rule_id": alert.RuleID, "rule_name": alert.RuleName,
				"severity": alert.Severity, "pod": alert.PodName, "namespace": alert.PodNS,
				"mitre": alert.MitreID, "description": alert.Description,
			}).Warn("SECURITY ALERT")

			c.sendAlertToSweetSecurity(ctx, alert)
		}
	}
}

func (c *Controller) sendAlertToSweetSecurity(ctx context.Context, alert *types.Alert) {
	c.sweetSecurityMu.RLock()
	client := c.sweetSecurity
	c.sweetSecurityMu.RUnlock()
	if client == nil {
		return
	}
	sweetAlert := &sweetsecurity.Alert{
		ID:           alert.ID,
		Timestamp:    alert.Timestamp,
		Severity:     alert.Severity,
		RuleID:       alert.RuleID,
		RuleName:     alert.RuleName,
		Description:  alert.Description,
		PodName:      alert.PodName,
		PodNamespace: alert.PodNS,
		MitreTactic:  alert.MitreTactic,
		MitreID:      alert.MitreID,
		EventIDs:     alert.EventIDs,
		Metadata: map[string]interface{}{
			"source":              "apss-autopilot-security-sensor",
			"recommended_actions": alert.Actions,
		},
	}
	go func() {
		if err := client.SendAlert(ctx, sweetAlert); err != nil {
			c.log.WithError(err).WithFields(logrus.Fields{"alert_id": alert.ID, "rule_id": alert.RuleID}).Error("Failed to send alert to Sweet Security API")
		}
	}()
}

func (c *Controller) checkAgentHealth(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.agentsMu.Lock()
			now := time.Now()
			for id, agent := range c.agents {
				if now.Sub(agent.LastSeen) > c.cfg.AgentStaleThreshold {
					c.log.WithField("agent_id", id).Warn("Agent appears offline")
					delete(c.agents, id)
				}
			}
			activeAgents.Set(float64(len(c.agents)))
			c.agentsMu.Unlock()
		}
	}
}
