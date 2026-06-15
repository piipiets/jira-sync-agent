# Jira to Google Sheets MCP Server

An MCP (Model Context Protocol) server written in Go that lets an AI client fetch Jira tickets and synchronize them into a Google Spreadsheet.

## Tech Stack

- **Language:** Go
- **Protocol:** Model Context Protocol (MCP)
- **MCP SDK:** `github.com/mark3labs/mcp-go`
- **Jira Integration:** Jira REST API v2
- **Google Sheets Integration:** Google Sheets API v4
- **Configuration:** Environment variables loaded from `.env`

## Project Flow

```text
MCP Client
    |
    | calls tool: sync_all / fetch_jira_tickets / upsert_to_sheets
    v
cmd/agent/main.go
    |
    | loads environment config
    v
internal/config
    |
    | initializes Jira and Google Sheets clients
    v
internal/jira  ---- fetches Jira issues
    |
    | maps issues into []models.Ticket
    v
pkg/models
    |
    | sends tickets to Sheets client
    v
internal/sheets ---- upserts to Google Sheets
    |
    v
Google Spreadsheet
```

## Project Structure

```text
jira-to-gsheet-agent/
|-- cmd/
|   `-- agent/          # Main MCP server entry point
|-- internal/
|   |-- config/         # Environment config loader
|   |-- jira/           # Jira REST API client
|   `-- sheets/         # Google Sheets API client
|-- pkg/
|   `-- models/         # Shared Ticket model
|-- .env                # Local environment variables
|-- go.mod              # Go module definition
|-- go.sum              # Dependency lock file
|-- README.md
`-- REQUIREMENTS.md
```

## How to Run

1. **Configure:** Create a `.env` file in the project root:
   ```env
   JIRA_URL=...
   JIRA_USER=...
   JIRA_TOKEN=...
   JIRA_JQL=...
   REVIEW_STATUSES="..."
   SPREADSHEET_ID=...
   SHEET_NAME=...
   GOOGLE_APPLICATION_CREDENTIALS=...
   COMMENT_AUTHOR=...
   ```

2. **Build:**
   ```bash
   go build -o agent.exe ./cmd/agent
   ```

3. **Run (via MCP host):**
   ```bash
   ./agent.exe
   ```
