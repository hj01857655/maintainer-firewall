package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type AlertLister interface {
	ListAlerts(ctx context.Context, limit int, offset int, eventType string, action string, suggestionType string) ([]store.AlertRecord, int64, error)
	ListAlertFilterOptions(ctx context.Context) (store.AlertFilterOptions, error)
}

type AlertsHandler struct {
	Store AlertLister
}

type listAlertsResponse struct {
	OK             bool                `json:"ok"`
	Items          []store.AlertRecord `json:"items"`
	Limit          int                 `json:"limit"`
	Offset         int                 `json:"offset"`
	Total          int64               `json:"total"`
	EventType      string              `json:"event_type,omitempty"`
	Action         string              `json:"action,omitempty"`
	SuggestionType string              `json:"suggestion_type,omitempty"`
}

func NewAlertsHandler(store AlertLister) *AlertsHandler {
	return &AlertsHandler{Store: store}
}

func (h *AlertsHandler) List(c *gin.Context) {
	if h.Store == nil {
		c.JSON(500, gin.H{"ok": false, "message": "alert store is not configured"})
		return
	}

	limit := parseIntOrDefault(c.Query("limit"), 20)
	offset := parseIntOrDefault(c.Query("offset"), 0)
	eventType := c.Query("event_type")
	action := c.Query("action")
	suggestionType := c.Query("suggestion_type")

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

	items, total, err := h.Store.ListAlerts(ctx, limit, offset, eventType, action, suggestionType)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "message": fmt.Sprintf("list alerts failed: %v", err)})
		return
	}

	c.JSON(200, listAlertsResponse{
		OK:             true,
		Items:          items,
		Limit:          limit,
		Offset:         offset,
		Total:          total,
		EventType:      eventType,
		Action:         action,
		SuggestionType: suggestionType,
	})
}

func (h *AlertsHandler) FilterOptions(c *gin.Context) {
	if h.Store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "alert store is not configured"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	options, err := h.Store.ListAlertFilterOptions(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("list alert filter options failed: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "options": options})
}
