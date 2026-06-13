package models

import "time"

// Ticket represents the core fields we want to track from Jira.
type Ticket struct {
	ID            string    `json:"id"`
	Summary       string    `json:"summary"`
	Status        string    `json:"status"`
	Assignee      string    `json:"assignee"`
	CreationDate  time.Time `json:"creation_date"`
	LatestComment string    `json:"latest_comment"`
}
