package main

import (
	"fmt"
	"time"

	"github.com/piipiets/jira-to-gsheet-agent/pkg/models"
)

func main() {
	fmt.Println("--- STARTING MOCK JIRA-TO-GSHEET SYNC ---")
	fmt.Println("[INFO] Simulating Jira connection...")
	
	// Mock Data from Jira
	mockTickets := []models.Ticket{
		{
			ID:           "PROJ-101",
			Summary:      "Implement OAuth2 Authentication",
			Status:       "In Progress",
			Assignee:     "John Doe",
			CreationDate: time.Now().AddDate(0, 0, -2),
		},
		{
			ID:           "PROJ-102",
			Summary:      "Fix bug in data mapping logic",
			Status:       "Done",
			Assignee:     "Jane Smith",
			CreationDate: time.Now().AddDate(0, 0, -1),
		},
		{
			ID:           "PROJ-103",
			Summary:      "Update API documentation for v2",
			Status:       "To Do",
			Assignee:     "Unassigned",
			CreationDate: time.Now(),
		},
	}

	fmt.Printf("[INFO] Fetched %d tickets from Jira.\n", len(mockTickets))
	fmt.Println("[INFO] Simulating Google Sheets update...")
	fmt.Println("\n--- FINAL GOOGLE SHEETS PREVIEW ---")
	fmt.Println("| Date (Col 1) | Summary [Key] (Col 2)          | Col 3 | Col 4 | Status (Col 5) |")
	fmt.Println("|--------------|--------------------------------|-------|-------|----------------|")

	now := time.Now().Format("2006-01-02")
	for _, t := range mockTickets {
		summaryWithKey := fmt.Sprintf("%s [%s]", t.Summary, t.ID)
		fmt.Printf("| %-12s | %-30s | %-5s | %-5s | %-14s |\n",
			now,
			summaryWithKey,
			"",
			"",
			t.Status,
		)
	}

	fmt.Println("--------------------------------------------------------------------------------------------")
	fmt.Println("[SUCCESS] 3 tickets successfully synced to Google Sheets!")
}
