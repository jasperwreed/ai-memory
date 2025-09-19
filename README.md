# AI Memory

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
go install github.com/jasper/ai-memory/cmd/ai-memory@latest
```

Or build from source:

```bash
git clone https://github.com/jasper/ai-memory.git
cd ai-memory
go build -o ai-memory cmd/ai-memory/main.go
```

## Usage

### Capture a Conversation

Pipe AI tool output directly:
```bash
claude | ai-memory capture --tool claude --project myapp
```

Capture from a file:
```bash
ai-memory capture --tool aider --project backend < conversation.txt
```

### Search Conversations

```bash
# Search for authentication-related conversations
ai-memory search "authentication JWT"

# Search with context
ai-memory search "database migration" --context

# Limit results
ai-memory search "error handling" --limit 5
```

### List Recent Conversations

```bash
# List all recent conversations
ai-memory list

# Filter by tool
ai-memory list --tool claude

# Filter by project
ai-memory list --project backend --limit 20
```

### Browse in TUI

```bash
# Open interactive browser
ai-memory browse
```

Navigation:
- `j/k` or arrow keys: Navigate list
- `Enter`: Select conversation
- `/`: Search
- `q`: Quit

### Export Conversations

```bash
# Export as JSON
ai-memory export --id 42 > conversation.json
```

### View Statistics

```bash
# Show usage statistics
ai-memory stats
```

### Delete Conversations

```bash
# Delete with confirmation
ai-memory delete --id 42

# Delete without confirmation
ai-memory delete --id 42 --yes
```

## Conversation Format

AI Memory automatically detects common conversation formats:

```
User: How do I implement a binary search?
Assistant: Here's how to implement binary search...