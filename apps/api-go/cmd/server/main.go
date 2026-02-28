package main

import (
	"fmt"

	"maintainer-firewall/api-go/internal/config"
	"maintainer-firewall/api-go/internal/http/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	webhookHandler := handlers.NewWebhookHandler(cfg.GitHubWebhookSecret)

	r := gin.Default()
	r.GET("/health", handlers.Health)
	r.POST("/webhook/github", webhookHandler.GitHub)

	addr := fmt.Sprintf(":%s", cfg.Port)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}
