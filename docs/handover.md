# Handover - maintainer-firewall

## Repository

- GitHub: https://github.com/hj01857655/maintainer-firewall
- Branch: `main`
- Local path: `e:\VSCodeSpace\reverse\maintainer-firewall`

## Current Tech Stack (V1)

- Backend: Go 1.23 + Gin
- Frontend: React + TypeScript + Vite
- Package manager: npm
- Runtime targets: local dev first, Docker later

## What is already done

1. Monorepo skeleton created
   - `apps/api-go`
   - `apps/web-react`
   - `infra/docker`
   - `docs`
2. Go API bootstrap
   - `GET /health`
   - env config loader (`PORT`, default 8080)
3. React bootstrap
   - app renders and fetches `/health`
   - Vite proxy routes `/health` to `http://localhost:8080`
4. Git initialized, first commit done, pushed to GitHub
5. `.gitignore` added for Go/Node/IDE/build artifacts

## Project Structure

```text
maintainer-firewall/
├─ apps/
│  ├─ api-go/
│  │  ├─ cmd/server/main.go
│  │  ├─ internal/config/config.go
│  │  ├─ internal/http/handlers/health.go
│  │  ├─ go.mod
│  │  └─ go.sum
│  └─ web-react/
│     ├─ src/App.tsx
│     ├─ src/main.tsx
│     ├─ package.json
│     ├─ tsconfig.json
│     └─ vite.config.ts
├─ docs/
│  └─ handover.md
├─ infra/
│  └─ docker/
├─ .gitignore
└─ README.md
```

## Run locally

### 1) API

```powershell
# e:\VSCodeSpace\reverse\maintainer-firewall\apps\api-go
go run .\cmd\server\main.go
```

Health endpoint:

- `http://localhost:8080/health`

### 2) Web

```powershell
# e:\VSCodeSpace\reverse\maintainer-firewall\apps\web-react
npm install
npm run dev
```

Web app:

- `http://localhost:5173`

## Next Milestones

### M1: GitHub App webhook ingestion

- Add `POST /webhook/github`
- Verify `X-Hub-Signature-256`
- Persist event metadata to PostgreSQL

### M2: Basic moderation engine

- Rule-based issue/PR classification
- Actions: label/comment/close suggestion

### M3: Dashboard

- Event list + filters
- Rule hit statistics
- Reviewer workload chart

## Reopen IDE Quick Resume Checklist

1. Open folder: `e:\VSCodeSpace\reverse\maintainer-firewall`
2. Start API (Go) and Web (React)
3. Verify `/health` in browser
4. Create issue `feat: github webhook endpoint`
5. Implement webhook signature verification first

## Notes

- If project path needs to move, use `git`-safe move and keep remote `origin` unchanged.
- Current repo remote is already configured and pushed.
