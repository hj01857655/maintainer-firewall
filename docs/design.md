# Technical Design - maintainer-firewall (MVP)

## 1. Scope

This design covers M3, M4, M5-v1 and post-MVP hardening done in this branch:

- M3: persist verified GitHub webhook events into PostgreSQL
- M4 base: query and display latest events via API + React console
- M5-v1: rule engine returns suggested label/comment actions
- M6: configurable rules API (`GET/POST /rules`) and DB-backed rule matching
- M7: optional GitHub auto actions (label/comment) execution
- M8: JWT login and protected API routes
- M9: action retry + failure recording without blocking webhook acceptance
- M11: extend existing `/events` with GitHub source pull and on-demand/periodic sync-to-DB (`source=github`, `mode=types|items`, `sync=true`, `/events/sync-status`, scheduler)
- M13: multi-tenant boundary + RBAC permission governance
- M14: rule version lifecycle (`publish`/`versions`/`rollback`/`replay`)

## 2. Runtime Components

- `cmd/server/main.go`
  - bootstraps config, DB store, and HTTP handlers
- `internal/config`
  - loads env-based runtime config
- `internal/store`
  - PostgreSQL/MySQL persistence for `webhook_events` and related tables
- `internal/service/github_executor.go`
  - GitHub API client for action execution and user-event pull
- `internal/service/github_sync_worker.go`
  - periodic scheduler for GitHub event sync
- `internal/http/handlers`
  - request parsing, signature verification, event extraction, store call
  - `/api/events` GitHub source mode (`mode=types|items`) + optional sync and `/api/events/sync-status`
  - `/api/events/filter-options` `/api/alerts/filter-options` `/api/rules/filter-options` full-dataset options APIs
  - RBAC middlewares: `RequirePermission`, `RequireDangerConfirm`
- `internal/tenantctx`
  - normalize and carry `tenant_id` through request context/store calls

## 3. Request/Data Flow

1. User logs in via `POST /auth/login` (supports `tenant_id`) and gets JWT (`tenant_id`, `role`, `permissions` claims)
2. Protected APIs are served under `/api/*` and require `Authorization: Bearer <jwt>`
3. GitHub sends `POST /webhook/github`
4. Handler validates `X-Hub-Signature-256` with `GITHUB_WEBHOOK_SECRET`
5. Handler extracts metadata from headers and JSON payload
6. Handler writes event into table `webhook_events`
7. Handler loads active rules from `webhook_rules` and evaluates suggestions
8. Handler writes matched suggestions into `webhook_alerts`
9. If configured, handler executes GitHub actions (`label`/`comment`) via GitHub API
10. Action execution uses retry policy and records failures when retries are exhausted
11. Webhook still returns success after core persistence path completes
12. `GET /api/events?source=github` defaults to `mode=types` and returns unique `event_types`
13. `GET /api/events?source=github&mode=items&limit=<n>&offset=<n>` returns paginated GitHub event items
14. `GET /api/events?source=github&sync=true` pulls recent user events and persists them into `webhook_events`
15. `GET /api/events/sync-status` exposes in-memory sync runtime status and last result counters
16. If `GITHUB_EVENTS_SYNC_INTERVAL_MINUTES > 0`, background worker periodically invokes the same sync path
17. Middleware injects `tenant_id` into request context and enforces permission layers: `read` / `write` / `admin`
18. Dangerous admin operations (`/api/users/:id` delete, `/api/config-update`, `/api/rules/rollback`, `/api/tenants/:id/active`) require `X-MF-Confirm: confirm`
19. React console calls `GET /api/events` / `GET /api/rules` / `GET /api/alerts`
20. React console loads full-dataset dropdown options from `/api/events/filter-options` `/api/alerts/filter-options` `/api/rules/filter-options`

## 4. Data Model

Table: `webhook_events`

- `id BIGSERIAL PRIMARY KEY`
- `delivery_id TEXT NOT NULL UNIQUE` (webhook delivery id or normalized GitHub event id like `gh-<id>`)
- `event_type TEXT NOT NULL`
- `action TEXT NOT NULL`
- `repository_full_name TEXT NOT NULL`
- `sender_login TEXT NOT NULL`
- `payload_json JSONB NOT NULL`
- `received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`

Indexes:

- `idx_webhook_events_received_at (received_at DESC)`
- `idx_webhook_events_event_type (event_type)`
- `idx_webhook_events_action (action)`
- `idx_webhook_events_event_action (event_type, action)`

## 5. Error Handling

- Invalid/missing signature -> `401`
- Bad body read / malformed JSON -> `400`
- DB unavailable / insert failure -> `500`
- Duplicate delivery id -> treated as success (`200`) for idempotency
- `GET /api/events?source=github` (`mode=types|items`) provider/token issues -> `500/502` with clear message
- Missing permission -> `403`
- Missing danger confirmation header -> `400`

## 6. Config

Core runtime config:

- `PORT` (default `8080`)
- `GITHUB_WEBHOOK_SECRET` (required for webhook signature verify)
- `DATABASE_URL` (required for persistence)

Auth config:

- `ADMIN_USERNAME` (required for login)
- `ADMIN_PASSWORD` (required for login)
- `JWT_SECRET` (preferred; used to sign/verify JWT)
- `ACCESS_TOKEN` (legacy fallback secret when `JWT_SECRET` is empty)

Automation config:

- `GITHUB_TOKEN` (optional; required only when enabling GitHub auto action execution)
- `GITHUB_EVENTS_SYNC_INTERVAL_MINUTES` (`0` disabled, `>0` periodic sync interval in minutes)

## 7. Verification

- `go test ./...`
- `go build ./...`
- `npm run build`
- periodic sync smoke:
  - set `GITHUB_EVENTS_SYNC_INTERVAL_MINUTES=5`
  - observe server log `github events sync done: saved=... total=...`
  - verify `webhook_events` grows with `delivery_id` prefix `gh-` (idempotent on repeats)
- GitHub source mode smoke:
  - `GET /api/events?source=github` returns `mode=types` and non-empty `event_types`
  - `GET /api/events?source=github&mode=items&limit=20&offset=0` returns `mode=items` and `items/total`
- Login:
  - `POST /auth/login` returns JWT on valid credentials (with `tenant_id`, `role`, `permissions`)
  - protected APIs reject missing/invalid bearer token
- Manual webhook smoke:
  - valid signature -> `200` and row inserted
  - invalid signature -> `401`, no row inserted
- Events/Rules/Alerts listing:
  - `GET /api/events?limit=20&offset=0&event_type=issues&action=opened` returns ordered filtered records
  - `GET /api/rules?...` returns configurable rules list
  - `GET /api/alerts?...` returns matched suggestions with pagination total
- Filter-options APIs:
  - `GET /api/events/filter-options` returns distinct `event_types/actions/repositories/senders`
  - `GET /api/alerts/filter-options` returns distinct `event_types/actions/suggestion_types/repositories/senders`
  - `GET /api/rules/filter-options` returns distinct `event_types/suggestion_types/active_states`

## 8. M4 Progress

Implemented:

- API `GET /events` with pagination params `limit` and `offset`
- API filtering by `event_type` and `action`
- React event list page with dynamic dropdown filters and page/total page display
- React alerts/rules pages use backend full-dataset filter-options to keep dropdowns stable across pagination

Next:

- Add endpoint tests for list/query validation
- Add server-side sorting and richer filters (repository/sender/date range)

## 9. Rules + Automation + Auth + Governance (current)

Implemented:

- Rule engine supports DB-configurable rules via `webhook_rules`
- Rule lifecycle APIs:
  - `POST /api/rules/publish`
  - `GET /api/rules/versions`
  - `POST /api/rules/rollback`
  - `POST /api/rules/replay`
- Webhook response includes `suggested_actions`
- Matched suggestions persisted in `webhook_alerts`
- Optional GitHub auto-action execution:
  - `label` action
  - `comment` action
- Action reliability hardening:
  - retry attempts for action execution
  - failed executions persisted to `webhook_action_failures`
  - webhook accept path remains non-blocking for action failures
- JWT auth for console APIs:
  - middleware-protected `/api/events`, `/api/alerts`, `/api/rules` and other protected routes
  - JWT claims include `tenant_id`, `role`, `permissions`
- RBAC:
  - read/write/admin permission layers at route groups
  - destructive admin APIs require `X-MF-Confirm: confirm`
- Multi-tenant:
  - `tenant_id` from login/JWT propagates into request context and store queries
