package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
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

type AdminUser struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"password_hash"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

type ActionExecutionFailure struct {
	DeliveryID         string    `json:"delivery_id"`
	EventType          string    `json:"event_type"`
	Action             string    `json:"action"`
	RepositoryFullName string    `json:"repository_full_name"`
	SuggestionType     string    `json:"suggestion_type"`
	SuggestionValue    string    `json:"suggestion_value"`
	ErrorMessage       string    `json:"error_message"`
	AttemptCount       int       `json:"attempt_count"`
	RetryCount         int       `json:"retry_count"`
	LastRetryStatus    string    `json:"last_retry_status"`
	LastRetryMessage   string    `json:"last_retry_message"`
	LastRetryAt        time.Time `json:"last_retry_at,omitempty"`
	IsResolved         bool      `json:"is_resolved"`
	OccurredAt         time.Time `json:"occurred_at,omitempty"`
}

type ActionExecutionFailureRecord struct {
	ID int64 `json:"id"`
	ActionExecutionFailure
}

type AuditLogRecord struct {
	ID        int64     `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	TargetID  string    `json:"target_id"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

type DeliveryMetric struct {
	EventType     string    `json:"event_type"`
	DeliveryID    string    `json:"delivery_id"`
	Success       bool      `json:"success"`
	ProcessingMS  int64     `json:"processing_ms"`
	RecordedAtUTC time.Time `json:"recorded_at_utc"`
}

type MetricsOverview struct {
	Events24h       int64   `json:"events_24h"`
	Alerts24h       int64   `json:"alerts_24h"`
	Failures24h     int64   `json:"failures_24h"`
	SuccessRate24h  float64 `json:"success_rate_24h"`
	P95LatencyMS24h float64 `json:"p95_latency_ms_24h"`
}

type MetricsTimePoint struct {
	BucketStart time.Time `json:"bucket_start"`
	Events      int64     `json:"events"`
	Alerts      int64     `json:"alerts"`
	Failures    int64     `json:"failures"`
}

type WebhookStore interface {
	Close()
	SaveEvent(ctx context.Context, evt WebhookEvent) error
	SaveAlert(ctx context.Context, alert AlertRecord) error
	ListEvents(ctx context.Context, limit int, offset int, eventType string, action string) ([]WebhookEventRecord, int64, error)
	ListAlerts(ctx context.Context, limit int, offset int, eventType string, action string, suggestionType string) ([]AlertRecord, int64, error)
	ListRules(ctx context.Context, limit int, offset int, eventType string, keyword string, activeOnly bool) ([]RuleRecord, int64, error)
	CreateRule(ctx context.Context, rule RuleRecord) (int64, error)
	UpdateRuleActive(ctx context.Context, id int64, isActive bool) error
	SaveActionExecutionFailure(ctx context.Context, item ActionExecutionFailure) error
	ListActionExecutionFailures(ctx context.Context, limit int, offset int, includeResolved bool) ([]ActionExecutionFailureRecord, int64, error)
	GetActionExecutionFailureByID(ctx context.Context, id int64) (ActionExecutionFailureRecord, error)
	UpdateActionFailureRetryResult(ctx context.Context, id int64, success bool, message string) error
	GetWebhookEventPayloadByDeliveryID(ctx context.Context, deliveryID string) (json.RawMessage, error)
	SaveAuditLog(ctx context.Context, item AuditLogRecord) error
	ListAuditLogs(ctx context.Context, limit int, offset int, actor string, action string, since *time.Time) ([]AuditLogRecord, int64, error)
	GetAdminUserByUsername(ctx context.Context, username string) (AdminUser, error)
	UpdateAdminUserLastLogin(ctx context.Context, id int64, at time.Time) error
	EnsureBootstrapAdminUser(ctx context.Context, username string, passwordHash string) error

	SaveDeliveryMetric(ctx context.Context, metric DeliveryMetric) error
	GetMetricsOverview(ctx context.Context, since time.Time) (MetricsOverview, error)
	GetMetricsTimeSeries(ctx context.Context, since time.Time, intervalMinutes int) ([]MetricsTimePoint, error)
}

func NewWebhookEventStore(ctx context.Context, databaseURL string) (WebhookStore, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DATABASE_URL is not configured")
	}

	if isMySQLDatabaseURL(databaseURL) {
		return newMySQLWebhookEventStore(ctx, databaseURL)
	}

	return newPostgresWebhookEventStore(ctx, databaseURL)
}

func newPostgresWebhookEventStore(ctx context.Context, databaseURL string) (*WebhookEventStore, error) {
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

func isMySQLDatabaseURL(databaseURL string) bool {
	u := strings.ToLower(strings.TrimSpace(databaseURL))
	return strings.HasPrefix(u, "mysql://")
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

func (s *WebhookEventStore) UpdateRuleActive(ctx context.Context, id int64, isActive bool) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE webhook_rules
		SET is_active = $2
		WHERE id = $1
	`, id, isActive)
	if err != nil {
		return fmt.Errorf("update webhook rule active: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

func (s *WebhookEventStore) SaveActionExecutionFailure(ctx context.Context, item ActionExecutionFailure) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_action_failures (
			delivery_id, event_type, action, repository_full_name,
			suggestion_type, suggestion_value, error_message, attempt_count,
			retry_count, last_retry_status, last_retry_message, last_retry_at, is_resolved
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,0,'never','',NULL,FALSE)
	`, item.DeliveryID, item.EventType, item.Action, item.RepositoryFullName, item.SuggestionType, item.SuggestionValue, item.ErrorMessage, item.AttemptCount)
	if err != nil {
		return fmt.Errorf("insert webhook action failure: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) ListActionExecutionFailures(ctx context.Context, limit int, offset int, includeResolved bool) ([]ActionExecutionFailureRecord, int64, error) {
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_action_failures WHERE ($1 OR NOT is_resolved)`, includeResolved).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count action failures: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, delivery_id, event_type, action, repository_full_name, suggestion_type, suggestion_value, error_message, attempt_count, retry_count, last_retry_status, last_retry_message, COALESCE(last_retry_at, 'epoch'::timestamptz), is_resolved, occurred_at
		FROM webhook_action_failures
		WHERE ($1 OR NOT is_resolved)
		ORDER BY occurred_at DESC
		LIMIT $2 OFFSET $3
	`, includeResolved, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query action failures: %w", err)
	}
	defer rows.Close()

	items := make([]ActionExecutionFailureRecord, 0, limit)
	for rows.Next() {
		var rec ActionExecutionFailureRecord
		if err := rows.Scan(&rec.ID, &rec.DeliveryID, &rec.EventType, &rec.Action, &rec.RepositoryFullName, &rec.SuggestionType, &rec.SuggestionValue, &rec.ErrorMessage, &rec.AttemptCount, &rec.RetryCount, &rec.LastRetryStatus, &rec.LastRetryMessage, &rec.LastRetryAt, &rec.IsResolved, &rec.OccurredAt); err != nil {
			return nil, 0, fmt.Errorf("scan action failure: %w", err)
		}
		if rec.LastRetryAt.Equal(time.Unix(0, 0).UTC()) {
			rec.LastRetryAt = time.Time{}
		}
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate action failures: %w", err)
	}
	return items, total, nil
}

func (s *WebhookEventStore) GetActionExecutionFailureByID(ctx context.Context, id int64) (ActionExecutionFailureRecord, error) {
	var rec ActionExecutionFailureRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, delivery_id, event_type, action, repository_full_name, suggestion_type, suggestion_value, error_message, attempt_count, retry_count, last_retry_status, last_retry_message, COALESCE(last_retry_at, 'epoch'::timestamptz), is_resolved, occurred_at
		FROM webhook_action_failures
		WHERE id = $1
	`, id).Scan(&rec.ID, &rec.DeliveryID, &rec.EventType, &rec.Action, &rec.RepositoryFullName, &rec.SuggestionType, &rec.SuggestionValue, &rec.ErrorMessage, &rec.AttemptCount, &rec.RetryCount, &rec.LastRetryStatus, &rec.LastRetryMessage, &rec.LastRetryAt, &rec.IsResolved, &rec.OccurredAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return rec, fmt.Errorf("action failure not found")
		}
		return rec, fmt.Errorf("get action failure by id: %w", err)
	}
	if rec.LastRetryAt.Equal(time.Unix(0, 0).UTC()) {
		rec.LastRetryAt = time.Time{}
	}
	return rec, nil
}

func (s *WebhookEventStore) UpdateActionFailureRetryResult(ctx context.Context, id int64, success bool, message string) error {
	status := "failed"
	resolved := false
	if success {
		status = "success"
		resolved = true
	}
	result, err := s.pool.Exec(ctx, `
		UPDATE webhook_action_failures
		SET retry_count = retry_count + 1,
		    last_retry_status = $2,
		    last_retry_message = $3,
		    last_retry_at = NOW(),
		    is_resolved = $4
		WHERE id = $1
	`, id, status, strings.TrimSpace(message), resolved)
	if err != nil {
		return fmt.Errorf("update action failure retry result: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("action failure not found")
	}
	return nil
}

func (s *WebhookEventStore) GetWebhookEventPayloadByDeliveryID(ctx context.Context, deliveryID string) (json.RawMessage, error) {
	var payload []byte
	err := s.pool.QueryRow(ctx, `SELECT payload_json FROM webhook_events WHERE delivery_id = $1`, strings.TrimSpace(deliveryID)).Scan(&payload)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return nil, fmt.Errorf("webhook event not found")
		}
		return nil, fmt.Errorf("get webhook event payload by delivery id: %w", err)
	}
	return json.RawMessage(payload), nil
}

func (s *WebhookEventStore) SaveAuditLog(ctx context.Context, item AuditLogRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_logs (actor, action, target, target_id, payload)
		VALUES ($1,$2,$3,$4,$5)
	`, strings.TrimSpace(item.Actor), strings.TrimSpace(item.Action), strings.TrimSpace(item.Target), strings.TrimSpace(item.TargetID), item.Payload)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) GetAdminUserByUsername(ctx context.Context, username string) (AdminUser, error) {
	var user AdminUser
	var lastLoginAt time.Time
	name := strings.TrimSpace(username)
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, is_active, created_at, updated_at, COALESCE(last_login_at, 'epoch'::timestamptz)
		FROM admin_users
		WHERE username = $1
		LIMIT 1
	`, name).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return user, fmt.Errorf("admin user not found")
		}
		return user, fmt.Errorf("get admin user by username: %w", err)
	}
	if !lastLoginAt.Equal(time.Unix(0, 0).UTC()) {
		ts := lastLoginAt.UTC()
		user.LastLoginAt = &ts
	}
	return user, nil
}

func (s *WebhookEventStore) UpdateAdminUserLastLogin(ctx context.Context, id int64, at time.Time) error {
	result, err := s.pool.Exec(ctx, `UPDATE admin_users SET last_login_at = $2, updated_at = NOW() WHERE id = $1`, id, at.UTC())
	if err != nil {
		return fmt.Errorf("update admin user last login: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("admin user not found")
	}
	return nil
}

func (s *WebhookEventStore) EnsureBootstrapAdminUser(ctx context.Context, username string, passwordHash string) error {
	name := strings.TrimSpace(username)
	hash := strings.TrimSpace(passwordHash)
	if name == "" || hash == "" {
		return nil
	}

	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&total); err != nil {
		return fmt.Errorf("count admin users: %w", err)
	}
	if total > 0 {
		return nil
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO admin_users (username, password_hash, is_active)
		VALUES ($1, $2, TRUE)
		ON CONFLICT (username) DO NOTHING
	`, name, hash)
	if err != nil {
		return fmt.Errorf("bootstrap admin user: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) ListAuditLogs(ctx context.Context, limit int, offset int, actor string, action string, since *time.Time) ([]AuditLogRecord, int64, error) {
	ac := strings.TrimSpace(actor)
	act := strings.TrimSpace(action)
	hasSince := since != nil
	var sinceTime time.Time
	if since != nil {
		sinceTime = since.UTC()
	}

	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs
		WHERE ($1 = '' OR actor = $1)
		  AND ($2 = '' OR action = $2)
		  AND (NOT $3 OR created_at >= $4)
	`, ac, act, hasSince, sinceTime).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, actor, action, target, target_id, payload, created_at
		FROM audit_logs
		WHERE ($1 = '' OR actor = $1)
		  AND ($2 = '' OR action = $2)
		  AND (NOT $3 OR created_at >= $4)
		ORDER BY created_at DESC
		LIMIT $5 OFFSET $6
	`, ac, act, hasSince, sinceTime, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	items := make([]AuditLogRecord, 0, limit)
	for rows.Next() {
		var rec AuditLogRecord
		if err := rows.Scan(&rec.ID, &rec.Actor, &rec.Action, &rec.Target, &rec.TargetID, &rec.Payload, &rec.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate audit logs: %w", err)
	}
	return items, total, nil
}

func (s *WebhookEventStore) SaveDeliveryMetric(ctx context.Context, metric DeliveryMetric) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_delivery_metrics (event_type, delivery_id, success, processing_ms, recorded_at)
		VALUES ($1,$2,$3,$4,$5)
	`, strings.TrimSpace(metric.EventType), strings.TrimSpace(metric.DeliveryID), metric.Success, metric.ProcessingMS, metric.RecordedAtUTC)
	if err != nil {
		return fmt.Errorf("insert delivery metric: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) GetMetricsOverview(ctx context.Context, since time.Time) (MetricsOverview, error) {
	var out MetricsOverview
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_events WHERE received_at >= $1`, since).Scan(&out.Events24h); err != nil {
		return out, fmt.Errorf("count events metrics: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_alerts WHERE created_at >= $1`, since).Scan(&out.Alerts24h); err != nil {
		return out, fmt.Errorf("count alerts metrics: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_action_failures WHERE occurred_at >= $1 AND NOT is_resolved`, since).Scan(&out.Failures24h); err != nil {
		return out, fmt.Errorf("count failures metrics: %w", err)
	}

	var total int64
	var success int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*), COALESCE(SUM(CASE WHEN success THEN 1 ELSE 0 END),0) FROM webhook_delivery_metrics WHERE recorded_at >= $1`, since).Scan(&total, &success); err != nil {
		return out, fmt.Errorf("count delivery metrics: %w", err)
	}
	if total > 0 {
		out.SuccessRate24h = (float64(success) / float64(total)) * 100
	}

	rows, err := s.pool.Query(ctx, `SELECT processing_ms FROM webhook_delivery_metrics WHERE recorded_at >= $1 ORDER BY processing_ms ASC`, since)
	if err != nil {
		return out, fmt.Errorf("query latency metrics: %w", err)
	}
	defer rows.Close()
	latencies := make([]int64, 0, 256)
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return out, fmt.Errorf("scan latency metric: %w", err)
		}
		latencies = append(latencies, v)
	}
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("iterate latency metrics: %w", err)
	}
	if len(latencies) > 0 {
		idx := int(float64(len(latencies)-1) * 0.95)
		out.P95LatencyMS24h = float64(latencies[idx])
	}
	return out, nil
}

func (s *WebhookEventStore) GetMetricsTimeSeries(ctx context.Context, since time.Time, intervalMinutes int) ([]MetricsTimePoint, error) {
	if intervalMinutes <= 0 {
		intervalMinutes = 60
	}
	step := time.Duration(intervalMinutes) * time.Minute
	start := since.UTC().Truncate(step)
	now := time.Now().UTC()

	buckets := make(map[time.Time]*MetricsTimePoint)
	for t := start; !t.After(now); t = t.Add(step) {
		tt := t
		buckets[tt] = &MetricsTimePoint{BucketStart: tt}
	}

	fill := func(query string, assign func(*MetricsTimePoint, int64)) error {
		rows, err := s.pool.Query(ctx, query, since)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var ts time.Time
			if err := rows.Scan(&ts); err != nil {
				return err
			}
			b := ts.UTC().Truncate(step)
			if p, ok := buckets[b]; ok {
				assign(p, 1)
			}
		}
		return rows.Err()
	}

	if err := fill(`SELECT received_at FROM webhook_events WHERE received_at >= $1`, func(p *MetricsTimePoint, _ int64) { p.Events++ }); err != nil {
		return nil, fmt.Errorf("fill events metrics timeseries: %w", err)
	}
	if err := fill(`SELECT created_at FROM webhook_alerts WHERE created_at >= $1`, func(p *MetricsTimePoint, _ int64) { p.Alerts++ }); err != nil {
		return nil, fmt.Errorf("fill alerts metrics timeseries: %w", err)
	}
	if err := fill(`SELECT occurred_at FROM webhook_action_failures WHERE occurred_at >= $1`, func(p *MetricsTimePoint, _ int64) { p.Failures++ }); err != nil {
		return nil, fmt.Errorf("fill failures metrics timeseries: %w", err)
	}

	out := make([]MetricsTimePoint, 0, len(buckets))
	for t := start; !t.After(now); t = t.Add(step) {
		if p, ok := buckets[t]; ok {
			out = append(out, *p)
		}
	}
	return out, nil
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

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_action_failures (
			id BIGSERIAL PRIMARY KEY,
			delivery_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			action TEXT NOT NULL,
			repository_full_name TEXT NOT NULL,
			suggestion_type TEXT NOT NULL,
			suggestion_value TEXT NOT NULL,
			error_message TEXT NOT NULL,
			attempt_count INT NOT NULL,
			retry_count INT NOT NULL DEFAULT 0,
			last_retry_status TEXT NOT NULL DEFAULT 'never',
			last_retry_message TEXT NOT NULL DEFAULT '',
			last_retry_at TIMESTAMPTZ NULL,
			is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
			occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create webhook_action_failures table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_action_failures_delivery
		ON webhook_action_failures (delivery_id)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_action_failures_delivery: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_action_failures_occurred_at
		ON webhook_action_failures (occurred_at DESC)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_action_failures_occurred_at: %w", err)
	}

	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS last_retry_status TEXT NOT NULL DEFAULT 'never'`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS last_retry_message TEXT NOT NULL DEFAULT ''`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS last_retry_at TIMESTAMPTZ NULL`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS is_resolved BOOLEAN NOT NULL DEFAULT FALSE`)

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id BIGSERIAL PRIMARY KEY,
			actor TEXT NOT NULL,
			action TEXT NOT NULL,
			target TEXT NOT NULL,
			target_id TEXT NOT NULL,
			payload TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create audit_logs table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at
		ON audit_logs (created_at DESC)
	`)
	if err != nil {
		return fmt.Errorf("create idx_audit_logs_created_at: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS admin_users (
			id BIGSERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			last_login_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create admin_users table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_admin_users_is_active
		ON admin_users (is_active)
	`)
	if err != nil {
		return fmt.Errorf("create idx_admin_users_is_active: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_action
		ON audit_logs (actor, action)
	`)
	if err != nil {
		return fmt.Errorf("create idx_audit_logs_actor_action: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_delivery_metrics (
			id BIGSERIAL PRIMARY KEY,
			event_type TEXT NOT NULL,
			delivery_id TEXT NOT NULL,
			success BOOLEAN NOT NULL,
			processing_ms BIGINT NOT NULL,
			recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create webhook_delivery_metrics table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_delivery_metrics_recorded_at
		ON webhook_delivery_metrics (recorded_at DESC)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_delivery_metrics_recorded_at: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_admin_users_username
		ON admin_users (username)
	`)
	if err != nil {
		return fmt.Errorf("create idx_admin_users_username: %w", err)
	}

	_, _ = s.pool.Exec(ctx, `ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ NULL`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`)

	return nil
}

func IsDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}
