package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

type MySQLWebhookEventStore struct {
	db *sql.DB
}

func newMySQLWebhookEventStore(ctx context.Context, databaseURL string) (*MySQLWebhookEventStore, error) {
	dsn, err := mysqlURLToDSN(databaseURL)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	store := &MySQLWebhookEventStore{db: db}
	if err := store.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func mysqlURLToDSN(databaseURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(databaseURL))
	if err != nil {
		return "", fmt.Errorf("parse mysql DATABASE_URL: %w", err)
	}
	if !strings.EqualFold(u.Scheme, "mysql") {
		return "", fmt.Errorf("unsupported mysql scheme: %s", u.Scheme)
	}
	if u.User == nil {
		return "", fmt.Errorf("mysql DATABASE_URL missing user")
	}
	if strings.TrimSpace(u.Host) == "" {
		u.Host = "127.0.0.1:3306"
	}

	username := u.User.Username()
	password, _ := u.User.Password()
	host := u.Host
	dbName := strings.TrimPrefix(u.Path, "/")
	if strings.TrimSpace(dbName) == "" {
		return "", fmt.Errorf("mysql DATABASE_URL missing database name")
	}

	params := u.Query()
	if params.Get("parseTime") == "" {
		params.Set("parseTime", "true")
	}
	if params.Get("charset") == "" {
		params.Set("charset", "utf8mb4")
	}
	if params.Get("loc") == "" {
		params.Set("loc", "UTC")
	}

	if strings.TrimSpace(host) == "" {
		host = "127.0.0.1:3306"
	}
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", username, password, host, dbName, params.Encode()), nil
}

func (s *MySQLWebhookEventStore) Close() {
	if s.db != nil {
		_ = s.db.Close()
	}
}

func (s *MySQLWebhookEventStore) SaveEvent(ctx context.Context, evt WebhookEvent) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_events (
			delivery_id, event_type, action,
			repository_full_name, sender_login, payload_json
		) VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE delivery_id = delivery_id
	`, evt.DeliveryID, evt.EventType, evt.Action, evt.RepositoryFullName, evt.SenderLogin, string(evt.PayloadJSON))
	if err != nil {
		return fmt.Errorf("insert webhook event: %w", err)
	}
	return nil
}

func (s *MySQLWebhookEventStore) SaveAlert(ctx context.Context, alert AlertRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_alerts (
			delivery_id, event_type, action, repository_full_name,
			sender_login, rule_matched, suggestion_type, suggestion_value, reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE delivery_id = delivery_id
	`, alert.DeliveryID, alert.EventType, alert.Action, alert.RepositoryFullName, alert.SenderLogin, alert.RuleMatched, alert.SuggestionType, alert.SuggestionValue, alert.Reason)
	if err != nil {
		return fmt.Errorf("insert webhook alert: %w", err)
	}
	return nil
}

func (s *MySQLWebhookEventStore) ListEvents(ctx context.Context, limit int, offset int, eventType string, action string) ([]WebhookEventRecord, int64, error) {
	et := strings.TrimSpace(eventType)
	ac := strings.TrimSpace(action)

	var total int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM webhook_events
		WHERE (? = '' OR event_type = ?)
		  AND (? = '' OR action = ?)
	`, et, et, ac, ac).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook events: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, delivery_id, event_type, action, repository_full_name, sender_login, payload_json, received_at
		FROM webhook_events
		WHERE (? = '' OR event_type = ?)
		  AND (? = '' OR action = ?)
		ORDER BY received_at DESC
		LIMIT ? OFFSET ?
	`, et, et, ac, ac, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query webhook events: %w", err)
	}
	defer rows.Close()

	items := make([]WebhookEventRecord, 0, limit)
	for rows.Next() {
		var rec WebhookEventRecord
		if err := rows.Scan(&rec.ID, &rec.DeliveryID, &rec.EventType, &rec.Action, &rec.RepositoryFullName, &rec.SenderLogin, &rec.PayloadJSON, &rec.ReceivedAt); err != nil {
			return nil, 0, fmt.Errorf("scan webhook event row: %w", err)
		}
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate webhook events: %w", err)
	}

	return items, total, nil
}

func (s *MySQLWebhookEventStore) ListAlerts(ctx context.Context, limit int, offset int, eventType string, action string, suggestionType string) ([]AlertRecord, int64, error) {
	et := strings.TrimSpace(eventType)
	ac := strings.TrimSpace(action)
	st := strings.TrimSpace(suggestionType)

	var total int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM webhook_alerts
		WHERE (? = '' OR event_type = ?)
		  AND (? = '' OR action = ?)
		  AND (? = '' OR suggestion_type = ?)
	`, et, et, ac, ac, st, st).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook alerts: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT delivery_id, event_type, action, repository_full_name, sender_login,
		       rule_matched, suggestion_type, suggestion_value, reason, created_at
		FROM webhook_alerts
		WHERE (? = '' OR event_type = ?)
		  AND (? = '' OR action = ?)
		  AND (? = '' OR suggestion_type = ?)
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, et, et, ac, ac, st, st, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query webhook alerts: %w", err)
	}
	defer rows.Close()

	items := make([]AlertRecord, 0, limit)
	for rows.Next() {
		var rec AlertRecord
		if err := rows.Scan(&rec.DeliveryID, &rec.EventType, &rec.Action, &rec.RepositoryFullName, &rec.SenderLogin, &rec.RuleMatched, &rec.SuggestionType, &rec.SuggestionValue, &rec.Reason, &rec.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan webhook alert row: %w", err)
		}
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate webhook alerts: %w", err)
	}

	return items, total, nil
}

func (s *MySQLWebhookEventStore) ListRules(ctx context.Context, limit int, offset int, eventType string, keyword string, activeOnly bool) ([]RuleRecord, int64, error) {
	et := strings.TrimSpace(eventType)
	kw := strings.TrimSpace(keyword)
	kwLike := "%" + kw + "%"

	var total int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM webhook_rules
		WHERE (? = '' OR event_type = ?)
		  AND (? = '' OR LOWER(keyword) LIKE LOWER(?))
		  AND (NOT ? OR is_active = true)
	`, et, et, kw, kwLike, activeOnly).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook rules: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, event_type, keyword, suggestion_type, suggestion_value, reason, is_active, created_at
		FROM webhook_rules
		WHERE (? = '' OR event_type = ?)
		  AND (? = '' OR LOWER(keyword) LIKE LOWER(?))
		  AND (NOT ? OR is_active = true)
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, et, et, kw, kwLike, activeOnly, limit, offset)
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

func listDistinctNonEmptyMySQL(ctx context.Context, db *sql.DB, q string) ([]string, error) {
	rows, err := db.QueryContext(ctx, q)
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

func (s *MySQLWebhookEventStore) ListEventFilterOptions(ctx context.Context) (EventFilterOptions, error) {
	et, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT event_type FROM webhook_events WHERE event_type <> '' ORDER BY event_type ASC`)
	if err != nil {
		return EventFilterOptions{}, fmt.Errorf("list distinct event_type from webhook_events: %w", err)
	}
	ac, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT action FROM webhook_events WHERE action <> '' ORDER BY action ASC`)
	if err != nil {
		return EventFilterOptions{}, fmt.Errorf("list distinct action from webhook_events: %w", err)
	}
	repo, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT repository_full_name FROM webhook_events WHERE repository_full_name <> '' ORDER BY repository_full_name ASC`)
	if err != nil {
		return EventFilterOptions{}, fmt.Errorf("list distinct repository from webhook_events: %w", err)
	}
	sender, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT sender_login FROM webhook_events WHERE sender_login <> '' ORDER BY sender_login ASC`)
	if err != nil {
		return EventFilterOptions{}, fmt.Errorf("list distinct sender from webhook_events: %w", err)
	}
	return EventFilterOptions{EventTypes: et, Actions: ac, Repositories: repo, Senders: sender}, nil
}

func (s *MySQLWebhookEventStore) ListAlertFilterOptions(ctx context.Context) (AlertFilterOptions, error) {
	et, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT event_type FROM webhook_alerts WHERE event_type <> '' ORDER BY event_type ASC`)
	if err != nil {
		return AlertFilterOptions{}, fmt.Errorf("list distinct event_type from webhook_alerts: %w", err)
	}
	ac, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT action FROM webhook_alerts WHERE action <> '' ORDER BY action ASC`)
	if err != nil {
		return AlertFilterOptions{}, fmt.Errorf("list distinct action from webhook_alerts: %w", err)
	}
	st, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT suggestion_type FROM webhook_alerts WHERE suggestion_type <> '' ORDER BY suggestion_type ASC`)
	if err != nil {
		return AlertFilterOptions{}, fmt.Errorf("list distinct suggestion_type from webhook_alerts: %w", err)
	}
	repo, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT repository_full_name FROM webhook_alerts WHERE repository_full_name <> '' ORDER BY repository_full_name ASC`)
	if err != nil {
		return AlertFilterOptions{}, fmt.Errorf("list distinct repository from webhook_alerts: %w", err)
	}
	sender, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT sender_login FROM webhook_alerts WHERE sender_login <> '' ORDER BY sender_login ASC`)
	if err != nil {
		return AlertFilterOptions{}, fmt.Errorf("list distinct sender from webhook_alerts: %w", err)
	}
	return AlertFilterOptions{EventTypes: et, Actions: ac, SuggestionTypes: st, Repositories: repo, Senders: sender}, nil
}

func (s *MySQLWebhookEventStore) ListRuleFilterOptions(ctx context.Context) (RuleFilterOptions, error) {
	et, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT event_type FROM webhook_rules WHERE event_type <> '' ORDER BY event_type ASC`)
	if err != nil {
		return RuleFilterOptions{}, fmt.Errorf("list distinct event_type from webhook_rules: %w", err)
	}
	st, err := listDistinctNonEmptyMySQL(ctx, s.db, `SELECT DISTINCT suggestion_type FROM webhook_rules WHERE suggestion_type <> '' ORDER BY suggestion_type ASC`)
	if err != nil {
		return RuleFilterOptions{}, fmt.Errorf("list distinct suggestion_type from webhook_rules: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT is_active FROM webhook_rules ORDER BY is_active DESC`)
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

func (s *MySQLWebhookEventStore) CreateRule(ctx context.Context, rule RuleRecord) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_rules (event_type, keyword, suggestion_type, suggestion_value, reason, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
	`, strings.TrimSpace(rule.EventType), strings.TrimSpace(rule.Keyword), strings.TrimSpace(rule.SuggestionType), strings.TrimSpace(rule.SuggestionValue), strings.TrimSpace(rule.Reason), rule.IsActive)
	if err != nil {
		return 0, fmt.Errorf("insert webhook rule: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get inserted webhook rule id: %w", err)
	}
	return id, nil
}

func (s *MySQLWebhookEventStore) UpdateRuleActive(ctx context.Context, id int64, isActive bool) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE webhook_rules
		SET is_active = ?
		WHERE id = ?
	`, isActive, id)
	if err != nil {
		return fmt.Errorf("update webhook rule active: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows for rule update: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

func (s *MySQLWebhookEventStore) SaveActionExecutionFailure(ctx context.Context, item ActionExecutionFailure) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_action_failures (
			delivery_id, event_type, action, repository_full_name,
			suggestion_type, suggestion_value, error_message, attempt_count,
			retry_count, last_retry_status, last_retry_message, last_retry_at, is_resolved
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 'never', '', NULL, FALSE)
	`, item.DeliveryID, item.EventType, item.Action, item.RepositoryFullName, item.SuggestionType, item.SuggestionValue, item.ErrorMessage, item.AttemptCount)
	if err != nil {
		return fmt.Errorf("insert webhook action failure: %w", err)
	}
	return nil
}

func (s *MySQLWebhookEventStore) ListActionExecutionFailures(ctx context.Context, limit int, offset int, includeResolved bool) ([]ActionExecutionFailureRecord, int64, error) {
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_action_failures WHERE (? OR NOT is_resolved)`, includeResolved).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count action failures: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, delivery_id, event_type, action, repository_full_name, suggestion_type, suggestion_value, error_message, attempt_count, retry_count, last_retry_status, last_retry_message, last_retry_at, is_resolved, occurred_at
		FROM webhook_action_failures
		WHERE (? OR NOT is_resolved)
		ORDER BY occurred_at DESC
		LIMIT ? OFFSET ?
	`, includeResolved, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query action failures: %w", err)
	}
	defer rows.Close()

	items := make([]ActionExecutionFailureRecord, 0, limit)
	for rows.Next() {
		var rec ActionExecutionFailureRecord
		var lastRetryAt sql.NullTime
		if err := rows.Scan(&rec.ID, &rec.DeliveryID, &rec.EventType, &rec.Action, &rec.RepositoryFullName, &rec.SuggestionType, &rec.SuggestionValue, &rec.ErrorMessage, &rec.AttemptCount, &rec.RetryCount, &rec.LastRetryStatus, &rec.LastRetryMessage, &lastRetryAt, &rec.IsResolved, &rec.OccurredAt); err != nil {
			return nil, 0, fmt.Errorf("scan action failure: %w", err)
		}
		normalizeLastRetryAt(&rec, lastRetryAt)
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate action failures: %w", err)
	}
	return items, total, nil
}

func (s *MySQLWebhookEventStore) GetActionExecutionFailureByID(ctx context.Context, id int64) (ActionExecutionFailureRecord, error) {
	var rec ActionExecutionFailureRecord
	var lastRetryAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, delivery_id, event_type, action, repository_full_name, suggestion_type, suggestion_value, error_message, attempt_count, retry_count, last_retry_status, last_retry_message, last_retry_at, is_resolved, occurred_at
		FROM webhook_action_failures
		WHERE id = ?
	`, id).Scan(&rec.ID, &rec.DeliveryID, &rec.EventType, &rec.Action, &rec.RepositoryFullName, &rec.SuggestionType, &rec.SuggestionValue, &rec.ErrorMessage, &rec.AttemptCount, &rec.RetryCount, &rec.LastRetryStatus, &rec.LastRetryMessage, &lastRetryAt, &rec.IsResolved, &rec.OccurredAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rec, fmt.Errorf("action failure not found")
		}
		return rec, fmt.Errorf("get action failure by id: %w", err)
	}
	normalizeLastRetryAt(&rec, lastRetryAt)
	return rec, nil
}

func normalizeLastRetryAt(rec *ActionExecutionFailureRecord, t sql.NullTime) {
	if !t.Valid {
		rec.LastRetryAt = time.Time{}
		return
	}
	rec.LastRetryAt = t.Time
}

func (s *MySQLWebhookEventStore) UpdateActionFailureRetryResult(ctx context.Context, id int64, success bool, message string) error {
	status := "failed"
	resolved := false
	if success {
		status = "success"
		resolved = true
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE webhook_action_failures
		SET retry_count = retry_count + 1,
		    last_retry_status = ?,
		    last_retry_message = ?,
		    last_retry_at = CURRENT_TIMESTAMP(6),
		    is_resolved = ?
		WHERE id = ?
	`, status, strings.TrimSpace(message), resolved, id)
	if err != nil {
		return fmt.Errorf("update action failure retry result: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows for action failure retry update: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("action failure not found")
	}
	return nil
}

func (s *MySQLWebhookEventStore) GetWebhookEventPayloadByDeliveryID(ctx context.Context, deliveryID string) (json.RawMessage, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload_json FROM webhook_events WHERE delivery_id = ?`, strings.TrimSpace(deliveryID)).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("webhook event not found")
		}
		return nil, fmt.Errorf("get webhook event payload by delivery id: %w", err)
	}
	return json.RawMessage(payload), nil
}

func (s *MySQLWebhookEventStore) SaveAuditLog(ctx context.Context, item AuditLogRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_logs (actor, action, target, target_id, payload)
		VALUES (?, ?, ?, ?, ?)
	`, strings.TrimSpace(item.Actor), strings.TrimSpace(item.Action), strings.TrimSpace(item.Target), strings.TrimSpace(item.TargetID), item.Payload)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (s *MySQLWebhookEventStore) ListAuditLogs(ctx context.Context, limit int, offset int, actor string, action string, since *time.Time) ([]AuditLogRecord, int64, error) {
	ac := strings.TrimSpace(actor)
	act := strings.TrimSpace(action)
	hasSince := since != nil
	sinceTime := time.Unix(0, 0).UTC()
	if since != nil {
		sinceTime = since.UTC()
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM audit_logs
		WHERE (? = '' OR actor = ?)
		  AND (? = '' OR action = ?)
		  AND (NOT ? OR created_at >= ?)
	`, ac, ac, act, act, hasSince, sinceTime).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, actor, action, target, target_id, payload, created_at
		FROM audit_logs
		WHERE (? = '' OR actor = ?)
		  AND (? = '' OR action = ?)
		  AND (NOT ? OR created_at >= ?)
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, ac, ac, act, act, hasSince, sinceTime, limit, offset)
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

func (s *MySQLWebhookEventStore) GetAdminUserByUsername(ctx context.Context, username string) (AdminUser, error) {
	var user AdminUser
	var lastLogin sql.NullTime
	name := strings.TrimSpace(username)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, is_active, created_at, updated_at, last_login_at
		FROM admin_users
		WHERE username = ?
		LIMIT 1
	`, name).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &lastLogin)
	if err != nil {
	if errors.Is(err, sql.ErrNoRows) {
			return user, fmt.Errorf("admin user not found")
		}
		return user, fmt.Errorf("get admin user by username: %w", err)
	}
	if lastLogin.Valid {
		t := lastLogin.Time.UTC()
		user.LastLoginAt = &t
	}
	return user, nil
}

func (s *MySQLWebhookEventStore) UpdateAdminUserLastLogin(ctx context.Context, id int64, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE admin_users SET last_login_at = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE id = ?`, at.UTC(), id)
	if err != nil {
		return fmt.Errorf("update admin user last login: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows for admin user update: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("admin user not found")
	}
	return nil
}

func (s *MySQLWebhookEventStore) EnsureBootstrapAdminUser(ctx context.Context, username string, passwordHash string) error {
	name := strings.TrimSpace(username)
	hash := strings.TrimSpace(passwordHash)
	if name == "" || hash == "" {
		return nil
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&total); err != nil {
		return fmt.Errorf("count admin users: %w", err)
	}
	if total > 0 {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_users (username, password_hash, is_active)
		VALUES (?, ?, TRUE)
		ON DUPLICATE KEY UPDATE username = username
	`, name, hash)
	if err != nil {
		return fmt.Errorf("bootstrap admin user: %w", err)
	}
	return nil
}

func (s *MySQLWebhookEventStore) SaveDeliveryMetric(ctx context.Context, metric DeliveryMetric) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_delivery_metrics (event_type, delivery_id, success, processing_ms, recorded_at)
		VALUES (?, ?, ?, ?, ?)
	`, strings.TrimSpace(metric.EventType), strings.TrimSpace(metric.DeliveryID), metric.Success, metric.ProcessingMS, metric.RecordedAtUTC)
	if err != nil {
		return fmt.Errorf("insert delivery metric: %w", err)
	}
	return nil
}

func (s *MySQLWebhookEventStore) GetMetricsOverview(ctx context.Context, since time.Time) (MetricsOverview, error) {
	var out MetricsOverview
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_events WHERE received_at >= ?`, since).Scan(&out.Events24h); err != nil {
		return out, fmt.Errorf("count events metrics: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_alerts WHERE created_at >= ?`, since).Scan(&out.Alerts24h); err != nil {
		return out, fmt.Errorf("count alerts metrics: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_action_failures WHERE occurred_at >= ? AND NOT is_resolved`, since).Scan(&out.Failures24h); err != nil {
		return out, fmt.Errorf("count failures metrics: %w", err)
	}

	var total int64
	var success int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(CASE WHEN success THEN 1 ELSE 0 END),0) FROM webhook_delivery_metrics WHERE recorded_at >= ?`, since).Scan(&total, &success); err != nil {
		return out, fmt.Errorf("count delivery metrics: %w", err)
	}
	if total > 0 {
		out.SuccessRate24h = (float64(success) / float64(total)) * 100
	}

	rows, err := s.db.QueryContext(ctx, `SELECT processing_ms FROM webhook_delivery_metrics WHERE recorded_at >= ? ORDER BY processing_ms ASC`, since)
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

func (s *MySQLWebhookEventStore) GetMetricsTimeSeries(ctx context.Context, since time.Time, intervalMinutes int) ([]MetricsTimePoint, error) {
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

	fill := func(query string, assign func(*MetricsTimePoint)) error {
		rows, err := s.db.QueryContext(ctx, query, since)
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
				assign(p)
			}
		}
		return rows.Err()
	}

	if err := fill(`SELECT received_at FROM webhook_events WHERE received_at >= ?`, func(p *MetricsTimePoint) { p.Events++ }); err != nil {
		return nil, fmt.Errorf("fill events metrics timeseries: %w", err)
	}
	if err := fill(`SELECT created_at FROM webhook_alerts WHERE created_at >= ?`, func(p *MetricsTimePoint) { p.Alerts++ }); err != nil {
		return nil, fmt.Errorf("fill alerts metrics timeseries: %w", err)
	}
	if err := fill(`SELECT occurred_at FROM webhook_action_failures WHERE occurred_at >= ?`, func(p *MetricsTimePoint) { p.Failures++ }); err != nil {
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

func (s *MySQLWebhookEventStore) ensureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS webhook_events (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			delivery_id VARCHAR(191) NOT NULL,
			event_type VARCHAR(128) NOT NULL,
			action VARCHAR(128) NOT NULL,
			repository_full_name VARCHAR(255) NOT NULL,
			sender_login VARCHAR(255) NOT NULL,
			payload_json JSON NOT NULL,
			received_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			UNIQUE KEY uk_webhook_events_delivery_id (delivery_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE INDEX idx_webhook_events_received_at ON webhook_events (received_at)`,
		`CREATE INDEX idx_webhook_events_event_type ON webhook_events (event_type)`,
		`CREATE INDEX idx_webhook_events_action ON webhook_events (action)`,
		`CREATE INDEX idx_webhook_events_event_action ON webhook_events (event_type, action)`,

		`CREATE TABLE IF NOT EXISTS webhook_alerts (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			delivery_id VARCHAR(191) NOT NULL,
			event_type VARCHAR(128) NOT NULL,
			action VARCHAR(128) NOT NULL,
			repository_full_name VARCHAR(255) NOT NULL,
			sender_login VARCHAR(255) NOT NULL,
			rule_matched VARCHAR(255) NOT NULL,
			suggestion_type VARCHAR(128) NOT NULL,
			suggestion_value VARCHAR(191) NOT NULL,
			reason TEXT NOT NULL,
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			UNIQUE KEY uk_webhook_alerts_dedup (delivery_id, suggestion_type, suggestion_value, rule_matched)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE INDEX idx_webhook_alerts_created_at ON webhook_alerts (created_at)`,
		`CREATE INDEX idx_webhook_alerts_event_action ON webhook_alerts (event_type, action)`,
		`CREATE INDEX idx_webhook_alerts_suggestion_type ON webhook_alerts (suggestion_type)`,

		`CREATE TABLE IF NOT EXISTS webhook_rules (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			event_type VARCHAR(128) NOT NULL,
			keyword VARCHAR(255) NOT NULL,
			suggestion_type VARCHAR(128) NOT NULL,
			suggestion_value VARCHAR(191) NOT NULL,
			reason TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE INDEX idx_webhook_rules_event_type ON webhook_rules (event_type)`,
		`CREATE INDEX idx_webhook_rules_active ON webhook_rules (is_active)`,

		`CREATE TABLE IF NOT EXISTS webhook_action_failures (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			delivery_id VARCHAR(191) NOT NULL,
			event_type VARCHAR(128) NOT NULL,
			action VARCHAR(128) NOT NULL,
			repository_full_name VARCHAR(255) NOT NULL,
			suggestion_type VARCHAR(128) NOT NULL,
			suggestion_value VARCHAR(191) NOT NULL,
			error_message TEXT NOT NULL,
			attempt_count INT NOT NULL,
			retry_count INT NOT NULL DEFAULT 0,
			last_retry_status VARCHAR(32) NOT NULL DEFAULT 'never',
			last_retry_message TEXT NOT NULL,
			last_retry_at DATETIME(6) NULL,
			is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
			occurred_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE INDEX idx_webhook_action_failures_delivery ON webhook_action_failures (delivery_id)`,
		`CREATE INDEX idx_webhook_action_failures_occurred_at ON webhook_action_failures (occurred_at)`,

		`CREATE TABLE IF NOT EXISTS audit_logs (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			actor VARCHAR(191) NOT NULL,
			action VARCHAR(191) NOT NULL,
			target VARCHAR(191) NOT NULL,
			target_id VARCHAR(191) NOT NULL,
			payload TEXT NOT NULL,
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at)`,
		`CREATE INDEX idx_audit_logs_actor_action ON audit_logs (actor, action)`,

		`CREATE TABLE IF NOT EXISTS admin_users (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(191) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			last_login_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			UNIQUE KEY uk_admin_users_username (username)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE INDEX idx_admin_users_is_active ON admin_users (is_active)`,

		`CREATE TABLE IF NOT EXISTS webhook_delivery_metrics (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			event_type VARCHAR(128) NOT NULL,
			delivery_id VARCHAR(191) NOT NULL,
			success BOOLEAN NOT NULL,
			processing_ms BIGINT NOT NULL,
			recorded_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE INDEX idx_webhook_delivery_metrics_recorded_at ON webhook_delivery_metrics (recorded_at)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			if isMySQLDuplicateIndexError(err) {
				continue
			}
			return fmt.Errorf("ensure mysql schema: %w", err)
		}
	}
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN retry_count INT NOT NULL DEFAULT 0`)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN last_retry_status VARCHAR(32) NOT NULL DEFAULT 'never'`)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN last_retry_message TEXT NOT NULL`)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN last_retry_at DATETIME(6) NULL`)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE webhook_action_failures ADD COLUMN is_resolved BOOLEAN NOT NULL DEFAULT FALSE`)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE admin_users ADD COLUMN last_login_at DATETIME(6) NULL`)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE admin_users ADD COLUMN updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)`)
	return nil
}

func isMySQLDuplicateIndexError(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1061
	}
	return false
}
