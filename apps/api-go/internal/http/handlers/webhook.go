package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"maintainer-firewall/api-go/internal/service"
	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type WebhookEventSaver interface {
	SaveEvent(ctx context.Context, evt store.WebhookEvent) error
	SaveAlert(ctx context.Context, alert store.AlertRecord) error
	SaveActionExecutionFailure(ctx context.Context, item store.ActionExecutionFailure) error
	SaveDeliveryMetric(ctx context.Context, metric store.DeliveryMetric) error
	ListRules(ctx context.Context, limit int, offset int, eventType string, keyword string, activeOnly bool) ([]store.RuleRecord, int64, error)
}

type WebhookActionExecutor interface {
	AddLabel(ctx context.Context, repositoryFullName string, number int, label string) error
	AddComment(ctx context.Context, repositoryFullName string, number int, body string) error
}

type WebhookHandler struct {
	Secret         string
	Store          WebhookEventSaver
	RuleEngine     *service.RuleEngine
	ActionExecutor WebhookActionExecutor
}

type webhookResponse struct {
	OK               bool                      `json:"ok"`
	Message          string                    `json:"message,omitempty"`
	Event            string                    `json:"event,omitempty"`
	SuggestedActions []service.SuggestedAction `json:"suggested_actions,omitempty"`
}

func NewWebhookHandler(secret string, eventStore WebhookEventSaver) *WebhookHandler {
	return &WebhookHandler{
		Secret:     secret,
		Store:      eventStore,
		RuleEngine: service.NewRuleEngine(),
	}
}

func (h *WebhookHandler) GitHub(c *gin.Context) {
	startedAt := time.Now().UTC()
	deliverySuccess := false

	defer func() {
		if h.Store == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		deliveryID := c.GetHeader("X-GitHub-Delivery")
		if strings.TrimSpace(deliveryID) == "" {
			deliveryID = fmt.Sprintf("missing-%d", startedAt.UnixNano())
		}
		eventType := c.GetHeader("X-GitHub-Event")
		if strings.TrimSpace(eventType) == "" {
			eventType = "unknown"
		}
		_ = h.Store.SaveDeliveryMetric(ctx, store.DeliveryMetric{
			EventType:     eventType,
			DeliveryID:    deliveryID,
			Success:       deliverySuccess,
			ProcessingMS:  time.Since(startedAt).Milliseconds(),
			RecordedAtUTC: time.Now().UTC(),
		})
	}()

	if strings.TrimSpace(h.Secret) == "" {
		c.JSON(500, webhookResponse{OK: false, Message: "GITHUB_WEBHOOK_SECRET is not configured"})
		return
	}
	if h.Store == nil {
		c.JSON(500, webhookResponse{OK: false, Message: "event store is not configured"})
		return
	}

	signature := c.GetHeader("X-Hub-Signature-256")
	if !strings.HasPrefix(signature, "sha256=") {
		c.JSON(401, webhookResponse{OK: false, Message: "missing or invalid X-Hub-Signature-256"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, webhookResponse{OK: false, Message: "failed to read request body"})
		return
	}

	if !verifyGitHubSignature(signature, body, h.Secret) {
		c.JSON(401, webhookResponse{OK: false, Message: "signature verification failed"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")
	if eventType == "" {
		eventType = "unknown"
	}
	deliveryID := c.GetHeader("X-GitHub-Delivery")
	if strings.TrimSpace(deliveryID) == "" {
		deliveryID = fmt.Sprintf("missing-%d", time.Now().UnixNano())
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(400, webhookResponse{OK: false, Message: "invalid JSON payload"})
		return
	}

	action, _ := payload["action"].(string)
	evt := store.WebhookEvent{
		DeliveryID:         deliveryID,
		EventType:          eventType,
		Action:             action,
		RepositoryFullName: extractRepositoryFullName(payload),
		SenderLogin:        extractSenderLogin(payload),
		PayloadJSON:        body,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	if err := h.Store.SaveEvent(ctx, evt); err != nil {
		c.JSON(500, webhookResponse{OK: false, Message: fmt.Sprintf("failed to persist event: %v", err)})
		return
	}

	suggestions := []service.SuggestedAction{}
	if h.RuleEngine != nil {
		rules, _, err := h.Store.ListRules(ctx, 200, 0, eventType, "", true)
		if err != nil {
			c.JSON(500, webhookResponse{OK: false, Message: fmt.Sprintf("failed to load rules: %v", err)})
			return
		}
		if len(rules) > 0 {
			defs := make([]service.RuleDefinition, 0, len(rules))
			for _, r := range rules {
				defs = append(defs, service.RuleDefinition{
					EventType:       r.EventType,
					Keyword:         r.Keyword,
					SuggestionType:  r.SuggestionType,
					SuggestionValue: r.SuggestionValue,
					Reason:          r.Reason,
				})
			}
			suggestions = h.RuleEngine.EvaluateWithRules(eventType, payload, defs)
		} else {
			suggestions = h.RuleEngine.Evaluate(eventType, payload)
		}
	}

	issueNumber := extractTargetNumber(eventType, payload)
	for _, s := range suggestions {
		alert := store.AlertRecord{
			DeliveryID:         deliveryID,
			EventType:          eventType,
			Action:             action,
			RepositoryFullName: evt.RepositoryFullName,
			SenderLogin:        evt.SenderLogin,
			RuleMatched:        s.Matched,
			SuggestionType:     s.Type,
			SuggestionValue:    s.Value,
			Reason:             s.Reason,
		}
		if err := h.Store.SaveAlert(ctx, alert); err != nil {
			c.JSON(500, webhookResponse{OK: false, Message: fmt.Sprintf("failed to persist alert: %v", err)})
			return
		}

		if h.ActionExecutor != nil && issueNumber > 0 && evt.RepositoryFullName != "unknown" {
			execErr, attempts := h.executeWithRetry(ctx, evt.RepositoryFullName, issueNumber, s)
			if execErr != nil {
				_ = h.Store.SaveActionExecutionFailure(ctx, store.ActionExecutionFailure{
					DeliveryID:         deliveryID,
					EventType:          eventType,
					Action:             action,
					RepositoryFullName: evt.RepositoryFullName,
					SuggestionType:     s.Type,
					SuggestionValue:    s.Value,
					ErrorMessage:       execErr.Error(),
					AttemptCount:       attempts,
				})
			}
		}
	}

	deliverySuccess = true
	c.JSON(200, webhookResponse{
		OK:               true,
		Message:          fmt.Sprintf("webhook accepted (action=%s)", action),
		Event:            eventType,
		SuggestedActions: suggestions,
	})
}

func extractRepositoryFullName(payload map[string]any) string {
	repo, ok := payload["repository"].(map[string]any)
	if !ok {
		return "unknown"
	}
	fullName, _ := repo["full_name"].(string)
	if strings.TrimSpace(fullName) == "" {
		return "unknown"
	}
	return fullName
}

func extractSenderLogin(payload map[string]any) string {
	sender, ok := payload["sender"].(map[string]any)
	if !ok {
		return "unknown"
	}
	login, _ := sender["login"].(string)
	if strings.TrimSpace(login) == "" {
		return "unknown"
	}
	return login
}

func verifyGitHubSignature(signatureHeader string, body []byte, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signatureHeader))
}

func extractTargetNumber(eventType string, payload map[string]any) int {
	if eventType == "issues" {
		if issue, ok := payload["issue"].(map[string]any); ok {
			if n, ok := issue["number"].(float64); ok {
				return int(n)
			}
		}
	}
	if eventType == "pull_request" {
		if pr, ok := payload["pull_request"].(map[string]any); ok {
			if n, ok := pr["number"].(float64); ok {
				return int(n)
			}
		}
	}
	return 0
}

func (h *WebhookHandler) executeWithRetry(ctx context.Context, repositoryFullName string, issueNumber int, action service.SuggestedAction) (error, int) {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		switch action.Type {
		case "label":
			lastErr = h.ActionExecutor.AddLabel(ctx, repositoryFullName, issueNumber, action.Value)
		case "comment":
			lastErr = h.ActionExecutor.AddComment(ctx, repositoryFullName, issueNumber, action.Value)
		default:
			return nil, attempt
		}
		if lastErr == nil {
			return nil, attempt
		}
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}
	}
	return lastErr, maxAttempts
}
