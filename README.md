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

## Key Features & Recent Updates

- **Date-Centric Sync:** Automatically ensures spreadsheet dates (Column A) are chronologically ordered.
- **Weekend Highlighting:** Automatically inserts rows for weekend gaps (Saturday/Sunday) and formats them with a red background (Columns A-H).
- **Intelligent Comment Updates:** 
  - For tickets in "Code Review" status (or other review statuses configured via `REVIEW_STATUSES`), the agent fetches the first reviewer comment from Jira.
  - Updates Column D with this comment only if Column D is blank (non-destructive).
- **Flexible Data Matching:** Uses robust regex to match ticket IDs anywhere in the summary, regardless of format.
- **Configurable Review Statuses:** Easily define which Jira statuses trigger comment fetching using the `REVIEW_STATUSES` environment variable.

## How It Works

The application starts as an MCP server over stdio. Once connected to an MCP host, it exposes tools that can fetch Jira issues, inspect ticket data, and write ticket rows into Google Sheets.

Available MCP tools:

1. **`sync_all`**
   - Fetches tickets from Jira based on the configured JQL.
   - Handles chronological date gap-filling.
   - Updates existing rows or appends new rows.
   - Synchronizes "Code Review" comments conditionally (Column D).

2. **`fetch_jira_tickets`**
   - Fetches Jira tickets using the configured JQL.
   - Returns the ticket data as formatted JSON for AI inspection.

3. **`upsert_to_sheets`**
   - Accepts a JSON array of ticket objects.
   - Updates matching rows in Google Sheets, respecting existing data.

## Project Structure

```text
jira-to-gsheet-agent/
|-- cmd/
|   `-- agent/          # Main MCP server entry point
|-- internal/
|   |-- config/         # Environment config loader (supports REVIEW_STATUSES)
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
JIRA_JQL=project = YOURPROJ
REVIEW_STATUSES="Code Review, Deploy Development, Staging, QC BC - Testing Staging"

SPREADSHEET_ID=your-spreadsheet-id
SHEET_NAME=Your Sheet Tab Name
GOOGLE_APPLICATION_CREDENTIALS=C:/path/to/service-account-key.json

COMMENT_AUTHOR=Reviewer Name
```

## Build & Run

```bash
# Build
go build -o agent.exe ./cmd/agent

# Run (via MCP host)
./agent.exe
```
