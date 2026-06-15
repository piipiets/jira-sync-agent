package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/piipiets/jira-to-gsheet-agent/internal/config"
	"github.com/piipiets/jira-to-gsheet-agent/internal/jira"
	"github.com/piipiets/jira-to-gsheet-agent/internal/sheets"
	"github.com/piipiets/jira-to-gsheet-agent/pkg/models"
)

func main() {
	// Load .env file
	_ = godotenv.Load()

	log.SetOutput(os.Stderr)

	cfg := config.LoadConfig()

	// Initialize Clients
	ctx := context.Background()
	jiraClient, err := jira.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error initializing Jira client: %v", err)
	}

	sheetsClient, err := sheets.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Error initializing Sheets client: %v", err)
	}

	// 1. Create MCP Server
	s := server.NewMCPServer(
		"Jira-to-GSheet Agent",
		"1.0.0",
	)

	// 2. Define Tools

	// Tool: sync_all
	syncTool := mcp.NewTool("sync_all",
		mcp.WithDescription("Executes a full synchronization: fetches tickets from Jira and upserts them into Google Sheets."),
	)
	s.AddTool(syncTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Println("MCP Tool Call: sync_all")
		tickets, err := jiraClient.GetTickets()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Jira fetch failed: %v", err)), nil
		}

		err = sheetsClient.UpsertTickets(tickets, jiraClient)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Sheets upsert failed: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully synced %d tickets to Google Sheets.", len(tickets))), nil
	})

	// Tool: fetch_jira_tickets
	fetchTool := mcp.NewTool("fetch_jira_tickets",
		mcp.WithDescription("Fetches raw ticket data from Jira based on the configured JQL."),
	)
	s.AddTool(fetchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Println("MCP Tool Call: fetch_jira_tickets")
		tickets, err := jiraClient.GetTickets()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Jira fetch failed: %v", err)), nil
		}

		data, _ := json.MarshalIndent(tickets, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	})

	// Tool: upsert_to_sheets
	upsertTool := mcp.NewTool("upsert_to_sheets",
		mcp.WithDescription("Updates or appends a specific list of ticket data into Google Sheets."),
		mcp.WithString("tickets_json",
			mcp.Required(),
			mcp.Description("A JSON array of ticket objects to upsert."),
		),
	)
	s.AddTool(upsertTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Println("MCP Tool Call: upsert_to_sheets")
		jsonStr, err := request.RequireString("tickets_json")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var tickets []models.Ticket
		if err := json.Unmarshal([]byte(jsonStr), &tickets); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid JSON format: %v", err)), nil
		}

		err = sheetsClient.UpsertTickets(tickets, jiraClient)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Sheets upsert failed: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully upserted %d tickets to Google Sheets.", len(tickets))), nil
	})

	// 3. Start Server
	log.Println("Starting MCP server on stdio...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
