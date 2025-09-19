# Mem - AI Conversation Memory

A lightweight, terminal-first tool for capturing, searching, and managing AI conversations across all CLI-based AI tools (Claude Code, Aider, GPT-CLI, etc.).

## Features

- **Universal Capture**: Works with any AI CLI tool through stdin
- **Full-Text Search**: Fast search across all conversations using SQLite FTS5
- **TUI Browser**: Interactive terminal UI for browsing conversations
- **JSON Export**: Export conversations for sharing or backup
- **Project Organization**: Tag conversations by project and tool
- **Token Tracking**: Estimate token usage and costs

## Installation

```bash
go install github.com/jasperwreed/ai-memory/cmd/mem@latest
```

Or build from source:

```bash
git clone https://github.com/jasperwreed/ai-memory.git
cd ai-memory
go build -tags "sqlite_fts5" -o mem cmd/mem/main.go
```

## Usage

### Capture a Conversation

Pipe AI tool output directly:
```bash
claude | mem capture --tool claude --project myapp
```

Capture from a file:
```bash
mem capture --tool aider --project backend < conversation.txt
```

### Import Claude Code Sessions

```bash
# Import specific session file
mem import --file ~/.claude/projects/myproject/session.jsonl

# Import from current project
mem import --claude-project
```

### Search Conversations

```bash
# Search for authentication-related conversations
mem search "authentication JWT"

# Search with context
mem search "database migration" --context

# Limit results
mem search "error handling" --limit 5
```

### List Recent Conversations

```bash
# List all recent conversations
mem list

# Filter by tool
mem list --tool claude

# Filter by project
mem list --project backend --limit 20
```

### Browse in TUI

```bash
# Open interactive browser
mem browse
```

Navigation:
- `j/k` or arrow keys: Navigate list
- `Enter`: Select conversation
- `/`: Search
- `q`: Quit

### Export Conversations

```bash
# Export as JSON
mem export --id 42 > conversation.json
```

### View Statistics

```bash
# Show usage statistics
mem stats
```

### Delete Conversations

```bash
# Delete with confirmation
mem delete --id 42

# Delete without confirmation
mem delete --id 42 --yes
```

## Conversation Format

AI Memory automatically detects common conversation formats:

```
User: How do I implement a binary search?
Assistant: Here's how to implement binary search...