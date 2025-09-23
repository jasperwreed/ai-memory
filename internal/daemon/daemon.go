package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/jasperwreed/ai-memory/internal/audit"
	"github.com/jasperwreed/ai-memory/internal/watcher"
)

// CaptureConfig holds configuration for the capture daemon
type CaptureConfig struct {
	AuditDir       string   `json:"audit_dir"`
	MaxShardSize   int64    `json:"max_shard_size"`
	CompressShards bool     `json:"compress_shards"`
	WatchDirs      []string `json:"watch_dirs"`
	BatchSize      int      `json:"batch_size"`
	FlushInterval  string   `json:"flush_interval"`
	EnableMetrics  bool     `json:"enable_metrics"`
}

// DefaultConfig returns default daemon configuration
func DefaultConfig() *CaptureConfig {
	home, _ := os.UserHomeDir()
	return &CaptureConfig{
		AuditDir:       filepath.Join(home, ".ai-memory", "audit"),
		MaxShardSize:   100 * 1024 * 1024, // 100MB shards
		CompressShards: true,
		WatchDirs: []string{
			"~/.claude-code/sessions",
			"~/.aider",
			"~/.cursor/sessions",
		},
		BatchSize:      100,
		FlushInterval:  "5s",
		EnableMetrics:  true,
	}
}

// CaptureDaemon manages the auto-capture process
type CaptureDaemon struct {
	config       *CaptureConfig
	watcher      *watcher.SessionWatcher
	auditLogger  *audit.AuditLogger
	eventQueue   chan watcher.Event
	metrics      *Metrics
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	pidFile      string
	statusFile   string
}

// Metrics tracks daemon performance
type Metrics struct {
	EventsReceived  int64     `json:"events_received"`
	EventsProcessed int64     `json:"events_processed"`
	EventsDropped   int64     `json:"events_dropped"`
	BytesWritten    int64     `json:"bytes_written"`
	ActiveSessions  int       `json:"active_sessions"`
	StartTime       time.Time `json:"start_time"`
	LastEventTime   time.Time `json:"last_event_time"`
	mu              sync.RWMutex
}

// NewCaptureDaemon creates a new capture daemon
func NewCaptureDaemon(config *CaptureConfig) (*CaptureDaemon, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create audit logger
	auditLogger, err := audit.NewAuditLogger(
		config.AuditDir,
		config.MaxShardSize,
		config.CompressShards,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	// Create session watcher
	sessionWatcher, err := watcher.NewSessionWatcher()
	if err != nil {
		auditLogger.Close()
		return nil, fmt.Errorf("failed to create session watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	daemon := &CaptureDaemon{
		config:      config,
		watcher:     sessionWatcher,
		auditLogger: auditLogger,
		eventQueue:  make(chan watcher.Event, 1000),
		metrics: &Metrics{
			StartTime: time.Now(),
		},
		ctx:        ctx,
		cancel:     cancel,
		pidFile:    filepath.Join(config.AuditDir, "daemon.pid"),
		statusFile: filepath.Join(config.AuditDir, "daemon.status"),
	}

	// Register event handler
	sessionWatcher.AddHandler(daemon.handleEvent)

	return daemon, nil
}

// Start starts the capture daemon
func (d *CaptureDaemon) Start() error {
	// Check if already running
	if d.isRunning() {
		return fmt.Errorf("daemon already running")
	}

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Start watching directories
	for _, dir := range d.config.WatchDirs {
		if err := d.watcher.WatchDirectory(dir, "*.jsonl"); err != nil {
			// Log error but continue with other directories
			fmt.Fprintf(os.Stderr, "Failed to watch %s: %v\n", dir, err)
		}
	}

	// Start watcher
	if err := d.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// Start processing workers
	workerCount := 3
	for i := 0; i < workerCount; i++ {
		d.wg.Add(1)
		go d.processEvents()
	}

	// Start metrics updater
	if d.config.EnableMetrics {
		d.wg.Add(1)
		go d.updateMetrics()
	}

	// Start status writer
	d.wg.Add(1)
	go d.writeStatus()

	return nil
}

// Stop stops the capture daemon
func (d *CaptureDaemon) Stop() error {
	// Signal shutdown
	d.cancel()

	// Stop watcher
	if err := d.watcher.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Error stopping watcher: %v\n", err)
	}

	// Close event queue
	close(d.eventQueue)

	// Wait for workers to finish
	d.wg.Wait()

	// Close audit logger
	if err := d.auditLogger.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing audit logger: %v\n", err)
	}

	// Remove PID file
	os.Remove(d.pidFile)
	os.Remove(d.statusFile)

	return nil
}

// Run runs the daemon until interrupted
func (d *CaptureDaemon) Run() error {
	if err := d.Start(); err != nil {
		return err
	}

	// Setup signal handlers
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-d.ctx.Done():
		// Context cancelled
	case sig := <-sigCh:
		fmt.Printf("Received signal %v, shutting down...\n", sig)
	}

	return d.Stop()
}

// handleEvent handles events from the watcher
func (d *CaptureDaemon) handleEvent(event watcher.Event) error {
	d.metrics.mu.Lock()
	d.metrics.EventsReceived++
	d.metrics.LastEventTime = time.Now()
	d.metrics.mu.Unlock()

	select {
	case d.eventQueue <- event:
		return nil
	case <-time.After(100 * time.Millisecond):
		// Queue full, drop event
		d.metrics.mu.Lock()
		d.metrics.EventsDropped++
		d.metrics.mu.Unlock()
		return fmt.Errorf("event queue full")
	}
}

// processEvents processes events from the queue
func (d *CaptureDaemon) processEvents() {
	defer d.wg.Done()

	batch := make([]watcher.Event, 0, d.config.BatchSize)
	flushInterval, _ := time.ParseDuration(d.config.FlushInterval)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			// Flush remaining batch
			d.flushBatch(batch)
			return

		case event, ok := <-d.eventQueue:
			if !ok {
				// Queue closed, flush and exit
				d.flushBatch(batch)
				return
			}

			batch = append(batch, event)
			if len(batch) >= d.config.BatchSize {
				d.flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				d.flushBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// flushBatch writes a batch of events to the audit log
func (d *CaptureDaemon) flushBatch(batch []watcher.Event) {
	for _, event := range batch {
		// Create audit event
		auditEvent := map[string]interface{}{
			"type":      event.Type,
			"tool":      event.Tool,
			"session":   event.SessionID,
			"path":      event.Path,
			"timestamp": event.Timestamp.Unix(),
		}

		// Add raw line if available
		if len(event.RawLine) > 0 {
			auditEvent["raw"] = string(event.RawLine)

			// Also write raw line to preserve original
			if err := d.auditLogger.WriteRawLine(event.RawLine); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write raw line: %v\n", err)
			}
		}

		// Add parsed data if available
		if event.Data != nil {
			auditEvent["data"] = event.Data
		}

		// Write to audit log
		if err := d.auditLogger.WriteEvent(auditEvent); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write audit event: %v\n", err)
			continue
		}

		d.metrics.mu.Lock()
		d.metrics.EventsProcessed++
		d.metrics.BytesWritten += int64(len(event.RawLine))
		d.metrics.mu.Unlock()
	}
}

// updateMetrics updates active session count
func (d *CaptureDaemon) updateMetrics() {
	defer d.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			sessions := d.watcher.GetActiveSessions()
			d.metrics.mu.Lock()
			d.metrics.ActiveSessions = len(sessions)
			d.metrics.mu.Unlock()
		}
	}
}

// writeStatus writes daemon status to file
func (d *CaptureDaemon) writeStatus() {
	defer d.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.writeStatusFile()
		}
	}
}

// writeStatusFile writes current status to file
func (d *CaptureDaemon) writeStatusFile() {
	d.metrics.mu.RLock()
	status := struct {
		PID           int       `json:"pid"`
		Status        string    `json:"status"`
		Config        *CaptureConfig `json:"config"`
		Metrics       *Metrics  `json:"metrics"`
		ActiveShards  []audit.ShardInfo `json:"active_shards"`
		UpdatedAt     time.Time `json:"updated_at"`
	}{
		PID:       os.Getpid(),
		Status:    "running",
		Config:    d.config,
		Metrics:   d.metrics,
		UpdatedAt: time.Now(),
	}
	d.metrics.mu.RUnlock()

	// Get active shards
	if shards, err := d.auditLogger.GetActiveShards(); err == nil {
		status.ActiveShards = shards
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return
	}

	// Write atomically
	tmpFile := d.statusFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return
	}
	os.Rename(tmpFile, d.statusFile)
}

// writePIDFile writes the process ID to file
func (d *CaptureDaemon) writePIDFile() error {
	pid := os.Getpid()
	return os.WriteFile(d.pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// isRunning checks if daemon is already running
func (d *CaptureDaemon) isRunning() bool {
	data, err := os.ReadFile(d.pidFile)
	if err != nil {
		return false
	}

	var pid int
	fmt.Sscanf(string(data), "%d", &pid)

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process is alive
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// GetStatus reads daemon status from file
func GetStatus(auditDir string) (*DaemonStatus, error) {
	statusFile := filepath.Join(auditDir, "daemon.status")
	data, err := os.ReadFile(statusFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &DaemonStatus{Status: "stopped"}, nil
		}
		return nil, err
	}

	var status DaemonStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// DaemonStatus represents the daemon status
type DaemonStatus struct {
	PID       int                `json:"pid"`
	Status    string             `json:"status"`
	Config    *CaptureConfig     `json:"config"`
	Metrics   *Metrics           `json:"metrics"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// StreamAuditLogs streams audit logs from shards
func StreamAuditLogs(auditDir string, follow bool, handler func([]byte) error) error {
	if follow {
		// Create iterator and follow new entries
		return followAuditLogs(auditDir, handler)
	}

	// Read all existing logs
	iterator, err := audit.NewShardIterator(auditDir)
	if err != nil {
		return err
	}
	defer iterator.Close()

	for {
		line, err := iterator.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := handler(line); err != nil {
			return err
		}
	}

	return nil
}

// followAuditLogs follows audit logs as they're written
func followAuditLogs(auditDir string, handler func([]byte) error) error {
	// Watch for new shards
	sessionWatcher, err := watcher.NewSessionWatcher()
	if err != nil {
		return err
	}
	defer sessionWatcher.Stop()

	// Handler for new events
	sessionWatcher.AddHandler(func(event watcher.Event) error {
		if len(event.RawLine) > 0 {
			return handler(event.RawLine)
		}
		return nil
	})

	// Start watching audit directory
	if err := sessionWatcher.WatchDirectory(auditDir, "shard_*.jsonl*"); err != nil {
		return err
	}

	if err := sessionWatcher.Start(); err != nil {
		return err
	}

	// Keep running until interrupted
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	return nil
}