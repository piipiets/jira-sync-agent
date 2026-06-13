# Jira to Google Sheets MCP Server

An MCP (Model Context Protocol) server written in Go that lets an AI client fetch Jira tickets and synchronize them into a Google Spreadsheet.

The server itself does not run an AI model. It exposes tools over MCP, and an external MCP-compatible client such as Claude Desktop, Cursor, or another agent host can call those tools.

## Tech Stack

- **Language:** Go
- **Protocol:** Model Context Protocol (MCP)
- **MCP SDK:** `github.com/mark3labs/mcp-go`
- **Jira Integration:** Jira REST API v2
- **Google Sheets Integration:** Google Sheets API v4
- **Configuration:** Environment variables loaded from `.env`

## How It Works

The application starts as an MCP server over stdio. Once connected to an MCP host, it exposes tools that can fetch Jira issues, inspect ticket data, and write ticket rows into Google Sheets.

Available MCP tools:

1. **`sync_all`**
   - Fetches tickets from Jira using the configured JQL.
   - Converts Jira issues into the internal ticket model.
   - Updates existing Google Sheets rows or appends new rows.

2. **`fetch_jira_tickets`**
   - Fetches Jira tickets using the configured JQL.
   - Returns the ticket data as formatted JSON.
   - Useful when the AI client needs to inspect Jira data before deciding what to do.

3. **`upsert_to_sheets`**
   - Accepts a JSON array of ticket objects.
   - Updates matching rows in Google Sheets by Jira ticket ID.
   - Appends rows for tickets that do not already exist in the sheet.

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
internal/jira  ---- fetches Jira issues using JQL
    |
    | maps issues into []models.Ticket
    v
pkg/models
    |
    | sends tickets to Sheets client
    v
internal/sheets ---- updates existing rows or appends new rows
    |
    v
Google Spreadsheet
```

## Data Mapping

Jira issues are mapped into the shared `Ticket` model:

```go
type Ticket struct {
    ID            string
    Summary       string
    Status        string
    Assignee      string
    CreationDate  time.Time
    LatestComment string
}
```

When writing to Google Sheets, the current implementation uses this layout:

| Column | Value |
| --- | --- |
| A | Sync date |
| B | Jira summary with ticket key, for example `Fix bug [PROJ-123]` |
| C | Blank |
| D | Latest tracked comment |
| E | Current Jira status |

Existing rows are detected by looking for a Jira key pattern like `[PROJ-123]` in column B.

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

## Configuration

Create a `.env` file in the project root:

```env
JIRA_URL=https://your-domain.atlassian.net
JIRA_USER=your-email@example.com
JIRA_TOKEN=your-jira-api-token
JIRA_JQL=project = YOURPROJ AND status in ("To Do", "In Progress", "Code Review")

SPREADSHEET_ID=your-spreadsheet-id
SHEET_NAME=Your Sheet Tab Name
GOOGLE_APPLICATION_CREDENTIALS=C:/path/to/service-account-key.json

COMMENT_AUTHOR=Reviewer Name
```

Notes:

- `SPREADSHEET_ID` can be either the raw spreadsheet ID or a full Google Sheets URL.
- `JIRA_URL` should normally be the Jira base URL, but the Jira client also attempts to extract JQL if a full Jira search URL is provided.
- `COMMENT_AUTHOR` is used when a ticket is in `Code Review`; the Jira client looks for comments from that author.

## Build

```bash
go build -o agent.exe ./cmd/agent
```

## Run Locally

```bash
./agent.exe
```

The server communicates over stdio, so it is usually launched by an MCP host rather than run manually in a terminal.

## MCP Client Example

Example Claude Desktop configuration:

```json
{
  "mcpServers": {
    "jira-sync": {
      "command": "C:/path/to/jira-to-gsheet-agent/agent.exe",
      "args": [],
      "env": {
        "JIRA_URL": "https://your-domain.atlassian.net",
        "JIRA_USER": "your-email@example.com",
        "JIRA_TOKEN": "your-jira-api-token",
        "JIRA_JQL": "project = YOURPROJ",
        "SPREADSHEET_ID": "your-spreadsheet-id",
        "SHEET_NAME": "Your Sheet Tab Name",
        "GOOGLE_APPLICATION_CREDENTIALS": "C:/path/to/service-account-key.json",
        "COMMENT_AUTHOR": "Reviewer Name"
      }
    }
  }
}
```

You can provide configuration through either the `.env` file or the MCP client config environment block.

## Current Behavior Notes

- The Jira client filters tickets to these statuses: `To Do`, `In Progress`, `Revisi`, `Code Review`, and `Task To Do`.
- For `Code Review` tickets, the client fetches detailed comments and stores the first comment matching `COMMENT_AUTHOR`.
- The Sheets client updates columns D and E for existing tickets.
- New rows are written starting after the existing data range.
