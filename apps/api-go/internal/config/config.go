package config

import "os"

type Config struct {
	Port                string
	GitHubWebhookSecret string
	GitHubToken         string
	DatabaseURL         string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:                port,
		GitHubWebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		GitHubToken:         os.Getenv("GITHUB_TOKEN"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
	}
}
