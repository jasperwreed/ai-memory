package storage

import (
	"fmt"
	"time"
)

// Config holds database configuration settings
type Config struct {
	Path            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	BusyTimeout     time.Duration
	CacheSizeKB     int
}

// DefaultConfig returns default database configuration
func DefaultConfig() *Config {
	return &Config{
		MaxOpenConns:    5,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		BusyTimeout:     5 * time.Second,
		CacheSizeKB:     64000,
	}
}

// pragmas returns SQLite PRAGMA statements based on configuration
func (c *Config) pragmas() []string {
	return []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA temp_store = memory",
		"PRAGMA mmap_size = 30000000000",
		"PRAGMA busy_timeout = " + formatMilliseconds(c.BusyTimeout),
		"PRAGMA foreign_keys = ON",
		"PRAGMA cache_size = -" + formatInt(c.CacheSizeKB),
	}
}

func formatMilliseconds(d time.Duration) string {
	return formatInt(int(d.Milliseconds()))
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}