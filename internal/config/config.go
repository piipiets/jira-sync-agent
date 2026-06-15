package config

import (
	"os"
	"strings"
)

type Config struct {
	JiraURL        string
	JiraUser       string
	JiraPassword   string
	JiraToken      string
	JiraProject    string
	JiraJQL        string
	SpreadsheetID  string
	GoogleAuthFile string
	CommentAuthor  string
	SheetName      string
	ReviewStatuses []string
}

func LoadConfig() *Config {
	rawID := os.Getenv("SPREADSHEET_ID")
	
	reviewStatusesRaw := os.Getenv("REVIEW_STATUSES")
	var reviewStatuses []string
	if reviewStatusesRaw != "" {
		for _, s := range strings.Split(reviewStatusesRaw, ",") {
			reviewStatuses = append(reviewStatuses, strings.TrimSpace(s))
		}
	}

	return &Config{
		JiraURL:        os.Getenv("JIRA_URL"),
		JiraUser:       os.Getenv("JIRA_USER"),
		JiraPassword:   os.Getenv("JIRA_PASSWORD"),
		JiraToken:      os.Getenv("JIRA_TOKEN"),
		JiraProject:    os.Getenv("JIRA_PROJECT"),
		JiraJQL:        os.Getenv("JIRA_JQL"),
		SpreadsheetID:  parseSpreadsheetID(rawID),
		GoogleAuthFile: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		CommentAuthor:  os.Getenv("COMMENT_AUTHOR"),
		SheetName:      os.Getenv("SHEET_NAME"),
		ReviewStatuses: reviewStatuses,
	}
}

// parseSpreadsheetID extracts the ID from a full URL or returns the string if it's already an ID.
func parseSpreadsheetID(input string) string {
	if !strings.Contains(input, "docs.google.com") {
		return input
	}
	// URL format: https://docs.google.com/spreadsheets/d/ID/edit...
	parts := strings.Split(input, "/d/")
	if len(parts) < 2 {
		return input
	}
	idParts := strings.Split(parts[1], "/")
	return idParts[0]
}
