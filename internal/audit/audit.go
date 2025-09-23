package audit

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditLogger handles raw log preservation with sharding
type AuditLogger struct {
	baseDir        string
	currentShard   *ShardWriter
	shardSize      int64
	maxShardSize   int64
	rotationMutex  sync.Mutex
	flushInterval  time.Duration
	compressShards bool
}

// ShardWriter represents a single audit shard file
type ShardWriter struct {
	file       *os.File
	writer     *bufio.Writer
	gzWriter   *gzip.Writer
	size       int64
	path       string
	startTime  time.Time
	compressed bool
}

// ShardInfo contains metadata about a shard
type ShardInfo struct {
	Path       string    `json:"path"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Size       int64     `json:"size"`
	LineCount  int64     `json:"line_count"`
	Compressed bool      `json:"compressed"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(baseDir string, maxShardSize int64, compress bool) (*AuditLogger, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit directory: %w", err)
	}

	logger := &AuditLogger{
		baseDir:        baseDir,
		maxShardSize:   maxShardSize,
		flushInterval:  5 * time.Second,
		compressShards: compress,
	}

	// Create initial shard
	if err := logger.rotateShard(); err != nil {
		return nil, fmt.Errorf("failed to create initial shard: %w", err)
	}

	// Start background flush routine
	go logger.backgroundFlush()

	return logger, nil
}

// WriteRawLine writes a raw JSON line to the current shard
func (a *AuditLogger) WriteRawLine(line []byte) error {
	a.rotationMutex.Lock()
	defer a.rotationMutex.Unlock()

	// Check if rotation is needed
	if a.currentShard.size >= a.maxShardSize {
		if err := a.rotateShard(); err != nil {
			return fmt.Errorf("failed to rotate shard: %w", err)
		}
	}

	// Write line with newline if not present
	if len(line) > 0 && line[len(line)-1] != '\n' {
		line = append(line, '\n')
	}

	var n int
	var err error
	if a.currentShard.compressed {
		n, err = a.currentShard.gzWriter.Write(line)
	} else {
		n, err = a.currentShard.writer.Write(line)
	}

	if err != nil {
		return fmt.Errorf("failed to write to shard: %w", err)
	}

	a.currentShard.size += int64(n)
	return nil
}

// WriteEvent writes a structured event to the audit log
func (a *AuditLogger) WriteEvent(event map[string]interface{}) error {
	// Add audit metadata
	event["_audit_timestamp"] = time.Now().Unix()
	event["_audit_shard"] = filepath.Base(a.currentShard.path)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return a.WriteRawLine(data)
}

// rotateShard creates a new shard and closes the current one
func (a *AuditLogger) rotateShard() error {
	// Close current shard if exists
	if a.currentShard != nil {
		if err := a.currentShard.Close(); err != nil {
			return fmt.Errorf("failed to close current shard: %w", err)
		}
	}

	// Generate new shard filename
	timestamp := time.Now().Format("20060102_150405")
	ext := ".jsonl"
	if a.compressShards {
		ext = ".jsonl.gz"
	}
	shardPath := filepath.Join(a.baseDir, fmt.Sprintf("shard_%s%s", timestamp, ext))

	// Create new shard file
	file, err := os.Create(shardPath)
	if err != nil {
		return fmt.Errorf("failed to create shard file: %w", err)
	}

	shard := &ShardWriter{
		file:       file,
		path:       shardPath,
		startTime:  time.Now(),
		compressed: a.compressShards,
	}

	if a.compressShards {
		shard.gzWriter = gzip.NewWriter(file)
		shard.writer = bufio.NewWriterSize(shard.gzWriter, 64*1024)
	} else {
		shard.writer = bufio.NewWriterSize(file, 64*1024)
	}

	a.currentShard = shard
	return nil
}

// backgroundFlush periodically flushes the current shard
func (a *AuditLogger) backgroundFlush() {
	ticker := time.NewTicker(a.flushInterval)
	defer ticker.Stop()

	for range ticker.C {
		a.rotationMutex.Lock()
		if a.currentShard != nil {
			a.currentShard.Flush()
		}
		a.rotationMutex.Unlock()
	}
}

// GetActiveShards returns information about active shards
func (a *AuditLogger) GetActiveShards() ([]ShardInfo, error) {
	pattern := filepath.Join(a.baseDir, "shard_*.jsonl*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var shards []ShardInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		shard := ShardInfo{
			Path:       file,
			StartTime:  info.ModTime(),
			Size:       info.Size(),
			Compressed: filepath.Ext(file) == ".gz",
		}
		shards = append(shards, shard)
	}

	return shards, nil
}

// Close closes the audit logger
func (a *AuditLogger) Close() error {
	a.rotationMutex.Lock()
	defer a.rotationMutex.Unlock()

	if a.currentShard != nil {
		return a.currentShard.Close()
	}
	return nil
}

// Flush forces a flush of the current shard
func (s *ShardWriter) Flush() error {
	if err := s.writer.Flush(); err != nil {
		return err
	}
	if s.compressed && s.gzWriter != nil {
		return s.gzWriter.Flush()
	}
	return nil
}

// Close closes the shard writer
func (s *ShardWriter) Close() error {
	if err := s.Flush(); err != nil {
		return err
	}
	if s.compressed && s.gzWriter != nil {
		if err := s.gzWriter.Close(); err != nil {
			return err
		}
	}
	return s.file.Close()
}