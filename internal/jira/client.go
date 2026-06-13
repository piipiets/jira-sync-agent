package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/piipiets/jira-to-gsheet-agent/internal/config"
	"github.com/piipiets/jira-to-gsheet-agent/pkg/models"
)

type Client struct {
	baseURL       string
	user          string
	token         string
	customJQL     string
	commentAuthor string
}

func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.JiraURL == "" || cfg.JiraToken == "" {
		return nil, fmt.Errorf("Jira configuration is incomplete (URL and Token are required)")
	}

	baseURL := cfg.JiraURL
	customJQL := cfg.JiraJQL

	if strings.Contains(cfg.JiraURL, "jql=") {
		fmt.Println("[DEBUG] Full search URL detected, extracting JQL...")
		u, err := url.Parse(cfg.JiraURL)
		if err == nil {
			baseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
			pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
			if len(pathParts) > 0 && pathParts[0] != "" && pathParts[0] != "issues" && pathParts[0] != "browse" && pathParts[0] != "rest" {
				baseURL = fmt.Sprintf("%s/%s", baseURL, pathParts[0])
			}
			customJQL = u.Query().Get("jql")
		}
		
		fmt.Printf("[DEBUG] Automatically set BaseURL: %s\n", baseURL)
		fmt.Printf("[DEBUG] Automatically set JQL: %s\n", customJQL)
	}

	return &Client{
		baseURL:       baseURL,
		user:          cfg.JiraUser,
		token:         cfg.JiraToken,
		customJQL:     customJQL,
		commentAuthor: cfg.CommentAuthor,
	}, nil
}

func (c *Client) getDetailedComments(issueKey string) ([]struct {
	Author struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
	Body string `json:"body"`
}, error) {
	detailURL := fmt.Sprintf("%s/rest/api/2/issue/%s?fields=comment", c.baseURL, issueKey)
	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return nil, err
	}

	if c.user == "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else {
		req.SetBasicAuth(c.user, c.token)
	}

	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch issue details for %s: %s", issueKey, resp.Status)
	}

	var result struct {
		Fields struct {
			Comment struct {
				Comments []struct {
					Author struct {
						DisplayName string `json:"displayName"`
					} `json:"author"`
					Body string `json:"body"`
				} `json:"comments"`
			} `json:"comment"`
		} `json:"fields"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Fields.Comment.Comments, nil
}

func (c *Client) GetTickets() ([]models.Ticket, error) {
	jql := c.customJQL
	if jql == "" {
		return nil, fmt.Errorf("no JQL provided or extracted from URL")
	}

	// Manual HTTP Request
	searchURL := fmt.Sprintf("%s/rest/api/2/search?jql=%s&os_authType=basic", c.baseURL, url.QueryEscape(jql))
	fmt.Printf("[DEBUG] Final Request URL: %s\n", searchURL)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	if c.user == "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		fmt.Println("[DEBUG] Attempting Authorization: Bearer <TOKEN>")
	} else {
		req.SetBasicAuth(c.user, c.token)
		fmt.Printf("[DEBUG] Attempting Authorization: Basic %s:<TOKEN>\n", c.user)
	}

	req.Header.Set("X-Atlassian-Token", "no-check")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[DEBUG] Response Status: %s\n", resp.Status)
		limit := len(body)
		if limit > 500 {
			limit = 500
		}
		fmt.Printf("[DEBUG] Response Body: %s\n", string(body[:limit]))
		return nil, fmt.Errorf("jira request failed with status: %s", resp.Status)
	}

	var result struct {
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary string `json:"summary"`
				Status  struct {
					Name string `json:"name"`
				} `json:"status"`
				Assignee struct {
					DisplayName string `json:"displayName"`
				} `json:"assignee"`
				Created string `json:"created"`
			} `json:"fields"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse jira response: %w", err)
	}

	// Allowed statuses for filtering
	allowedStatuses := map[string]bool{
		"To Do":       true,
		"In Progress": true,
		"Revisi":      true,
		"Code Review": true,
		"Task To Do":  true,
	}

	var tickets []models.Ticket
	for _, issue := range result.Issues {
		statusName := issue.Fields.Status.Name
		if !allowedStatuses[statusName] {
			continue
		}

		createdAt, err := time.Parse("2006-01-02T15:04:05.000-0700", issue.Fields.Created)
		if err != nil {
			createdAt, _ = time.Parse(time.RFC3339, issue.Fields.Created)
		}
		
		assignee := "Unassigned"
		if issue.Fields.Assignee.DisplayName != "" {
			assignee = issue.Fields.Assignee.DisplayName
		}

		latestComment := ""
		if statusName == "Code Review" {
			fmt.Printf("[DEBUG] Fetching details for Code Review ticket: %s\n", issue.Key)
			comments, err := c.getDetailedComments(issue.Key)
			if err != nil {
				fmt.Printf("[WARN] Failed to fetch comments for %s: %v\n", issue.Key, err)
			} else {
				// Find the first comment from the configured author
				for i := 0; i < len(comments); i++ {
					if comments[i].Author.DisplayName == c.commentAuthor {
						latestComment = comments[i].Body
						break
					}
				}
			}
		}

		tickets = append(tickets, models.Ticket{
			ID:            issue.Key,
			Summary:       issue.Fields.Summary,
			Status:        statusName,
			Assignee:      assignee,
			CreationDate:  createdAt,
			LatestComment: latestComment,
		})
	}

	return tickets, nil
}
