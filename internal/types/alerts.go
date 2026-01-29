package types

import "time"

// Alert is a generated security alert from the detection engine.
type Alert struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"`
	RuleID      string    `json:"rule_id"`
	RuleName    string    `json:"rule_name"`
	Description string    `json:"description"`
	EventIDs    []string  `json:"event_ids"`
	PodName     string    `json:"pod_name"`
	PodNS       string    `json:"pod_namespace"`
	MitreTactic string    `json:"mitre_tactic,omitempty"`
	MitreID     string    `json:"mitre_id,omitempty"`
	Actions     []string  `json:"recommended_actions"`
}

// AgentInfo tracks a connected agent for the controller.
type AgentInfo struct {
	ID           string    `json:"id"`
	PodName      string    `json:"pod_name"`
	PodNamespace string    `json:"pod_namespace"`
	ConnectedAt  time.Time `json:"connected_at"`
	LastSeen     time.Time `json:"last_seen"`
	EventCount   int64     `json:"event_count"`
}
