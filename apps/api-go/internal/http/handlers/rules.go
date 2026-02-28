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

type RuleManager interface {
	ListRules(ctx context.Context, limit int, offset int, eventType string, keyword string, activeOnly bool) ([]store.RuleRecord, int64, error)
	CreateRule(ctx context.Context, rule store.RuleRecord) (int64, error)
}

type RulesHandler struct {
	Store RuleManager
}

type listRulesResponse struct {
	OK         bool               `json:"ok"`
	Items      []store.RuleRecord `json:"items"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
	Total      int64              `json:"total"`
	EventType  string             `json:"event_type,omitempty"`
	Keyword    string             `json:"keyword,omitempty"`
	ActiveOnly bool               `json:"active_only"`
}

type createRuleRequest struct {
	EventType       string `json:"event_type"`
	Keyword         string `json:"keyword"`
	SuggestionType  string `json:"suggestion_type"`
	SuggestionValue string `json:"suggestion_value"`
	Reason          string `json:"reason"`
	IsActive        bool   `json:"is_active"`
}

func NewRulesHandler(store RuleManager) *RulesHandler {
	return &RulesHandler{Store: store}
}

func (h *RulesHandler) List(c *gin.Context) {
	if h.Store == nil {
		c.JSON(500, gin.H{"ok": false, "message": "rule store is not configured"})
		return
	}

	limit := parseIntOrDefault(c.Query("limit"), 20)
	offset := parseIntOrDefault(c.Query("offset"), 0)
	eventType := c.Query("event_type")
	keyword := c.Query("keyword")
	activeOnly := strings.EqualFold(c.DefaultQuery("active_only", "true"), "true")

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

	items, total, err := h.Store.ListRules(ctx, limit, offset, eventType, keyword, activeOnly)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "message": fmt.Sprintf("list rules failed: %v", err)})
		return
	}

	c.JSON(200, listRulesResponse{
		OK:         true,
		Items:      items,
		Limit:      limit,
		Offset:     offset,
		Total:      total,
		EventType:  eventType,
		Keyword:    keyword,
		ActiveOnly: activeOnly,
	})
}

func (h *RulesHandler) Create(c *gin.Context) {
	if h.Store == nil {
		c.JSON(500, gin.H{"ok": false, "message": "rule store is not configured"})
		return
	}

	var req createRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid JSON payload"})
		return
	}

	req.EventType = strings.TrimSpace(req.EventType)
	req.Keyword = strings.TrimSpace(req.Keyword)
	req.SuggestionType = strings.TrimSpace(req.SuggestionType)
	req.SuggestionValue = strings.TrimSpace(req.SuggestionValue)
	req.Reason = strings.TrimSpace(req.Reason)

	if req.EventType == "" || req.Keyword == "" || req.SuggestionType == "" || req.SuggestionValue == "" || req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "event_type, keyword, suggestion_type, suggestion_value, reason are required"})
		return
	}

	if req.EventType != "issues" && req.EventType != "pull_request" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "event_type must be issues or pull_request"})
		return
	}
	if req.SuggestionType != "label" && req.SuggestionType != "comment" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "suggestion_type must be label or comment"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	id, err := h.Store.CreateRule(ctx, store.RuleRecord{
		EventType:       req.EventType,
		Keyword:         req.Keyword,
		SuggestionType:  req.SuggestionType,
		SuggestionValue: req.SuggestionValue,
		Reason:          req.Reason,
		IsActive:        req.IsActive,
	})
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "message": fmt.Sprintf("create rule failed: %v", err)})
		return
	}

	c.JSON(200, gin.H{"ok": true, "id": id})
}
