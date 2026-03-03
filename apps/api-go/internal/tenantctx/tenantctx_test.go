package tenantctx

import (
	"context"
	"testing"
)

func TestWithTenantIDAndFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenantID(ctx, "acme")

	got, ok := FromContext(ctx)
	if !ok || got != "acme" {
		t.Fatalf("want acme, got %q ok=%v", got, ok)
	}
}

func TestWithTenantID_EmptyFallbackToDefault(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenantID(ctx, "  ")

	got, ok := FromContext(ctx)
	if !ok || got != DefaultTenantID {
		t.Fatalf("want default tenant, got %q ok=%v", got, ok)
	}
}

func TestMustFromContext_WithFallback(t *testing.T) {
	got := MustFromContext(context.Background(), "")
	if got != DefaultTenantID {
		t.Fatalf("want default fallback, got %q", got)
	}

	got = MustFromContext(context.Background(), "team-x")
	if got != "team-x" {
		t.Fatalf("want fallback tenant team-x, got %q", got)
	}
}
