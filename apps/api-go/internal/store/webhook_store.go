package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"maintainer-firewall/api-go/internal/tenantctx"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
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
	ID                 int64           `json:"id"`
	DeliveryID         string          `json:"delivery_id"`
	EventType          string          `json:"event_type"`
	Action             string          `json:"action"`
	RepositoryFullName string          `json:"repository_full_name"`
	SenderLogin        string          `json:"sender_login"`
	PayloadJSON        json.RawMessage `json:"payload_json,omitempty"`
	ReceivedAt         time.Time       `json:"received_at"`
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

type RuleVersionRecord struct {
	Version       int64     `json:"version"`
	RuleCount     int       `json:"rule_count"`
	CreatedBy     string    `json:"created_by"`
	SourceVersion int64     `json:"source_version,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type AdminUser struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"password_hash"`
	IsActive     bool       `json:"is_active"`
	Role         string     `json:"role"`        // admin, editor, viewer
	Permissions  []string   `json:"permissions"` // read, write, admin
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

type EventFilterOptions struct {
	EventTypes   []string `json:"event_types"`
	Actions      []string `json:"actions"`
	Repositories []string `json:"repositories"`
	Senders      []string `json:"senders"`
}

type AlertFilterOptions struct {
	EventTypes      []string `json:"event_types"`
	Actions         []string `json:"actions"`
	SuggestionTypes []string `json:"suggestion_types"`
	Repositories    []string `json:"repositories"`
	Senders         []string `json:"senders"`
}

type RuleFilterOptions struct {
	EventTypes      []string `json:"event_types"`
	SuggestionTypes []string `json:"suggestion_types"`
	ActiveStates    []string `json:"active_states"`
}

type TenantRecord struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	Events24h                 int64   `json:"events_24h"`
	Alerts24h                 int64   `json:"alerts_24h"`
	Failures24h               int64   `json:"failures_24h"`
	SuccessRate24h            float64 `json:"success_rate_24h"`
	P95LatencyMS24h           float64 `json:"p95_latency_ms_24h"`
	DeliveryAttempts          int64   `json:"delivery_attempts"`
	DeliverySuccess           int64   `json:"delivery_success"`
	ResolvedFailures24h       int64   `json:"resolved_failures_24h"`
	AutomationCoverage24h     float64 `json:"automation_coverage_24h"`
	FailureRate24h            float64 `json:"failure_rate_24h"`
	EstimatedManualMinutes24h float64 `json:"estimated_manual_minutes_24h"`
	AvgProcessingMS24h        float64 `json:"avg_processing_ms_24h"`
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
	ListEventFilterOptions(ctx context.Context) (EventFilterOptions, error)
	ListAlertFilterOptions(ctx context.Context) (AlertFilterOptions, error)
	ListRuleFilterOptions(ctx context.Context) (RuleFilterOptions, error)
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
	ListTenants(ctx context.Context) ([]TenantRecord, error)
	CreateTenant(ctx context.Context, id string, name string) error
	UpdateTenantActive(ctx context.Context, id string, isActive bool) error
	CreateRuleVersionSnapshot(ctx context.Context, createdBy string, sourceVersion int64) (int64, int, error)
	ListRuleVersions(ctx context.Context, limit int, offset int) ([]RuleVersionRecord, int64, error)
	GetRulesByVersion(ctx context.Context, version int64) ([]RuleRecord, error)
	RestoreRulesFromVersion(ctx context.Context, version int64) (int, error)
	UserStore
}

type UserStore interface {
	ListAdminUsers(ctx context.Context, limit int, offset int) ([]AdminUser, int64, error)
	CreateAdminUser(ctx context.Context, user AdminUser) (int64, error)
	UpdateAdminUser(ctx context.Context, id int64, user AdminUser) error
	DeleteAdminUser(ctx context.Context, id int64) error
	GetAdminUserByID(ctx context.Context, id int64) (AdminUser, error)
	UpdateAdminUserActive(ctx context.Context, id int64, isActive bool) error
	SaveAuditLog(ctx context.Context, item AuditLogRecord) error
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

func tenantIDFromCtx(ctx context.Context) string {
	return tenantctx.MustFromContext(ctx, tenantctx.DefaultTenantID)
}

func (s *WebhookEventStore) SaveEvent(ctx context.Context, evt WebhookEvent) error {
	tenantID := tenantIDFromCtx(ctx)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_events (
			tenant_id, delivery_id, event_type, action,
			repository_full_name, sender_login, payload_json
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, delivery_id) DO NOTHING
	`, tenantID, evt.DeliveryID, evt.EventType, evt.Action, evt.RepositoryFullName, evt.SenderLogin, evt.PayloadJSON)
	if err != nil {
		return fmt.Errorf("insert webhook event: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) SaveAlert(ctx context.Context, alert AlertRecord) error {
	tenantID := tenantIDFromCtx(ctx)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_alerts (
			tenant_id, delivery_id, event_type, action, repository_full_name,
			sender_login, rule_matched, suggestion_type, suggestion_value, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_id, delivery_id, suggestion_type, suggestion_value, rule_matched) DO NOTHING
	`, tenantID, alert.DeliveryID, alert.EventType, alert.Action, alert.RepositoryFullName, alert.SenderLogin, alert.RuleMatched, alert.SuggestionType, alert.SuggestionValue, alert.Reason)
	if err != nil {
		return fmt.Errorf("insert webhook alert: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) ListEvents(ctx context.Context, limit int, offset int, eventType string, action string) ([]WebhookEventRecord, int64, error) {
	tenantID := tenantIDFromCtx(ctx)
	et := strings.TrimSpace(eventType)
	ac := strings.TrimSpace(action)

	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM webhook_events
		WHERE tenant_id = $1
		  AND ($2 = '' OR event_type = $2)
		  AND ($3 = '' OR action = $3)
	`, tenantID, et, ac).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook events: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, delivery_id, event_type, action, repository_full_name, sender_login, payload_json, received_at
		FROM webhook_events
		WHERE tenant_id = $1
		  AND ($2 = '' OR event_type = $2)
		  AND ($3 = '' OR action = $3)
		ORDER BY received_at DESC
		LIMIT $4 OFFSET $5
	`, tenantID, et, ac, limit, offset)
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
			&item.PayloadJSON,
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
	tenantID := tenantIDFromCtx(ctx)
	et := strings.TrimSpace(eventType)
	ac := strings.TrimSpace(action)
	st := strings.TrimSpace(suggestionType)

	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM webhook_alerts
		WHERE tenant_id = $1
		  AND ($2 = '' OR event_type = $2)
		  AND ($3 = '' OR action = $3)
		  AND ($4 = '' OR suggestion_type = $4)
	`, tenantID, et, ac, st).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook alerts: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT delivery_id, event_type, action, repository_full_name, sender_login,
		       rule_matched, suggestion_type, suggestion_value, reason, created_at
		FROM webhook_alerts
		WHERE tenant_id = $1
		  AND ($2 = '' OR event_type = $2)
		  AND ($3 = '' OR action = $3)
		  AND ($4 = '' OR suggestion_type = $4)
		ORDER BY created_at DESC
		LIMIT $5 OFFSET $6
	`, tenantID, et, ac, st, limit, offset)
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
	tenantID := tenantIDFromCtx(ctx)
	et := strings.TrimSpace(eventType)
	kw := strings.TrimSpace(keyword)

	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM webhook_rules
		WHERE tenant_id = $1
		  AND ($2 = '' OR event_type = $2)
		  AND ($3 = '' OR keyword ILIKE '%' || $3 || '%')
		  AND (NOT $4 OR is_active = true)
	`, tenantID, et, kw, activeOnly).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook rules: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, event_type, keyword, suggestion_type, suggestion_value, reason, is_active, created_at
		FROM webhook_rules
		WHERE tenant_id = $1
		  AND ($2 = '' OR event_type = $2)
		  AND ($3 = '' OR keyword ILIKE '%' || $3 || '%')
		  AND (NOT $4 OR is_active = true)
		ORDER BY created_at DESC
		LIMIT $5 OFFSET $6
	`, tenantID, et, kw, activeOnly, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query webhook rules: %w", err)
	}
	defer rows.Close()

	items := make([]RuleRecord, 0, limit)
	for rows.Next() {
		var rec RuleRecord
		if err := rows.Scan(&rec.ID, &rec.EventType, &rec.Keyword, &rec.SuggestionType, &rec.SuggestionValue, &rec.Reason, &rec.IsActive, &rec.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan webhook rule row: %w", err)
		}
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate webhook rules: %w", err)
	}
	return items, total, nil
}

func listDistinctNonEmpty(ctx context.Context, pool *pgxpool.Pool, q string, args ...any) ([]string, error) {
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, 32)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *WebhookEventStore) ListEventFilterOptions(ctx context.Context) (EventFilterOptions, error) {
	tenantID := tenantIDFromCtx(ctx)
	et, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT event_type FROM webhook_events WHERE tenant_id = $1 AND event_type <> '' ORDER BY event_type ASC`, tenantID)
	if err != nil {
		return EventFilterOptions{}, fmt.Errorf("list distinct event_type from webhook_events: %w", err)
	}
	ac, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT action FROM webhook_events WHERE tenant_id = $1 AND action <> '' ORDER BY action ASC`, tenantID)
	if err != nil {
		return EventFilterOptions{}, fmt.Errorf("list distinct action from webhook_events: %w", err)
	}
	repo, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT repository_full_name FROM webhook_events WHERE tenant_id = $1 AND repository_full_name <> '' ORDER BY repository_full_name ASC`, tenantID)
	if err != nil {
		return EventFilterOptions{}, fmt.Errorf("list distinct repository from webhook_events: %w", err)
	}
	sender, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT sender_login FROM webhook_events WHERE tenant_id = $1 AND sender_login <> '' ORDER BY sender_login ASC`, tenantID)
	if err != nil {
		return EventFilterOptions{}, fmt.Errorf("list distinct sender from webhook_events: %w", err)
	}
	return EventFilterOptions{EventTypes: et, Actions: ac, Repositories: repo, Senders: sender}, nil
}

func (s *WebhookEventStore) ListAlertFilterOptions(ctx context.Context) (AlertFilterOptions, error) {
	tenantID := tenantIDFromCtx(ctx)
	et, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT event_type FROM webhook_alerts WHERE tenant_id = $1 AND event_type <> '' ORDER BY event_type ASC`, tenantID)
	if err != nil {
		return AlertFilterOptions{}, fmt.Errorf("list distinct event_type from webhook_alerts: %w", err)
	}
	ac, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT action FROM webhook_alerts WHERE tenant_id = $1 AND action <> '' ORDER BY action ASC`, tenantID)
	if err != nil {
		return AlertFilterOptions{}, fmt.Errorf("list distinct action from webhook_alerts: %w", err)
	}
	st, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT suggestion_type FROM webhook_alerts WHERE tenant_id = $1 AND suggestion_type <> '' ORDER BY suggestion_type ASC`, tenantID)
	if err != nil {
		return AlertFilterOptions{}, fmt.Errorf("list distinct suggestion_type from webhook_alerts: %w", err)
	}
	repo, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT repository_full_name FROM webhook_alerts WHERE tenant_id = $1 AND repository_full_name <> '' ORDER BY repository_full_name ASC`, tenantID)
	if err != nil {
		return AlertFilterOptions{}, fmt.Errorf("list distinct repository from webhook_alerts: %w", err)
	}
	sender, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT sender_login FROM webhook_alerts WHERE tenant_id = $1 AND sender_login <> '' ORDER BY sender_login ASC`, tenantID)
	if err != nil {
		return AlertFilterOptions{}, fmt.Errorf("list distinct sender from webhook_alerts: %w", err)
	}
	return AlertFilterOptions{EventTypes: et, Actions: ac, SuggestionTypes: st, Repositories: repo, Senders: sender}, nil
}

func (s *WebhookEventStore) ListRuleFilterOptions(ctx context.Context) (RuleFilterOptions, error) {
	tenantID := tenantIDFromCtx(ctx)
	et, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT event_type FROM webhook_rules WHERE tenant_id = $1 AND event_type <> '' ORDER BY event_type ASC`, tenantID)
	if err != nil {
		return RuleFilterOptions{}, fmt.Errorf("list distinct event_type from webhook_rules: %w", err)
	}
	st, err := listDistinctNonEmpty(ctx, s.pool, `SELECT DISTINCT suggestion_type FROM webhook_rules WHERE tenant_id = $1 AND suggestion_type <> '' ORDER BY suggestion_type ASC`, tenantID)
	if err != nil {
		return RuleFilterOptions{}, fmt.Errorf("list distinct suggestion_type from webhook_rules: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT is_active FROM webhook_rules WHERE tenant_id = $1 ORDER BY is_active DESC`, tenantID)
	if err != nil {
		return RuleFilterOptions{}, fmt.Errorf("list distinct is_active from webhook_rules: %w", err)
	}
	defer rows.Close()
	activeStates := make([]string, 0, 2)
	for rows.Next() {
		var v bool
		if err := rows.Scan(&v); err != nil {
			return RuleFilterOptions{}, fmt.Errorf("scan distinct is_active: %w", err)
		}
		if v {
			activeStates = append(activeStates, "active")
		} else {
			activeStates = append(activeStates, "inactive")
		}
	}
	if err := rows.Err(); err != nil {
		return RuleFilterOptions{}, fmt.Errorf("iterate distinct is_active: %w", err)
	}
	return RuleFilterOptions{EventTypes: et, SuggestionTypes: st, ActiveStates: activeStates}, nil
}

func (s *WebhookEventStore) CreateRule(ctx context.Context, rule RuleRecord) (int64, error) {
	tenantID := tenantIDFromCtx(ctx)
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO webhook_rules (tenant_id, event_type, keyword, suggestion_type, suggestion_value, reason, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, tenantID, strings.TrimSpace(rule.EventType), strings.TrimSpace(rule.Keyword), strings.TrimSpace(rule.SuggestionType), strings.TrimSpace(rule.SuggestionValue), strings.TrimSpace(rule.Reason), rule.IsActive).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert webhook rule: %w", err)
	}
	return id, nil
}

func (s *WebhookEventStore) UpdateRuleActive(ctx context.Context, id int64, isActive bool) error {
	tenantID := tenantIDFromCtx(ctx)
	result, err := s.pool.Exec(ctx, `
		UPDATE webhook_rules
		SET is_active = $2
		WHERE id = $1
		  AND tenant_id = $3
	`, id, isActive, tenantID)
	if err != nil {
		return fmt.Errorf("update webhook rule active: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

func (s *WebhookEventStore) CreateRuleVersionSnapshot(ctx context.Context, createdBy string, sourceVersion int64) (int64, int, error) {
	tenantID := tenantIDFromCtx(ctx)
	rules, err := s.listAllRulesByTenant(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}

	payload, err := json.Marshal(rules)
	if err != nil {
		return 0, 0, fmt.Errorf("marshal rules snapshot: %w", err)
	}

	var version int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM webhook_rule_versions
		WHERE tenant_id = $1
	`, tenantID).Scan(&version); err != nil {
		return 0, 0, fmt.Errorf("next rule version: %w", err)
	}

	var src any
	if sourceVersion > 0 {
		src = sourceVersion
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO webhook_rule_versions (tenant_id, version, rules_json, rule_count, created_by, source_version)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tenantID, version, payload, len(rules), strings.TrimSpace(createdBy), src)
	if err != nil {
		return 0, 0, fmt.Errorf("insert rule version snapshot: %w", err)
	}
	return version, len(rules), nil
}

func (s *WebhookEventStore) ListRuleVersions(ctx context.Context, limit int, offset int) ([]RuleVersionRecord, int64, error) {
	tenantID := tenantIDFromCtx(ctx)
	var total int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM webhook_rule_versions
		WHERE tenant_id = $1
	`, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count rule versions: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT version, rule_count, created_by, COALESCE(source_version, 0), created_at
		FROM webhook_rule_versions
		WHERE tenant_id = $1
		ORDER BY version DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query rule versions: %w", err)
	}
	defer rows.Close()

	items := make([]RuleVersionRecord, 0, limit)
	for rows.Next() {
		var item RuleVersionRecord
		if err := rows.Scan(&item.Version, &item.RuleCount, &item.CreatedBy, &item.SourceVersion, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan rule version: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate rule versions: %w", err)
	}
	return items, total, nil
}

func (s *WebhookEventStore) GetRulesByVersion(ctx context.Context, version int64) ([]RuleRecord, error) {
	tenantID := tenantIDFromCtx(ctx)
	var payload []byte
	err := s.pool.QueryRow(ctx, `
		SELECT rules_json
		FROM webhook_rule_versions
		WHERE tenant_id = $1
		  AND version = $2
		LIMIT 1
	`, tenantID, version).Scan(&payload)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return nil, fmt.Errorf("rule version not found")
		}
		return nil, fmt.Errorf("get rules by version: %w", err)
	}

	var items []RuleRecord
	if err := json.Unmarshal(payload, &items); err != nil {
		return nil, fmt.Errorf("unmarshal rules snapshot: %w", err)
	}
	return items, nil
}

func (s *WebhookEventStore) RestoreRulesFromVersion(ctx context.Context, version int64) (int, error) {
	tenantID := tenantIDFromCtx(ctx)
	rules, err := s.GetRulesByVersion(ctx, version)
	if err != nil {
		return 0, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin restore rules tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM webhook_rules WHERE tenant_id = $1`, tenantID); err != nil {
		return 0, fmt.Errorf("clear tenant rules before restore: %w", err)
	}

	for _, r := range rules {
		_, err := tx.Exec(ctx, `
			INSERT INTO webhook_rules (tenant_id, event_type, keyword, suggestion_type, suggestion_value, reason, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, tenantID, strings.TrimSpace(r.EventType), strings.TrimSpace(r.Keyword), strings.TrimSpace(r.SuggestionType), strings.TrimSpace(r.SuggestionValue), strings.TrimSpace(r.Reason), r.IsActive)
		if err != nil {
			return 0, fmt.Errorf("restore webhook rule: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit restore rules tx: %w", err)
	}
	return len(rules), nil
}

func (s *WebhookEventStore) listAllRulesByTenant(ctx context.Context, tenantID string) ([]RuleRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, event_type, keyword, suggestion_type, suggestion_value, reason, is_active, created_at
		FROM webhook_rules
		WHERE tenant_id = $1
		ORDER BY created_at ASC, id ASC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query all webhook rules: %w", err)
	}
	defer rows.Close()

	items := make([]RuleRecord, 0, 64)
	for rows.Next() {
		var rec RuleRecord
		if err := rows.Scan(&rec.ID, &rec.EventType, &rec.Keyword, &rec.SuggestionType, &rec.SuggestionValue, &rec.Reason, &rec.IsActive, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan all webhook rule rows: %w", err)
		}
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all webhook rules: %w", err)
	}
	return items, nil
}

func (s *WebhookEventStore) SaveActionExecutionFailure(ctx context.Context, item ActionExecutionFailure) error {
	tenantID := tenantIDFromCtx(ctx)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_action_failures (
			tenant_id, delivery_id, event_type, action, repository_full_name,
			suggestion_type, suggestion_value, error_message, attempt_count,
			retry_count, last_retry_status, last_retry_message, last_retry_at, is_resolved
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,0,'never','',NULL,FALSE)
	`, tenantID, item.DeliveryID, item.EventType, item.Action, item.RepositoryFullName, item.SuggestionType, item.SuggestionValue, item.ErrorMessage, item.AttemptCount)
	if err != nil {
		return fmt.Errorf("insert webhook action failure: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) ListActionExecutionFailures(ctx context.Context, limit int, offset int, includeResolved bool) ([]ActionExecutionFailureRecord, int64, error) {
	tenantID := tenantIDFromCtx(ctx)
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_action_failures WHERE tenant_id = $1 AND ($2 OR is_resolved = FALSE)`, tenantID, includeResolved).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count action failures: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, delivery_id, event_type, action, repository_full_name, suggestion_type, suggestion_value, error_message, attempt_count, retry_count, last_retry_status, last_retry_message, COALESCE(last_retry_at, 'epoch'::timestamptz), is_resolved, occurred_at
		FROM webhook_action_failures
		WHERE tenant_id = $1
		  AND ($2 OR is_resolved = FALSE)

		LIMIT $3 OFFSET $4
	`, tenantID, includeResolved, limit, offset)
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
	tenantID := tenantIDFromCtx(ctx)
	var rec ActionExecutionFailureRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, delivery_id, event_type, action, repository_full_name, suggestion_type, suggestion_value, error_message, attempt_count, retry_count, last_retry_status, last_retry_message, COALESCE(last_retry_at, 'epoch'::timestamptz), is_resolved, occurred_at
		FROM webhook_action_failures
		WHERE id = $1
		  AND tenant_id = $2
	`, id, tenantID).Scan(&rec.ID, &rec.DeliveryID, &rec.EventType, &rec.Action, &rec.RepositoryFullName, &rec.SuggestionType, &rec.SuggestionValue, &rec.ErrorMessage, &rec.AttemptCount, &rec.RetryCount, &rec.LastRetryStatus, &rec.LastRetryMessage, &rec.LastRetryAt, &rec.IsResolved, &rec.OccurredAt)
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
	tenantID := tenantIDFromCtx(ctx)
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
		  AND tenant_id = $5
	`, id, status, strings.TrimSpace(message), resolved, tenantID)
	if err != nil {
		return fmt.Errorf("update action failure retry result: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("action failure not found")
	}
	return nil
}

func (s *WebhookEventStore) GetWebhookEventPayloadByDeliveryID(ctx context.Context, deliveryID string) (json.RawMessage, error) {
	tenantID := tenantIDFromCtx(ctx)
	var payload []byte
	err := s.pool.QueryRow(ctx, `SELECT payload_json FROM webhook_events WHERE tenant_id = $1 AND delivery_id = $2`, tenantID, strings.TrimSpace(deliveryID)).Scan(&payload)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return nil, fmt.Errorf("webhook event not found")
		}
		return nil, fmt.Errorf("get webhook event payload by delivery id: %w", err)
	}
	return json.RawMessage(payload), nil
}

func (s *WebhookEventStore) SaveAuditLog(ctx context.Context, item AuditLogRecord) error {
	tenantID := tenantIDFromCtx(ctx)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_logs (tenant_id, actor, action, target, target_id, payload)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, tenantID, strings.TrimSpace(item.Actor), strings.TrimSpace(item.Action), strings.TrimSpace(item.Target), strings.TrimSpace(item.TargetID), item.Payload)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) GetAdminUserByUsername(ctx context.Context, username string) (AdminUser, error) {
	tenantID := tenantIDFromCtx(ctx)
	var user AdminUser
	var lastLoginAt time.Time
	var permissionsJSON string
	name := strings.TrimSpace(username)
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, is_active, role, permissions, created_at, updated_at, COALESCE(last_login_at, 'epoch'::timestamptz)
		FROM admin_users
		WHERE tenant_id = $1
		  AND username = $2
		LIMIT 1
	`, tenantID, name).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsActive, &user.Role, &permissionsJSON, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return user, fmt.Errorf("admin user not found")
		}
		return user, fmt.Errorf("get admin user by username: %w", err)
	}
	if err := json.Unmarshal([]byte(permissionsJSON), &user.Permissions); err != nil {
		return user, fmt.Errorf("parse permissions: %w", err)
	}
	if !lastLoginAt.Equal(time.Unix(0, 0).UTC()) {
		ts := lastLoginAt.UTC()
		user.LastLoginAt = &ts
	}
	return user, nil
}
func (s *WebhookEventStore) ListAdminUsers(ctx context.Context, limit int, offset int) ([]AdminUser, int64, error) {
	tenantID := tenantIDFromCtx(ctx)
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users WHERE tenant_id = $1`, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count admin users: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, username, password_hash, is_active, role, permissions, created_at, updated_at, COALESCE(last_login_at, 'epoch'::timestamptz)
		FROM admin_users
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query admin users: %w", err)
	}
	defer rows.Close()

	items := make([]AdminUser, 0, limit)
	for rows.Next() {
		var user AdminUser
		var lastLoginAt time.Time
		var permissionsJSON string
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsActive, &user.Role, &permissionsJSON, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt); err != nil {
			return nil, 0, fmt.Errorf("scan admin user: %w", err)
		}

		// 解析permissions JSON
		if err := json.Unmarshal([]byte(permissionsJSON), &user.Permissions); err != nil {
			return nil, 0, fmt.Errorf("parse permissions: %w", err)
		}

		if !lastLoginAt.IsZero() && lastLoginAt.Unix() > 0 {
			user.LastLoginAt = &lastLoginAt
		}

		items = append(items, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate admin users: %w", err)
	}

	return items, total, nil
}

func (s *WebhookEventStore) CreateAdminUser(ctx context.Context, user AdminUser) (int64, error) {
	tenantID := tenantIDFromCtx(ctx)
	permissionsJSON, err := json.Marshal(user.Permissions)
	if err != nil {
		return 0, fmt.Errorf("marshal permissions: %w", err)
	}

	var id int64
	err = s.pool.QueryRow(ctx, `
		INSERT INTO admin_users (tenant_id, username, password_hash, is_active, role, permissions)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, tenantID, strings.TrimSpace(user.Username), user.PasswordHash, user.IsActive, strings.TrimSpace(user.Role), permissionsJSON).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert admin user: %w", err)
	}

	return id, nil
}

func (s *WebhookEventStore) UpdateAdminUser(ctx context.Context, id int64, user AdminUser) error {
	tenantID := tenantIDFromCtx(ctx)
	permissionsJSON, err := json.Marshal(user.Permissions)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}

	result, err := s.pool.Exec(ctx, `
		UPDATE admin_users
		SET username = $1, password_hash = $2, is_active = $3, role = $4, permissions = $5, updated_at = NOW()
		WHERE id = $6
		  AND tenant_id = $7
	`, strings.TrimSpace(user.Username), user.PasswordHash, user.IsActive, strings.TrimSpace(user.Role), permissionsJSON, id, tenantID)
	if err != nil {
		return fmt.Errorf("update admin user: %w", err)
	}

	affected := result.RowsAffected()
	_ = affected // 使用变量避免unused错误
	if affected == 0 {
		return fmt.Errorf("admin user not found")
	}

	return nil
}

func (s *WebhookEventStore) DeleteAdminUser(ctx context.Context, id int64) error {
	tenantID := tenantIDFromCtx(ctx)
	result, err := s.pool.Exec(ctx, `DELETE FROM admin_users WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete admin user: %w", err)
	}

	affected := result.RowsAffected()
	_ = affected // 使用变量避免unused错误
	if affected == 0 {
		return fmt.Errorf("admin user not found")
	}

	return nil
}

func (s *WebhookEventStore) GetAdminUserByID(ctx context.Context, id int64) (AdminUser, error) {
	tenantID := tenantIDFromCtx(ctx)
	var user AdminUser
	var lastLoginAt time.Time
	var permissionsJSON string
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, is_active, role, permissions, created_at, updated_at, COALESCE(last_login_at, 'epoch'::timestamptz)
		FROM admin_users
		WHERE id = $1
		  AND tenant_id = $2
	`, id, tenantID).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsActive, &user.Role, &permissionsJSON, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, fmt.Errorf("admin user not found")
		}
		return user, fmt.Errorf("get admin user by id: %w", err)
	}

	// 解析permissions JSON
	if err := json.Unmarshal([]byte(permissionsJSON), &user.Permissions); err != nil {
		return user, fmt.Errorf("parse permissions: %w", err)
	}

	if !lastLoginAt.IsZero() && lastLoginAt.Unix() > 0 {
		user.LastLoginAt = &lastLoginAt
	}

	return user, nil
}

func (s *WebhookEventStore) UpdateAdminUserActive(ctx context.Context, id int64, isActive bool) error {
	tenantID := tenantIDFromCtx(ctx)
	result, err := s.pool.Exec(ctx, `
		UPDATE admin_users
		SET is_active = $1, updated_at = NOW()
		WHERE id = $2
		  AND tenant_id = $3
	`, isActive, id, tenantID)
	if err != nil {
		return fmt.Errorf("update admin user active: %w", err)
	}

	affected := result.RowsAffected()
	_ = affected // 使用变量避免unused错误
	if affected == 0 {
		return fmt.Errorf("admin user not found")
	}

	return nil
}

func (s *WebhookEventStore) UpdateAdminUserLastLogin(ctx context.Context, id int64, at time.Time) error {
	tenantID := tenantIDFromCtx(ctx)
	result, err := s.pool.Exec(ctx, `UPDATE admin_users SET last_login_at = $2, updated_at = NOW() WHERE id = $1 AND tenant_id = $3`, id, at.UTC(), tenantID)
	if err != nil {
		return fmt.Errorf("update admin user last login: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("admin user not found")
	}
	return nil
}

func (s *WebhookEventStore) EnsureBootstrapAdminUser(ctx context.Context, username string, passwordHash string) error {
	tenantID := tenantIDFromCtx(ctx)
	name := strings.TrimSpace(username)
	hash := strings.TrimSpace(passwordHash)
	if name == "" || hash == "" {
		return nil
	}

	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users WHERE tenant_id = $1`, tenantID).Scan(&total); err != nil {
		return fmt.Errorf("count admin users: %w", err)
	}
	if total > 0 {
		return nil
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO admin_users (tenant_id, username, password_hash, is_active, role, permissions)
		VALUES ($1, $2, $3, TRUE, 'admin', '["read","write","admin"]'::jsonb)
		ON CONFLICT (tenant_id, username) DO NOTHING
	`, tenantID, name, hash)
	if err != nil {
		return fmt.Errorf("bootstrap admin user: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) ListAuditLogs(ctx context.Context, limit int, offset int, actor string, action string, since *time.Time) ([]AuditLogRecord, int64, error) {
	tenantID := tenantIDFromCtx(ctx)
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
		WHERE tenant_id = $1
		  AND ($2 = '' OR actor = $2)
		  AND ($3 = '' OR action = $3)
		  AND (NOT $4 OR created_at >= $5)
	`, tenantID, ac, act, hasSince, sinceTime).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, actor, action, target, target_id, payload, created_at
		FROM audit_logs
		WHERE tenant_id = $1
		  AND ($2 = '' OR actor = $2)
		  AND ($3 = '' OR action = $3)
		  AND (NOT $4 OR created_at >= $5)
		ORDER BY created_at DESC
		LIMIT $6 OFFSET $7
	`, tenantID, ac, act, hasSince, sinceTime, limit, offset)
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
	tenantID := tenantIDFromCtx(ctx)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_delivery_metrics (tenant_id, event_type, delivery_id, success, processing_ms, recorded_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, tenantID, strings.TrimSpace(metric.EventType), strings.TrimSpace(metric.DeliveryID), metric.Success, metric.ProcessingMS, metric.RecordedAtUTC)
	if err != nil {
		return fmt.Errorf("insert delivery metric: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) GetMetricsOverview(ctx context.Context, since time.Time) (MetricsOverview, error) {
	tenantID := tenantIDFromCtx(ctx)
	var out MetricsOverview
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_events WHERE tenant_id = $1 AND received_at >= $2`, tenantID, since).Scan(&out.Events24h); err != nil {
		return out, fmt.Errorf("count events metrics: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_alerts WHERE tenant_id = $1 AND created_at >= $2`, tenantID, since).Scan(&out.Alerts24h); err != nil {
		return out, fmt.Errorf("count alerts metrics: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_action_failures WHERE tenant_id = $1 AND occurred_at >= $2 AND is_resolved = FALSE`, tenantID, since).Scan(&out.Failures24h); err != nil {
		return out, fmt.Errorf("count failures metrics: %w", err)
	}

	var total int64
	var success int64
	var avgProcessing sql.NullFloat64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*), COALESCE(SUM(CASE WHEN success THEN 1 ELSE 0 END),0), AVG(processing_ms)::float8 FROM webhook_delivery_metrics WHERE tenant_id = $1 AND recorded_at >= $2`, tenantID, since).Scan(&total, &success, &avgProcessing); err != nil {
		return out, fmt.Errorf("count delivery metrics: %w", err)
	}
	out.DeliveryAttempts = total
	out.DeliverySuccess = success
	if total > 0 {
		out.SuccessRate24h = (float64(success) / float64(total)) * 100
		out.FailureRate24h = 100 - out.SuccessRate24h
		out.EstimatedManualMinutes24h = float64(total-success) * 2
		if out.Events24h > 0 {
			out.AutomationCoverage24h = (float64(total) / float64(out.Events24h)) * 100
			if out.AutomationCoverage24h > 100 {
				out.AutomationCoverage24h = 100
			}
		}
	}
	if avgProcessing.Valid {
		out.AvgProcessingMS24h = avgProcessing.Float64
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_action_failures WHERE tenant_id = $1 AND occurred_at >= $2 AND is_resolved`, tenantID, since).Scan(&out.ResolvedFailures24h); err != nil {
		return out, fmt.Errorf("count resolved failures metrics: %w", err)
	}

	rows, err := s.pool.Query(ctx, `SELECT processing_ms FROM webhook_delivery_metrics WHERE tenant_id = $1 AND recorded_at >= $2 ORDER BY processing_ms ASC`, tenantID, since)
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
		if out.AvgProcessingMS24h == 0 {
			var sum int64
			for _, v := range latencies {
				sum += v
			}
			out.AvgProcessingMS24h = float64(sum) / float64(len(latencies))
		}
	}
	return out, nil
}

func (s *WebhookEventStore) GetMetricsTimeSeries(ctx context.Context, since time.Time, intervalMinutes int) ([]MetricsTimePoint, error) {
	tenantID := tenantIDFromCtx(ctx)
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
		rows, err := s.pool.Query(ctx, query, tenantID, since)
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

	if err := fill(`SELECT received_at FROM webhook_events WHERE tenant_id = $1 AND received_at >= $2`, func(p *MetricsTimePoint, _ int64) { p.Events++ }); err != nil {
		return nil, fmt.Errorf("fill events metrics timeseries: %w", err)
	}
	if err := fill(`SELECT created_at FROM webhook_alerts WHERE tenant_id = $1 AND created_at >= $2`, func(p *MetricsTimePoint, _ int64) { p.Alerts++ }); err != nil {
		return nil, fmt.Errorf("fill alerts metrics timeseries: %w", err)
	}
	if err := fill(`SELECT occurred_at FROM webhook_action_failures WHERE tenant_id = $1 AND occurred_at >= $2 AND is_resolved = FALSE`, func(p *MetricsTimePoint, _ int64) { p.Failures++ }); err != nil {
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

func (s *WebhookEventStore) ListTenants(ctx context.Context) ([]TenantRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, is_active, created_at, updated_at
		FROM tenants
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query tenants: %w", err)
	}
	defer rows.Close()

	items := make([]TenantRecord, 0, 16)
	for rows.Next() {
		var item TenantRecord
		if err := rows.Scan(&item.ID, &item.Name, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenants: %w", err)
	}
	return items, nil
}

func (s *WebhookEventStore) CreateTenant(ctx context.Context, id string, name string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, is_active)
		VALUES ($1, $2, TRUE)
	`, strings.TrimSpace(id), strings.TrimSpace(name))
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return nil
}

func (s *WebhookEventStore) UpdateTenantActive(ctx context.Context, id string, isActive bool) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE tenants
		SET is_active = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, strings.TrimSpace(id), isActive)
	if err != nil {
		return fmt.Errorf("update tenant active: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}

func (s *WebhookEventStore) ensureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tenants (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create tenants table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, is_active)
		VALUES ('default', 'Default Tenant', TRUE)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("seed default tenant: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_events (
			id BIGSERIAL PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			delivery_id TEXT NOT NULL,
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
		CREATE TABLE IF NOT EXISTS webhook_alerts (
			id BIGSERIAL PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
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
			UNIQUE (tenant_id, delivery_id, suggestion_type, suggestion_value, rule_matched)
		)
	`)
	if err != nil {
		return fmt.Errorf("create webhook_alerts table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_rules (
			id BIGSERIAL PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			event_type TEXT NOT NULL,
			keyword TEXT NOT NULL,
			suggestion_type TEXT NOT NULL,
			suggestion_value TEXT NOT NULL,
			reason TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (tenant_id, event_type, keyword, suggestion_type, suggestion_value)
		)
	`)
	if err != nil {
		return fmt.Errorf("create webhook_rules table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_rule_versions (
			id BIGSERIAL PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			version BIGINT NOT NULL,
			rules_json JSONB NOT NULL,
			rule_count INT NOT NULL,
			created_by TEXT NOT NULL,
			source_version BIGINT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (tenant_id, version)
		)
	`)
	if err != nil {
		return fmt.Errorf("create webhook_rule_versions table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_action_failures (
			id BIGSERIAL PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
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
		CREATE TABLE IF NOT EXISTS audit_logs (
			id BIGSERIAL PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
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
		CREATE TABLE IF NOT EXISTS admin_users (
			id BIGSERIAL PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			username TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			role TEXT NOT NULL DEFAULT 'viewer',
			permissions JSONB NOT NULL DEFAULT '["read"]'::jsonb,
			last_login_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (tenant_id, username)
		)
	`)
	if err != nil {
		return fmt.Errorf("create admin_users table: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_delivery_metrics (
			id BIGSERIAL PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
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

	// 兼容历史表结构
	_, _ = s.pool.Exec(ctx, `ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'viewer'`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS permissions JSONB NOT NULL DEFAULT '["read"]'::jsonb`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ NULL`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS last_retry_status TEXT NOT NULL DEFAULT 'never'`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS last_retry_message TEXT NOT NULL DEFAULT ''`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS last_retry_at TIMESTAMPTZ NULL`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS is_resolved BOOLEAN NOT NULL DEFAULT FALSE`)

	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_events ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default'`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_alerts ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default'`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_rules ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default'`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_rule_versions ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default'`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default'`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default'`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE webhook_delivery_metrics ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default'`)
	_, _ = s.pool.Exec(ctx, `ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default'`)

	_, _ = s.pool.Exec(ctx, `UPDATE webhook_events SET tenant_id = 'default' WHERE tenant_id IS NULL OR tenant_id = ''`)
	_, _ = s.pool.Exec(ctx, `UPDATE webhook_alerts SET tenant_id = 'default' WHERE tenant_id IS NULL OR tenant_id = ''`)
	_, _ = s.pool.Exec(ctx, `UPDATE webhook_rules SET tenant_id = 'default' WHERE tenant_id IS NULL OR tenant_id = ''`)
	_, _ = s.pool.Exec(ctx, `UPDATE webhook_rule_versions SET tenant_id = 'default' WHERE tenant_id IS NULL OR tenant_id = ''`)
	_, _ = s.pool.Exec(ctx, `UPDATE webhook_action_failures SET tenant_id = 'default' WHERE tenant_id IS NULL OR tenant_id = ''`)
	_, _ = s.pool.Exec(ctx, `UPDATE audit_logs SET tenant_id = 'default' WHERE tenant_id IS NULL OR tenant_id = ''`)
	_, _ = s.pool.Exec(ctx, `UPDATE webhook_delivery_metrics SET tenant_id = 'default' WHERE tenant_id IS NULL OR tenant_id = ''`)
	_, _ = s.pool.Exec(ctx, `UPDATE admin_users SET tenant_id = 'default' WHERE tenant_id IS NULL OR tenant_id = ''`)

	_, _ = s.pool.Exec(ctx, `DO $$ BEGIN IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'webhook_events_delivery_id_key') THEN ALTER TABLE webhook_events DROP CONSTRAINT webhook_events_delivery_id_key; END IF; END $$;`)
	_, _ = s.pool.Exec(ctx, `DO $$ BEGIN IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'admin_users_username_key') THEN ALTER TABLE admin_users DROP CONSTRAINT admin_users_username_key; END IF; END $$;`)

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
		CREATE INDEX IF NOT EXISTS idx_webhook_events_tenant_id
		ON webhook_events (tenant_id)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_events_tenant_id: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS uk_webhook_events_tenant_delivery_id
		ON webhook_events (tenant_id, delivery_id)
	`)
	if err != nil {
		return fmt.Errorf("create uk_webhook_events_tenant_delivery_id: %w", err)
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
		CREATE INDEX IF NOT EXISTS idx_webhook_alerts_tenant_id
		ON webhook_alerts (tenant_id)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_alerts_tenant_id: %w", err)
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
		CREATE INDEX IF NOT EXISTS idx_webhook_rules_tenant_id
		ON webhook_rules (tenant_id)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_rules_tenant_id: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_rule_versions_tenant_version
		ON webhook_rule_versions (tenant_id, version DESC)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_rule_versions_tenant_version: %w", err)
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
	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_action_failures_tenant_id
		ON webhook_action_failures (tenant_id)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_action_failures_tenant_id: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at
		ON audit_logs (created_at DESC)
	`)
	if err != nil {
		return fmt.Errorf("create idx_audit_logs_created_at: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_action
		ON audit_logs (actor, action)
	`)
	if err != nil {
		return fmt.Errorf("create idx_audit_logs_actor_action: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id
		ON audit_logs (tenant_id)
	`)
	if err != nil {
		return fmt.Errorf("create idx_audit_logs_tenant_id: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_admin_users_is_active
		ON admin_users (is_active)
	`)
	if err != nil {
		return fmt.Errorf("create idx_admin_users_is_active: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_admin_users_username
		ON admin_users (tenant_id, username)
	`)
	if err != nil {
		return fmt.Errorf("create idx_admin_users_username: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS uk_admin_users_tenant_username
		ON admin_users (tenant_id, username)
	`)
	if err != nil {
		return fmt.Errorf("create uk_admin_users_tenant_username: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_delivery_metrics_recorded_at
		ON webhook_delivery_metrics (recorded_at DESC)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_delivery_metrics_recorded_at: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_webhook_delivery_metrics_tenant_id
		ON webhook_delivery_metrics (tenant_id)
	`)
	if err != nil {
		return fmt.Errorf("create idx_webhook_delivery_metrics_tenant_id: %w", err)
	}

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
