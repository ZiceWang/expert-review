# Expert Review MCP

AI task review system with human approval workflow.

## Overview

AI completes a task → calls `request_review` → **BLOCKS waiting** → Human reviews via web UI → Decision returned → AI continues (or fixes and re-reviews)

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────┐
│ Claude Code │────▶│  Go MCP Server   │────▶│ Vue Frontend│
│   (MCP)     │◀────│   (blocking)     │◀────│  :3100      │
└─────────────┘     └──────────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   SQLite    │
                    │  (FTS5)    │
                    └─────────────┘
```

## Quick Start

```bash
# Build
./build.sh        # Linux/macOS/Git Bash
pwsh ./build.ps1   # Windows PowerShell
cmd /c build       # Windows CMD

# Run
cd build
./expert-review.exe    # Windows
./expert-review         # Linux/macOS

# Access frontend
open http://localhost:3100
```

## Claude Code Integration

Add to your `settings.json`:

```json
{
  "mcpServers": {
    "expert-review": {
      "command": "/path/to/build/expert-review",
      "args": []
    }
  }
}
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `request_review` | **BLOCKING** - Wait for human review decision |
| `search_review_history` | FTS5 full-text search through reviews |
| `get_recent_reviews` | Get N most recent reviews |
| `get_review_from_id` | Get specific review by ID |
| `get_status` | Server health and statistics |

### `request_review` (BLOCKING)

```json
{
  "tool": "request_review",
  "arguments": {
    "taskResult": {
      "summary": "Fixed authentication bug",
      "details": "Changed token validation logic..."
    }
  }
}
```

Returns after human decision:

```json
{
  "reviewId": "uuid",
  "decision": "approve|reject|needs_revision",
  "comments": "Looks good",
  "reviewedAt": "2024-01-01T00:00:00Z",
  "reviewedBy": "reviewer-name"
}
```

## HTTP API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/reviews` | List all reviews |
| GET | `/api/reviews/pending` | List pending reviews |
| GET | `/api/reviews/recent?limit=10` | Recent reviews |
| GET | `/api/reviews/search?q=keyword` | FTS5 search |
| GET | `/api/reviews/:id` | Get by ID |
| POST | `/api/reviews/:id/submit` | Submit decision |
| GET | `/health` | Health check |

## Project Structure

```
.
├── build.sh/.ps1/.bat    # Build scripts
├── mcp-server/           # Go backend source
│   └── main.go           # MCP server + HTTP API + SQLite
├── frontend/             # Vue 3 frontend source
│   ├── src/              # Vue components
│   └── public/           # Static assets
└── build/                # Build output (after build)
    ├── expert-review     # Go binary
    └── public/           # Vue static files
```

## Workflow

1. AI completes task, calls `request_review`
2. MCP server stores review, blocks on goroutine
3. Human sees pending review at `http://localhost:3100`
4. Human reviews and clicks: **Approve** / **Reject** / **Needs Revision**
5. MCP receives decision, returns to AI
6. AI fixes if rejected/needs_revision, re-requests review
7. Loop until approved, then AI returns result to user

## Tech Stack

- **Backend**: Go 1.21+, gorilla/mux, modernc.org/sqlite (FTS5)
- **Frontend**: Vue 3, Vite
- **Protocol**: MCP over stdio
- **Storage**: SQLite with FTS5 full-text search
