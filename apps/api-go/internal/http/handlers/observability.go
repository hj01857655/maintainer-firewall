package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type ObservabilityStore interface {
	ListActionExecutionFailures(ctx context.Context, limit int, offset int) ([]store.ActionExecutionFailureRecord, int64, error)
	ListAuditLogs(ctx context.Context, limit int, offset int, actor string, action string) ([]store.AuditLogRecord, int64, error)
	GetMetricsOverview(ctx context.Context, since time.Time) (store.MetricsOverview, error)
	GetMetricsTimeSeries(ctx context.Context, since time.Time, intervalMinutes int) ([]store.MetricsTimePoint, error)
}

type ObservabilityHandler struct {
	Store ObservabilityStore
}

func NewObservabilityHandler(s ObservabilityStore) *ObservabilityHandler {
	return &ObservabilityHandler{Store: s}
}

func (h *ObservabilityHandler) MetricsOverview(c *gin.Context) {
	if h.Store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "store is not configured"})
		return
	}

	window := strings.TrimSpace(c.DefaultQuery("window", "24h"))
	since, err := parseWindowStart(window)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	overview, err := h.Store.GetMetricsOverview(ctx, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("get metrics overview failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"window":  window,
		"since":   since,
		"overview": overview,
	})
}

func (h *ObservabilityHandler) MetricsTimeSeries(c *gin.Context) {
	if h.Store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "store is not configured"})
		return
	}

	window := strings.TrimSpace(c.DefaultQuery("window", "24h"))
	intervalMinutes := parseIntOrDefault(c.DefaultQuery("interval_minutes", "60"), 60)
	if intervalMinutes <= 0 {
		intervalMinutes = 60
	}

	since, err := parseWindowStart(window)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	items, err := h.Store.GetMetricsTimeSeries(ctx, since, intervalMinutes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("get metrics timeseries failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":               true,
		"window":           window,
		"interval_minutes": intervalMinutes,
		"since":            since,
		"items":            items,
	})
}

func (h *ObservabilityHandler) ActionFailures(c *gin.Context) {
	if h.Store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "store is not configured"})
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

	items, total, err := h.Store.ListActionExecutionFailures(ctx, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("list action failures failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":     true,
		"items":  items,
		"limit":  limit,
		"offset": offset,
		"total":  total,
	})
}

func (h *ObservabilityHandler) AuditLogs(c *gin.Context) {
	if h.Store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "store is not configured"})
		return
	}

	limit := parseIntOrDefault(c.Query("limit"), 20)
	offset := parseIntOrDefault(c.Query("offset"), 0)
	actor := strings.TrimSpace(c.Query("actor"))
	action := strings.TrimSpace(c.Query("action"))
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

	items, total, err := h.Store.ListAuditLogs(ctx, limit, offset, actor, action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("list audit logs failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":     true,
		"items":  items,
		"limit":  limit,
		"offset": offset,
		"total":  total,
		"actor":  actor,
		"action": action,
	})
}

func parseWindowStart(v string) (time.Time, error) {
	now := time.Now().UTC()
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "24h", "1d", "day":
		return now.Add(-24 * time.Hour), nil
	case "12h":
		return now.Add(-12 * time.Hour), nil
	case "6h":
		return now.Add(-6 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("window must be one of: 6h, 12h, 24h")
	}
}
