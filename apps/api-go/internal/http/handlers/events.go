package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type WebhookEventLister interface {
	ListEvents(ctx context.Context, limit int, offset int) ([]store.WebhookEventRecord, error)
}

type EventsHandler struct {
	Store WebhookEventLister
}

type listEventsResponse struct {
	OK     bool                      `json:"ok"`
	Items  []store.WebhookEventRecord `json:"items"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

func NewEventsHandler(store WebhookEventLister) *EventsHandler {
	return &EventsHandler{Store: store}
}

func (h *EventsHandler) List(c *gin.Context) {
	if h.Store == nil {
		c.JSON(500, gin.H{"ok": false, "message": "event store is not configured"})
		return
	}

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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	items, err := h.Store.ListEvents(ctx, limit, offset)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "message": fmt.Sprintf("list events failed: %v", err)})
		return
	}

	c.JSON(200, listEventsResponse{
		OK:     true,
		Items:  items,
		Limit:  limit,
		Offset: offset,
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
