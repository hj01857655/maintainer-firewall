package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookEvent struct {
	DeliveryID         string
	EventType          string
	Action             string
	RepositoryFullName string
	SenderLogin        string
	PayloadJSON        json.RawMessage
}

type WebhookEventStore struct {
	pool *pgxpool.Pool
}

type WebhookEventRecord struct {
	ID                 int64     `json:"id"`
	DeliveryID         string    `json:"delivery_id"`
	EventType          string    `json:"event_type"`
	Action             string    `json:"action"`
	RepositoryFullName string    `json:"repository_full_name"`
	SenderLogin        string    `json:"sender_login"`
	ReceivedAt         time.Time `json:"received_at"`
}

type AlertRecord struct {
	DeliveryID         string    `json:"delivery_id"`
	EventType          string    `json:"event_type"`
	Action             string    `json:"action"`
	RepositoryFullName string    `json:"repository_full_name"`
	SenderLogin        string    `json:"sender_login"`
	RuleMatched        string    `json:"rule_matched"`
	SuggestionType     string    `json:"suggestion_type"`
	SuggestionValue    string    `json:"suggestion_value"`
	Reason             string    `json:"reason"`
	CreatedAt          time.Time `json:"created_at,omitempty"`
}

type RuleRecord struct {
	ID              int64     `json:"id"`
	EventType       string    `json:"event_type"`
	Keyword         string    `json:"keyword"`
	SuggestionType  string    `json:"suggestion_type"`
	SuggestionValue string    `json:"suggestion_value"`
	Reason          string    `json:"reason"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}

func NewWebhookEventStore(ctx context.Context, databaseURL string) (*WebhookEventStore, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DATABASE_URL is not configured")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	store := &WebhookEventStore{pool: pool}
	if err := store.ensureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return store, nil
}

func (s *WebhookEventStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *WebhookEventStore) SaveEvent(ctx context.Context, evt WebhookEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_events (
			delivery_id, event_type, action,
			repository_full_name, sender_login, payload_json
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (delivery_id) DO NOTHING
	`, evt.DeliveryID, evt.EventType, evt.Action, evt.RepositoryFullName, evt.SenderLogin, evt.PayloadJSON)
	if err != nil {
		return fmt.Errorf("insert webhook event: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) SaveAlert(ctx context.Context, alert AlertRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_alerts (
			delivery_id, event_type, action, repository_full_name,
			sender_login, rule_matched, suggestion_type, suggestion_value, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (delivery_id, suggestion_type, suggestion_value, rule_matched) DO NOTHING
	`, alert.DeliveryID, alert.EventType, alert.Action, alert.RepositoryFullName, alert.SenderLogin, alert.RuleMatched, alert.SuggestionType, alert.SuggestionValue, alert.Reason)
	if err != nil {
		return fmt.Errorf("insert webhook alert: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) ListEvents(ctx context.Context, limit int, offset int, eventType string, action string) ([]WebhookEventRecord, int64, error) {
	et := strings.TrimSpace(eventType)
	ac := strings.TrimSpace(action)

	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM webhook_events
		WHERE ($1 = '' OR event_type = $1)
		  AND ($2 = '' OR action = $2)
	`, et, ac).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook events: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, delivery_id, event_type, action, repository_full_name, sender_login, received_at
		FROM webhook_events
		WHERE ($1 = '' OR event_type = $1)
		  AND ($2 = '' OR action = $2)
		ORDER BY received_at DESC
		LIMIT $3 OFFSET $4
	`, et, ac, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query webhook events: %w", err)
	}
	defer rows.Close()

	items := make([]WebhookEventRecord, 0, limit)
	for rows.Next() {
		var item WebhookEventRecord
		if err := rows.Scan(
			&item.ID,
			&item.DeliveryID,
			&item.EventType,
			&item.Action,
			&item.RepositoryFullName,
			&item.SenderLogin,
			&item.ReceivedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan webhook event: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate webhook events: %w", err)
	}

	return items, total, nil
}

func (s *WebhookEventStore) ListAlerts(ctx context.Context, limit int, offset int, eventType string, action string, suggestionType string) ([]AlertRecord, int64, error) {
	et := strings.TrimSpace(eventType)
	ac := strings.TrimSpace(action)
	st := strings.TrimSpace(suggestionType)

	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM webhook_alerts
		WHERE ($1 = '' OR event_type = $1)
		  AND ($2 = '' OR action = $2)
		  AND ($3 = '' OR suggestion_type = $3)
	`, et, ac, st).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook alerts: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT delivery_id, event_type, action, repository_full_name, sender_login,
		       rule_matched, suggestion_type, suggestion_value, reason, created_at
		FROM webhook_alerts
		WHERE ($1 = '' OR event_type = $1)
		  AND ($2 = '' OR action = $2)
		  AND ($3 = '' OR suggestion_type = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`, et, ac, st, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query webhook alerts: %w", err)
	}
	defer rows.Close()

	items := make([]AlertRecord, 0, limit)
	for rows.Next() {
		var item AlertRecord
		if err := rows.Scan(
			&item.DeliveryID,
			&item.EventType,
			&item.Action,
			&item.RepositoryFullName,
			&item.SenderLogin,
			&item.RuleMatched,
			&item.SuggestionType,
			&item.SuggestionValue,
			&item.Reason,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan webhook alert: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate webhook alerts: %w", err)
	}

	return items, total, nil
}

func (s *WebhookEventStore) ListRules(ctx context.Context, limit int, offset int, eventType string, keyword string, activeOnly bool) ([]RuleRecord, int64, error) {
	et := strings.TrimSpace(eventType)
	kw := strings.TrimSpace(keyword)

	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM webhook_rules
		WHERE ($1 = '' OR event_type = $1)
		  AND ($2 = '' OR keyword ILIKE '%' || $2 || '%')
		  AND (NOT $3 OR is_active = true)
	`, et, kw, activeOnly).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook rules: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, event_type, keyword, suggestion_type, suggestion_value, reason, is_active, created_at
		FROM webhook_rules
		WHERE ($1 = '' OR event_type = $1)
		  AND ($2 = '' OR keyword ILIKE '%' || $2 || '%')
		  AND (NOT $3 OR is_active = true)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`, et, kw, activeOnly, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query webhook rules: %w", err)
	}
	defer rows.Close()

	items := make([]RuleRecord, 0, limit)
	for rows.Next() {
		var item RuleRecord
		if err := rows.Scan(&item.ID, &item.EventType, &item.Keyword, &item.SuggestionType, &item.SuggestionValue, &item.Reason, &item.IsActive, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan webhook rule: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate webhook rules: %w", err)
	}
	return items, total, nil
}

func (s *WebhookEventStore) CreateRule(ctx context.Context, rule RuleRecord) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO webhook_rules (event_type, keyword, suggestion_type, suggestion_value, reason, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, strings.TrimSpace(rule.EventType), strings.TrimSpace(rule.Keyword), strings.TrimSpace(rule.SuggestionType), strings.TrimSpace(rule.SuggestionValue), strings.TrimSpace(rule.Reason), rule.IsActive).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert webhook rule: %w", err)
	}
	return id, nil
}

func (s *WebhookEventStore) ensureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_events (
			id BIGSERIAL PRIMARY KEY,
			delivery_id TEXT NOT NULL UNIQUE,
			event_type TEXT NOT NULL,
			action TEXT NOT NULL,
			repository_full_name TEXT NOT NULL,
			sender_login TEXT NOT NULL,
			payload_json JSONB NOT NULL,
			received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create webhook_events table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_events_received_at
		ON webhook_events (received_at DESC)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_events_received_at: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_events_event_type
		ON webhook_events (event_type)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_events_event_type: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_events_action
		ON webhook_events (action)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_events_action: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_events_event_action
		ON webhook_events (event_type, action)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_events_event_action: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_alerts (
			id BIGSERIAL PRIMARY KEY,
			delivery_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			action TEXT NOT NULL,
			repository_full_name TEXT NOT NULL,
			sender_login TEXT NOT NULL,
			rule_matched TEXT NOT NULL,
			suggestion_type TEXT NOT NULL,
			suggestion_value TEXT NOT NULL,
			reason TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (delivery_id, suggestion_type, suggestion_value, rule_matched)
		)
	`)
	if err != nil {
		return fmt.Errorf("create webhook_alerts table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_alerts_created_at
		ON webhook_alerts (created_at DESC)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_alerts_created_at: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_alerts_event_action
		ON webhook_alerts (event_type, action)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_alerts_event_action: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_alerts_suggestion_type
		ON webhook_alerts (suggestion_type)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_alerts_suggestion_type: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_rules (
			id BIGSERIAL PRIMARY KEY,
			event_type TEXT NOT NULL,
			keyword TEXT NOT NULL,
			suggestion_type TEXT NOT NULL,
			suggestion_value TEXT NOT NULL,
			reason TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (event_type, keyword, suggestion_type, suggestion_value)
		)
	`)
	if err != nil {
		return fmt.Errorf("create webhook_rules table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_rules_event_type
		ON webhook_rules (event_type)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_rules_event_type: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_rules_active
		ON webhook_rules (is_active)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_rules_active: %w", err)
	}

	return nil
}

func IsDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
