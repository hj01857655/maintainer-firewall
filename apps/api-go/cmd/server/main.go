package main

import (
	"context"
	"fmt"
	"log"

	"maintainer-firewall/api-go/internal/config"
	"maintainer-firewall/api-go/internal/http/handlers"
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
	eventsHandler := handlers.NewEventsHandler(eventStore)
	alertsHandler := handlers.NewAlertsHandler(eventStore)
	rulesHandler := handlers.NewRulesHandler(eventStore)

	r := gin.Default()
	r.GET("/health", handlers.Health)
	r.GET("/events", eventsHandler.List)
	r.GET("/alerts", alertsHandler.List)
	r.GET("/rules", rulesHandler.List)
	r.POST("/rules", rulesHandler.Create)
	r.POST("/webhook/github", webhookHandler.GitHub)

	addr := fmt.Sprintf(":%s", cfg.Port)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}
