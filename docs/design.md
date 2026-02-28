# Technical Design - maintainer-firewall (MVP)

## 1. Scope

This design covers M3, M4, and M5-v1:

- M3: persist verified GitHub webhook events into PostgreSQL
- M4 base: query and display latest events via API + React console
- M5-v1: rule engine returns suggested label/comment actions

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

1. GitHub sends `POST /webhook/github`
2. Handler validates `X-Hub-Signature-256` with `GITHUB_WEBHOOK_SECRET`
3. Handler extracts metadata from headers and JSON payload
4. Handler writes event into PostgreSQL table `webhook_events`
5. Handler returns `200` when accepted and persisted
6. Rule engine evaluates payload text and produces suggested actions
7. React console calls `GET /events` to render latest records

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

## 5. Error Handling

- Invalid/missing signature -> `401`
- Bad body read / malformed JSON -> `400`
- DB unavailable / insert failure -> `500`
- Duplicate delivery id -> treated as success (`200`) for idempotency

## 6. Config

Required for M3:

- `PORT` (default `8080`)
- `GITHUB_WEBHOOK_SECRET` (required for webhook)
- `DATABASE_URL` (required for persistence)

## 7. Verification

- `go build ./...`
- `npm run build`
- Manual webhook smoke:
  - valid signature -> `200` and row inserted
  - invalid signature -> `401`, no row inserted
- Events listing:
- `GET /events?limit=20&offset=0&event_type=issues&action=opened` returns ordered filtered records

## 8. M4 Progress

Implemented:

- API `GET /events` with pagination params `limit` and `offset`
- API filtering by `event_type` and `action`
- React event list page with filter inputs and prev/next pagination controls

Next:

- Add total count and pagination controls in UI (offset controls done, total count pending)
- Add endpoint tests for list/query validation

## 9. M5 Rule Engine v1

Implemented:

- Added keyword-based rule engine for `issues` and `pull_request` events
- Returns suggested actions in webhook response:
  - `label` suggestion
  - `comment` suggestion

Current built-in keyword rules:

- `duplicate` -> `needs-triage`
- `help wanted` -> `help-wanted`
- `urgent` -> `priority-high`

Response contract update:

- `POST /webhook/github` now may include `suggested_actions` array
