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
}

type WebhookHandler struct {
	Secret     string
	Store      WebhookEventSaver
	RuleEngine *service.RuleEngine
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
		suggestions = h.RuleEngine.Evaluate(eventType, payload)
	}

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
