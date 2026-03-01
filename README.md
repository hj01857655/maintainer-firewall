# 🚀 Maintainer Firewall

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8)](https://golang.org)
[![React Version](https://img.shields.io/badge/React-18.3+-61DAFB)](https://reactjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.6+-3178C6)](https://www.typescriptlang.org)
[![Vite](https://img.shields.io/badge/Vite-5.4+-646CFF)](https://vitejs.dev)

**开源的维护者工作流自动化平台**

Maintainer Firewall 是一个基于 Go + React 的开源项目，为维护者提供强大的 GitHub webhook 自动化处理能力。通过可配置的规则系统，实现智能的事件处理、自动标签添加、评论回复等功能，大幅提升开源项目维护效率。

## ✨ 核心特性

### 🔐 安全可靠
- **Webhook 签名验证** - 确保请求来源可靠
- **JWT 身份认证** - 保护 API 和管理界面
- **环境隔离** - 支持开发/生产环境配置

### 🎯 智能自动化
- **规则引擎** - 可配置的事件匹配和自动化处理
- **多事件支持** - Issues、PR、Comments 等 GitHub 事件
- **批量操作** - 支持标签添加、评论回复等操作

### 📊 数据洞察
- **实时监控** - Dashboard 展示系统运行状态
- **性能指标** - Webhook 处理延迟、成功率统计
- **审计日志** - 完整的操作记录和追踪

### 🌐 现代化界面
- **响应式设计** - 支持桌面和移动设备
- **深色模式** - 护眼的深色主题支持
- **国际化** - 中英文双语界面
- **无障碍访问** - 完整的键盘导航和屏幕阅读器支持

## 🏗️ 项目架构

```
maintainer-firewall/
├── apps/
│   ├── api-go/          # Go API 服务 (Gin 框架)
│   └── web-react/       # React 管理控制台 (Vite + TypeScript)
├── docs/                # 项目文档
├── scripts/             # 部署和测试脚本
└── README.md           # 项目说明
```

### 技术栈

| 组件 | 技术栈 | 说明 |
|------|--------|------|
| **后端** | Go + Gin | 高性能 HTTP 服务框架 |
| **数据库** | PostgreSQL/MySQL | 关系型数据存储 |
| **前端** | React + TypeScript | 现代化用户界面 |
| **构建工具** | Vite | 快速的开发和构建工具 |
| **样式** | Tailwind CSS | 实用优先的 CSS 框架 |
| **状态管理** | React Query | 强大的数据获取和缓存 |

## 🚀 快速开始

### 环境要求

- **Go** 1.21+
- **Node.js** 18+
- **PostgreSQL** 12+ 或 **MySQL** 8.0+
- **Git** 2.0+

### 启动后端 API

```bash
# 1. 配置环境变量
cp .env.example .env
# 编辑 .env 文件，设置 DATABASE_URL

# 2. 安装依赖
cd apps/api-go
go mod tidy

# 3. 启动服务
go run ./cmd/server/main.go
```

**默认配置**：
- 端口：8080
- 管理员账号：`admin` / `admin123`
- 数据库：需要手动配置

### 启动前端控制台

```bash
# 1. 安装依赖
cd apps/web-react
npm install

# 2. 启动开发服务器
npm run dev
```

访问 `http://localhost:5173` 即可使用管理控制台。

## 📚 API 接口

### 认证接口

```http
POST /auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

### Webhook 接口

```http
POST /webhook/github
X-Hub-Signature-256: sha256=...
Content-Type: application/json

{
  "action": "opened",
  "issue": { ... },
  "repository": { ... }
}
```

### 数据接口

| 接口 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 服务健康检查 |
| `/events` | GET | 获取 webhook 事件列表 |
| `/rules` | GET/POST | 规则管理 |
| `/alerts` | GET | 获取告警列表 |
| `/metrics/*` | GET | 系统性能指标 |

## 🧪 测试和验证

### 自动化测试

```bash
# 前端单元测试
cd apps/web-react
npm run test

# 端到端测试
cd scripts
./e2e.ps1
```

### 手动验证

```powershell
# 登录获取 Token
$token = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/auth/login" -Body '{"username":"admin","password":"admin123"}' -ContentType "application/json"

# 查看事件数据
Invoke-RestMethod "http://localhost:8080/events?limit=5" -Headers @{Authorization="Bearer $token"}
```

## 📖 使用指南

### 1. 配置 GitHub Webhook

1. 在 GitHub 仓库设置中添加 webhook
2. URL：`https://your-domain/webhook/github`
3. Content-Type：`application/json`
4. Secret：配置 `GITHUB_WEBHOOK_SECRET`

### 2. 创建自动化规则

1. 登录管理控制台
2. 进入 "规则管理" 页面
3. 点击 "新建规则"
4. 配置事件类型、关键词、自动化操作

### 3. 监控系统状态

- **Dashboard**：查看系统概览和性能指标
- **事件流**：监控所有 webhook 事件
- **告警中心**：查看规则匹配结果
- **审计日志**：追踪所有操作记录

## 🤝 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. **Fork** 本项目
2. 创建特性分支：`git checkout -b feature/amazing-feature`
3. 提交更改：`git commit -m 'Add amazing feature'`
4. 推送分支：`git push origin feature/amazing-feature`
5. 提交 **Pull Request**

### 开发环境设置

```bash
# 克隆项目
git clone https://github.com/your-username/maintainer-firewall.git
cd maintainer-firewall

# 启动后端
cd apps/api-go
go run ./cmd/server/main.go

# 启动前端（新终端）
cd apps/web-react
npm install
npm run dev
```

## 📄 许可证

本项目采用 **MIT 许可证** - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

感谢所有为这个项目贡献的开发者！

- **Gin** - 优秀的 Go Web 框架
- **React** - 用户界面库
- **Tailwind CSS** - 实用优先的样式框架
- **Vite** - 下一代前端构建工具

---

**⭐ 如果这个项目对你有帮助，请给我们一个 Star！**

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
- AUTH_ENV_FALLBACK=true (set false to force DB-admin-only login)
- JWT_SECRET=dev-jwt-secret (or ACCESS_TOKEN fallback)
- GITHUB_WEBHOOK_SECRET=dev-webhook-secret
- GITHUB_TOKEN is optional (empty by default)
- DATABASE_URL is still required (set in `.env`/environment; if omitted, API starts but store initialization will fail)
- `GITHUB_EVENTS_SYNC_INTERVAL_MINUTES` controls periodic GitHub event sync (`0`=disabled, `5`=every 5 minutes)


API endpoints:

- `GET http://localhost:8080/health`
- `POST http://localhost:8080/auth/login` (default local account in development: `admin` / `admin123`)
- `POST http://localhost:8080/webhook/github`
- `GET http://localhost:8080/events?limit=20&offset=0&event_type=issues&action=opened` (auth required)
  - response includes `total` for pagination
- `GET http://localhost:8080/events/filter-options` (auth required)
  - return full-dataset filter options: `event_types/actions/repositories/senders`
- `GET http://localhost:8080/events?source=github` (auth required)
  - default `mode=types`, return unique `event_types`
- `GET http://localhost:8080/events?source=github&mode=items&limit=20&offset=0` (auth required)
  - return paginated recent GitHub event items
- `GET http://localhost:8080/events?source=github&sync=true` (auth required)
  - pull current GitHub user recent events and sync into `webhook_events`
- `GET http://localhost:8080/events/sync-status` (auth required)
  - return GitHub sync runtime status and last sync counters
- `GET http://localhost:8080/alerts?limit=20&offset=0&event_type=issues&action=opened&suggestion_type=label` (auth required)
  - response includes `total` for pagination
- `GET http://localhost:8080/alerts/filter-options` (auth required)
  - return full-dataset filter options: `event_types/actions/suggestion_types/repositories/senders`
- `GET/POST http://localhost:8080/rules` (auth required)
- `GET http://localhost:8080/rules/filter-options` (auth required)
  - return full-dataset filter options: `event_types/suggestion_types/active_states`
- `PATCH http://localhost:8080/rules/:id/active` (auth required)
- `GET http://localhost:8080/action-failures` (auth required)
- `POST http://localhost:8080/action-failures/:id/retry` (auth required)
- `GET http://localhost:8080/audit-logs` (auth required)
- `GET http://localhost:8080/metrics/overview` (auth required)
- `GET http://localhost:8080/metrics/timeseries` (auth required)
- `GET http://localhost:8080/config-status` (auth required)
- `GET http://localhost:8080/config-view` (auth required)
- `POST http://localhost:8080/config-update` (auth required)

## Run Web

```powershell
# <repo-root>/apps/web-react
npm install
npm run dev
```

Web app:

- `http://localhost:5173`
- automatically proxies `/health` / `/auth` / `/events` / `/alerts` / `/rules` / `/action-failures` / `/audit-logs` / `/metrics` / `/config-*` to `http://localhost:8080`

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
- webhook payload now requires real test target parameters (`-GitHubTestRepo`, `-GitHubTestIssueNumber`) to avoid placeholder dirty data

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

# list failures and retry one item
Invoke-RestMethod "http://localhost:8080/action-failures?limit=20&offset=0&include_resolved=true" -Headers $headers
# Invoke-RestMethod -Method Post "http://localhost:8080/action-failures/<ID>/retry" -Headers $headers

# metrics and audit
Invoke-RestMethod "http://localhost:8080/metrics/overview?window=24h" -Headers $headers
Invoke-RestMethod "http://localhost:8080/audit-logs?limit=20&offset=0" -Headers $headers
```

## CI

- GitHub Actions workflow: `.github/workflows/ci.yml`
- Runs Go test/build + Web build on push/pull_request for changed app paths

## Main-flow checklist (done)

- Webhook signature validation
- Persist webhook events
- Rule suggestion generation (`label` / `comment`)
- Persist rule-hit alerts
- Configurable rules API (`GET/POST /rules`, `PATCH /rules/:id/active`)
- Auto execute GitHub actions (label/comment)
- Action retry + failure recording (`webhook_action_failures`) + retry API
- `/events` GitHub source mode (`source=github`, `mode=types|items`) + on-demand sync (`sync=true`) + `/events/sync-status` + periodic sync worker
- Login + protected API/UI routes (JWT)
- Query events/rules/alerts with pagination/filter + `total`
- Full-dataset filter-options APIs for Events/Rules/Alerts
- Failures/audit/metrics/config APIs
- Console pages: events/rules/alerts/failures/audit/system-config
- CI checks for API/Web build

## Secondary (next)

- Dashboard alert summary widgets
- Rich filters (repository/sender/date range)
- Export & reporting
## License

MIT
