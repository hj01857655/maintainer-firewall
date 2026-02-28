package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"maintainer-firewall/api-go/internal/config"
	"maintainer-firewall/api-go/internal/http/handlers"
	"maintainer-firewall/api-go/internal/service"
	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	eventStore, err := store.NewWebhookEventStore(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to init webhook event store: %v", err)
	}
	defer eventStore.Close()

	webhookHandler := handlers.NewWebhookHandler(cfg.GitHubWebhookSecret, eventStore)
	webhookHandler.ActionExecutor = service.NewGitHubActionExecutor(cfg.GitHubToken)
	eventsHandler := handlers.NewEventsHandler(eventStore)
	alertsHandler := handlers.NewAlertsHandler(eventStore)
	rulesHandler := handlers.NewRulesHandler(eventStore)
	authHandler := handlers.NewAuthHandler(cfg.AdminUsername, cfg.AdminPassword, cfg.JWTSecret, 24*time.Hour)

	r := gin.Default()
	r.GET("/health", handlers.Health)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/webhook/github", webhookHandler.GitHub)

	api := r.Group("/")
	api.Use(handlers.AuthMiddleware(cfg.JWTSecret))
	api.GET("/events", eventsHandler.List)
	api.GET("/alerts", alertsHandler.List)
	api.GET("/rules", rulesHandler.List)
	api.POST("/rules", rulesHandler.Create)

	addr := fmt.Sprintf(":%s", cfg.Port)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}
