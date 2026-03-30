# Expert Review MCP System

## Architecture

```
Claude Code  →  MCP Server (blocking)  →  AI waits
                     │
                     ├── stores review in memory
                     │
                     └── HTTP API (3100) → Vue Frontend → Human Reviewer
                                                  ↓
                                            submit → unblocks AI
```

## Build

```bash
cd mcp-server
npm install
npm run build
```

## Run

**Direct execution:**
```bash
node mcp-server/dist/server.cjs
```

**In settings.json (Claude Code):**
```json
{
  "mcpServers": {
    "expert-review": {
      "command": "node",
      "args": ["/absolute/path/to/mcp-server/dist/server.cjs"]
    }
  }
}
```

## Frontend

```bash
cd frontend
npm install
npm run dev
```
Visit http://localhost:5173

## MCP Tools

### `request_review` (BLOCKING)
Blocks until the human reviewer submits their decision in the frontend.

```json
{
  "tool": "request_review",
  "arguments": {
    "taskResult": {
      "taskId": "task-123",
      "summary": "Fixed login bug",
      "details": "Changed authentication middleware..."
    }
  }
}
```

Returns after human submits:
```json
{
  "content": [{
    "type": "text",
    "text": "{\n  \"reviewId\": \"...\",\n  \"decision\": \"approve\",\n  \"comments\": \"Looks good!\",\n  \"reviewedAt\": \"...\",\n  \"reviewedBy\": \"anonymous\"\n}"
  }]
}
```

### `get_review_history`
Query past completed reviews.

```json
{
  "tool": "get_review_history",
  "arguments": { "limit": 10 }
}
```

## HTTP API (for frontend polling)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/reviews/pending` | List pending reviews |
| GET | `/api/reviews` | List all reviews |
| GET | `/api/reviews/:id` | Get specific review |
| POST | `/api/reviews/:id/submit` | Submit review result |
| GET | `/api/reviews/:id/result` | Poll for result |
| GET | `/health` | Health check |
