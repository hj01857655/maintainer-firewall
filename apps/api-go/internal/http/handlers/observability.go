package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type ObservabilityStore interface {
	ListActionExecutionFailures(ctx context.Context, limit int, offset int, includeResolved bool) ([]store.ActionExecutionFailureRecord, int64, error)
	ListAuditLogs(ctx context.Context, limit int, offset int, actor string, action string, since *time.Time) ([]store.AuditLogRecord, int64, error)
	GetMetricsOverview(ctx context.Context, since time.Time) (store.MetricsOverview, error)
	GetMetricsTimeSeries(ctx context.Context, since time.Time, intervalMinutes int) ([]store.MetricsTimePoint, error)
}

type RuntimeConfigStatus struct {
	GitHubTokenConfigured         bool
	GitHubWebhookSecretConfigured bool
	DatabaseURLConfigured         bool
	JWTSecretConfigured           bool
	AdminUsernameConfigured       bool
	AdminPasswordConfigured       bool
}

type ObservabilityHandler struct {
	Store         ObservabilityStore
	RuntimeConfig RuntimeConfigStatus
}

func NewObservabilityHandler(s ObservabilityStore, cfg RuntimeConfigStatus) *ObservabilityHandler {
	return &ObservabilityHandler{Store: s, RuntimeConfig: cfg}
}

func (h *ObservabilityHandler) ConfigStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":                               true,
		"github_token_configured":          h.RuntimeConfig.GitHubTokenConfigured,
		"github_webhook_secret_configured": h.RuntimeConfig.GitHubWebhookSecretConfigured,
		"database_url_configured":          h.RuntimeConfig.DatabaseURLConfigured,
		"jwt_secret_configured":            h.RuntimeConfig.JWTSecretConfigured,
		"admin_username_configured":        h.RuntimeConfig.AdminUsernameConfigured,
		"admin_password_configured":        h.RuntimeConfig.AdminPasswordConfigured,
	})
}

type ConfigUpdateRequest struct {
	DatabaseURL         *string `json:"database_url"`
	AdminUsername       *string `json:"admin_username"`
	AdminPassword       *string `json:"admin_password"`
	JWTSecret           *string `json:"jwt_secret"`
	GitHubWebhookSecret *string `json:"github_webhook_secret"`
	GitHubToken         *string `json:"github_token"`
}

func (h *ObservabilityHandler) ConfigView(c *gin.Context) {
	vals, err := readEnvFile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("read .env failed: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":                    true,
		"database_url":          vals["DATABASE_URL"],
		"admin_username":        vals["ADMIN_USERNAME"],
		"admin_password_masked": maskSecret(vals["ADMIN_PASSWORD"]),
		"jwt_secret_masked":     maskSecret(vals["JWT_SECRET"]),
		"github_webhook_secret_masked": maskSecret(vals["GITHUB_WEBHOOK_SECRET"]),
		"github_token_masked":          maskSecret(vals["GITHUB_TOKEN"]),
	})
}

func (h *ObservabilityHandler) ConfigUpdate(c *gin.Context) {
	var req ConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": fmt.Sprintf("invalid body: %v", err)})
		return
	}
	vals, err := readEnvFile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("read .env failed: %v", err)})
		return
	}
	if req.DatabaseURL != nil {
		vals["DATABASE_URL"] = strings.TrimSpace(*req.DatabaseURL)
		_ = os.Setenv("DATABASE_URL", vals["DATABASE_URL"])
	}
	if req.AdminUsername != nil {
		vals["ADMIN_USERNAME"] = strings.TrimSpace(*req.AdminUsername)
		_ = os.Setenv("ADMIN_USERNAME", vals["ADMIN_USERNAME"])
	}
	if req.AdminPassword != nil {
		vals["ADMIN_PASSWORD"] = strings.TrimSpace(*req.AdminPassword)
		_ = os.Setenv("ADMIN_PASSWORD", vals["ADMIN_PASSWORD"])
	}
	if req.JWTSecret != nil {
		vals["JWT_SECRET"] = strings.TrimSpace(*req.JWTSecret)
		_ = os.Setenv("JWT_SECRET", vals["JWT_SECRET"])
	}
	if req.GitHubWebhookSecret != nil {
		vals["GITHUB_WEBHOOK_SECRET"] = strings.TrimSpace(*req.GitHubWebhookSecret)
		_ = os.Setenv("GITHUB_WEBHOOK_SECRET", vals["GITHUB_WEBHOOK_SECRET"])
	}
	if req.GitHubToken != nil {
		vals["GITHUB_TOKEN"] = strings.TrimSpace(*req.GitHubToken)
		_ = os.Setenv("GITHUB_TOKEN", vals["GITHUB_TOKEN"])
	}
	if err := writeEnvFile(vals); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("write .env failed: %v", err)})
		return
	}
	h.RuntimeConfig = RuntimeConfigStatus{
		GitHubTokenConfigured:         strings.TrimSpace(vals["GITHUB_TOKEN"]) != "",
		GitHubWebhookSecretConfigured: strings.TrimSpace(vals["GITHUB_WEBHOOK_SECRET"]) != "",
		DatabaseURLConfigured:         strings.TrimSpace(vals["DATABASE_URL"]) != "",
		JWTSecretConfigured:           strings.TrimSpace(vals["JWT_SECRET"]) != "",
		AdminUsernameConfigured:       strings.TrimSpace(vals["ADMIN_USERNAME"]) != "",
		AdminPasswordConfigured:       strings.TrimSpace(vals["ADMIN_PASSWORD"]) != "",
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "config saved", "restart_required": true})
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

	includeResolved := strings.EqualFold(c.DefaultQuery("include_resolved", "false"), "true")
	items, total, err := h.Store.ListActionExecutionFailures(ctx, limit, offset, includeResolved)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("list action failures failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":               true,
		"items":            items,
		"limit":            limit,
		"offset":           offset,
		"total":            total,
		"include_resolved": includeResolved,
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

	sinceStr := strings.TrimSpace(c.Query("since"))
	var since *time.Time
	if sinceStr != "" {
		parsed, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "since must be RFC3339"})
			return
		}
		t := parsed.UTC()
		since = &t
	}

	items, total, err := h.Store.ListAuditLogs(ctx, limit, offset, actor, action, since)
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
		"since":  since,
	})
}

func readEnvFile() (map[string]string, error) {
	path := filepath.Clean(".env")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return out, nil
}

func writeEnvFile(vals map[string]string) error {
	ordered := []string{"DATABASE_URL", "PORT", "ADMIN_USERNAME", "ADMIN_PASSWORD", "JWT_SECRET", "GITHUB_WEBHOOK_SECRET", "GITHUB_TOKEN"}
	lines := make([]string, 0, len(vals))
	seen := map[string]bool{}
	for _, k := range ordered {
		if v, ok := vals[k]; ok {
			lines = append(lines, fmt.Sprintf("%s=%s", k, v))
			seen[k] = true
		}
	}
	extra := make([]string, 0)
	for k := range vals {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		lines = append(lines, fmt.Sprintf("%s=%s", k, vals[k]))
	}
	return os.WriteFile(filepath.Clean(".env"), []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

func maskSecret(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return "******"
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
