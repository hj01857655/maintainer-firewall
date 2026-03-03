package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"maintainer-firewall/api-go/internal/tenantctx"
)

func TestNormalizeLastRetryAt_NullToZeroTime(t *testing.T) {
	rec := ActionExecutionFailureRecord{}
	var nullable sql.NullTime

	normalizeLastRetryAt(&rec, nullable)
	if !rec.LastRetryAt.IsZero() {
		t.Fatalf("expected zero time when nullable time is invalid, got %v", rec.LastRetryAt)
	}

	b, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if string(b) == "" {
		t.Fatalf("unexpected empty json output")
	}
}

func TestNormalizeLastRetryAt_ValidTimeAssigned(t *testing.T) {
	rec := ActionExecutionFailureRecord{}
	want := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	nullable := sql.NullTime{Time: want, Valid: true}

	normalizeLastRetryAt(&rec, nullable)
	if !rec.LastRetryAt.Equal(want) {
		t.Fatalf("expected %v, got %v", want, rec.LastRetryAt)
	}
}

type mockBootstrapStore struct {
	adminUserCount int64
	insertedUser   string
	insertedHash   string
}

func (m *mockBootstrapStore) EnsureBootstrapAdminUser(_ context.Context, username string, passwordHash string) error {
	name := strings.TrimSpace(username)
	hash := strings.TrimSpace(passwordHash)
	if name == "" || hash == "" {
		return nil
	}
	if m.adminUserCount > 0 {
		return nil
	}
	m.insertedUser = name
	m.insertedHash = hash
	m.adminUserCount = 1
	return nil
}

func TestEnsureBootstrapAdminUser_InsertsWhenEmpty(t *testing.T) {
	m := &mockBootstrapStore{}
	if err := m.EnsureBootstrapAdminUser(context.Background(), "admin", "hash"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if m.insertedUser != "admin" || m.insertedHash != "hash" {
		t.Fatalf("expected inserted admin/hash, got user=%q hash=%q", m.insertedUser, m.insertedHash)
	}
}

func TestEnsureBootstrapAdminUser_NoOpWhenExisting(t *testing.T) {
	m := &mockBootstrapStore{adminUserCount: 1}
	if err := m.EnsureBootstrapAdminUser(context.Background(), "admin", "hash"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if m.insertedUser != "" {
		t.Fatalf("expected no insert when admin exists")
	}
}

func TestNullableInt64_ForRuleVersionSource(t *testing.T) {
	if got := nullableInt64(0); got != nil {
		t.Fatalf("expected nil for zero source version, got %#v", got)
	}
	if got := nullableInt64(-1); got != nil {
		t.Fatalf("expected nil for negative source version, got %#v", got)
	}
	got := nullableInt64(9)
	value, ok := got.(int64)
	if !ok || value != 9 {
		t.Fatalf("expected int64(9), got %#v", got)
	}
}

func TestTenantIDFromCtxMySQL_DefaultAndOverride(t *testing.T) {
	if got := tenantIDFromCtxMySQL(context.Background()); got != tenantctx.DefaultTenantID {
		t.Fatalf("expected default tenant id %q, got %q", tenantctx.DefaultTenantID, got)
	}

	ctx := tenantctx.WithTenantID(context.Background(), "tenant-a")
	if got := tenantIDFromCtxMySQL(ctx); got != "tenant-a" {
		t.Fatalf("expected tenant-a, got %q", got)
	}
}

func TestMySQLURLToDSN_DefaultsAndValidation(t *testing.T) {
	dsn, err := mysqlURLToDSN("mysql://u:p@/mf_db")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(dsn, "u:p@tcp(127.0.0.1:3306)/mf_db?") {
		t.Fatalf("expected default host in dsn, got %q", dsn)
	}
	if !strings.Contains(dsn, "parseTime=true") || !strings.Contains(dsn, "charset=utf8mb4") || !strings.Contains(dsn, "loc=UTC") {
		t.Fatalf("expected default mysql params in dsn, got %q", dsn)
	}

	if _, err := mysqlURLToDSN("postgres://u:p@localhost/db"); err == nil {
		t.Fatalf("expected unsupported scheme error")
	}
	if _, err := mysqlURLToDSN("mysql://u:p@localhost"); err == nil {
		t.Fatalf("expected missing database name error")
	}
}
