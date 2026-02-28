# maintainer-firewall

Go + React open-source project skeleton for maintainer workflow automation.

Current status: webhook signature verification + DB event persistence (PostgreSQL/MySQL) + rule suggestion + alerts persistence + configurable rules + auto action execution + JWT-protected API/UI + action retry/failure recording + E2E acceptance script are implemented.

## Structure

- `apps/api-go`: Go API service (Gin + PostgreSQL/MySQL event/alert store, supports auto-loading env from `.env` and auto-creating `.env` from `.env.example`)
- `apps/web-react`: React console (Vite + TS + React Router, dashboard + events + alerts pages)
- `docs`: architecture/docs (requirements/design/handover)

## Run API

Before starting API, you can create a `.env` once (or export env vars manually):

```powershell
# <repo-root>
Copy-Item .env.example .env
# edit .env and set DATABASE_URL (others can keep defaults)
```

If `.env` is missing at startup, API will auto-create it from `.env.example`.

Then run API:

```powershell
# <repo-root>/apps/api-go
go mod tidy
go run .\cmd\server\main.go
```

Local defaults when omitted (development only):

- PORT=8080
- ADMIN_USERNAME=admin
- ADMIN_PASSWORD=admin123
- JWT_SECRET=dev-jwt-secret (or ACCESS_TOKEN fallback)
- GITHUB_WEBHOOK_SECRET=dev-webhook-secret
- GITHUB_TOKEN is optional (empty by default)
- DATABASE_URL is still required (set in `.env`/environment; if omitted, API starts but store initialization will fail)


API endpoints:

- `GET http://localhost:8080/health`
- `POST http://localhost:8080/auth/login` (default local account in development: `admin` / `admin123`)
- `GET http://localhost:8080/events?limit=20&offset=0&event_type=issues&action=opened` (auth required)
  - response includes `total` for pagination
- `GET http://localhost:8080/alerts?limit=20&offset=0&event_type=issues&action=opened&suggestion_type=label` (auth required)
  - response includes `total` for pagination
- `GET/POST http://localhost:8080/rules` (auth required)
- `POST http://localhost:8080/webhook/github`

## Run Web

```powershell
# <repo-root>/apps/web-react
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
# <repo-root>
.\scripts\demo.ps1
```

Script does:

- set env vars (including auth secret)

- login and get JWT token
- create demo rule
- send signed webhook
- query events/alerts and print summary

## E2E acceptance (automated)

```powershell
# <repo-root> (PostgreSQL example)
.\scripts\e2e.ps1 `
  -AdminUsername "admin" `
  -AdminPassword "<YOUR_ADMIN_PASSWORD>" `
  -JWTSecret "<YOUR_JWT_SECRET>" `
  -GitHubWebhookSecret "<YOUR_WEBHOOK_SECRET>" `
  -DatabaseURL "postgres://postgres:postgres@localhost:5432/maintainer_firewall?sslmode=disable"

# <repo-root> (MySQL example)
# .\scripts\e2e.ps1 `
#   -AdminUsername "admin" `
#   -AdminPassword "<YOUR_ADMIN_PASSWORD>" `
#   -JWTSecret "<YOUR_JWT_SECRET>" `
#   -GitHubWebhookSecret "<YOUR_WEBHOOK_SECRET>" `
#   -DatabaseURL "mysql://<MYSQL_USER>:<MYSQL_PASSWORD>@127.0.0.1:3306/maintainer_firewall"
```

What it verifies automatically:

- health endpoint is up
- login returns JWT
- signed webhook accepted
- events/alerts contain the new delivery_id
- alerts include expected suggestion value
- works with either PostgreSQL or MySQL via `-DatabaseURL`

## Quick API check (manual)

```powershell
# login (returns JWT)
$loginBody = @{ username = "admin"; password = "<YOUR_ADMIN_PASSWORD>" } | ConvertTo-Json
$login = Invoke-RestMethod -Method Post -Uri http://localhost:8080/auth/login -ContentType "application/json" -Body $loginBody
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
- Action retry + failure recording (`webhook_action_failures`)
- Login + protected API/UI routes (JWT)

- Query alerts with pagination/filter + `total`
- Web pages for events/alerts
- CI checks for API/Web build

## Secondary (next)

- Dashboard alert summary widgets
- Rich filters (repository/sender/date range)
- Export & reporting

## License

MIT
