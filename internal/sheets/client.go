package sheets

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/piipiets/jira-to-gsheet-agent/internal/config"
	"github.com/piipiets/jira-to-gsheet-agent/pkg/models"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Client struct {
	service       *sheets.Service
	spreadsheetID string
	sheetName     string
}

func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	if cfg.GoogleAuthFile == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}

	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(cfg.GoogleAuthFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	return &Client{
		service:       srv,
		spreadsheetID: cfg.SpreadsheetID,
		sheetName:     cfg.SheetName,
	}, nil
}

func (c *Client) UpsertTickets(tickets []models.Ticket) error {
	sheetName := c.sheetName
	
	// 1. Get spreadsheet info
	spreadsheet, err := c.service.Spreadsheets.Get(c.spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("failed to get spreadsheet info: %w", err)
	}

	found := false
	for _, s := range spreadsheet.Sheets {
		if s.Properties.Title == sheetName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("tab '%s' not found", sheetName)
	}

	// 2. Read existing data to find matches and the last row
	readRange := fmt.Sprintf("%s!A:E", sheetName)
	resp, err := c.service.Spreadsheets.Values.Get(c.spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("failed to read existing data: %w", err)
	}

	existingRows := resp.Values
	keyToRowIndex := make(map[string]int)
	keyRegex := regexp.MustCompile(`\[([A-Z0-9]+-\d+)\]`)

	for i, row := range existingRows {
		if len(row) < 2 {
			continue
		}
		if cellVal, ok := row[1].(string); ok {
			matches := keyRegex.FindStringSubmatch(cellVal)
			if len(matches) > 1 {
				keyToRowIndex[matches[1]] = i
			}
		}
	}

	now := time.Now().Format("2006-01-02")
	var valueUpdates []*sheets.ValueRange
	var newRows [][]interface{}

	lastRow := len(existingRows)
	
	for _, t := range tickets {
		if rowIndex, exists := keyToRowIndex[t.ID]; exists {
			// UPDATE EXISTING: 
			// Target Column D (index 3) for Comment
			// Target Column E (index 4) for Status
			updateRange := fmt.Sprintf("%s!D%d:E%d", sheetName, rowIndex+1, rowIndex+1)
			valueUpdates = append(valueUpdates, &sheets.ValueRange{
				Range:  updateRange,
				Values: [][]interface{}{{t.LatestComment, t.Status}},
			})
		} else {
			// ADD NEW: Prepare row data
			summaryWithKey := fmt.Sprintf("%s [%s]", t.Summary, t.ID)
			newRows = append(newRows, []interface{}{
				now, summaryWithKey, "", t.LatestComment, t.Status,
			})
		}
	}

	// 3. Update Existing Statuses
	if len(valueUpdates) > 0 {
		_, err = c.service.Spreadsheets.Values.BatchUpdate(c.spreadsheetID, &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "RAW",
			Data:             valueUpdates,
		}).Do()
		if err != nil {
			return fmt.Errorf("failed status updates: %w", err)
		}
	}

	// 4. Append New Rows to last empty row
	if len(newRows) > 0 {
		// Use Update instead of Append for more row index control
		writeRange := fmt.Sprintf("%s!A%d", sheetName, lastRow+1)
		_, err = c.service.Spreadsheets.Values.Update(c.spreadsheetID, writeRange, &sheets.ValueRange{
			Values: newRows,
		}).ValueInputOption("RAW").Do()
		if err != nil {
			return fmt.Errorf("failed to add new rows: %w", err)
		}
	}

	return nil
}
