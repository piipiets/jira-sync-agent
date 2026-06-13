# Jira to Google Sheets MCP Server

An MCP (Model Context Protocol) server built in Go that allows LLMs to synchronize Jira ticket statuses and reviewer comments into a Google Spreadsheet.

## 🚀 How It Works

The agent acts as an MCP server, exposing tools that an LLM (like Claude Desktop) can call:

1.  **`sync_all`**: Executes a full synchronization (Fetch from Jira -> Upsert to Sheets).
2.  **`fetch_jira_tickets`**: Returns a list of Jira tickets to the LLM for inspection.
3.  **`upsert_to_sheets`**: Allows the LLM to send specific ticket data to the spreadsheet.

## 🛠 Tech Stack

*   **Language**: Go (Golang) 1.21+
*   **Protocol**: Model Context Protocol (MCP)
*   **APIs**: Jira REST API v2, Google Sheets API v4
*   **Libraries**:
    *   `github.com/mark3labs/mcp-go`: MCP SDK for Go.
    *   `google.golang.org/api/sheets/v4`: Official Google Sheets client.

## 🏃 Setup & Configuration

### 1. Build the Binary
```bash
go build -o agent.exe ./cmd/agent
```

### 2. Configure Environment
Create a `.env` file in the project root:
```env
JIRA_URL=https://your-domain.atlassian.net
JIRA_TOKEN=your-jira-token
JIRA_JQL=project = YOURPROJ AND status = "To Do"
SPREADSHEET_ID=your-spreadsheet-id
SHEET_NAME="Your Tab Name"
COMMENT_AUTHOR="Author Name to Track"
GOOGLE_APPLICATION_CREDENTIALS=C:/path/to/service-account.json
```

### 3. Add to Claude Desktop
Add the following to your `claude_desktop_config.json` (usually in `%APPDATA%\Claude\config.json` on Windows):

```json
{
  "mcpServers": {
    "jira-sync": {
      "command": "C:/path/to/jira-to-gsheet-agent/agent.exe",
      "args": [],
      "env": {
        "JIRA_URL": "...",
        "JIRA_TOKEN": "...",
        "SPREADSHEET_ID": "...",
        "GOOGLE_APPLICATION_CREDENTIALS": "..."
      }
    }
  }
}
```
*Note: You can either use the `.env` file in the project root or provide the environment variables directly in the config JSON.*

## 📁 Project Structure

```text
jira-to-gsheet-agent/
├── cmd/
│   └── agent/          # MCP Server entry point
├── internal/
│   ├── ai/             # Gemini AI integration logic (optional)
│   ├── config/         # Config loader (safe for MCP stdio)
│   ├── jira/           # Jira API client
│   └── sheets/         # Google Sheets API client
├── pkg/
│   └── models/         # Shared Ticket model
└── .env                # Environment variables
```
