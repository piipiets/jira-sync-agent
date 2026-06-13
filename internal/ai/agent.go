package ai

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/piipiets/jira-to-gsheet-agent/internal/config"
	"google.golang.org/api/option"
)

type Agent struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewAgent(ctx context.Context, cfg *config.Config) (*Agent, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.GeminiAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	model := client.GenerativeModel("gemini-1.5-flash")

	return &Agent{
		client: client,
		model:  model,
	}, nil
}

func (a *Agent) ProcessDescription(ctx context.Context, description string) (string, error) {
	prompt := genai.Text(fmt.Sprintf("Summarize the following Jira description into a single concise sentence for a spreadsheet summary field: %s", description))
	resp, err := a.model.GenerateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	// Simplistic extraction of the first text part
	if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		return string(text), nil
	}

	return "", fmt.Errorf("unexpected response format")
}
