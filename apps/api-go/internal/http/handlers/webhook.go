package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
)

type WebhookHandler struct {
	Secret string
}

type webhookResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Event   string `json:"event,omitempty"`
}

type githubWebhookPayload struct {
	Action string `json:"action"`
}

func NewWebhookHandler(secret string) *WebhookHandler {
	return &WebhookHandler{Secret: secret}
}

func (h *WebhookHandler) GitHub(c *gin.Context) {
	if strings.TrimSpace(h.Secret) == "" {
		c.JSON(500, webhookResponse{OK: false, Message: "GITHUB_WEBHOOK_SECRET is not configured"})
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

	event := c.GetHeader("X-GitHub-Event")
	if event == "" {
		event = "unknown"
	}

	var payload githubWebhookPayload
	_ = json.Unmarshal(body, &payload)

	c.JSON(200, webhookResponse{
		OK:      true,
		Message: fmt.Sprintf("webhook accepted (action=%s)", payload.Action),
		Event:   event,
	})
}

func verifyGitHubSignature(signatureHeader string, body []byte, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signatureHeader))
}
