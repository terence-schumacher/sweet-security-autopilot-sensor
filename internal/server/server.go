// Package server provides the HTTP server and API handlers for the APSS controller.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/internal/config"
	"github.com/invisible-tech/autopilot-security-sensor/internal/controller"
	"github.com/invisible-tech/autopilot-security-sensor/internal/types"
	"github.com/invisible-tech/autopilot-security-sensor/internal/version"
)

// Server is the HTTP server for the controller API.
type Server struct {
	cfg        config.ControllerConfig
	controller *controller.Controller
	log        *logrus.Logger
	httpServer *http.Server
}

// New creates a new HTTP server that uses the given controller.
func New(cfg config.ControllerConfig, ctrl *controller.Controller, log *logrus.Logger) *Server {
	mux := http.NewServeMux()
	s := &Server{cfg: cfg, controller: ctrl, log: log}
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/v1/events", s.handleEvents)
	mux.HandleFunc("/api/v1/agents", s.handleAgents)
	mux.HandleFunc("/api/v1/alerts", s.handleAlerts)
	mux.Handle("/metrics", promhttp.Handler())

	s.httpServer = &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

// ListenAndServe starts the HTTP server. It blocks until the server is closed.
func (s *Server) ListenAndServe() error {
	s.log.WithField("addr", s.cfg.HTTPAddr).Info("Controller listening")
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"version": version.Version,
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var event types.SecurityEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := s.controller.IngestEvent(r.Context(), &event); err != nil {
		http.Error(w, "Event buffer full", http.StatusServiceUnavailable)
		return
	}
	if event.Severity == "CRITICAL" || event.Severity == "HIGH" {
		s.controller.SendHighSeverityEvent(r.Context(), &event)
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	agents := s.controller.GetAgents()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := s.controller.GetAlerts(100)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}
