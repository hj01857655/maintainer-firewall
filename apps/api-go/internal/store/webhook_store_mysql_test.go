package store

import (
	"database/sql"
	"encoding/json"
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
