# Technical Design - maintainer-firewall (MVP)

## 1. Scope

This design covers M3, M4, M5-v1 and post-MVP hardening done in this branch:

- M3: persist verified GitHub webhook events into PostgreSQL
- M4 base: query and display latest events via API + React console
- M5-v1: rule engine returns suggested label/comment actions
- M6: configurable rules API (`GET/POST /rules`) and DB-backed rule matching
- M7: optional GitHub auto actions (label/comment) execution
- M8: JWT login and protected API routes

## 2. Runtime Components

- `cmd/server/main.go`
  - bootstraps config, DB store, and HTTP handlers
- `internal/config`
  - loads env-based runtime config
- `internal/store`
  - PostgreSQL connection and `webhook_events` persistence
- `internal/http/handlers`
  - request parsing, signature verification, event extraction, store call

## 3. Request/Data Flow

1. User logs in via `POST /auth/login` and gets JWT
2. Protected APIs (`/events`, `/alerts`, `/rules`) require `Authorization: Bearer <jwt>`
3. GitHub sends `POST /webhook/github`
4. Handler validates `X-Hub-Signature-256` with `GITHUB_WEBHOOK_SECRET`
5. Handler extracts metadata from headers and JSON payload
6. Handler writes event into PostgreSQL table `webhook_events`
7. Handler loads active rules from `webhook_rules` and evaluates suggestions
8. Handler writes matched suggestions into `webhook_alerts`
9. If configured, handler executes GitHub actions (`label`/`comment`) via GitHub API
10. React console calls `GET /events` / `GET /alerts` / `GET /rules`

## 4. Data Model

Table: `webhook_events`

- `id BIGSERIAL PRIMARY KEY`
- `delivery_id TEXT NOT NULL UNIQUE`
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

## 7. Verification

- `go test ./...`
- `go build ./...`
- `npm run build`
- Login:
  - `POST /auth/login` returns JWT on valid credentials
  - protected APIs reject missing/invalid bearer token
- Manual webhook smoke:
  - valid signature -> `200` and row inserted
  - invalid signature -> `401`, no row inserted
- Events/Alerts/Rules listing:
  - `GET /events?limit=20&offset=0&event_type=issues&action=opened` returns ordered filtered records
  - `GET /alerts?...` returns matched suggestions with pagination total
  - `GET /rules?...` returns configurable rules list

## 8. M4 Progress

Implemented:

- API `GET /events` with pagination params `limit` and `offset`
- API filtering by `event_type` and `action`
- React event list page with filter inputs and page/total page display

Next:

- Add endpoint tests for list/query validation
- Add server-side sorting and richer filters (repository/sender/date range)

## 9. Rules + Automation + Auth (current)

Implemented:

- Rule engine supports DB-configurable rules via `webhook_rules`
- `GET /rules` and `POST /rules` for rule management (protected)
- Webhook response includes `suggested_actions`
- Matched suggestions persisted in `webhook_alerts`
- Optional GitHub auto-action execution:
  - `label` action
  - `comment` action
- JWT auth for console APIs:
  - `POST /auth/login`
  - middleware-protected `/events`, `/alerts`, `/rules`
