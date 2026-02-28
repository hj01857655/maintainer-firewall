# maintainer-firewall

Go + React open-source project skeleton for maintainer workflow automation.

Current status: webhook signature verification + PostgreSQL event persistence + multi-page UI + event listing API/UI (filter + total pagination) are implemented.

## Structure

- `apps/api-go`: Go API service (Gin + PostgreSQL event store)
- `apps/web-react`: React console (Vite + TS + React Router, dashboard + events pages)

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
- `POST http://localhost:8080/webhook/github`


## Run Web

```powershell
# e:\VSCodeSpace\reverse\maintainer-firewall\apps\web-react
npm install
npm run dev
```

Web app:

- `http://localhost:5173`
- automatically proxies `/health` and `/events` to `http://localhost:8080`

## Docs

- Requirements: `docs/requirements.md`
- Design: `docs/design.md`
- Handover: `docs/handover.md`

## Quick API check

```powershell
# health
Invoke-RestMethod http://localhost:8080/health

# list events (with optional filters + total)
Invoke-RestMethod "http://localhost:8080/events?limit=20&offset=0&event_type=issues&action=opened"

