# Requirements - maintainer-firewall (MVP)

## 1. Background

Open source maintainers are receiving increasing low-quality issues/PRs and repetitive submissions, which consume reviewer time. The project aims to provide a maintainability layer that receives GitHub events, classifies them by rules, and supports follow-up automation.

## 2. Product Goal

Build a self-hostable service that helps maintainers reduce noisy triage work and improve response efficiency.

## 3. Non-Goals (MVP)

- No full AI auto-review of code correctness
- No org-wide SSO / enterprise IAM in MVP
- No multi-platform integrations beyond GitHub webhook in MVP

## 4. Target Users

- Open source maintainers and repo owners
- Small teams managing high-volume GitHub issues/PRs

## 5. Core Use Cases

1. As a maintainer, I can receive GitHub webhook events reliably.
2. As a maintainer, I can verify webhook authenticity using signature validation.
3. As a maintainer, I can persist event metadata for audit and analysis.
4. As a maintainer, I can view basic event status in a web console.

## 6. Functional Requirements (MVP+)

### FR-1 Webhook Ingestion
- `POST /webhook/github` endpoint
- Validate `X-Hub-Signature-256` using HMAC-SHA256
- Return clear status for invalid signature / malformed payload

### FR-2 Event Persistence
- Save accepted events to PostgreSQL from two sources:
  - verified webhook ingest (`POST /webhook/github`)
  - GitHub pull sync (`GET /events?source=github&sync=true`)
- Event persistence must be idempotent by `delivery_id`.
- Minimum fields:
  - `delivery_id`
  - `event_type`
  - `action`
  - `repository_full_name`
  - `sender_login`
  - `received_at`
  - `payload_json`

### FR-3 Runtime Config + Health
- `GET /health`
- Runtime config APIs (protected):
  - `GET /config-status`
  - `GET /config-view`
  - `POST /config-update`
- Environment-driven config:
  - `PORT`
  - `GITHUB_WEBHOOK_SECRET`
  - `DATABASE_URL`
  - `ADMIN_USERNAME`
  - `ADMIN_PASSWORD`
  - `AUTH_ENV_FALLBACK` (whether auth can fallback to env credentials when DB admin user is missing)
  - `JWT_SECRET` (preferred)
  - `ACCESS_TOKEN` (legacy fallback secret)
  - `GITHUB_TOKEN` (optional for auto actions)

### FR-4 Console (Protected)
- React login page (`/login`)
- Protected console routes (dashboard/events/alerts/rules/failures/audit/system-config)
- Event list page with latest records
- Alerts list page with latest records
- Rules page with status toggle
- Failures page with retry actions and status hints
- Audit logs page with filters
- System config page for runtime env update

### FR-5 Rule Engine + Configurable Rules
- For `issues` and `pull_request` events, run rule matching
- Return suggested actions in webhook response (`label` / `comment`)
- Provide protected rules API:
  - `GET /rules`
  - `POST /rules`

### FR-6 Alert Persistence and Query
- Persist matched rule hits into alert records
- Provide protected `GET /alerts` with pagination/filter/total

### FR-7 Optional Action Automation
- If `GITHUB_TOKEN` is configured, execute suggested GitHub actions:
  - add label
  - add comment

### FR-8 Action Reliability
- Action execution should retry on transient failures
- If retries are exhausted, record failure details for audit/troubleshooting
- Action failure must not block core webhook acceptance after event/alert persistence
- Provide protected failure APIs:
  - `GET /action-failures`
  - `POST /action-failures/:id/retry`

### FR-9 Observability APIs
- Provide protected observability APIs:
  - `GET /metrics/overview`
  - `GET /metrics/timeseries`
  - `GET /audit-logs`

### FR-10 GitHub Source Event Pull (Protected)
- Extend existing events API with GitHub source mode (no new endpoint path):
  - `GET /events?source=github`
    - fetch recent GitHub user events and return unique `event_types`
  - `GET /events?source=github&sync=true`
    - fetch recent GitHub user events and persist into `webhook_events`
    - return sync summary (`saved`, `total`)
- Must use configured `GITHUB_TOKEN` and reject when token/provider is unavailable.
- Delivery id for synced GitHub events should be normalized as `gh-<github_event_id>` for idempotent upsert.

## 7. Non-Functional Requirements

- **Security**: Reject unsigned/invalid webhooks; protected APIs require bearer JWT.
- **Reliability**: Service should keep accepting events under normal failures with retry-ready design.
- **Performance**: P95 webhook processing < 500ms (excluding DB outage).
- **Observability**: Structured logs for webhook/auth/action/github-sync paths and metrics/audit querying APIs.
- **Maintainability**: Clear package boundaries (`config`, `handlers`, `store`, `service`).

## 8. Acceptance Criteria (Definition of Done)

For current main-flow completion, all are required:

1. Webhook endpoint works with signature validation.
2. Valid events are persisted in PostgreSQL.
3. Invalid signatures return `401` and are not persisted.
4. `POST /auth/login` returns JWT on valid credentials.
5. Protected APIs (`/events`, `/alerts`, `/rules`, `/metrics/*`, `/audit-logs`, `/action-failures*`, `/config-*`) reject invalid/missing bearer token.
6. Event list endpoint returns latest records with pagination/filter/total.
7. Alerts list endpoint returns latest records with pagination/filter/total.
8. Rules API supports list/create and active status update (`PATCH /rules/:id/active`).
9. Rule engine returns suggested actions for matched keywords.
10. When `GITHUB_TOKEN` is set, suggested label/comment execution path is available.
11. Action execution failures are retried and persisted when exhausted; retry API available.
12. Action failure does not prevent webhook success after core persistence.
13. `GET /events?source=github` returns recent GitHub `event_types`.
14. `GET /events?source=github&sync=true` persists pulled events into `webhook_events` idempotently.
15. Observability endpoints provide overview/timeseries/audit data.
16. Runtime config endpoints provide view/update/status capabilities.
17. `go test ./...` and `go build ./...` pass.
18. `npm run build` passes.
19. README/docs include setup and run instructions.

## 9. Milestones

- **M1**: Go + React skeleton, health endpoint, GitHub repo setup (done)
- **M2**: Webhook endpoint + signature validation (done)
- **M3**: PostgreSQL persistence for webhook events (done)
- **M4**: Event list page + basic filtering (done)
- **M5**: Rule engine v1 suggestions (done)
- **M6**: Configurable rules API + DB-backed matching (done)
- **M7**: Alerts persistence + alerts API/UI (done)
- **M8**: Optional GitHub action execution (label/comment) (done)
- **M9**: JWT login + protected API/UI routes (done)
- **M10**: action retry + failure recording without blocking webhook core path (done)
- **M11**: `/events` GitHub source mode + on-demand sync-to-DB (`source=github`, `sync=true`) (in progress)

## 10. Risks and Mitigations

- **Risk**: Invalid webhook signature handling bugs
  - **Mitigation**: Deterministic signature verification tests.
- **Risk**: Auth misconfiguration (JWT secret / admin creds)
  - **Mitigation**: Startup/env checklist + auth handler tests.
- **Risk**: DB connection instability
  - **Mitigation**: Connection pooling + clear error paths.
- **Risk**: GitHub API failures during auto actions
  - **Mitigation**: Keep webhook core persistence path independent from optional action automation.
- **Risk**: Scope creep
  - **Mitigation**: Keep “main-flow first” and stage secondary features separately.
