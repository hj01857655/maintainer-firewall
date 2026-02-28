# maintainer-firewall

Go + React open-source project skeleton for maintainer workflow automation.

Current status: webhook signature verification + PostgreSQL event persistence + rule suggestion + alerts persistence + events/alerts list API/UI (filter + total pagination) are implemented.

## Structure

- `apps/api-go`: Go API service (Gin + PostgreSQL event/alert store)
- `apps/web-react`: React console (Vite + TS + React Router, dashboard + events + alerts pages)
- `docs`: architecture/docs (requirements/design/handover)

## Run API

Before starting API, set required environment variables:

```powershell
# e:\VSCodeSpace\reverse\maintainer-firewall\apps\api-go
$env:GITHUB_WEBHOOK_SECRET="replace_with_webhook_secret"
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/maintainer_firewall?sslmode=disable"
go mod tidy
go run .\cmd\server\main.go
```

API endpoints:

- `GET http://localhost:8080/health`
- `GET http://localhost:8080/events?limit=20&offset=0&event_type=issues&action=opened`
  - response includes `total` for pagination
- `GET http://localhost:8080/alerts?limit=20&offset=0&event_type=issues&action=opened&suggestion_type=label`
  - response includes `total` for pagination
- `POST http://localhost:8080/webhook/github`

## Run Web

```powershell
# e:\VSCodeSpace\reverse\maintainer-firewall\apps\web-react
npm install
npm run dev
```

Web app:

- `http://localhost:5173`
- automatically proxies `/health` / `/events` / `/alerts` to `http://localhost:8080`

## Docs

- Requirements: `docs/requirements.md`
- Design: `docs/design.md`
- Handover: `docs/handover.md`

## Quick API check (main flow)

```powershell
# health
Invoke-RestMethod http://localhost:8080/health

# 1) send webhook (replace secret/signature/payload as needed)
$secret = "replace_with_webhook_secret"
$body = '{"action":"opened","repository":{"full_name":"owner/repo"},"sender":{"login":"alice"},"issue":{"title":"urgent duplicate bug"}}'
$hmac = New-Object System.Security.Cryptography.HMACSHA256
$hmac.Key = [Text.Encoding]::UTF8.GetBytes($secret)
$hashBytes = $hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($body))
$signature = "sha256=" + ([BitConverter]::ToString($hashBytes).Replace("-","").ToLower())

Invoke-RestMethod -Method Post `
  -Uri http://localhost:8080/webhook/github `
  -Headers @{ "X-Hub-Signature-256"=$signature; "X-GitHub-Event"="issues"; "X-GitHub-Delivery"="demo-delivery-1" } `
  -Body $body `
  -ContentType "application/json"

# 2) list events
Invoke-RestMethod "http://localhost:8080/events?limit=20&offset=0&event_type=issues&action=opened"

# 3) list alerts
Invoke-RestMethod "http://localhost:8080/alerts?limit=20&offset=0&event_type=issues&action=opened&suggestion_type=label"
```

## CI

- GitHub Actions workflow: `.github/workflows/ci.yml`
- Runs Go test/build + Web build on push/pull_request for changed app paths

## Main-flow checklist (done)

- Webhook signature validation
- Persist webhook events
- Rule suggestion generation (`label` / `comment`)
- Persist rule-hit alerts
- Query events with pagination/filter + `total`
- Query alerts with pagination/filter + `total`
- Web pages for events/alerts
- CI checks for API/Web build

## Secondary (next)

- Dashboard alert summary widgets
- E2E test for webhook -> events/alerts visibility
- Rich filters (repository/sender/date range)
- Export & reporting

## License

MIT
