package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"maintainer-firewall/api-go/internal/service"
	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type WebhookEventStore interface {
	ListEvents(ctx context.Context, limit int, offset int, eventType string, action string) ([]store.WebhookEventRecord, int64, error)
	SaveEvent(ctx context.Context, evt store.WebhookEvent) error
}

type GitHubEventTypesProvider interface {
	ListRecentEventTypes(ctx context.Context) ([]string, error)
	ListRecentEvents(ctx context.Context) ([]service.GitHubUserEvent, error)
}

type EventsHandler struct {
	Store          WebhookEventStore
	GitHubProvider GitHubEventTypesProvider
}

type listEventsResponse struct {
	OK        bool                       `json:"ok"`
	Items     []store.WebhookEventRecord `json:"items"`
	Limit     int                        `json:"limit"`
	Offset    int                        `json:"offset"`
	Total     int64                      `json:"total"`
	EventType string                     `json:"event_type,omitempty"`
	Action    string                     `json:"action,omitempty"`
}

func NewEventsHandler(store WebhookEventStore, githubProvider GitHubEventTypesProvider) *EventsHandler {
	return &EventsHandler{Store: store, GitHubProvider: githubProvider}
}

func (h *EventsHandler) List(c *gin.Context) {
	source := strings.ToLower(strings.TrimSpace(c.Query("source")))
	if source == "github" {
		if h.GitHubProvider == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "github provider is not configured"})
			return
		}
		syncEnabled := strings.EqualFold(strings.TrimSpace(c.Query("sync")), "true")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()

		if syncEnabled {
			if h.Store == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "event store is not configured"})
				return
			}
			events, err := h.GitHubProvider.ListRecentEvents(ctx)
			if err != nil {
				status := http.StatusBadGateway
				errMsg := strings.ToLower(err.Error())
				if strings.Contains(errMsg, "not configured") {
					status = http.StatusInternalServerError
				}
				c.JSON(status, gin.H{"ok": false, "message": fmt.Sprintf("sync github events failed: %v", err)})
				return
			}
			saved := 0
			for _, evt := range events {
				saveErr := h.Store.SaveEvent(ctx, store.WebhookEvent{
					DeliveryID:         evt.DeliveryID,
					EventType:          evt.EventType,
					Action:             evt.Action,
					RepositoryFullName: evt.RepositoryFullName,
					SenderLogin:        evt.SenderLogin,
					PayloadJSON:        evt.PayloadJSON,
				})
				if saveErr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("save github event failed: %v", saveErr)})
					return
				}
				saved++
			}
			c.JSON(http.StatusOK, gin.H{
				"ok":     true,
				"source": "github",
				"sync":   true,
				"saved":  saved,
				"total":  len(events),
			})
			return
		}

		types, err := h.GitHubProvider.ListRecentEventTypes(ctx)
		if err != nil {
			status := http.StatusBadGateway
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "not configured") {
				status = http.StatusInternalServerError
			}
			c.JSON(status, gin.H{"ok": false, "message": fmt.Sprintf("list github events failed: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"ok":          true,
			"source":      "github",
			"event_types": types,
			"total":       len(types),
		})
		return
	}

	if h.Store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "event store is not configured"})
		return
	}

	limit := parseIntOrDefault(c.Query("limit"), 20)
	offset := parseIntOrDefault(c.Query("offset"), 0)
	eventType := c.Query("event_type")
	action := c.Query("action")

	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	items, total, err := h.Store.ListEvents(ctx, limit, offset, eventType, action)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "message": fmt.Sprintf("list events failed: %v", err)})
		return
	}

	c.JSON(200, listEventsResponse{
		OK:        true,
		Items:     items,
		Limit:     limit,
		Offset:    offset,
		Total:     total,
		EventType: eventType,
		Action:    action,
	})
}

func parseIntOrDefault(v string, fallback int) int {
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
