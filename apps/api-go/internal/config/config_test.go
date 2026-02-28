package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_UsesDevDefaultsWhenEnvMissing(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("ADMIN_USERNAME", "")
	t.Setenv("ADMIN_PASSWORD", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ACCESS_TOKEN", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("BREEZELL_TEST_DOTENV_CONTENT", "")
	t.Setenv("BREEZELL_TEST_DOTENV_PATH", filepath.Join(t.TempDir(), "not-found.env"))

	cfg := Load()

	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %q", cfg.Port)
	}
	if cfg.AdminUsername != "admin" {
		t.Fatalf("expected default admin username, got %q", cfg.AdminUsername)
	}
	if cfg.AdminPassword != "admin123" {
		t.Fatalf("expected default admin password, got %q", cfg.AdminPassword)
	}
	if cfg.JWTSecret != "dev-jwt-secret" {
		t.Fatalf("expected default jwt secret, got %q", cfg.JWTSecret)
	}
	if cfg.GitHubWebhookSecret != "dev-webhook-secret" {
		t.Fatalf("expected default webhook secret, got %q", cfg.GitHubWebhookSecret)
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("expected database url to stay empty when not set, got %q", cfg.DatabaseURL)
	}
}

func TestLoad_JWTSecretFallsBackToAccessToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ACCESS_TOKEN", "legacy-token")
	t.Setenv("BREEZELL_TEST_DOTENV_CONTENT", "")
	t.Setenv("BREEZELL_TEST_DOTENV_PATH", filepath.Join(t.TempDir(), "not-found.env"))

	cfg := Load()
	if cfg.JWTSecret != "legacy-token" {
		t.Fatalf("expected jwt secret fallback to access token, got %q", cfg.JWTSecret)
	}
}

func TestLoad_UsesDotenvWhenEnvMissing(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("ADMIN_USERNAME", "")
	t.Setenv("ADMIN_PASSWORD", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ACCESS_TOKEN", "")
	t.Setenv("DATABASE_URL", "")

	t.Setenv("BREEZELL_TEST_DOTENV_CONTENT", "DATABASE_URL=mysql://dotenv-user:dotenv-pass@127.0.0.1:3306/dotenv_db\nADMIN_USERNAME=dotenv-admin\nJWT_SECRET=dotenv-jwt")
	t.Setenv("BREEZELL_TEST_DOTENV_PATH", "")

	cfg := Load()
	if cfg.DatabaseURL != "mysql://dotenv-user:dotenv-pass@127.0.0.1:3306/dotenv_db" {
		t.Fatalf("expected DATABASE_URL from dotenv, got %q", cfg.DatabaseURL)
	}
	if cfg.AdminUsername != "dotenv-admin" {
		t.Fatalf("expected ADMIN_USERNAME from dotenv, got %q", cfg.AdminUsername)
	}
	if cfg.JWTSecret != "dotenv-jwt" {
		t.Fatalf("expected JWT_SECRET from dotenv, got %q", cfg.JWTSecret)
	}
}

func TestLoad_AutoCreatesDotenvFromExample(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("ADMIN_USERNAME", "")
	t.Setenv("ADMIN_PASSWORD", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ACCESS_TOKEN", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("BREEZELL_TEST_DOTENV_CONTENT", "")

	tempDir := t.TempDir()
	dotenvPath := filepath.Join(tempDir, ".env")
	dotenvExamplePath := filepath.Join(tempDir, ".env.example")
	exampleContent := "DATABASE_URL=mysql://auto-user:auto-pass@127.0.0.1:3306/auto_db\nADMIN_USERNAME=auto-admin\n"
	if err := os.WriteFile(dotenvExamplePath, []byte(exampleContent), 0o600); err != nil {
		t.Fatalf("write .env.example: %v", err)
	}
	t.Setenv("BREEZELL_TEST_DOTENV_PATH", dotenvPath)

	cfg := Load()
	if cfg.DatabaseURL != "mysql://auto-user:auto-pass@127.0.0.1:3306/auto_db" {
		t.Fatalf("expected DATABASE_URL from auto-created dotenv, got %q", cfg.DatabaseURL)
	}

	generated, err := os.ReadFile(dotenvPath)
	if err != nil {
		t.Fatalf("expected .env generated from .env.example, read failed: %v", err)
	}
	if !strings.Contains(string(generated), "DATABASE_URL=mysql://auto-user:auto-pass@127.0.0.1:3306/auto_db") {
		t.Fatalf("generated .env content mismatch: %q", string(generated))
	}
}
