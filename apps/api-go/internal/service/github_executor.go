package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type GitHubActionExecutor struct {
	Token      string
	HTTPClient *http.Client
}

func NewGitHubActionExecutor(token string) *GitHubActionExecutor {
	return &GitHubActionExecutor{
		Token:      strings.TrimSpace(token),
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (e *GitHubActionExecutor) AddLabel(ctx context.Context, repositoryFullName string, number int, label string) error {
	if strings.TrimSpace(e.Token) == "" {
		return fmt.Errorf("GITHUB_TOKEN is not configured")
	}
	if strings.TrimSpace(repositoryFullName) == "" || repositoryFullName == "unknown" {
		return fmt.Errorf("invalid repository full name")
	}
	if number <= 0 {
		return fmt.Errorf("invalid issue/pull_request number")
	}
	if strings.TrimSpace(label) == "" {
		return fmt.Errorf("empty label")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/labels", repositoryFullName, number)
	body, _ := json.Marshal(map[string]any{"labels": []string{label}})
	return e.doJSONRequest(ctx, http.MethodPost, url, body)
}

func (e *GitHubActionExecutor) AddComment(ctx context.Context, repositoryFullName string, number int, comment string) error {
	if strings.TrimSpace(e.Token) == "" {
		return fmt.Errorf("GITHUB_TOKEN is not configured")
	}
	if strings.TrimSpace(repositoryFullName) == "" || repositoryFullName == "unknown" {
		return fmt.Errorf("invalid repository full name")
	}
	if number <= 0 {
		return fmt.Errorf("invalid issue/pull_request number")
	}
	if strings.TrimSpace(comment) == "" {
		return fmt.Errorf("empty comment")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments", repositoryFullName, number)
	body, _ := json.Marshal(map[string]any{"body": comment})
	return e.doJSONRequest(ctx, http.MethodPost, url, body)
}

func (e *GitHubActionExecutor) doJSONRequest(ctx context.Context, method string, url string, body []byte) error {
	client := e.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("github api status: %d", resp.StatusCode)
}
