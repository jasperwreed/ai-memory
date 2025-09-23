# AI Memory API

A lightweight REST API for interacting with the AI conversation database. This is a separate Go module independent from the main application.

## Module Structure

The API is a completely independent Go module (`github.com/jasperwreed/ai-memory/api`) with its own `go.mod` file. It only depends on the SQLite driver and doesn't import any code from the parent application.

## Building the API

```bash
# From the api directory
cd api
go build -o api-server .

# Or from the repository root (requires Go 1.20+)
go build -C api -o api/api-server .
```

## Running the API

```bash
# Run directly
./api-server --port 8080 --db ~/.ai-memory/conversations.db

# Or using go run
go run api/main.go --port 8080 --db ~/.ai-memory/conversations.db
```

### Command Line Options
- `--port` - Port to run the API server on (default: 8080)
- `--db` - Path to SQLite database file (default: ~/.ai-memory/conversations.db)

## API Endpoints

### Health Check
```
GET /api/health
```
Returns server health status.

### List Conversations
```
GET /api/conversations?limit=50&offset=0&tool=claude&project=myproject
```
Query Parameters:
- `limit` - Number of results (default: 50)
- `offset` - Pagination offset (default: 0)
- `tool` - Filter by tool name (optional)
- `project` - Filter by project name (optional)

### Get Conversation
```
GET /api/conversations/{id}
```
Returns a single conversation by ID.

### Create Conversation
```
POST /api/conversations
Content-Type: application/json

{
  "tool": "claude",
  "project": "myproject",
  "summary": "Discussion about API design",
  "messages": [
    {
      "role": "user",
      "content": "How should I design a REST API?"
    },
    {
      "role": "assistant",
      "content": "Here are some best practices..."
    }
  ]
}
```

### Delete Conversation
```
DELETE /api/conversations/{id}
```
Deletes a conversation and all its messages.

### Get Messages
```
GET /api/conversations/{id}/messages
```
Returns all messages for a conversation.

### Search
```
GET /api/search?q=authentication&limit=20
```
Query Parameters:
- `q` - Search query (required)
- `limit` - Number of results (default: 20)

Searches across all conversations using full-text search.

### Statistics
```
GET /api/stats
```
Returns usage statistics including:
- Total conversations
- Total messages
- Total tokens
- Tools breakdown
- Top projects

### Export Conversation
```
GET /api/conversations/{id}/export
```
Downloads a conversation as a JSON file.

## Response Format

All successful responses return JSON:
```json
{
  "data": "...",
  "metadata": "..."
}
```

Error responses:
```json
{
  "error": "Error message"
}
```

## Database Schema

The API interacts directly with the SQLite database tables:
- `conversations` - Stores conversation metadata
- `messages` - Individual messages within conversations
- `conversations_fts` - Full-text search index

## CORS

The API includes CORS headers to allow cross-origin requests from web applications.

## Authentication

Currently, the API does not include authentication. Add authentication middleware as needed for production use.