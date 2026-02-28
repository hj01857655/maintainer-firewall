# Technical Design - maintainer-firewall (MVP)

## 1. Scope

This design covers M3 in requirements: persist verified GitHub webhook events into PostgreSQL while keeping existing signature verification behavior.

## 2. Runtime Components

- `cmd/server/main.go`
  - bootstraps config, DB store, and HTTP handlers
- `internal/config`
  - loads env-based runtime config
- `internal/store`
  - PostgreSQL connection and `webhook_events` persistence
- `internal/http/handlers`
  - request parsing, signature verification, event extraction, store call

## 3. Request Flow

1. GitHub sends `POST /webhook/github`
2. Handler validates `X-Hub-Signature-256` with `GITHUB_WEBHOOK_SECRET`
3. Handler extracts metadata from headers and JSON payload
4. Handler writes event into PostgreSQL table `webhook_events`
5. Handler returns `200` when accepted and persisted

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
- Manual webhook smoke:
  - valid signature -> `200` and row inserted
  - invalid signature -> `401`, no row inserted

## 8. Next Design Step (M4)

- Add API `GET /events` with pagination and filter by event type/action
- Build web page for event list and status summary
