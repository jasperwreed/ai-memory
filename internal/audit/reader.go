package audit

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ShardReader reads from audit shards
type ShardReader struct {
	file       *os.File
	reader     *bufio.Reader
	gzReader   *gzip.Reader
	compressed bool
}

// OpenShard opens a shard for reading
func OpenShard(path string) (*ShardReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open shard: %w", err)
	}

	reader := &ShardReader{
		file:       file,
		compressed: filepath.Ext(path) == ".gz",
	}

	if reader.compressed {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		reader.gzReader = gzReader
		reader.reader = bufio.NewReader(gzReader)
	} else {
		reader.reader = bufio.NewReader(file)
	}

	return reader, nil
}

// ReadLine reads a single line from the shard
func (r *ShardReader) ReadLine() ([]byte, error) {
	return r.reader.ReadBytes('\n')
}

// Close closes the shard reader
func (r *ShardReader) Close() error {
	if r.compressed && r.gzReader != nil {
		r.gzReader.Close()
	}
	return r.file.Close()
}

// ShardIterator provides sequential access to all shards
type ShardIterator struct {
	shards       []string
	currentIndex int
	currentShard *ShardReader
}

// NewShardIterator creates a new iterator over audit shards
func NewShardIterator(baseDir string) (*ShardIterator, error) {
	pattern := filepath.Join(baseDir, "shard_*.jsonl*")
	shards, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Sort shards by name (which includes timestamp)
	sort.Strings(shards)

	return &ShardIterator{
		shards:       shards,
		currentIndex: -1,
	}, nil
}

// Next reads the next line across all shards
func (it *ShardIterator) Next() ([]byte, error) {
	for {
		// Try to read from current shard
		if it.currentShard != nil {
			line, err := it.currentShard.ReadLine()
			if err == nil {
				return line, nil
			}
			if err != io.EOF {
				return nil, err
			}
			// EOF reached, close current shard
			it.currentShard.Close()
			it.currentShard = nil
		}

		// Move to next shard
		it.currentIndex++
		if it.currentIndex >= len(it.shards) {
			return nil, io.EOF
		}

		// Open next shard
		shard, err := OpenShard(it.shards[it.currentIndex])
		if err != nil {
			return nil, err
		}
		it.currentShard = shard
	}
}

// Close closes the iterator
func (it *ShardIterator) Close() error {
	if it.currentShard != nil {
		return it.currentShard.Close()
	}
	return nil
}

// Streamer provides real-time streaming of audit logs
type Streamer struct {
	baseDir      string
	tailInterval time.Duration
	stopCh       chan struct{}
}

// NewStreamer creates a new audit log streamer
func NewStreamer(baseDir string) *Streamer {
	return &Streamer{
		baseDir:      baseDir,
		tailInterval: 500 * time.Millisecond,
		stopCh:       make(chan struct{}),
	}
}

// Stream streams audit logs to the provided handler
func (s *Streamer) Stream(handler func([]byte) error, follow bool) error {
	// First, read all existing logs
	iterator, err := NewShardIterator(s.baseDir)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iterator.Close()

	for {
		line, err := iterator.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read line: %w", err)
		}

		if err := handler(line); err != nil {
			return fmt.Errorf("handler error: %w", err)
		}
	}

	// If not following, we're done
	if !follow {
		return nil
	}

	// Follow mode: watch for new entries
	return s.followNewEntries(handler)
}

// followNewEntries watches for new audit log entries
func (s *Streamer) followNewEntries(handler func([]byte) error) error {
	// Get the latest shard
	latestShard, err := s.findLatestShard()
	if err != nil {
		return fmt.Errorf("failed to find latest shard: %w", err)
	}

	if latestShard == "" {
		// No shards exist yet, wait for one to be created
		return s.waitForFirstShard(handler)
	}

	// Open the latest shard for tailing
	return s.tailShard(latestShard, handler)
}

// findLatestShard finds the most recent shard file
func (s *Streamer) findLatestShard() (string, error) {
	pattern := filepath.Join(s.baseDir, "shard_*.jsonl*")
	shards, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	if len(shards) == 0 {
		return "", nil
	}

	// Sort and get the latest
	sort.Strings(shards)
	return shards[len(shards)-1], nil
}

// tailShard tails a specific shard file
func (s *Streamer) tailShard(shardPath string, handler func([]byte) error) error {
	reader, err := OpenShard(shardPath)
	if err != nil {
		return fmt.Errorf("failed to open shard for tailing: %w", err)
	}
	defer reader.Close()

	ticker := time.NewTicker(s.tailInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return nil
		case <-ticker.C:
			// Check for new lines
			for {
				line, err := reader.ReadLine()
				if err == io.EOF {
					// Check if a new shard has been created
					latestShard, _ := s.findLatestShard()
					if latestShard != shardPath {
						// New shard created, switch to it
						reader.Close()
						return s.tailShard(latestShard, handler)
					}
					break
				}
				if err != nil {
					return fmt.Errorf("failed to read line: %w", err)
				}

				if err := handler(line); err != nil {
					return fmt.Errorf("handler error: %w", err)
				}
			}
		}
	}
}

// waitForFirstShard waits for the first shard to be created
func (s *Streamer) waitForFirstShard(handler func([]byte) error) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return nil
		case <-ticker.C:
			shard, err := s.findLatestShard()
			if err != nil {
				return err
			}
			if shard != "" {
				return s.tailShard(shard, handler)
			}
		}
	}
}

// Stop stops the streamer
func (s *Streamer) Stop() {
	close(s.stopCh)
}

// StreamFilter represents a filter for streaming logs
type StreamFilter struct {
	SessionID string
	Tool      string
	StartTime time.Time
	EndTime   time.Time
}

// FilteredStream streams audit logs with filtering
func (s *Streamer) FilteredStream(filter StreamFilter, handler func([]byte) error, follow bool) error {
	// Wrap the handler with filtering logic
	filteredHandler := func(line []byte) error {
		// Parse the line to check if it matches the filter
		var event map[string]interface{}
		if err := json.Unmarshal(line, &event); err != nil {
			// If we can't parse it, let it through
			return handler(line)
		}

		// Apply filters
		if filter.SessionID != "" {
			if sid, ok := event["session"].(string); ok && sid != filter.SessionID {
				return nil // Skip this line
			}
		}

		if filter.Tool != "" {
			if tool, ok := event["tool"].(string); ok && tool != filter.Tool {
				return nil // Skip this line
			}
		}

		if !filter.StartTime.IsZero() {
			if ts, ok := event["timestamp"].(float64); ok {
				eventTime := time.Unix(int64(ts), 0)
				if eventTime.Before(filter.StartTime) {
					return nil // Skip this line
				}
			}
		}

		if !filter.EndTime.IsZero() {
			if ts, ok := event["timestamp"].(float64); ok {
				eventTime := time.Unix(int64(ts), 0)
				if eventTime.After(filter.EndTime) {
					return nil // Skip this line
				}
			}
		}

		return handler(line)
	}

	return s.Stream(filteredHandler, follow)
}