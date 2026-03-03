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

type RuleManager interface {
	ListRules(ctx context.Context, limit int, offset int, eventType string, keyword string, activeOnly bool) ([]store.RuleRecord, int64, error)
	ListRuleFilterOptions(ctx context.Context) (store.RuleFilterOptions, error)
	CreateRule(ctx context.Context, rule store.RuleRecord) (int64, error)
	UpdateRuleActive(ctx context.Context, id int64, isActive bool) error
	CreateRuleVersionSnapshot(ctx context.Context, createdBy string, sourceVersion int64) (int64, int, error)
	ListRuleVersions(ctx context.Context, limit int, offset int) ([]store.RuleVersionRecord, int64, error)
	GetRulesByVersion(ctx context.Context, version int64) ([]store.RuleRecord, error)
	RestoreRulesFromVersion(ctx context.Context, version int64) (int, error)
	SaveAuditLog(ctx context.Context, item store.AuditLogRecord) error
}

type RulesHandler struct {
	Store      RuleManager
	RuleEngine *service.RuleEngine
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

type updateRuleActiveRequest struct {
	IsActive bool `json:"is_active"`
}

type publishRulesVersionResponse struct {
	OK        bool  `json:"ok"`
	Version   int64 `json:"version"`
	RuleCount int   `json:"rule_count"`
}

type listRuleVersionsResponse struct {
	OK     bool                      `json:"ok"`
	Items  []store.RuleVersionRecord `json:"items"`
	Total  int64                     `json:"total"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

type rollbackRulesRequest struct {
	Version int64 `json:"version"`
}

type replayRulesRequest struct {
	Version   int64          `json:"version"`
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
}

func NewRulesHandler(store RuleManager) *RulesHandler {
	return &RulesHandler{Store: store, RuleEngine: service.NewRuleEngine()}
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

func (h *RulesHandler) FilterOptions(c *gin.Context) {
	if h.Store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "rule store is not configured"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	options, err := h.Store.ListRuleFilterOptions(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("list rule filter options failed: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "options": options})
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

	actor := strings.TrimSpace(c.GetString("actor"))
	if actor == "" {
		actor = "unknown"
	}
	_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
		Actor:    actor,
		Action:   "rule.create",
		Target:   "rule",
		TargetID: fmt.Sprintf("%d", id),
		Payload:  fmt.Sprintf(`{"event_type":"%s","keyword":"%s","suggestion_type":"%s","is_active":%t}`, req.EventType, req.Keyword, req.SuggestionType, req.IsActive),
	})

	c.JSON(200, gin.H{"ok": true, "id": id})
}

func (h *RulesHandler) UpdateActive(c *gin.Context) {
	if h.Store == nil {
		c.JSON(500, gin.H{"ok": false, "message": "rule store is not configured"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid rule id"})
		return
	}

	var req updateRuleActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid JSON payload"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	if err := h.Store.UpdateRuleActive(ctx, id, req.IsActive); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("update rule active failed: %v", err)})
		return
	}

	actor := strings.TrimSpace(c.GetString("actor"))
	if actor == "" {
		actor = "unknown"
	}
	_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
		Actor:    actor,
		Action:   "rule.update_active",
		Target:   "rule",
		TargetID: fmt.Sprintf("%d", id),
		Payload:  fmt.Sprintf(`{"is_active":%t}`, req.IsActive),
	})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *RulesHandler) PublishVersion(c *gin.Context) {
	if h.Store == nil {
		c.JSON(500, gin.H{"ok": false, "message": "rule store is not configured"})
		return
	}
	actor := strings.TrimSpace(c.GetString("actor"))
	if actor == "" {
		actor = "unknown"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	version, count, err := h.Store.CreateRuleVersionSnapshot(ctx, actor, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("publish rule version failed: %v", err)})
		return
	}
	_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
		Actor:    actor,
		Action:   "rule.publish",
		Target:   "rule_version",
		TargetID: fmt.Sprintf("%d", version),
		Payload:  fmt.Sprintf(`{"version":%d,"rule_count":%d}`, version, count),
	})
	c.JSON(http.StatusOK, publishRulesVersionResponse{OK: true, Version: version, RuleCount: count})
}

func (h *RulesHandler) ListVersions(c *gin.Context) {
	if h.Store == nil {
		c.JSON(500, gin.H{"ok": false, "message": "rule store is not configured"})
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
	items, total, err := h.Store.ListRuleVersions(ctx, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("list rule versions failed: %v", err)})
		return
	}
	c.JSON(http.StatusOK, listRuleVersionsResponse{
		OK:     true,
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *RulesHandler) Rollback(c *gin.Context) {
	if h.Store == nil {
		c.JSON(500, gin.H{"ok": false, "message": "rule store is not configured"})
		return
	}
	var req rollbackRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "version is required"})
		return
	}
	actor := strings.TrimSpace(c.GetString("actor"))
	if actor == "" {
		actor = "unknown"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	restoredCount, err := h.Store.RestoreRulesFromVersion(ctx, req.Version)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "rule version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("rollback rules failed: %v", err)})
		return
	}
	newVersion, _, err := h.Store.CreateRuleVersionSnapshot(ctx, actor, req.Version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("snapshot after rollback failed: %v", err)})
		return
	}
	_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
		Actor:    actor,
		Action:   "rule.rollback",
		Target:   "rule_version",
		TargetID: fmt.Sprintf("%d", req.Version),
		Payload:  fmt.Sprintf(`{"from_version":%d,"to_version":%d,"restored_count":%d}`, req.Version, newVersion, restoredCount),
	})
	c.JSON(http.StatusOK, gin.H{
		"ok":             true,
		"from_version":   req.Version,
		"to_version":     newVersion,
		"restored_count": restoredCount,
	})
}

func (h *RulesHandler) Replay(c *gin.Context) {
	if h.Store == nil {
		c.JSON(500, gin.H{"ok": false, "message": "rule store is not configured"})
		return
	}
	var req replayRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid JSON payload"})
		return
	}
	req.EventType = strings.TrimSpace(req.EventType)
	if req.EventType != "issues" && req.EventType != "pull_request" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "event_type must be issues or pull_request"})
		return
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}
	if h.RuleEngine == nil {
		h.RuleEngine = service.NewRuleEngine()
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var rules []store.RuleRecord
	var err error
	if req.Version > 0 {
		rules, err = h.Store.GetRulesByVersion(ctx, req.Version)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "rule version not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("load rule version failed: %v", err)})
			return
		}
	} else {
		rules, _, err = h.Store.ListRules(ctx, 1000, 0, "", "", true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("list rules failed: %v", err)})
			return
		}
	}

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

	suggestions := h.RuleEngine.EvaluateWithRules(req.EventType, req.Payload, defs)
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"version":     req.Version,
		"rule_count":  len(rules),
		"suggestions": suggestions,
	})
}
