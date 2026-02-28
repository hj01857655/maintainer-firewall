# maintainer-firewall

Go + React open-source project skeleton for maintainer workflow automation.

## Structure

- `apps/api-go`: Go API service (Gin)
- `apps/web-react`: React console (Vite + TS)
- `infra/docker`: docker files (next step)
- `docs`: architecture/docs

## Run API

```powershell
# e:\VSCodeSpace\reverse\maintainer-firewall\apps\api-go
go mod tidy
go run .\cmd\server\main.go
```

API health:

- `GET http://localhost:8080/health`

## Run Web

```powershell
# e:\VSCodeSpace\reverse\maintainer-firewall\apps\web-react
npm install
npm run dev
```

Web app:

- `http://localhost:5173`
- automatically proxies `/health` to `http://localhost:8080`
