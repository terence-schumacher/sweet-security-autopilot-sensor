package fileintegrity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/pkg/collector"
)

// Config for file integrity monitoring
type Config struct {
	WatchPaths []string
	EventChan  chan<- collector.SecurityEvent
}

// FileHash stores the baseline hash of a file
type FileHash struct {
	Path    string
	Hash    string
	Mode    os.FileMode
	ModTime time.Time
	Size    int64
}

// FileMonitor monitors critical files for changes
type FileMonitor struct {
	cfg     Config
	log     *logrus.Logger
	watcher *fsnotify.Watcher

	// Baseline file hashes
	baseline map[string]*FileHash
	mu       sync.RWMutex
}

// New creates a new FileMonitor
func New(cfg Config, log *logrus.Logger) (*FileMonitor, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fm := &FileMonitor{
		cfg:      cfg,
		log:      log,
		watcher:  watcher,
		baseline: make(map[string]*FileHash),
	}

	// Build initial baseline
	for _, path := range cfg.WatchPaths {
		fm.addWatchRecursive(path)
	}

	return fm, nil
}

// addWatchRecursive adds a path and all subdirectories to the watcher
func (fm *FileMonitor) addWatchRecursive(path string) {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		fm.log.WithError(err).WithField("path", path).Debug("Cannot watch path")
		return
	}

	if info.IsDir() {
		// Walk directory and add all subdirectories
		filepath.Walk(path, func(walkPath string, walkInfo os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if walkInfo.IsDir() {
				if err := fm.watcher.Add(walkPath); err != nil {
					fm.log.WithError(err).WithField("path", walkPath).Debug("Failed to add watch")
				}
			} else {
				// Hash the file for baseline
				fm.hashFile(walkPath)
			}
			return nil
		})
	} else {
		// Watch the parent directory for the file
		dir := filepath.Dir(path)
		if err := fm.watcher.Add(dir); err != nil {
			fm.log.WithError(err).WithField("path", dir).Debug("Failed to add watch")
		}
		fm.hashFile(path)
	}
}

// hashFile computes and stores the hash of a file
func (fm *FileMonitor) hashFile(path string) *FileHash {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	// Don't hash directories or special files
	if info.IsDir() || !info.Mode().IsRegular() {
		return nil
	}

	// Skip large files (>10MB) to avoid performance issues
	if info.Size() > 10*1024*1024 {
		fm.log.WithField("path", path).Debug("Skipping large file")
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil
	}

	hash := &FileHash{
		Path:    path,
		Hash:    hex.EncodeToString(hasher.Sum(nil)),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		Size:    info.Size(),
	}

	fm.mu.Lock()
	fm.baseline[path] = hash
	fm.mu.Unlock()

	return hash
}

// Start begins file integrity monitoring
func (fm *FileMonitor) Start(ctx context.Context) {
	fm.log.Info("Starting file integrity monitor")

	for {
		select {
		case <-ctx.Done():
			fm.log.Info("File monitor stopping")
			fm.watcher.Close()
			return

		case event, ok := <-fm.watcher.Events:
			if !ok {
				return
			}
			fm.handleFsEvent(ctx, event)

		case err, ok := <-fm.watcher.Errors:
			if !ok {
				return
			}
			fm.log.WithError(err).Error("Watcher error")
		}
	}
}

// handleFsEvent processes a filesystem event
func (fm *FileMonitor) handleFsEvent(ctx context.Context, event fsnotify.Event) {
	path := event.Name

	// Determine event type
	var eventType collector.EventType
	var operation string
	severity := collector.SeverityMedium

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		eventType = collector.EventTypeFileCreate
		operation = "create"
	case event.Op&fsnotify.Write == fsnotify.Write:
		eventType = collector.EventTypeFileModify
		operation = "modify"
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = collector.EventTypeFileDelete
		operation = "delete"
		severity = collector.SeverityHigh
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = collector.EventTypeFileModify
		operation = "rename"
	case event.Op&fsnotify.Chmod == fsnotify.Chmod:
		eventType = collector.EventTypeFileModify
		operation = "chmod"
	default:
		return // Ignore other events
	}

	// Check severity based on path
	severity = fm.classifySeverity(path, operation, severity)

	// Get old hash if available
	fm.mu.RLock()
	oldHash := fm.baseline[path]
	fm.mu.RUnlock()

	// Compute new hash if file still exists
	var newHash *FileHash
	if event.Op&fsnotify.Remove == 0 {
		newHash = fm.hashFile(path)
	} else {
		// Remove from baseline
		fm.mu.Lock()
		delete(fm.baseline, path)
		fm.mu.Unlock()
	}

	fileEvent := &collector.FileEvent{
		Path:      path,
		Operation: operation,
	}

	if oldHash != nil {
		fileEvent.OldHash = oldHash.Hash
	}
	if newHash != nil {
		fileEvent.NewHash = newHash.Hash
		fileEvent.SizeBytes = newHash.Size
		fileEvent.Permissions = newHash.Mode.String()
	}

	secEvent := collector.SecurityEvent{
		Type:      eventType,
		Severity:  severity,
		Timestamp: time.Now(),
		File:      fileEvent,
		Metadata: map[string]string{
			"fsnotify_op": event.Op.String(),
		},
	}

	select {
	case fm.cfg.EventChan <- secEvent:
	case <-ctx.Done():
	default:
		fm.log.Debug("Event channel full, dropping file event")
	}

	// If a new directory was created, watch it
	if event.Op&fsnotify.Create == fsnotify.Create {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			fm.watcher.Add(path)
		}
	}
}

// classifySeverity determines event severity based on the path
func (fm *FileMonitor) classifySeverity(path, operation string, defaultSeverity collector.Severity) collector.Severity {
	// Critical paths
	criticalPaths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/etc/ssh/sshd_config",
		"/root/.ssh/authorized_keys",
	}

	for _, critical := range criticalPaths {
		if path == critical {
			return collector.SeverityCritical
		}
	}

	// High severity paths
	highPaths := []string{
		"/etc/crontab",
		"/var/spool/cron",
		"/etc/cron.d",
		"/etc/profile",
		"/etc/bashrc",
		"/root/.bashrc",
		"/root/.profile",
	}

	for _, high := range highPaths {
		if path == high || filepath.Dir(path) == high {
			return collector.SeverityHigh
		}
	}

	// Check for suspicious file extensions
	ext := filepath.Ext(path)
	suspiciousExts := []string{".sh", ".py", ".pl", ".rb", ".elf", ".so"}
	for _, sext := range suspiciousExts {
		if ext == sext && operation == "create" {
			return collector.SeverityMedium
		}
	}

	return defaultSeverity
}
