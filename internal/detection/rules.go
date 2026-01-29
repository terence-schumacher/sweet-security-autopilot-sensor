// Package detection provides the detection rules engine for evaluating
// security events and generating alerts.
package detection

import (
	"fmt"
	"time"

	"github.com/invisible-tech/autopilot-security-sensor/internal/types"
)

// Rule defines a detection rule: condition and metadata.
type Rule struct {
	ID          string
	Name        string
	Description string
	Severity    string
	MitreTactic string
	MitreID     string
	Condition   func(event *types.SecurityEvent) bool
	Actions     []string
}

// Engine evaluates events against rules and produces alerts.
type Engine struct {
	rules []*Rule
}

// NewEngine creates a detection engine with the default rule set.
func NewEngine() *Engine {
	e := &Engine{}
	e.rules = defaultRules()
	return e
}

// Evaluate runs all rules against the event and returns any matching alerts.
func (e *Engine) Evaluate(event *types.SecurityEvent) []*types.Alert {
	var alerts []*types.Alert
	for _, rule := range e.rules {
		if rule.Condition(event) {
			alerts = append(alerts, &types.Alert{
				ID:          fmt.Sprintf("alert-%d", time.Now().UnixNano()),
				Timestamp:   time.Now(),
				Severity:    rule.Severity,
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				Description: rule.Description,
				EventIDs:    []string{event.ID},
				PodName:     event.PodName,
				PodNS:       event.PodNamespace,
				MitreTactic: rule.MitreTactic,
				MitreID:     rule.MitreID,
				Actions:     rule.Actions,
			})
		}
	}
	return alerts
}

// Rules returns the loaded rules (read-only).
func (e *Engine) Rules() []*Rule {
	return e.rules
}

func defaultRules() []*Rule {
	return []*Rule{
		{
			ID:          "APSS-001",
			Name:        "Potential Reverse Shell",
			Description: "Detected network connection matching reverse shell pattern",
			Severity:    "CRITICAL",
			MitreTactic: "Command and Control",
			MitreID:     "T1059.004",
			Condition: func(e *types.SecurityEvent) bool {
				if e.Network == nil {
					return false
				}
				suspiciousPorts := map[int]bool{4444: true, 5555: true, 6666: true, 1337: true}
				return e.Network.IsExternal && suspiciousPorts[e.Network.DstPort]
			},
			Actions: []string{"Investigate pod immediately", "Check for unauthorized processes", "Review pod logs"},
		},
		{
			ID:          "APSS-002",
			Name:        "Cryptominer Detected",
			Description: "Process matching known cryptocurrency miner patterns",
			Severity:    "CRITICAL",
			MitreTactic: "Impact",
			MitreID:     "T1496",
			Condition: func(e *types.SecurityEvent) bool {
				if e.Process == nil {
					return false
				}
				for _, ind := range e.Process.SuspiciousIndicators {
					if ind == "possible_cryptominer" {
						return true
					}
				}
				return false
			},
			Actions: []string{"Terminate pod", "Investigate container image", "Review deployment source"},
		},
		{
			ID:          "APSS-003",
			Name:        "Sensitive File Modified",
			Description: "Critical system file was modified",
			Severity:    "HIGH",
			MitreTactic: "Persistence",
			MitreID:     "T1546",
			Condition: func(e *types.SecurityEvent) bool {
				if e.File == nil {
					return false
				}
				criticalPaths := []string{"/etc/passwd", "/etc/shadow", "/etc/sudoers"}
				for _, p := range criticalPaths {
					if e.File.Path == p && e.File.Operation == "modify" {
						return true
					}
				}
				return false
			},
			Actions: []string{"Review file changes", "Check for privilege escalation", "Audit container"},
		},
		{
			ID:          "APSS-004",
			Name:        "Shell Spawned in Container",
			Description: "Interactive shell was spawned inside container",
			Severity:    "MEDIUM",
			MitreTactic: "Execution",
			MitreID:     "T1059",
			Condition: func(e *types.SecurityEvent) bool {
				if e.Process == nil {
					return false
				}
				for _, ind := range e.Process.SuspiciousIndicators {
					if ind == "shell_spawn" {
						return true
					}
				}
				return false
			},
			Actions: []string{"Verify if expected (kubectl exec)", "Review user activity", "Check for lateral movement"},
		},
		{
			ID:          "APSS-005",
			Name:        "External Database Connection",
			Description: "Connection to external database detected",
			Severity:    "MEDIUM",
			MitreTactic: "Exfiltration",
			MitreID:     "T1048",
			Condition: func(e *types.SecurityEvent) bool {
				if e.Network == nil {
					return false
				}
				dbPorts := map[int]bool{3306: true, 5432: true, 27017: true, 6379: true, 9200: true}
				return e.Network.IsExternal && dbPorts[e.Network.DstPort]
			},
			Actions: []string{"Verify database connection is authorized", "Review network policies", "Check for data exfiltration"},
		},
	}
}
