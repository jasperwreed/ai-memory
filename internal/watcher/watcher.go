package watcher

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// SessionWatcher watches for AI tool session files and streams updates
type SessionWatcher struct {
	watcher       *fsnotify.Watcher
	watchedPaths  map[string]bool
	activeSessions map[string]*SessionTail
	handlers      []EventHandler
	mu            sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// SessionTail tracks a single session file
type SessionTail struct {
	Path       string
	File       *os.File
	Reader     *bufio.Reader
	LastOffset int64
	SessionID  string
	Tool       string
	StartTime  time.Time
}

// Event represents a captured event from a session file
type Event struct {
	Type      string                 `json:"type"`
	Tool      string                 `json:"tool"`
	SessionID string                 `json:"session_id"`
	Path      string                 `json:"path"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	RawLine   []byte                 `json:"-"`
}

// EventHandler processes captured events
type EventHandler func(event Event) error

// NewSessionWatcher creates a new session watcher
func NewSessionWatcher() (*SessionWatcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fs watcher: %w", err)
	}

	return &SessionWatcher{
		watcher:        fsWatcher,
		watchedPaths:   make(map[string]bool),
		activeSessions: make(map[string]*SessionTail),
		handlers:       []EventHandler{},
		stopCh:         make(chan struct{}),
	}, nil
}

// AddHandler adds an event handler
func (w *SessionWatcher) AddHandler(handler EventHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers = append(w.handlers, handler)
}

// WatchDirectory adds a directory to watch for session files
func (w *SessionWatcher) WatchDirectory(dir string, pattern string) error {
	// Expand home directory
	if dir[:2] == "~/" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, dir[2:])
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Add to watcher if not already watching
	if !w.watchedPaths[dir] {
		if err := w.watcher.Add(dir); err != nil {
			return fmt.Errorf("failed to watch directory %s: %w", dir, err)
		}
		w.watchedPaths[dir] = true
	}

	// Scan for existing session files
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	for _, path := range matches {
		if err := w.startTailing(path); err != nil {
			// Log error but continue with other files
			fmt.Fprintf(os.Stderr, "Failed to tail %s: %v\n", path, err)
		}
	}

	return nil
}

// WatchClaudeCodeSessions watches for Claude Code session files
func (w *SessionWatcher) WatchClaudeCodeSessions() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Claude Code stores sessions in ~/.claude-code/sessions/
	sessionDir := filepath.Join(home, ".claude-code", "sessions")

	// Create directory if it doesn't exist (for new installations)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	return w.WatchDirectory(sessionDir, "*.jsonl")
}

// Start begins watching for file changes
func (w *SessionWatcher) Start() error {
	w.wg.Add(1)
	go w.watchLoop()

	w.wg.Add(1)
	go w.tailLoop()

	return nil
}

// Stop stops the watcher
func (w *SessionWatcher) Stop() error {
	close(w.stopCh)
	w.wg.Wait()

	w.mu.Lock()
	defer w.mu.Unlock()

	// Close all active sessions
	for _, session := range w.activeSessions {
		session.File.Close()
	}

	return w.watcher.Close()
}

// watchLoop monitors file system events
func (w *SessionWatcher) watchLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopCh:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Handle file events
			if event.Op&fsnotify.Write == fsnotify.Write {
				w.handleFileWrite(event.Name)
			} else if event.Op&fsnotify.Create == fsnotify.Create {
				// Check if it's a JSONL file
				if filepath.Ext(event.Name) == ".jsonl" {
					w.startTailing(event.Name)
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				w.stopTailing(event.Name)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}

// tailLoop periodically checks for new data in tracked files
func (w *SessionWatcher) tailLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.checkAllSessions()
		}
	}
}

// handleFileWrite handles write events to tracked files
func (w *SessionWatcher) handleFileWrite(path string) {
	w.mu.RLock()
	session, exists := w.activeSessions[path]
	w.mu.RUnlock()

	if exists {
		w.readNewLines(session)
	}
}

// startTailing begins tailing a session file
func (w *SessionWatcher) startTailing(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if already tailing
	if _, exists := w.activeSessions[path]; exists {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	// Get file info to start from end
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat file: %w", err)
	}

	session := &SessionTail{
		Path:       path,
		File:       file,
		Reader:     bufio.NewReader(file),
		LastOffset: 0, // Start from beginning for existing files
		Tool:       detectToolFromPath(path),
		StartTime:  time.Now(),
	}

	// For new files, start from end
	if info.Size() == 0 {
		session.LastOffset = 0
	} else {
		// Read existing content first
		w.mu.Unlock()
		w.readNewLines(session)
		w.mu.Lock()
		session.LastOffset = info.Size()
	}

	w.activeSessions[path] = session

	// Send session start event
	event := Event{
		Type:      "session_start",
		Tool:      session.Tool,
		Path:      path,
		Timestamp: time.Now(),
	}
	w.notifyHandlers(event)

	return nil
}

// stopTailing stops tailing a session file
func (w *SessionWatcher) stopTailing(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if session, exists := w.activeSessions[path]; exists {
		session.File.Close()
		delete(w.activeSessions, path)

		// Send session end event
		event := Event{
			Type:      "session_end",
			Tool:      session.Tool,
			SessionID: session.SessionID,
			Path:      path,
			Timestamp: time.Now(),
		}
		w.notifyHandlers(event)
	}
}

// checkAllSessions checks all active sessions for new data
func (w *SessionWatcher) checkAllSessions() {
	w.mu.RLock()
	sessions := make([]*SessionTail, 0, len(w.activeSessions))
	for _, session := range w.activeSessions {
		sessions = append(sessions, session)
	}
	w.mu.RUnlock()

	for _, session := range sessions {
		w.readNewLines(session)
	}
}

// readNewLines reads new lines from a session file
func (w *SessionWatcher) readNewLines(session *SessionTail) {
	// Seek to last known position
	session.File.Seek(session.LastOffset, 0)
	session.Reader.Reset(session.File)

	for {
		line, err := session.Reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", session.Path, err)
			}
			break
		}

		// Update offset
		session.LastOffset += int64(len(line))

		// Parse and emit event
		event := Event{
			Type:      "message",
			Tool:      session.Tool,
			SessionID: session.SessionID,
			Path:      session.Path,
			Timestamp: time.Now(),
			RawLine:   line,
		}

		// Try to parse as JSON
		var data map[string]interface{}
		if err := json.Unmarshal(line, &data); err == nil {
			event.Data = data

			// Extract session ID if available
			if sid, ok := data["sessionId"].(string); ok && session.SessionID == "" {
				session.SessionID = sid
			}
		}

		w.notifyHandlers(event)
	}
}

// notifyHandlers sends event to all registered handlers
func (w *SessionWatcher) notifyHandlers(event Event) {
	w.mu.RLock()
	handlers := make([]EventHandler, len(w.handlers))
	copy(handlers, w.handlers)
	w.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(event); err != nil {
			fmt.Fprintf(os.Stderr, "Handler error: %v\n", err)
		}
	}
}

// detectToolFromPath tries to detect the tool from file path
func detectToolFromPath(path string) string {
	dir := filepath.Dir(path)

	if filepath.Base(dir) == "sessions" && filepath.Base(filepath.Dir(dir)) == ".claude-code" {
		return "claude-code"
	}

	// Add more detection logic for other tools
	if filepath.Base(dir) == ".aider" {
		return "aider"
	}

	return "unknown"
}

// GetActiveSessions returns information about active sessions
func (w *SessionWatcher) GetActiveSessions() []SessionInfo {
	w.mu.RLock()
	defer w.mu.RUnlock()

	sessions := make([]SessionInfo, 0, len(w.activeSessions))
	for _, session := range w.activeSessions {
		info := SessionInfo{
			Path:      session.Path,
			Tool:      session.Tool,
			SessionID: session.SessionID,
			StartTime: session.StartTime,
			Active:    true,
		}

		// Get file size
		if fi, err := session.File.Stat(); err == nil {
			info.Size = fi.Size()
		}

		sessions = append(sessions, info)
	}

	return sessions
}

// SessionInfo contains information about a session
type SessionInfo struct {
	Path      string    `json:"path"`
	Tool      string    `json:"tool"`
	SessionID string    `json:"session_id"`
	StartTime time.Time `json:"start_time"`
	Size      int64     `json:"size"`
	Active    bool      `json:"active"`
}