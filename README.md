# maintainer-firewall

Go + React open-source project skeleton for maintainer workflow automation.

Current status: webhook signature verification + PostgreSQL event persistence + rule suggestion + alerts persistence + configurable rules + auto action execution + JWT-protected API/UI are implemented.

## Structure

- `apps/api-go`: Go API service (Gin + PostgreSQL event/alert store)
- `apps/web-react`: React console (Vite + TS + React Router, dashboard + events + alerts pages)
- `docs`: architecture/docs (requirements/design/handover)

## Run API

Before starting API, set required environment variables:

```powershell
# e:\VSCodeSpace\reverse\maintainer-firewall\apps\api-go
$env:GITHUB_WEBHOOK_SECRET="replace_with_webhook_secret"
$env:GITHUB_TOKEN="optional_github_pat_for_auto_actions"
$env:ADMIN_USERNAME="admin"
$env:ADMIN_PASSWORD="admin123"
$env:JWT_SECRET="mf-demo-jwt-secret"
# backward-compat fallback if JWT_SECRET is empty:
# $env:ACCESS_TOKEN="legacy-shared-secret"
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/maintainer_firewall?sslmode=disable"
go mod tidy
go run .\cmd\server\main.go
```

API endpoints:

- `GET http://localhost:8080/health`
- `POST http://localhost:8080/auth/login`
- `GET http://localhost:8080/events?limit=20&offset=0&event_type=issues&action=opened` (auth required)
  - response includes `total` for pagination
- `GET http://localhost:8080/alerts?limit=20&offset=0&event_type=issues&action=opened&suggestion_type=label` (auth required)
  - response includes `total` for pagination
- `GET/POST http://localhost:8080/rules` (auth required)
- `POST http://localhost:8080/webhook/github`

## Run Web

```powershell
# e:\VSCodeSpace\reverse\maintainer-firewall\apps\web-react
npm install
npm run dev
```

Web app:

- `http://localhost:5173`
- automatically proxies `/health` / `/auth` / `/events` / `/alerts` / `/rules` to `http://localhost:8080`

## Docs

- Requirements: `docs/requirements.md`
- Design: `docs/design.md`
- Handover: `docs/handover.md`

## 3-minute demo (recommended)

```powershell
# e:\VSCodeSpace\reverse\maintainer-firewall
.\scripts\demo.ps1
```

Script does:

- set env vars (including auth secret)
- start API in background
- login and get JWT token
- create demo rule
- send signed webhook
- query events/alerts and print summary

## Quick API check (manual)

```powershell
# login (returns JWT)
$login = Invoke-RestMethod -Method Post -Uri http://localhost:8080/auth/login -ContentType "application/json" -Body '{"username":"admin","password":"admin123"}'
$headers = @{ Authorization = "Bearer $($login.token)" }

# list events (auth required)
Invoke-RestMethod "http://localhost:8080/events?limit=20&offset=0&event_type=issues&action=opened" -Headers $headers

# list alerts (auth required)
Invoke-RestMethod "http://localhost:8080/alerts?limit=20&offset=0&event_type=issues&action=opened&suggestion_type=label" -Headers $headers
```

## CI

- GitHub Actions workflow: `.github/workflows/ci.yml`
- Runs Go test/build + Web build on push/pull_request for changed app paths

## Main-flow checklist (done)

- Webhook signature validation
- Persist webhook events
- Rule suggestion generation (`label` / `comment`)
- Persist rule-hit alerts
- Configurable rules API (`GET/POST /rules`)
- Auto execute GitHub actions (label/comment)
- Login + protected API/UI routes (JWT)
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
