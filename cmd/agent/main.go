package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/pkg/monitor"
)

var (
	version = "0.1.0"
	log     = logrus.New()
)

func main() {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	log.WithFields(logrus.Fields{
		"version":   version,
		"pod":       os.Getenv("POD_NAME"),
		"namespace": os.Getenv("POD_NAMESPACE"),
	}).Info("Starting APSS Sidecar Agent")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize agent configuration
	cfg := &monitor.AgentConfig{
		AgentID:            os.Getenv("AGENT_ID"),
		PodName:            os.Getenv("POD_NAME"),
		PodNamespace:       os.Getenv("POD_NAMESPACE"),
		NodeName:           os.Getenv("NODE_NAME"),
		ControllerEndpoint: getEnv("CONTROLLER_ENDPOINT", "apss-controller.apss-system.svc.cluster.local:8080"),
		
		// Monitoring intervals
		ProcScanInterval: 5 * time.Second,
		NetScanInterval:  10 * time.Second,
		FileScanInterval: 30 * time.Second,
		
		// Paths to monitor for file integrity
		WatchPaths: []string{
			"/etc/passwd",
			"/etc/shadow",
			"/etc/sudoers",
			"/root/.ssh",
			"/etc/crontab",
			"/var/spool/cron",
		},
		
		// Suspicious process patterns
		SuspiciousProcesses: []string{
			"nc", "ncat", "netcat",
			"nmap", "masscan",
			"tcpdump", "wireshark",
			"python -c", "perl -e", "ruby -e",
			"bash -i", "sh -i",
			"xmrig", "minerd", "cpuminer",
			"socat", "curl.*|.*sh", "wget.*|.*sh",
		},
		
		// Suspicious ports
		SuspiciousPorts: []int{
			4444, 5555, 6666, 1337,  // Common reverse shell ports
			3389,                      // RDP
			5900, 5901,               // VNC
			6379,                      // Redis
			27017,                     // MongoDB
		},
	}

	// Create and start the monitor
	mon, err := monitor.New(cfg, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to create monitor")
	}

	// Start monitoring in background
	go func() {
		if err := mon.Start(ctx); err != nil {
			log.WithError(err).Error("Monitor error")
			cancel()
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.WithField("signal", sig.String()).Info("Received shutdown signal")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := mon.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("Error during shutdown")
	}

	log.Info("Agent shutdown complete")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
