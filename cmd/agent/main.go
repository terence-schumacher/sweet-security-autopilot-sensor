package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/internal/config"
	"github.com/invisible-tech/autopilot-security-sensor/internal/version"
	"github.com/invisible-tech/autopilot-security-sensor/pkg/monitor"
)

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	log.WithFields(logrus.Fields{
		"version":   version.Version,
		"pod":       os.Getenv("POD_NAME"),
		"namespace": os.Getenv("POD_NAMESPACE"),
	}).Info("Starting APSS Sidecar Agent")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cfg := config.DefaultAgentConfig()
	monCfg := &monitor.AgentConfig{
		AgentID:             cfg.AgentID,
		PodName:             cfg.PodName,
		PodNamespace:        cfg.PodNamespace,
		NodeName:            cfg.NodeName,
		ControllerEndpoint:  cfg.ControllerEndpoint,
		ProcScanInterval:    cfg.ProcScanInterval,
		NetScanInterval:     cfg.NetScanInterval,
		FileScanInterval:    cfg.FileScanInterval,
		WatchPaths:          cfg.WatchPaths,
		SuspiciousProcesses: cfg.SuspiciousProcesses,
		SuspiciousPorts:     cfg.SuspiciousPorts,
	}

	mon, err := monitor.New(monCfg, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to create monitor")
	}

	go func() {
		if err := mon.Start(ctx); err != nil {
			log.WithError(err).Error("Monitor error")
			cancel()
		}
	}()

	sig := <-sigChan
	log.WithField("signal", sig.String()).Info("Received shutdown signal")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := mon.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("Error during shutdown")
	}

	log.Info("Agent shutdown complete")
}
