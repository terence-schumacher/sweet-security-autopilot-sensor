package procmon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/pkg/collector"
)

// Config for process monitoring
type Config struct {
	ScanInterval        time.Duration
	SuspiciousProcesses []string
	EventChan           chan<- collector.SecurityEvent
}

// ProcessInfo holds information about a running process
type ProcessInfo struct {
	PID         int
	PPID        int
	Name        string
	Exe         string
	Cmdline     []string
	User        string
	UID         int
	StartTime   time.Time
	CmdlineHash string
}

// ProcessMonitor monitors processes within the container namespace
type ProcessMonitor struct {
	cfg Config
	log *logrus.Logger

	// Track known processes to detect new ones
	knownProcs map[int]*ProcessInfo
	mu         sync.RWMutex

	// Compiled suspicious patterns
	suspiciousPatterns []*regexp.Regexp
}

// New creates a new ProcessMonitor
func New(cfg Config, log *logrus.Logger) *ProcessMonitor {
	pm := &ProcessMonitor{
		cfg:        cfg,
		log:        log,
		knownProcs: make(map[int]*ProcessInfo),
	}

	// Compile suspicious process patterns
	for _, pattern := range cfg.SuspiciousProcesses {
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.WithError(err).WithField("pattern", pattern).Warn("Invalid process pattern")
			continue
		}
		pm.suspiciousPatterns = append(pm.suspiciousPatterns, re)
	}

	return pm
}

// Start begins process monitoring
func (pm *ProcessMonitor) Start(ctx context.Context) {
	pm.log.Info("Starting process monitor")

	// Initial scan
	pm.scanProcesses(ctx)

	ticker := time.NewTicker(pm.cfg.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			pm.log.Info("Process monitor stopping")
			return
		case <-ticker.C:
			pm.scanProcesses(ctx)
		}
	}
}

// scanProcesses scans /proc for all processes
func (pm *ProcessMonitor) scanProcesses(ctx context.Context) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		pm.log.WithError(err).Error("Failed to read /proc")
		return
	}

	currentPids := make(map[int]bool)

	for _, entry := range entries {
		// Skip non-numeric entries (not PIDs)
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		currentPids[pid] = true

		// Check if this is a new process
		pm.mu.RLock()
		_, exists := pm.knownProcs[pid]
		pm.mu.RUnlock()

		if !exists {
			proc, err := pm.getProcessInfo(pid)
			if err != nil {
				continue // Process may have exited
			}

			pm.mu.Lock()
			pm.knownProcs[pid] = proc
			pm.mu.Unlock()

			// Check for suspicious activity and emit event
			pm.analyzeNewProcess(ctx, proc)
		}
	}

	// Detect exited processes
	pm.mu.Lock()
	for pid, proc := range pm.knownProcs {
		if !currentPids[pid] {
			delete(pm.knownProcs, pid)
			pm.emitProcessExit(ctx, proc)
		}
	}
	pm.mu.Unlock()
}

// getProcessInfo reads process information from /proc
func (pm *ProcessMonitor) getProcessInfo(pid int) (*ProcessInfo, error) {
	procPath := fmt.Sprintf("/proc/%d", pid)

	// Read cmdline
	cmdlineBytes, err := os.ReadFile(filepath.Join(procPath, "cmdline"))
	if err != nil {
		return nil, err
	}
	cmdline := strings.Split(strings.TrimRight(string(cmdlineBytes), "\x00"), "\x00")

	// Read exe (symlink to actual executable)
	exe, _ := os.Readlink(filepath.Join(procPath, "exe"))

	// Read stat for process name, ppid, start time
	statBytes, err := os.ReadFile(filepath.Join(procPath, "stat"))
	if err != nil {
		return nil, err
	}
	name, ppid, startTime := parseStatFile(string(statBytes))

	// Read status for UID
	uid := pm.getProcessUID(procPath)

	// Hash the cmdline for comparison
	hash := sha256.Sum256(cmdlineBytes)

	return &ProcessInfo{
		PID:         pid,
		PPID:        ppid,
		Name:        name,
		Exe:         exe,
		Cmdline:     cmdline,
		UID:         uid,
		StartTime:   startTime,
		CmdlineHash: hex.EncodeToString(hash[:8]),
	}, nil
}

// parseStatFile extracts name, ppid, and start time from /proc/[pid]/stat
func parseStatFile(stat string) (name string, ppid int, startTime time.Time) {
	// Format: pid (comm) state ppid ...
	// Find the comm field between parentheses
	start := strings.Index(stat, "(")
	end := strings.LastIndex(stat, ")")
	if start != -1 && end != -1 {
		name = stat[start+1 : end]
		fields := strings.Fields(stat[end+2:])
		if len(fields) >= 2 {
			ppid, _ = strconv.Atoi(fields[1])
		}
		if len(fields) >= 20 {
			// Field 22 is starttime in clock ticks
			ticks, _ := strconv.ParseInt(fields[19], 10, 64)
			// Convert to time (approximate)
			bootTime := getBootTime()
			startTime = bootTime.Add(time.Duration(ticks) * time.Second / 100)
		}
	}
	return
}

// getBootTime returns system boot time
func getBootTime() time.Time {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Now()
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				btime, _ := strconv.ParseInt(fields[1], 10, 64)
				return time.Unix(btime, 0)
			}
		}
	}
	return time.Now()
}

// getProcessUID reads the UID from /proc/[pid]/status
func (pm *ProcessMonitor) getProcessUID(procPath string) int {
	data, err := os.ReadFile(filepath.Join(procPath, "status"))
	if err != nil {
		return -1
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				uid, _ := strconv.Atoi(fields[1])
				return uid
			}
		}
	}
	return -1
}

// analyzeNewProcess checks if a new process is suspicious
func (pm *ProcessMonitor) analyzeNewProcess(ctx context.Context, proc *ProcessInfo) {
	cmdlineStr := strings.Join(proc.Cmdline, " ")
	indicators := []string{}
	severity := collector.SeverityInfo

	// Check against suspicious patterns
	for _, pattern := range pm.suspiciousPatterns {
		if pattern.MatchString(cmdlineStr) || pattern.MatchString(proc.Name) {
			indicators = append(indicators, fmt.Sprintf("matches_pattern:%s", pattern.String()))
			severity = collector.SeverityHigh
		}
	}

	// Check for common attack patterns
	if pm.isReverseShell(cmdlineStr) {
		indicators = append(indicators, "possible_reverse_shell")
		severity = collector.SeverityCritical
	}

	if pm.isCryptoMiner(proc.Name, cmdlineStr) {
		indicators = append(indicators, "possible_cryptominer")
		severity = collector.SeverityCritical
	}

	if pm.isShellSpawn(proc) {
		indicators = append(indicators, "shell_spawn")
		if severity < collector.SeverityMedium {
			severity = collector.SeverityMedium
		}
	}

	// Emit event
	event := collector.SecurityEvent{
		Type:      collector.EventTypeProcessStart,
		Severity:  severity,
		Timestamp: time.Now(),
		Process: &collector.ProcessEvent{
			PID:                  proc.PID,
			PPID:                 proc.PPID,
			Name:                 proc.Name,
			ExePath:             proc.Exe,
			Cmdline:              proc.Cmdline,
			UID:                  proc.UID,
			StartTime:            proc.StartTime,
			SuspiciousIndicators: indicators,
		},
		Metadata: map[string]string{
			"cmdline_hash": proc.CmdlineHash,
		},
	}

	select {
	case pm.cfg.EventChan <- event:
	case <-ctx.Done():
	default:
		pm.log.Warn("Event channel full, dropping process event")
	}
}

// emitProcessExit emits an event when a process exits
func (pm *ProcessMonitor) emitProcessExit(ctx context.Context, proc *ProcessInfo) {
	event := collector.SecurityEvent{
		Type:      collector.EventTypeProcessExit,
		Severity:  collector.SeverityInfo,
		Timestamp: time.Now(),
		Process: &collector.ProcessEvent{
			PID:       proc.PID,
			PPID:      proc.PPID,
			Name:      proc.Name,
			ExePath:  proc.Exe,
			Cmdline:   proc.Cmdline,
			StartTime: proc.StartTime,
		},
	}

	select {
	case pm.cfg.EventChan <- event:
	case <-ctx.Done():
	default:
		// Don't log for exit events, they're less critical
	}
}

// isReverseShell detects common reverse shell patterns
func (pm *ProcessMonitor) isReverseShell(cmdline string) bool {
	patterns := []string{
		`bash\s+-i.*>&\s*/dev/tcp`,
		`nc\s+.*-e\s+/bin/(ba)?sh`,
		`python.*socket.*connect`,
		`perl.*socket.*connect`,
		`ruby.*TCPSocket`,
		`php.*fsockopen`,
		`socat.*exec`,
		`/dev/tcp/`,
		`mkfifo.*nc`,
	}

	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p, cmdline); matched {
			return true
		}
	}
	return false
}

// isCryptoMiner detects common cryptocurrency miners
func (pm *ProcessMonitor) isCryptoMiner(name, cmdline string) bool {
	miners := []string{
		"xmrig", "minerd", "cpuminer", "cgminer", "bfgminer",
		"ethminer", "stratum", "cryptonight", "randomx",
	}

	nameLower := strings.ToLower(name)
	cmdlineLower := strings.ToLower(cmdline)

	for _, miner := range miners {
		if strings.Contains(nameLower, miner) || strings.Contains(cmdlineLower, miner) {
			return true
		}
	}

	// Check for mining pool connections
	poolPatterns := []string{
		`stratum\+tcp://`,
		`pool\..*:\d+`,
		`-o\s+.*pool`,
		`--url.*mining`,
	}
	for _, p := range poolPatterns {
		if matched, _ := regexp.MatchString(p, cmdlineLower); matched {
			return true
		}
	}

	return false
}

// isShellSpawn detects shell spawning (potential breakout attempt)
func (pm *ProcessMonitor) isShellSpawn(proc *ProcessInfo) bool {
	shells := []string{"sh", "bash", "zsh", "fish", "csh", "tcsh", "dash", "ash"}
	for _, shell := range shells {
		if proc.Name == shell {
			// Check if interactive (-i flag or allocated TTY)
			for _, arg := range proc.Cmdline {
				if arg == "-i" || arg == "-il" || arg == "-li" {
					return true
				}
			}
		}
	}
	return false
}
