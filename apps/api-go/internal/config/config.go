package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

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
	loadDotenvIfPresent()

	port := getenvOrDefault("PORT", "8080")
	adminUsername := getenvOrDefault("ADMIN_USERNAME", "admin")
	adminPassword := getenvOrDefault("ADMIN_PASSWORD", "admin123")
	githubWebhookSecret := getenvOrDefault("GITHUB_WEBHOOK_SECRET", "dev-webhook-secret")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = os.Getenv("ACCESS_TOKEN")
	}
	if jwtSecret == "" {
		jwtSecret = "dev-jwt-secret"
	}

	return Config{
		Port:                port,
		GitHubWebhookSecret: githubWebhookSecret,
		GitHubToken:         os.Getenv("GITHUB_TOKEN"),
		AdminUsername:       adminUsername,
		AdminPassword:       adminPassword,
		JWTSecret:           jwtSecret,
		DatabaseURL:         os.Getenv("DATABASE_URL"),
	}
}

func getenvOrDefault(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func loadDotenvIfPresent() {
	if content := os.Getenv("BREEZELL_TEST_DOTENV_CONTENT"); content != "" {
		applyDotenvContent(content)
		return
	}

	dotenvPath := os.Getenv("BREEZELL_TEST_DOTENV_PATH")
	if dotenvPath == "" {
		dotenvPath = ".env"
	}

	if data, err := os.ReadFile(dotenvPath); err == nil {
		applyDotenvContent(string(data))
		return
	}

	if exampleData, ok := readDotenvExample(dotenvPath); ok {
		dotenvDir := filepath.Dir(dotenvPath)
		_ = os.MkdirAll(dotenvDir, 0o755)
		_ = os.WriteFile(dotenvPath, exampleData, 0o600)
		applyDotenvContent(string(exampleData))
		return
	}

	if absPath, absErr := filepath.Abs(dotenvPath); absErr == nil {
		if parentData, parentErr := os.ReadFile(filepath.Join(filepath.Dir(absPath), "..", "..", ".env")); parentErr == nil {
			applyDotenvContent(string(parentData))
		}
	}
}

func readDotenvExample(dotenvPath string) ([]byte, bool) {
	dotenvDir := filepath.Dir(dotenvPath)
	candidates := []string{
		filepath.Join(dotenvDir, ".env.example"),
		".env.example",
		filepath.Join("..", ".env.example"),
		filepath.Join("..", "..", ".env.example"),
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			return data, true
		}
	}
	return nil, false
}

func applyDotenvContent(content string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")
		value = strings.Trim(value, "'")
		if key == "" {
			continue
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}
