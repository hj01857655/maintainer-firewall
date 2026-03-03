package tenantctx

import (
	"context"
	"strings"
)

type contextKey string

const (
	tenantIDKey     contextKey = "tenant_id"
	DefaultTenantID            = "default"
)

func normalizeTenantID(raw string) string {
	tenantID := strings.TrimSpace(raw)
	if tenantID == "" {
		return DefaultTenantID
	}
	return tenantID
}

func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, normalizeTenantID(tenantID))
}

func FromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	v := ctx.Value(tenantIDKey)
	if v == nil {
		return "", false
	}

	tenantID, ok := v.(string)
	if !ok {
		return "", false
	}

	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return "", false
	}

	return tenantID, true
}

func MustFromContext(ctx context.Context, fallback string) string {
	if tenantID, ok := FromContext(ctx); ok {
		return tenantID
	}
	return normalizeTenantID(fallback)
}
