# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build
```bash
# Build the mem binary
go build -o mem cmd/mem/main.go

# Build using Taskfile
task build

# Install to system (/usr/local/bin)
./install.sh
# OR
task install
```

### Test
```bash
# Run unit tests only
go test ./...
task test

# Run tests with coverage
go test -v -cover ./...
task test:coverage

# Run integration tests only (requires tag)
go test -v -tags=integration ./test/integration
task test:integration

# Run all tests (unit and integration)
go test -tags=integration ./...
task test:all

# Run specific package tests
go test ./internal/storage
go test ./internal/capture
go test ./internal/cli

# Run a single test by name
go test -run TestFunctionName ./internal/cli
```

### Lint & Format
```bash
# Format code
go fmt ./...

# Run go vet for static analysis
go vet ./...
```

### Dependencies
```bash
# Download dependencies
go mod download

# Update dependencies
go mod tidy
```

## Architecture Overview

This is a Go CLI application for capturing and managing AI conversation history. The project follows a standard Go layout with modular internal packages.

### Core Components

- **cmd/mem/main.go**: Entry point that delegates to the CLI package
- **internal/cli**: Command-line interface using Cobra, handles all user commands (capture, search, list, browse, import, export, delete, stats, scan, daemon)
- **internal/storage**: SQLite persistence layer with FTS5 full-text search capabilities, manages conversations and messages
- **internal/capture**: Conversation parsing logic that detects various AI tool formats (Claude, GPT, Aider, etc.)
  - **claude_code.go**: Specialized parser for Claude Code JSONL session files
- **internal/models**: Core data models (Conversation, Message, Statistics)
- **internal/tui**: Terminal UI implementation using Bubble Tea for interactive browsing
  - **enhanced_browser.go**: Vim-style TUI with command mode
- **internal/scanner**: File system scanning for AI tool session files
  - **claude_scanner.go**: Finds Claude Code project sessions
- **internal/search**: Search functionality leveraging SQLite FTS5
- **internal/daemon**: Background service for watching file changes
- **internal/watcher**: File system monitoring for auto-capture
- **internal/audit**: Audit logging for conversation shards

### Key Design Patterns

- **Single Write / Multiple Read DB Connections**: The SQLiteStore uses separate database connections for writes (single) and reads (pooled) to optimize performance
- **Conversation Detection**: Automatically detects conversation format from input using pattern matching in `patterns.go`
- **Project Context**: Conversations are tagged with tool and project metadata for organization
- **Token Estimation**: Built-in token counting for tracking AI usage
- **Test Organization**: Unit tests live alongside source files (*_test.go), integration tests are in test/integration/ and require build tags

### Database Schema

Uses SQLite with FTS5 extension for full-text search. Main tables:
- `conversations`: Stores conversation metadata with project, tool, and session tracking
- `messages`: Individual messages with role (user/assistant/system) and content
- `conversations_fts`: Full-text search virtual table for fast searching
- `projects`: Project metadata and configuration

The database uses WAL mode and optimized pragmas defined in `internal/storage/config.go` for better concurrency.

### Command Structure

All commands are defined in `internal/cli/` with validation handled in `internal/cli/validation.go`. The root command is in `root.go` and uses Cobra for command parsing. The TUI mode supports vim-style commands (`:search`, `:scan`, `:capture`, etc.) when launched without arguments.

### Testing Strategy

- Unit tests are co-located with source files (e.g., internal/cli/capture_test.go)
- Integration tests require the `integration` build tag and are located in test/integration/
- Test coverage is tracked and can be generated with `go test -cover`