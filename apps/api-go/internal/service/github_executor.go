package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type GitHubActionExecutor struct {
	Token      string
	HTTPClient *http.Client
}

type GitHubUserEvent struct {
	DeliveryID         string
	EventType          string
	Action             string
	RepositoryFullName string
	SenderLogin        string
	PayloadJSON        json.RawMessage
}

func NewGitHubActionExecutor(token string) *GitHubActionExecutor {
	return &GitHubActionExecutor{
		Token:      strings.TrimSpace(token),
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (e *GitHubActionExecutor) AddLabel(ctx context.Context, repositoryFullName string, number int, label string) error {
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

func (e *GitHubActionExecutor) ListRecentEventTypes(ctx context.Context) ([]string, error) {
	events, err := e.ListRecentEvents(ctx)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(events))
	for _, evt := range events {
		t := strings.TrimSpace(evt.EventType)
		if t == "" {
			continue
		}
		set[t] = struct{}{}
	}
	types := make([]string, 0, len(set))
	for t := range set {
		types = append(types, t)
	}
	sort.Strings(types)
	return types, nil
}

func (e *GitHubActionExecutor) ListRecentEvents(ctx context.Context) ([]GitHubUserEvent, error) {
	login, err := e.getAuthenticatedLogin(ctx)
	if err != nil {
		return nil, err
	}
	body, err := e.doRequest(ctx, http.MethodGet, fmt.Sprintf("https://api.github.com/users/%s/events?per_page=100", login), nil)
	if err != nil {
		return nil, err
	}

	var raw []map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode github events: %w", err)
	}

	out := make([]GitHubUserEvent, 0, len(raw))
	for _, item := range raw {
		payload, _ := json.Marshal(item)
		eventID, _ := item["id"].(string)
		typ, _ := item["type"].(string)
		action := "unknown"
		if p, ok := item["payload"].(map[string]any); ok {
			if a, ok := p["action"].(string); ok && strings.TrimSpace(a) != "" {
				action = strings.TrimSpace(a)
			}
		}
		repo := "unknown"
		if r, ok := item["repo"].(map[string]any); ok {
			if n, ok := r["name"].(string); ok && strings.TrimSpace(n) != "" {
				repo = strings.TrimSpace(n)
			}
		}
		sender := "unknown"
		if a, ok := item["actor"].(map[string]any); ok {
			if l, ok := a["login"].(string); ok && strings.TrimSpace(l) != "" {
				sender = strings.TrimSpace(l)
			}
		}
		deliveryID := "gh-unknown"
		if strings.TrimSpace(eventID) != "" {
			deliveryID = "gh-" + strings.TrimSpace(eventID)
		}
		out = append(out, GitHubUserEvent{
			DeliveryID:         deliveryID,
			EventType:          strings.TrimSpace(typ),
			Action:             action,
			RepositoryFullName: repo,
			SenderLogin:        sender,
			PayloadJSON:        json.RawMessage(payload),
		})
	}

	return out, nil
}

func (e *GitHubActionExecutor) getAuthenticatedLogin(ctx context.Context) (string, error) {
	body, err := e.doRequest(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", err
	}
	var user struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return "", fmt.Errorf("decode github user: %w", err)
	}
	login := strings.TrimSpace(user.Login)
	if login == "" {
		return "", fmt.Errorf("github user login is empty")
	}
	return login, nil
}

func (e *GitHubActionExecutor) doJSONRequest(ctx context.Context, method string, url string, body []byte) error {
	_, err := e.doRequest(ctx, method, url, body)
	return err
}

func (e *GitHubActionExecutor) doRequest(ctx context.Context, method string, url string, body []byte) ([]byte, error) {
	if strings.TrimSpace(e.Token) == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is not configured")
	}
	client := e.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request github api: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}
	return nil, fmt.Errorf("github api status: %d", resp.StatusCode)
}
