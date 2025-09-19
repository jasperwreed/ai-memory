# AI Memory - Terminal-Based AI Conversation Manager

## Project Overview

A lightweight, terminal-first tool for capturing, searching, and managing AI conversations across all CLI-based AI tools (Claude Code, Aider, Codex, GPT-CLI, etc.). Think of it as a "second brain" for your AI interactions - never lose valuable solutions or insights from your AI sessions again.

## Problem Statement

Developers using AI CLI tools face several challenges:
- Conversations are ephemeral - valuable solutions get lost
- No way to search across past AI interactions
- Can't build a knowledge base from accumulated AI wisdom
- No visibility into token usage/costs across different tools
- Each AI tool has its own session format with no interoperability

## Core Concept

```bash
# Automatic capture from ANY AI CLI tool
ai-memory record claude code --project "omnicast"
# Now it captures everything from this session

# Later, search across all conversations
ai-memory search "tmux resurrect config"
# Shows relevant snippets from past conversations

# Browse in a TUI (like htop but for AI chats)
ai-memory browse
# ┌─ Sessions ──────┬─ Conversation ─────────────┐
# │ claude: omnicast│ Q: How do I save tmux...   │
# │ aider: auth fix │ A: Use tmux-resurrect...   │
# │ codex: api      │ [Show full response]       │
```

## Key Features

### 1. Universal Capture
- **Hook into ANY AI CLI** - Works with Claude, Aider, Codex, GPT-CLI, etc.
- **Auto-detect formats** - Intelligently parse different response formats
- **Project tagging** - Organize by project, date, tool, topic
- **Passive recording** - Runs in background, no workflow disruption
- **Process monitoring** - Detect AI tool launches automatically

### 2. Smart Search
- **Full-text search** - Search across all conversations instantly
- **Semantic search** - Find conceptually similar conversations (local embeddings, no API)
- **Multi-facet filtering**:
  - By tool (claude, aider, etc.)
  - By date range
  - By project/tags
  - By token count/cost
- **Code-aware search** - Regex support for finding code patterns
- **Context preservation** - Show surrounding Q&A for search results

### 3. TUI Browser
- **Split-pane interface**:
  - Left: Sessions list (sortable, filterable)
  - Right: Conversation view (scrollable, searchable)
- **Vim-like navigation** - j/k for movement, / for search, etc.
- **Syntax highlighting** - Automatic for code blocks
- **Conversation threading** - See full context of exchanges
- **Quick actions**:
  - Export selection (Markdown, JSON)
  - Copy to clipboard
  - Open in editor
  - Tag/annotate

### 4. Knowledge Base Building
- **Auto-extract code snippets** - Build searchable code library
- **Pattern recognition** - Identify recurring questions/solutions
- **TIL generation** - Auto-generate "Today I Learned" summaries
- **Relationship mapping** - Link related conversations
- **Solution templates** - Extract reusable patterns
- **Personal documentation** - Generate docs from your AI interactions

### 5. Analytics Dashboard
- **Token usage tracking**:
  - Cost per tool/model
  - Usage over time graphs
  - Project cost allocation
- **Tool comparison**:
  - Response quality metrics
  - Speed comparisons
  - Success rate tracking
- **Query patterns**:
  - Most asked questions
  - Topic frequency
  - Problem domain analysis
- **Efficiency metrics**:
  - Time saved estimations
  - Solution reuse stats

## Technical Architecture

### Storage Layer
- **Primary DB**: SQLite with FTS5 (full-text search) extension
  - Conversations table
  - Messages table
  - Projects/Tags tables
  - Analytics/Metrics tables
- **Optional Vector DB**: For semantic search (ChromaDB or similar)
- **File-based cache**: JSON exports for quick access
- **Compression**: For old conversations to save space

### Capture Methods
1. **Wrapper approach**: `ai-memory record <command>`
2. **Process monitoring**: Watch for AI tool processes
3. **Plugin system**: Tool-specific adapters
4. **Pipe support**: `claude | ai-memory capture`

### Export Formats
- **Markdown**: For documentation/sharing
- **JSON**: For data portability
- **CSV**: For analytics
- **HTML**: For web viewing

## Implementation Plan

### Phase 1: Core Capture & Storage
- Basic SQLite schema
- Claude Code capture adapter
- Simple CLI interface
- JSON import/export

### Phase 2: Search & Browse
- FTS5 full-text search
- Basic TUI browser (using Bubble Tea or similar)
- Filtering and sorting
- Markdown export

### Phase 3: Advanced Features
- Semantic search with local embeddings
- Multi-tool support (Aider, Codex, etc.)
- Analytics dashboard
- Knowledge base features

### Phase 4: Polish & Extend
- Plugin system for custom tools
- Team sharing features
- Cloud backup options
- API for integration

## Technology Stack

### Language Options
- **Rust**: For performance and single binary distribution
- **Go**: For simplicity and good TUI libraries (Bubble Tea)
- **Python**: For ML features and rapid prototyping

### Key Libraries
- **TUI**: Ratatui (Rust), Bubble Tea (Go), or Textual (Python)
- **Database**: SQLite with FTS5
- **Embeddings**: Sentence-transformers or similar (local)
- **CLI**: Clap (Rust), Cobra (Go), or Click (Python)

## Usage Examples

```bash
# Start recording a session
ai-memory record claude code --project backend --tags "auth,debugging"

# Search for past solutions
ai-memory search "authentication JWT"

# Browse all conversations in TUI
ai-memory browse

# Export project-specific knowledge
ai-memory export --project backend --format markdown > backend-ai-knowledge.md

# Show analytics
ai-memory stats --period month

# Find similar conversations
ai-memory similar "How do I implement OAuth?"

# Quick access to last conversation
ai-memory last

# Pipe support
claude | ai-memory capture --auto-tag
```

## Benefits

### For Individual Developers
- Never lose valuable AI solutions
- Build personal knowledge base
- Track AI tool costs
- Learn from past interactions
- Improve prompting over time

### For Teams
- Share AI-discovered solutions
- Reduce duplicate questions
- Onboard new developers faster
- Document architectural decisions
- Build team knowledge base

## Competitive Landscape

### What Exists
- **GPTCache/ModelCache**: Caching libraries, not CLI tools
- **LangChain/LangSmith**: Enterprise platforms, web-based
- **Claude Squad**: Multi-agent manager, not conversation storage

### Our Differentiation
- **Terminal-first**: Built for CLI workflow
- **Tool-agnostic**: Works with ANY AI CLI
- **Local-first**: Your data stays on your machine
- **Developer-focused**: Features developers actually need
- **Lightweight**: No heavy dependencies or web servers

## Success Metrics
- Conversations captured without data loss
- Search queries return relevant results in <100ms
- TUI responsive with 10k+ conversations
- Export formats preserve all important information
- Zero impact on AI tool performance

## Future Possibilities
- Browser extension for web-based AI tools
- VSCode extension integration
- Team collaboration features
- AI-powered insights from conversation history
- Prompt optimization suggestions based on history
- Integration with documentation tools

## Open Questions
1. Should we support web-based AI tools or stay CLI-only?
2. How to handle sensitive information in conversations?
3. Should we build tool-specific optimizations or stay generic?
4. What's the best way to handle real-time streaming responses?
5. Should we support cloud sync or stay local-only initially?

## Next Steps
1. Create a new project directory
2. Set up basic project structure
3. Implement minimal SQLite storage
4. Build Claude Code capture adapter
5. Create simple CLI for search
6. Develop basic TUI browser
7. Release MVP for feedback

---

*This tool would fill a real gap in the AI CLI ecosystem - providing the "memory layer" that developers need to make their AI interactions truly valuable over time.*