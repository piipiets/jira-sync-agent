package sheets

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/piipiets/jira-to-gsheet-agent/internal/config"
	"github.com/piipiets/jira-to-gsheet-agent/internal/jira"
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

func (c *Client) UpsertTickets(tickets []models.Ticket, jiraClient *jira.Client) error {
	sheetName := c.sheetName

	// 1. Get spreadsheet info to find the sheet ID
	spreadsheet, err := c.service.Spreadsheets.Get(c.spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("failed to get spreadsheet info: %w", err)
	}

	var sheetID int64
	found := false
	for _, s := range spreadsheet.Sheets {
		if s.Properties.Title == sheetName {
			sheetID = s.Properties.SheetId
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("tab '%s' not found", sheetName)
	}

	// 2. Read existing data
	readRange := fmt.Sprintf("%s!A:E", sheetName)
	resp, err := c.service.Spreadsheets.Values.Get(c.spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("failed to read existing data: %w", err)
	}

	existingRows := resp.Values
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 2.5 Find the last valid date strictly BEFORE today
	var lastDate time.Time
	for i := len(existingRows) - 1; i >= 0; i-- {
		row := existingRows[i]
		if len(row) > 0 {
			if dateStr, ok := row[0].(string); ok && dateStr != "" {
				parsedDate, err := parseFlexDate(dateStr, now.Location())
				if err == nil {
					if parsedDate.Before(todayMidnight) {
						lastDate = parsedDate
						break
					}
				}
			}
		}
	}

	var requests []*sheets.Request
	var valueUpdates []*sheets.ValueRange
	
	// Find the first row that has today's date
	firstTodayRowIndex := -1
	for i, row := range existingRows {
		if len(row) > 0 {
			if dateStr, ok := row[0].(string); ok && (dateStr == todayStr || isSameDate(dateStr, todayMidnight)) {
				firstTodayRowIndex = i
				break
			}
		}
	}

	currentRowCount := len(existingRows)

	// 3. Fill gaps if dates are absent
	insertCount := 0
	if !lastDate.IsZero() {
		for d := lastDate.AddDate(0, 0, 1); d.Before(todayMidnight); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			
			// If today rows exist, we must INSERT rows to keep order
			if firstTodayRowIndex != -1 {
				// Insert a row at the firstTodayRowIndex
				requests = append(requests, &sheets.Request{
					InsertDimension: &sheets.InsertDimensionRequest{
						Range: &sheets.DimensionRange{
							SheetId:    sheetID,
							Dimension:  "ROWS",
							StartIndex: int64(firstTodayRowIndex + insertCount),
							EndIndex:   int64(firstTodayRowIndex + insertCount + 1),
						},
						InheritFromBefore: true,
					},
				})
				// Set values for the inserted row
				writeRange := fmt.Sprintf("%s!A%d", sheetName, firstTodayRowIndex + insertCount + 1)
				valueUpdates = append(valueUpdates, &sheets.ValueRange{
					Range:  writeRange,
					Values: [][]interface{}{{dateStr, "", "", "", ""}},
				})
				if isWeekend(d) {
					requests = append(requests, createRedRowRequest(sheetID, int64(firstTodayRowIndex + insertCount)))
				}
				insertCount++
				currentRowCount++
			} else {
				// No today rows, just append at the end
				newRow := []interface{}{dateStr, "", "", "", ""}
				writeRange := fmt.Sprintf("%s!A%d", sheetName, currentRowCount+1)
				valueUpdates = append(valueUpdates, &sheets.ValueRange{
					Range:  writeRange,
					Values: [][]interface{}{newRow},
				})
				if isWeekend(d) {
					requests = append(requests, createRedRowRequest(sheetID, int64(currentRowCount)))
				}
				insertCount++
				currentRowCount++
			}
		}
	}

	// Find rows that match today's date to decide between update or append
	todayTicketToRow := make(map[string]int)
	// Match pattern PROJ-123 anywhere
	keyRegex := regexp.MustCompile(`([A-Z0-9]+-\d+)`)
	for i, row := range existingRows {
		if len(row) < 2 {
			continue
		}
		dateCell, ok1 := row[0].(string)
		summaryCell, ok2 := row[1].(string)
		if ok1 && ok2 && (dateCell == todayStr || isSameDate(dateCell, todayMidnight)) {
			match := keyRegex.FindString(summaryCell)
			if match != "" {
				todayTicketToRow[match] = i
			}
		}
	}


	// Track which rows we've already prepared updates for
	updatedRowIndices := make(map[int]bool)

	for _, t := range tickets {
		if rowIndex, exists := todayTicketToRow[t.ID]; exists {
			actualRowIndex := rowIndex + insertCount
			updatedRowIndices[actualRowIndex] = true
			
			row := existingRows[rowIndex]
			currentComment := ""
			statusInSheet := ""
			if len(row) > 3 {
				currentComment = strings.TrimSpace(row[3].(string))
			}
			if len(row) > 4 {
				statusInSheet = strings.TrimSpace(row[4].(string))
			}

			// LOGIC: Only set comment if it's currently blank AND status in sheet is "Code Review"
			if statusInSheet == "Code Review" && currentComment == "" {
				updateRange := fmt.Sprintf("%s!D%d", sheetName, actualRowIndex+1)
				valueUpdates = append(valueUpdates, &sheets.ValueRange{
					Range:  updateRange,
					Values: [][]interface{}{{t.LatestComment}},
				})
			}

		} else {
			// APPEND: date mismatch or ticket not found today
			summaryWithKey := fmt.Sprintf("%s [%s]", t.Summary, t.ID)
			comment := ""
			if t.Status == "Code Review" {
				comment = t.LatestComment
			}
			newRow := []interface{}{todayStr, summaryWithKey, "", comment, t.Status}
			
			writeRange := fmt.Sprintf("%s!A%d", sheetName, currentRowCount+1)
			valueUpdates = append(valueUpdates, &sheets.ValueRange{
				Range:  writeRange,
				Values: [][]interface{}{newRow},
			})

			if isWeekend(now) {
				requests = append(requests, createRedRowRequest(sheetID, int64(currentRowCount)))
			}
			currentRowCount++
		}
	}

	// 4.5 Extra Scan: Check the last 5 rows of the spreadsheet
	ticketMap := make(map[string]models.Ticket)
	for _, t := range tickets {
		ticketMap[t.ID] = t
	}

	scanStartIdx := len(existingRows) - 5
	if scanStartIdx < 0 {
		scanStartIdx = 0
	}

	for i := scanStartIdx; i < len(existingRows); i++ {
		actualIdx := i
		if i >= firstTodayRowIndex && firstTodayRowIndex != -1 {
			actualIdx += insertCount
		}

		if updatedRowIndices[actualIdx] {
			continue
		}

		row := existingRows[i]
		if len(row) < 5 {
			continue
		}

		currentComment := ""
		if len(row) > 3 {
			currentComment = strings.TrimSpace(row[3].(string))
		}
		status := ""
		if len(row) > 4 {
			status = strings.TrimSpace(row[4].(string))
		}
		summary, _ := row[1].(string)
		
		if status == "Code Review" && currentComment == "" {
			matches := keyRegex.FindStringSubmatch(summary)
			if len(matches) > 1 {
				ticketID := matches[1]
				
				// Dynamic fetch for Code Review tickets not in the sync list
				if jiraClient != nil {
					if fetchedT, err := jiraClient.GetTicket(ticketID); err == nil && fetchedT.LatestComment != "" {
						updateRange := fmt.Sprintf("%s!D%d", sheetName, actualIdx+1)
						valueUpdates = append(valueUpdates, &sheets.ValueRange{
							Range:  updateRange,
							Values: [][]interface{}{{fetchedT.LatestComment}},
						})
						updatedRowIndices[actualIdx] = true
					}
				}
			}
		}
	}

	// 5. Execute Batch Update for formatting and insertions FIRST
	if len(requests) > 0 {
		_, err = c.service.Spreadsheets.BatchUpdate(c.spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: requests,
		}).Do()
		if err != nil {
			return fmt.Errorf("failed to apply formatting and insertions: %w", err)
		}
	}

	// 6. Execute Batch Updates for values
	if len(valueUpdates) > 0 {
		_, err = c.service.Spreadsheets.Values.BatchUpdate(c.spreadsheetID, &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "RAW",
			Data:             valueUpdates,
		}).Do()
		if err != nil {
			return fmt.Errorf("failed to update sheet values: %w", err)
		}
	}

	return nil
}

func parseFlexDate(dateStr string, loc *time.Location) (time.Time, error) {
	// Try YYYY-MM-DD
	if t, err := time.ParseInLocation("2006-01-02", dateStr, loc); err == nil {
		return t, nil
	}

	// Try DD Month YYYY (Indonesian)
	indonesianMonths := map[string]string{
		"Januari": "January", "Februari": "February", "Maret": "March",
		"April": "April", "Mei": "May", "Juni": "June",
		"Juli": "July", "Agustus": "August", "September": "September",
		"Oktober": "October", "November": "November", "Desember": "December",
	}

	parts := regexp.MustCompile(`\s+`).Split(dateStr, -1)
	if len(parts) == 3 {
		day := parts[0]
		month := parts[1]
		year := parts[2]

		if engMonth, ok := indonesianMonths[month]; ok {
			engDateStr := fmt.Sprintf("%s %s %s", day, engMonth, year)
			if t, err := time.ParseInLocation("02 January 2006", engDateStr, loc); err == nil {
				return t, nil
			}
		}
		// Also try standard English month names
		if t, err := time.ParseInLocation("02 January 2006", dateStr, loc); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported date format: %s", dateStr)
}

func isSameDate(dateStr string, target time.Time) bool {
	t, err := parseFlexDate(dateStr, target.Location())
	if err != nil {
		return false
	}
	return t.Year() == target.Year() && t.Month() == target.Month() && t.Day() == target.Day()
}

func isWeekend(t time.Time) bool {
	return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}

func createRedRowRequest(sheetID int64, rowIndex int64) *sheets.Request {
	return &sheets.Request{
		RepeatCell: &sheets.RepeatCellRequest{
			Range: &sheets.GridRange{
				SheetId:          sheetID,
				StartRowIndex:    rowIndex,
				EndRowIndex:      rowIndex + 1,
				StartColumnIndex: 0,
				EndColumnIndex:   8,
			},
			Cell: &sheets.CellData{
				UserEnteredFormat: &sheets.CellFormat{
					BackgroundColor: &sheets.Color{
						Red:   1.0,
						Green: 0.0,
						Blue:  0.0,
					},
				},
			},
			Fields: "userEnteredFormat.backgroundColor",
		},
	}
}
