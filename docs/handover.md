# Handover - maintainer-firewall

## Repository

- GitHub: https://github.com/hj01857655/maintainer-firewall
- Branch: `main`
- Local path: `<repo-root>/maintainer-firewall`

## Current Tech Stack (V1)

- Backend: Go + Gin + PostgreSQL (pgx)
- Frontend: React + TypeScript + Vite + React Router
- Auth: JWT bearer auth + RBAC (`read/write/admin`) + danger confirmation gate
- Package manager: npm
- Runtime targets: local dev first, Docker later

## What is already done (current main)

1. Monorepo skeleton created
   - `apps/api-go`
   - `apps/web-react`
   - `infra/docker`
   - `docs`
2. Core backend APIs
    - `GET /health`
    - `POST /webhook/github` (HMAC signature verification)
    - `POST /auth/login` (JWT issue; supports `tenant_id`)
    - protected APIs moved under `/api/*`:
      - read: events/alerts/rules/users/tenants/metrics/audit/config/action-failures queries
      - write: rules create/update/publish, user mutate, failure retry
      - admin: tenant create
      - admin+danger confirm (`X-MF-Confirm: confirm`): user delete, tenant active toggle, config update, rule rollback
    - rule version lifecycle:
      - `POST /api/rules/publish`
      - `GET /api/rules/versions`
      - `POST /api/rules/rollback`
      - `POST /api/rules/replay`
3. Data persistence
    - `webhook_events`
    - `webhook_alerts`
    - `webhook_rules`
    - `webhook_rule_versions`
    - `admin_users` (role/permissions + tenant scope)
    - `tenants`
4. Rule engine + automation
    - DB-backed rule matching
    - Suggested actions (`label` / `comment`)
    - optional GitHub auto action execution via `GITHUB_TOKEN`
5. Tenant + permission governance
   - request context carries `tenant_id`
   - JWT claims include `tenant_id` / `role` / `permissions`
   - storage read/write paths enforce tenant isolation by `tenant_id`
6. Frontend pages
   - login/dashboard/events/rules/alerts/failures/audit/system-config
   - Events/Alerts/Rules dropdown filters use full-dataset filter-options APIs
   - protected route guard using bearer token
7. Reliability hardening
   - action execution retry
   - failure persistence (`webhook_action_failures`)
   - webhook core persistence path remains non-blocking for action failures
8. CI and docs
   - GitHub Actions for Go/Web build
   - README + requirements + design aligned to current flow

## Project Structure (key files)

```text
maintainer-firewall/
├─ apps/
│  ├─ api-go/
│  │  ├─ cmd/server/main.go
│  │  ├─ internal/config/config.go
│  │  ├─ internal/http/handlers/
│  │  │  ├─ health.go
│  │  │  ├─ auth.go
│  │  │  ├─ webhook.go
│  │  │  ├─ events.go
│  │  │  ├─ alerts.go
│  │  │  └─ rules.go
│  │  ├─ internal/store/webhook_store.go
│  │  ├─ internal/service/
│  │  │  ├─ rule_engine.go
│  │  │  └─ github_executor.go
│  │  ├─ go.mod
│  │  └─ go.sum
│  └─ web-react/
│     ├─ src/main.tsx
│     ├─ src/AppRouter.tsx
│     ├─ src/auth.ts
│     ├─ src/layout/AppLayout.tsx
│     ├─ src/pages/
│     │  ├─ LoginPage.tsx
│     │  ├─ DashboardPage.tsx
│     │  ├─ EventsPage.tsx
│     │  ├─ RulesPage.tsx
│     │  ├─ AlertsPage.tsx
│     │  ├─ FailuresPage.tsx
│     │  ├─ AuditLogsPage.tsx
│     │  └─ SystemConfigPage.tsx
│     ├─ package.json
│     └─ vite.config.ts
├─ docs/
│  ├─ requirements.md
│  ├─ design.md
│  └─ handover.md
├─ scripts/demo.ps1
└─ README.md
```

## Run locally

### 1) API

```powershell
# <repo-root>/apps/api-go
$env:GITHUB_WEBHOOK_SECRET="replace_with_webhook_secret"
$env:GITHUB_TOKEN="optional_github_pat_for_auto_actions"
$env:ADMIN_USERNAME="admin"
$env:ADMIN_PASSWORD="CHANGE_ME_ADMIN_PASSWORD"
$env:JWT_SECRET="CHANGE_ME_JWT_SECRET"
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/maintainer_firewall?sslmode=disable"
go run .\cmd\server\main.go
```

### 2) Web

```powershell
# <repo-root>/apps/web-react
npm install
npm run dev
```

Web app:

- `http://localhost:5173`
- login first, then access dashboard/events/rules/alerts

## Next Milestones (remaining)

### R1: Dashboard value upgrades

- Alert summary metrics
- Rule hit trend snapshots
- Better empty/error/loading states

### R2: E2E automation

- Alert summary metrics
- Rule hit trend snapshots
- Better empty/error/loading states

- End-to-end test: webhook -> events/rules/alerts visible in UI
- Filter-options smoke: events/alerts/rules dropdown options come from dedicated filter-options APIs (not current page rows)

## Reopen IDE Quick Resume Checklist

1. Open folder: `<repo-root>/maintainer-firewall`
2. Run `go test ./...` in `apps/api-go`
3. Run `npm run build` in `apps/web-react`
4. Run one-command demo from repo root: `./scripts/demo.ps1`
5. Verify login + protected `/api/events` / `/api/events/filter-options` / `/api/rules` / `/api/rules/filter-options` / `/api/alerts` / `/api/alerts/filter-options`
6. Verify rule version flow: publish -> versions list -> rollback (with `X-MF-Confirm: confirm`) -> replay
7. Verify retry/failure record behavior for action execution path

## Notes

- If project path needs to move, use `git`-safe move and keep remote `origin` unchanged.
- Current repo remote is already configured and pushed.
