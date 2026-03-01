package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	Store store.UserStore
}

type createUserRequest struct {
	Username   string   `json:"username" binding:"required,min=3,max=50"`
	Password   string   `json:"password" binding:"required,min=6"`
	Role       string   `json:"role" binding:"required,oneof=admin editor viewer"`
	Permissions []string `json:"permissions" binding:"required,min=1"`
	IsActive   bool     `json:"is_active"`
}

type updateUserRequest struct {
	Username   string   `json:"username" binding:"required,min=3,max=50"`
	Role       string   `json:"role" binding:"required,oneof=admin editor viewer"`
	Permissions []string `json:"permissions" binding:"required,min=1"`
	IsActive   bool     `json:"is_active"`
}

type updatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func NewUserHandler(store store.UserStore) *UserHandler {
	return &UserHandler{Store: store}
}

func (h *UserHandler) List(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	users, total, err := h.Store.ListAdminUsers(ctx, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("list users failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"users": users,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

func (h *UserHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid user id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	user, err := h.Store.GetAdminUserByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("get user failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "user": user})
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}

	// 验证角色和权限
	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid role"})
		return
	}

	if !areValidPermissions(req.Permissions) {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid permissions"})
		return
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "password hashing failed"})
		return
	}

	user := store.AdminUser{
		Username:     strings.TrimSpace(req.Username),
		PasswordHash: string(hashedPassword),
		IsActive:     req.IsActive,
		Role:         req.Role,
		Permissions:  req.Permissions,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, err := h.Store.CreateAdminUser(ctx, user)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			c.JSON(http.StatusConflict, gin.H{"ok": false, "message": "username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("create user failed: %v", err)})
		return
	}

	// 记录审计日志
	actor := strings.TrimSpace(c.GetString("actor"))
	if actor == "" {
		actor = "unknown"
	}
	_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
		Actor:    actor,
		Action:   "user.create",
		Target:   "user",
		TargetID: fmt.Sprintf("%d", id),
		Payload:  fmt.Sprintf(`{"username":"%s","role":"%s","is_active":%t}`, user.Username, user.Role, user.IsActive),
	})

	c.JSON(http.StatusCreated, gin.H{"ok": true, "id": id})
}

func (h *UserHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid user id"})
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}

	// 验证角色和权限
	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid role"})
		return
	}

	if !areValidPermissions(req.Permissions) {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid permissions"})
		return
	}

	user := store.AdminUser{
		Username:    strings.TrimSpace(req.Username),
		IsActive:    req.IsActive,
		Role:        req.Role,
		Permissions: req.Permissions,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err = h.Store.UpdateAdminUser(ctx, id, user)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "user not found"})
			return
		}
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			c.JSON(http.StatusConflict, gin.H{"ok": false, "message": "username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("update user failed: %v", err)})
		return
	}

	// 记录审计日志
	actor := strings.TrimSpace(c.GetString("actor"))
	if actor == "" {
		actor = "unknown"
	}
	_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
		Actor:    actor,
		Action:   "user.update",
		Target:   "user",
		TargetID: fmt.Sprintf("%d", id),
		Payload:  fmt.Sprintf(`{"username":"%s","role":"%s","is_active":%t}`, user.Username, user.Role, user.IsActive),
	})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *UserHandler) UpdatePassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid user id"})
		return
	}

	var req updatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}

	// 获取当前用户信息
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	user, err := h.Store.GetAdminUserByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("get user failed: %v", err)})
		return
	}

	// 验证当前密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "current password is incorrect"})
		return
	}

	// 哈希新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "password hashing failed"})
		return
	}

	// 更新密码
	user.PasswordHash = string(hashedPassword)
	err = h.Store.UpdateAdminUser(ctx, id, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("update password failed: %v", err)})
		return
	}

	// 记录审计日志
	actor := strings.TrimSpace(c.GetString("actor"))
	if actor == "" {
		actor = "unknown"
	}
	_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
		Actor:    actor,
		Action:   "user.update_password",
		Target:   "user",
		TargetID: fmt.Sprintf("%d", id),
		Payload:  `{"password_updated":true}`,
	})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *UserHandler) UpdateActive(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid user id"})
		return
	}

	var req struct {
		IsActive bool `json:"is_active" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err = h.Store.UpdateAdminUserActive(ctx, id, req.IsActive)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("update user active failed: %v", err)})
		return
	}

	// 记录审计日志
	actor := strings.TrimSpace(c.GetString("actor"))
	if actor == "" {
		actor = "unknown"
	}
	_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
		Actor:    actor,
		Action:   "user.update_active",
		Target:   "user",
		TargetID: fmt.Sprintf("%d", id),
		Payload:  fmt.Sprintf(`{"is_active":%t}`, req.IsActive),
	})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *UserHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid user id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err = h.Store.DeleteAdminUser(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": fmt.Sprintf("delete user failed: %v", err)})
		return
	}

	// 记录审计日志
	actor := strings.TrimSpace(c.GetString("actor"))
	if actor == "" {
		actor = "unknown"
	}
	_ = h.Store.SaveAuditLog(ctx, store.AuditLogRecord{
		Actor:    actor,
		Action:   "user.delete",
		Target:   "user",
		TargetID: fmt.Sprintf("%d", id),
		Payload:  `{"deleted":true}`,
	})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// 辅助函数
func isValidRole(role string) bool {
	validRoles := []string{"admin", "editor", "viewer"}
	for _, r := range validRoles {
		if r == role {
			return true
		}
	}
	return false
}

func areValidPermissions(permissions []string) bool {
	validPermissions := []string{"read", "write", "admin"}
	for _, p := range permissions {
		found := false
		for _, vp := range validPermissions {
			if vp == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return len(permissions) > 0
}
