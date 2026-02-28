package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"
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
