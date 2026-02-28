package config

import "os"

type Config struct {
	Port                string
	GitHubWebhookSecret string
	GitHubToken         string
	AdminUsername       string
	AdminPassword       string
	JWTSecret           string
	DatabaseURL         string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = os.Getenv("ACCESS_TOKEN")
	}

	return Config{
		Port:                port,
		GitHubWebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		GitHubToken:         os.Getenv("GITHUB_TOKEN"),
		AdminUsername:       os.Getenv("ADMIN_USERNAME"),
		AdminPassword:       os.Getenv("ADMIN_PASSWORD"),
		JWTSecret:           jwtSecret,
		DatabaseURL:         os.Getenv("DATABASE_URL"),
	}
}
