package netpolicy

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/invisible-tech/autopilot-security-sensor/pkg/collector"
)

// Config for network monitoring
type Config struct {
	ScanInterval    time.Duration
	SuspiciousPorts []int
	EventChan       chan<- collector.SecurityEvent
}

// Connection represents a network connection
type Connection struct {
	Protocol  string
	LocalIP   net.IP
	LocalPort int
	RemoteIP  net.IP
	RemotePort int
	State      string
	Inode      uint64
	UID        int
}

// NetworkMonitor monitors network connections within the container
type NetworkMonitor struct {
	cfg Config
	log *logrus.Logger

	// Track known connections
	knownConns map[string]*Connection
	mu         sync.RWMutex

	// Suspicious ports as a set for fast lookup
	suspiciousPorts map[int]bool

	// Private IP ranges
	privateRanges []*net.IPNet
}

// New creates a new NetworkMonitor
func New(cfg Config, log *logrus.Logger) *NetworkMonitor {
	nm := &NetworkMonitor{
		cfg:             cfg,
		log:             log,
		knownConns:      make(map[string]*Connection),
		suspiciousPorts: make(map[int]bool),
	}

	for _, port := range cfg.SuspiciousPorts {
		nm.suspiciousPorts[port] = true
	}

	// Initialize private IP ranges
	privateRangeStrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16", // Link-local
	}
	for _, cidr := range privateRangeStrs {
		_, ipnet, _ := net.ParseCIDR(cidr)
		nm.privateRanges = append(nm.privateRanges, ipnet)
	}

	return nm
}

// Start begins network monitoring
func (nm *NetworkMonitor) Start(ctx context.Context) {
	nm.log.Info("Starting network monitor")

	ticker := time.NewTicker(nm.cfg.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			nm.log.Info("Network monitor stopping")
			return
		case <-ticker.C:
			nm.scanConnections(ctx)
		}
	}
}

// scanConnections reads /proc/net/tcp and /proc/net/udp
func (nm *NetworkMonitor) scanConnections(ctx context.Context) {
	currentConns := make(map[string]bool)

	// Scan TCP connections
	tcpConns, err := nm.parseNetFile("/proc/net/tcp", "tcp")
	if err != nil {
		nm.log.WithError(err).Debug("Failed to read /proc/net/tcp")
	}

	// Scan TCP6 connections
	tcp6Conns, err := nm.parseNetFile("/proc/net/tcp6", "tcp6")
	if err != nil {
		nm.log.WithError(err).Debug("Failed to read /proc/net/tcp6")
	}

	// Scan UDP connections
	udpConns, err := nm.parseNetFile("/proc/net/udp", "udp")
	if err != nil {
		nm.log.WithError(err).Debug("Failed to read /proc/net/udp")
	}

	allConns := append(tcpConns, tcp6Conns...)
	allConns = append(allConns, udpConns...)

	for _, conn := range allConns {
		key := nm.connectionKey(conn)
		currentConns[key] = true

		nm.mu.RLock()
		_, exists := nm.knownConns[key]
		nm.mu.RUnlock()

		if !exists {
			nm.mu.Lock()
			nm.knownConns[key] = conn
			nm.mu.Unlock()

			nm.analyzeConnection(ctx, conn)
		}
	}

	// Clean up closed connections
	nm.mu.Lock()
	for key := range nm.knownConns {
		if !currentConns[key] {
			delete(nm.knownConns, key)
		}
	}
	nm.mu.Unlock()
}

// parseNetFile parses /proc/net/tcp or /proc/net/udp
func (nm *NetworkMonitor) parseNetFile(path, protocol string) ([]*Connection, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var conns []*Connection
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum == 1 {
			continue // Skip header
		}

		conn, err := nm.parseLine(scanner.Text(), protocol)
		if err != nil {
			continue
		}
		conns = append(conns, conn)
	}

	return conns, scanner.Err()
}

// parseLine parses a single line from /proc/net/tcp or udp
func (nm *NetworkMonitor) parseLine(line, protocol string) (*Connection, error) {
	fields := strings.Fields(line)
	if len(fields) < 12 {
		return nil, fmt.Errorf("invalid line format")
	}

	localIP, localPort, err := nm.parseAddress(fields[1])
	if err != nil {
		return nil, err
	}

	remoteIP, remotePort, err := nm.parseAddress(fields[2])
	if err != nil {
		return nil, err
	}

	state := nm.parseState(fields[3])
	uid, _ := strconv.Atoi(fields[7])
	inode, _ := strconv.ParseUint(fields[9], 10, 64)

	return &Connection{
		Protocol:   protocol,
		LocalIP:    localIP,
		LocalPort:  localPort,
		RemoteIP:   remoteIP,
		RemotePort: remotePort,
		State:      state,
		UID:        uid,
		Inode:      inode,
	}, nil
}

// parseAddress parses an address from hex format (e.g., "0100007F:0050")
func (nm *NetworkMonitor) parseAddress(s string) (net.IP, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, 0, fmt.Errorf("invalid address format")
	}

	ipHex := parts[0]
	var ip net.IP

	if len(ipHex) == 8 {
		// IPv4
		ipBytes, err := hex.DecodeString(ipHex)
		if err != nil {
			return nil, 0, err
		}
		// Reverse byte order (little endian)
		ip = net.IPv4(ipBytes[3], ipBytes[2], ipBytes[1], ipBytes[0])
	} else if len(ipHex) == 32 {
		// IPv6
		ipBytes, err := hex.DecodeString(ipHex)
		if err != nil {
			return nil, 0, err
		}
		// Handle IPv6 byte order
		ip = make(net.IP, 16)
		for i := 0; i < 4; i++ {
			start := i * 4
			binary.LittleEndian.PutUint32(ip[start:start+4], binary.BigEndian.Uint32(ipBytes[start:start+4]))
		}
	}

	port, err := strconv.ParseInt(parts[1], 16, 32)
	if err != nil {
		return nil, 0, err
	}

	return ip, int(port), nil
}

// parseState converts TCP state hex to string
func (nm *NetworkMonitor) parseState(s string) string {
	states := map[string]string{
		"01": "ESTABLISHED",
		"02": "SYN_SENT",
		"03": "SYN_RECV",
		"04": "FIN_WAIT1",
		"05": "FIN_WAIT2",
		"06": "TIME_WAIT",
		"07": "CLOSE",
		"08": "CLOSE_WAIT",
		"09": "LAST_ACK",
		"0A": "LISTEN",
		"0B": "CLOSING",
	}
	if state, ok := states[strings.ToUpper(s)]; ok {
		return state
	}
	return "UNKNOWN"
}

// connectionKey generates a unique key for a connection
func (nm *NetworkMonitor) connectionKey(conn *Connection) string {
	return fmt.Sprintf("%s:%s:%d->%s:%d:%s",
		conn.Protocol,
		conn.LocalIP.String(),
		conn.LocalPort,
		conn.RemoteIP.String(),
		conn.RemotePort,
		conn.State)
}

// analyzeConnection checks if a connection is suspicious
func (nm *NetworkMonitor) analyzeConnection(ctx context.Context, conn *Connection) {
	severity := collector.SeverityInfo
	eventType := collector.EventTypeNetworkConnect

	if conn.State == "LISTEN" {
		eventType = collector.EventTypeNetworkListen
	}

	isExternal := !nm.isPrivateIP(conn.RemoteIP)
	isSuspiciousPort := nm.suspiciousPorts[conn.RemotePort] || nm.suspiciousPorts[conn.LocalPort]

	// Elevate severity based on suspicious indicators
	if conn.State == "ESTABLISHED" && isExternal {
		severity = collector.SeverityLow
	}

	if isSuspiciousPort {
		severity = collector.SeverityHigh
	}

	// Check for potential reverse shell indicators
	if conn.State == "ESTABLISHED" && isExternal && nm.isPotentialReverseShell(conn) {
		severity = collector.SeverityCritical
	}

	// Only emit events for non-trivial connections
	if conn.RemotePort == 0 && conn.RemoteIP.Equal(net.IPv4zero) {
		return // Skip local sockets with no remote
	}

	event := collector.SecurityEvent{
		Type:      eventType,
		Severity:  severity,
		Timestamp: time.Now(),
		Network: &collector.NetworkEvent{
			Protocol:        conn.Protocol,
			SrcIP:           conn.LocalIP.String(),
			SrcPort:         conn.LocalPort,
			DstIP:           conn.RemoteIP.String(),
			DstPort:         conn.RemotePort,
			State:           conn.State,
			IsExternal:      isExternal,
			IsSuspiciousPort: isSuspiciousPort,
		},
	}

	select {
	case nm.cfg.EventChan <- event:
	case <-ctx.Done():
	default:
		nm.log.Debug("Event channel full, dropping network event")
	}
}

// isPrivateIP checks if an IP is in a private range
func (nm *NetworkMonitor) isPrivateIP(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() {
		return true
	}
	for _, ipnet := range nm.privateRanges {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

// isPotentialReverseShell checks connection patterns
func (nm *NetworkMonitor) isPotentialReverseShell(conn *Connection) bool {
	// Common reverse shell ports
	rsShellPorts := []int{4444, 5555, 6666, 1337, 1234, 31337, 9001, 9999}
	for _, port := range rsShellPorts {
		if conn.RemotePort == port || conn.LocalPort == port {
			return true
		}
	}
	return false
}
