package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

var tenantIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{1,62}$`)

type TenantStore interface {
	ListTenants(ctx context.Context) ([]store.TenantRecord, error)
	CreateTenant(ctx context.Context, id string, name string) error
	UpdateTenantActive(ctx context.Context, id string, isActive bool) error
}

type TenantsHandler struct {
	Store TenantStore
}

type createTenantRequest struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type updateTenantActiveRequest struct {
	IsActive bool `json:"is_active"`
}

func NewTenantsHandler(store TenantStore) *TenantsHandler {
	return &TenantsHandler{Store: store}
}

func (h *TenantsHandler) List(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	items, err := h.Store.ListTenants(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("list tenants failed: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "items": items})
}

func (h *TenantsHandler) Create(c *gin.Context) {
	var req createTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid JSON payload"})
		return
	}

	tenantID := strings.TrimSpace(req.ID)
	tenantName := strings.TrimSpace(req.Name)
	if !tenantIDPattern.MatchString(tenantID) {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid tenant id"})
		return
	}
	if tenantName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "tenant name is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	if err := h.Store.CreateTenant(ctx, tenantID, tenantName); err != nil {
		if store.IsDuplicateKeyError(err) || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"ok": false, "message": "tenant already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("create tenant failed: %v", err)})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

func (h *TenantsHandler) UpdateActive(c *gin.Context) {
	tenantID := strings.TrimSpace(c.Param("id"))
	if !tenantIDPattern.MatchString(tenantID) {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid tenant id"})
		return
	}

	var req updateTenantActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid JSON payload"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	if err := h.Store.UpdateTenantActive(ctx, tenantID, req.IsActive); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "tenant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("update tenant failed: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
