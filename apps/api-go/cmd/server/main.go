package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"maintainer-firewall/api-go/internal/config"
	"maintainer-firewall/api-go/internal/http/handlers"
	"maintainer-firewall/api-go/internal/service"
	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()

	eventStore, err := store.NewWebhookEventStore(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to init webhook event store: %v", err)
	}
	defer eventStore.Close()

	if cfg.BootstrapAdmin {
		adminName := strings.TrimSpace(cfg.AdminUsername)
		adminPass := strings.TrimSpace(cfg.AdminPassword)
		if adminName != "" && adminPass != "" {
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
			if hashErr != nil {
				log.Fatalf("failed to hash bootstrap admin password: %v", hashErr)
			}
			if err := eventStore.EnsureBootstrapAdminUser(context.Background(), adminName, string(hash)); err != nil {
				log.Fatalf("failed to bootstrap admin user: %v", err)
			}
		}
	}

	webhookHandler := handlers.NewWebhookHandler(cfg.GitHubWebhookSecret, eventStore)
	githubExecutor := service.NewGitHubActionExecutor(cfg.GitHubToken)
	webhookHandler.ActionExecutor = githubExecutor
	actionFailureRetryHandler := handlers.NewActionFailureRetryHandler(eventStore, githubExecutor)
	eventsHandler := handlers.NewEventsHandler(eventStore, githubExecutor)
	if cfg.GitHubSyncIntervalMinute > 0 {
		interval := time.Duration(cfg.GitHubSyncIntervalMinute) * time.Minute
		service.StartGitHubEventsSyncWorker(context.Background(), interval, eventsHandler.SyncGitHubEvents)
		log.Printf("github events sync worker enabled: interval=%s", interval)
	}
	alertsHandler := handlers.NewAlertsHandler(eventStore)
	rulesHandler := handlers.NewRulesHandler(eventStore)
	usersHandler := handlers.NewUserHandler(eventStore)
	tenantsHandler := handlers.NewTenantsHandler(eventStore)
	observabilityHandler := handlers.NewObservabilityHandler(eventStore, handlers.RuntimeConfigStatus{
		GitHubTokenConfigured:         cfg.GitHubToken != "",
		GitHubWebhookSecretConfigured: cfg.GitHubWebhookSecret != "",
		DatabaseURLConfigured:         cfg.DatabaseURL != "",
		JWTSecretConfigured:           cfg.JWTSecret != "",
		AdminUsernameConfigured:       cfg.AdminUsername != "",
		AdminPasswordConfigured:       cfg.AdminPassword != "",
	})
	authHandler := handlers.NewAuthHandlerWithStore(eventStore, cfg.AdminUsername, cfg.AdminPassword, cfg.JWTSecret, 24*time.Hour, cfg.AuthEnvFallback)

	r := gin.Default()

	// CORS中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/health", handlers.Health)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/webhook/github", webhookHandler.GitHub)

	api := r.Group("/api")
	api.Use(handlers.AuthMiddleware(cfg.JWTSecret))
	readAPI := api.Group("")
	readAPI.Use(handlers.RequirePermission("read"))
	readAPI.GET("/events", eventsHandler.List)
	readAPI.GET("/events/filter-options", eventsHandler.FilterOptions)
	readAPI.GET("/events/sync-status", eventsHandler.GitHubSyncStatus)
	readAPI.GET("/alerts", alertsHandler.List)
	readAPI.GET("/alerts/filter-options", alertsHandler.FilterOptions)
	readAPI.GET("/rules", rulesHandler.List)
	readAPI.GET("/rules/filter-options", rulesHandler.FilterOptions)
	readAPI.GET("/rules/versions", rulesHandler.ListVersions)
	readAPI.POST("/rules/replay", rulesHandler.Replay)
	readAPI.GET("/users", usersHandler.List)
	readAPI.GET("/users/:id", usersHandler.GetByID)
	readAPI.GET("/tenants", tenantsHandler.List)
	readAPI.GET("/config-status", observabilityHandler.ConfigStatus)
	readAPI.GET("/config-view", observabilityHandler.ConfigView)
	readAPI.GET("/metrics/overview", observabilityHandler.MetricsOverview)
	readAPI.GET("/metrics/timeseries", observabilityHandler.MetricsTimeSeries)
	readAPI.GET("/action-failures", observabilityHandler.ActionFailures)
	readAPI.GET("/audit-logs", observabilityHandler.AuditLogs)

	writeAPI := api.Group("")
	writeAPI.Use(handlers.RequirePermission("write"))
	writeAPI.POST("/rules", rulesHandler.Create)
	writeAPI.PATCH("/rules/:id/active", rulesHandler.UpdateActive)
	writeAPI.POST("/rules/publish", rulesHandler.PublishVersion)
	writeAPI.POST("/users", usersHandler.Create)
	writeAPI.PUT("/users/:id", usersHandler.Update)
	writeAPI.PUT("/users/:id/password", usersHandler.UpdatePassword)
	writeAPI.PATCH("/users/:id/active", usersHandler.UpdateActive)
	writeAPI.POST("/action-failures/:id/retry", actionFailureRetryHandler.Retry)

	adminAPI := api.Group("")
	adminAPI.Use(handlers.RequirePermission("admin"))
	adminAPI.POST("/tenants", tenantsHandler.Create)

	dangerAdminAPI := api.Group("")
	dangerAdminAPI.Use(handlers.RequirePermission("admin"), handlers.RequireDangerConfirm())
	dangerAdminAPI.DELETE("/users/:id", usersHandler.Delete)
	dangerAdminAPI.PATCH("/tenants/:id/active", tenantsHandler.UpdateActive)
	dangerAdminAPI.POST("/config-update", observabilityHandler.ConfigUpdate)
	dangerAdminAPI.POST("/rules/rollback", rulesHandler.Rollback)

	addr := fmt.Sprintf(":%s", cfg.Port)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}
