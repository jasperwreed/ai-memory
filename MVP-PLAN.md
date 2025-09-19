# AI Memory MVP Implementation Plan

## Technology Stack Decision: Go

Based on research of successful CLI tools (ripgrep, fzf, bat, exa, docker, kubectl, gh):
- **Language**: Go - proven track record for CLI tools with excellent performance
- **CLI Framework**: Cobra (used by docker, kubectl, gh)
- **TUI Framework**: Bubble Tea (used by GitHub's gh CLI)
- **Database**: SQLite with FTS5 for full-text search
- **Single binary distribution** - key advantage of Go

## MVP Feature Set (Reduced Scope)

### Phase 1: Core Functionality
1. **Basic Capture**
   - Pipe input support: `claude | ai-memory capture`
   - Simple stdin parsing
   - Auto-detect Q&A format
   - Store in SQLite

2. **Storage Layer**
   - SQLite database with conversations and messages tables
   - Simple tagging system (project, tool)
   - Timestamp tracking

3. **Search**
   - Full-text search using FTS5
   - Filter by project/tool/date
   - Return relevant snippets with context

4. **CLI Commands**
   - `ai-memory capture` - capture from stdin
   - `ai-memory search <query>` - search conversations
   - `ai-memory list` - list recent conversations
   - `ai-memory export --id <id>` - export as JSON

### Phase 2: TUI Browser
- Split-pane interface using Bubble Tea
- Left pane: conversation list
- Right pane: conversation details
- Vim-like navigation (j/k, /, q)
- Syntax highlighting for code blocks

## Implementation Tasks

1. **Set up Go project structure** ✓
   - Created directories: cmd, internal/storage, internal/capture, internal/search, internal/tui, internal/models

2. **Create SQLite database schema**
   - Conversations table
   - Messages table
   - FTS5 virtual table for search

3. **Implement conversation capture**
   - Read from stdin
   - Parse Q&A format
   - Store in database

4. **Build CLI with Cobra**
   - capture command
   - search command
   - list command
   - export command

5. **Add FTS5 search**
   - Create virtual table
   - Implement search queries
   - Return context snippets

6. **Create TUI browser**
   - List view with Bubble Tea
   - Detail view
   - Navigation

7. **Add JSON export**
   - Export single conversation
   - Export search results

8. **Write tests**
   - Unit tests for core functions
   - Integration tests for CLI

## File Structure
```
ai-memory/
├── cmd/
│   └── ai-memory/
│       └── main.go          # Entry point
├── internal/
│   ├── models/
│   │   └── conversation.go  # Data models
│   ├── storage/
│   │   └── sqlite.go        # Database layer
│   ├── capture/
│   │   └── capture.go       # Input capture logic
│   ├── search/
│   │   └── search.go        # Search implementation
│   └── tui/
│       └── browser.go       # TUI browser
├── go.mod
├── go.sum
└── README.md
```

## Key Libraries
- `github.com/spf13/cobra` - CLI framework
- `github.com/mattn/go-sqlite3` - SQLite driver with FTS5
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - TUI components
- `github.com/charmbracelet/lipgloss` - TUI styling

## Next Steps
1. Initialize Go module
2. Install dependencies
3. Create models
4. Implement SQLite storage with FTS5
5. Build capture functionality
6. Add CLI commands
7. Implement search
8. Build TUI browser

## Success Metrics
- Capture conversations without data loss
- Search returns results in <100ms
- TUI responsive with 1000+ conversations
- Single binary under 20MB
- Zero configuration required to start using