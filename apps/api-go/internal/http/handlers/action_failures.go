package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"maintainer-firewall/api-go/internal/service"
	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type ActionFailureRetryStore interface {
	GetActionExecutionFailureByID(ctx context.Context, id int64) (store.ActionExecutionFailureRecord, error)
	UpdateActionFailureRetryResult(ctx context.Context, id int64, success bool, message string) error
	GetWebhookEventPayloadByDeliveryID(ctx context.Context, deliveryID string) (json.RawMessage, error)
	SaveAuditLog(ctx context.Context, item store.AuditLogRecord) error
}

type ActionFailureRetryHandler struct {
	Store    ActionFailureRetryStore
	Executor *service.GitHubActionExecutor
}

func NewActionFailureRetryHandler(s ActionFailureRetryStore, exec *service.GitHubActionExecutor) *ActionFailureRetryHandler {
	return &ActionFailureRetryHandler{Store: s, Executor: exec}
}

func (h *ActionFailureRetryHandler) Retry(c *gin.Context) {
	if h.Store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "store is not configured"})
		return
	}
	if h.Executor == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "executor is not configured"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid failure id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	failure, err := h.Store.GetActionExecutionFailureByID(ctx, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "failure not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("load failure failed: %v", err)})
		return
	}

	payloadBytes, err := h.Store.GetWebhookEventPayloadByDeliveryID(ctx, failure.DeliveryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("load related event failed: %v", err)})
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("parse related event failed: %v", err)})
		return
	}

	number := extractTargetNumber(failure.EventType, payload)
	if number <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid issue/pr number for retry"})
		return
	}

	switch failure.SuggestionType {
	case "label":
		err = h.Executor.AddLabel(ctx, failure.RepositoryFullName, number, failure.SuggestionValue)
	case "comment":
		err = h.Executor.AddComment(ctx, failure.RepositoryFullName, number, failure.SuggestionValue)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "unsupported suggestion type"})
		return
	}

	actor := strings.TrimSpace(c.GetString("actor"))
	if actor == "" {
		actor = "unknown"
	}

	if err != nil {
		_ = h.Store.UpdateActionFailureRetryResult(ctx, failure.ID, false, err.Error())
		_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
			Actor:    actor,
			Action:   "failure.retry.failed",
			Target:   "action_failure",
			TargetID: fmt.Sprintf("%d", failure.ID),
			Payload:  fmt.Sprintf(`{"delivery_id":"%s","error":"%s"}`, failure.DeliveryID, strings.ReplaceAll(err.Error(), `"`, `'`)),
		})

		status := http.StatusBadGateway
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "not configured") || strings.Contains(errMsg, "invalid ") || strings.Contains(errMsg, "empty ") || strings.Contains(errMsg, "unsupported") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"ok": false, "message": fmt.Sprintf("retry failed: %v", err)})
		return
	}

	_ = h.Store.UpdateActionFailureRetryResult(ctx, failure.ID, true, "retry succeeded")
	_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
		Actor:    actor,
		Action:   "failure.retry.success",
		Target:   "action_failure",
		TargetID: fmt.Sprintf("%d", failure.ID),
		Payload:  fmt.Sprintf(`{"delivery_id":"%s"}`, failure.DeliveryID),
	})

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "retry succeeded"})
}
