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

## 6. Functional Requirements (MVP)

### FR-1 Webhook Ingestion
- `POST /webhook/github` endpoint
- Validate `X-Hub-Signature-256` using HMAC-SHA256
- Return clear status for invalid signature / malformed payload

### FR-2 Event Persistence
- Save each accepted event to PostgreSQL
- Minimum fields:
  - `delivery_id`
  - `event_type`
  - `action`
  - `repository_full_name`
  - `sender_login`
  - `received_at`
  - `payload_json`

### FR-3 Health and Runtime Config
- `GET /health`
- Environment-driven config:
  - `PORT`
  - `GITHUB_WEBHOOK_SECRET`
  - `DATABASE_URL`

### FR-4 Basic Console
- React page to show API health status
- (Next step) event list page with latest records

## 7. Non-Functional Requirements

- **Security**: Reject unsigned/invalid webhooks.
- **Reliability**: Service should keep accepting events under normal failures with retry-ready design.
- **Performance**: P95 webhook processing < 500ms (excluding DB outage).
- **Observability**: Structured logs for every webhook request.
- **Maintainability**: Clear package boundaries (`config`, `handlers`, `store`).

## 8. Acceptance Criteria (Definition of Done)

For MVP completion, all are required:

1. Webhook endpoint works with signature validation.
2. Valid events are persisted in PostgreSQL.
3. Invalid signatures return `401` and are not persisted.
4. `go build ./...` passes.
5. `npm run build` passes.
6. README/docs include setup and run instructions.

## 9. Milestones

- **M1**: Go + React skeleton, health endpoint, GitHub repo setup (done)
- **M2**: Webhook endpoint + signature validation (done)
- **M3**: PostgreSQL persistence for webhook events (next)
- **M4**: Event list page + basic filtering
- **M5**: Rule engine v1 (label/reply suggestion)

## 10. Risks and Mitigations

- **Risk**: Invalid webhook signature handling bugs
  - **Mitigation**: Add deterministic signature verification tests.
- **Risk**: DB connection instability
  - **Mitigation**: Connection pooling + retry strategy + clear error paths.
- **Risk**: Scope creep
  - **Mitigation**: Freeze MVP scope to FR-1..FR-4.
