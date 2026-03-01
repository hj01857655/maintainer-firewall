package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"maintainer-firewall/api-go/internal/service"
	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type WebhookEventStore interface {
	ListEvents(ctx context.Context, limit int, offset int, eventType string, action string) ([]store.WebhookEventRecord, int64, error)
	ListEventFilterOptions(ctx context.Context) (store.EventFilterOptions, error)
	SaveEvent(ctx context.Context, evt store.WebhookEvent) error
}

type GitHubEventTypesProvider interface {
	ListRecentEventTypes(ctx context.Context) ([]string, error)
	ListRecentEvents(ctx context.Context) ([]service.GitHubUserEvent, error)
}

type GitHubSyncStatus struct {
	Running        bool       `json:"running"`
	LastStartedAt  *time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt *time.Time `json:"last_finished_at,omitempty"`
	LastSuccessAt  *time.Time `json:"last_success_at,omitempty"`
	LastSaved      int        `json:"last_saved"`
	LastTotal      int        `json:"last_total"`
	LastError      string     `json:"last_error,omitempty"`
	SuccessCount   int64      `json:"success_count"`
	FailureCount   int64      `json:"failure_count"`
}

type EventsHandler struct {
	Store          WebhookEventStore
	GitHubProvider GitHubEventTypesProvider

	syncMu     sync.Mutex
	syncStatus GitHubSyncStatus
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
		mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
		if mode == "" {
			mode = "types"
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()

		if syncEnabled {
			saved, total, err := h.SyncGitHubEvents(ctx)
			if err != nil {
				status := http.StatusBadGateway
				errMsg := strings.ToLower(err.Error())
				if strings.Contains(errMsg, "not configured") || strings.Contains(errMsg, "save github event failed") {
					status = http.StatusInternalServerError
				}
				if strings.Contains(errMsg, "already running") {
					status = http.StatusConflict
				}
				c.JSON(status, gin.H{"ok": false, "message": fmt.Sprintf("sync github events failed: %v", err)})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"ok":     true,
				"source": "github",
				"sync":   true,
				"saved":  saved,
				"total":  total,
			})
			return
		}

		if mode == "items" {
			limit := parseIntOrDefault(c.Query("limit"), 20)
			offset := parseIntOrDefault(c.Query("offset"), 0)
			if limit < 1 {
				limit = 1
			}
			if limit > 100 {
				limit = 100
			}
			if offset < 0 {
				offset = 0
			}

			events, err := h.GitHubProvider.ListRecentEvents(ctx)
			if err != nil {
				status := http.StatusBadGateway
				errMsg := strings.ToLower(err.Error())
				if strings.Contains(errMsg, "not configured") {
					status = http.StatusInternalServerError
				}
				c.JSON(status, gin.H{"ok": false, "message": fmt.Sprintf("list github events failed: %v", err)})
				return
			}

			total := len(events)
			if offset > total {
				offset = total
			}
			end := offset + limit
			if end > total {
				end = total
			}
			items := events[offset:end]

			c.JSON(http.StatusOK, gin.H{
				"ok":     true,
				"source": "github",
				"mode":   "items",
				"items":  items,
				"limit":  limit,
				"offset": offset,
				"total":  total,
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
			"mode":        "types",
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

func (h *EventsHandler) SyncGitHubEvents(ctx context.Context) (int, int, error) {
	h.syncMu.Lock()
	if h.syncStatus.Running {
		h.syncMu.Unlock()
		return 0, 0, fmt.Errorf("github events sync is already running")
	}
	now := time.Now().UTC()
	h.syncStatus.Running = true
	h.syncStatus.LastStartedAt = &now
	h.syncMu.Unlock()

	finish := func(saved int, total int, err error) (int, int, error) {
		h.syncMu.Lock()
		defer h.syncMu.Unlock()
		ended := time.Now().UTC()
		h.syncStatus.Running = false
		h.syncStatus.LastFinishedAt = &ended
		h.syncStatus.LastSaved = saved
		h.syncStatus.LastTotal = total
		if err != nil {
			h.syncStatus.LastError = err.Error()
			h.syncStatus.FailureCount++
			return saved, total, err
		}
		h.syncStatus.LastError = ""
		h.syncStatus.SuccessCount++
		h.syncStatus.LastSuccessAt = &ended
		return saved, total, nil
	}

	if h.GitHubProvider == nil {
		return finish(0, 0, fmt.Errorf("github provider is not configured"))
	}
	if h.Store == nil {
		return finish(0, 0, fmt.Errorf("event store is not configured"))
	}
	events, err := h.GitHubProvider.ListRecentEvents(ctx)
	if err != nil {
		return finish(0, 0, fmt.Errorf("sync github events failed: %w", err))
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
			return finish(saved, len(events), fmt.Errorf("save github event failed: %w", saveErr))
		}
		saved++
	}
	return finish(saved, len(events), nil)
}

func (h *EventsHandler) GitHubSyncStatus(c *gin.Context) {
	h.syncMu.Lock()
	status := h.syncStatus
	h.syncMu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"ok":     true,
		"source": "github",
		"status": status,
	})
}

func (h *EventsHandler) FilterOptions(c *gin.Context) {
	if h.Store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "event store is not configured"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	options, err := h.Store.ListEventFilterOptions(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("list event filter options failed: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "options": options})
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
