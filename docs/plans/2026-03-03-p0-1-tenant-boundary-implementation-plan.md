# P0-1 Tenant Boundary Foundation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在不破坏现有 V1 主流程的前提下，为系统引入可验证的租户边界能力（数据隔离 + 租户上下文 + 基础租户管理）。

**Architecture:** 采用“上下文透传租户 ID”的增量方案：HTTP 层解析租户并写入 `context.Context`，Store 层从上下文读取租户并做 SQL 过滤/写入，不改大部分接口签名。数据库通过 idempotent schema ensure 方式新增 `tenants` 表和业务表 `tenant_id` 列，并把历史数据回填到默认租户 `default`。登录令牌增加 `tenant_id` claim，受保护接口强制按 claim 隔离。

**Tech Stack:** Go + Gin + PostgreSQL/MySQL + React + TypeScript + Vitest（前端测试可后补）.

---

### Task 1: 租户上下文基础设施（后端）

**Files:**
- Create: `apps/api-go/internal/tenantctx/tenantctx.go`
- Create: `apps/api-go/internal/tenantctx/tenantctx_test.go`

**Step 1: Write the failing test**

```go
func TestWithTenantIDAndFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenantID(ctx, "acme")
	got, ok := FromContext(ctx)
	if !ok || got != "acme" {
		t.Fatalf("want acme, got %q ok=%v", got, ok)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tenantctx -count=1`  
Expected: FAIL (`WithTenantID` / `FromContext` not found)

**Step 3: Write minimal implementation**

```go
type key string
const tenantKey key = "tenant_id"
```

- 提供 `WithTenantID(ctx, tenantID)`、`FromContext(ctx)`、`MustFromContext(ctx, fallback)`。
- 统一去空格，空值回退 `default`。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tenantctx -count=1`  
Expected: PASS

**Step 5: Commit**

```bash
git add apps/api-go/internal/tenantctx/tenantctx.go apps/api-go/internal/tenantctx/tenantctx_test.go
git commit -m "feat(api): add tenant context helpers"
```

---

### Task 2: JWT + 中间件注入租户上下文

**Files:**
- Modify: `apps/api-go/internal/http/handlers/auth.go`
- Modify: `apps/api-go/internal/http/handlers/auth_test.go`

**Step 1: Write the failing test**

- 新增测试覆盖：
1. 登录成功返回的 token 包含 `tenant_id` claim。
2. `AuthMiddleware` 将 `tenant_id` 写入 gin context（`c.GetString("tenant_id")`）和 request context。
3. token 不带 `tenant_id` 时兼容回退到 `default`（用于平滑升级）。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/http/handlers -run "TestAuth.*Tenant|TestAuthMiddleware.*Tenant" -count=1`  
Expected: FAIL（claim/上下文字段缺失）

**Step 3: Write minimal implementation**

- `issueJWT` 增加 `tenant_id` claim 参数。
- 登录流程优先从请求体读取 `tenant_id`（为空则 `default`）。
- `AuthMiddleware` 解析 claim 后设置：
1. `c.Set("tenant_id", tenantID)`
2. `c.Request = c.Request.WithContext(tenantctx.WithTenantID(...))`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/http/handlers -run "TestAuth.*Tenant|TestAuthMiddleware.*Tenant" -count=1`  
Expected: PASS

**Step 5: Commit**

```bash
git add apps/api-go/internal/http/handlers/auth.go apps/api-go/internal/http/handlers/auth_test.go
git commit -m "feat(api): inject tenant id into jwt and auth middleware context"
```

---

### Task 3: 数据库租户化 schema（PostgreSQL + MySQL）

**Files:**
- Modify: `apps/api-go/internal/store/webhook_store.go`
- Modify: `apps/api-go/internal/store/webhook_store_mysql.go`
- Modify: `apps/api-go/internal/store/webhook_store_mysql_test.go`

**Step 1: Write the failing test**

- 新增 schema 语句构造/迁移辅助测试（至少验证）：
1. 存在 `tenants` 表创建逻辑。
2. 关键业务表包含 `tenant_id` 列（`webhook_events`、`webhook_alerts`、`webhook_rules`、`webhook_action_failures`、`audit_logs`、`admin_users`）。
3. `webhook_events` 的唯一约束变更为 `(tenant_id, delivery_id)`。
4. `admin_users` 的唯一约束变更为 `(tenant_id, username)`。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run "Test.*Tenant.*Schema|Test.*Tenant.*Unique" -count=1`  
Expected: FAIL（无租户 schema）

**Step 3: Write minimal implementation**

- 在 `ensureSchema` / MySQL 对应函数中新增：
1. `tenants(id, name, is_active, created_at, updated_at)`
2. `INSERT ... default tenant`
3. 所有核心表 `ADD COLUMN tenant_id ... DEFAULT 'default'`
4. 新增租户相关索引（`tenant_id` 单列和组合索引）
5. 旧唯一约束降级并新增租户复合唯一约束

**Step 4: Run test to verify it passes**

Run: `go test ./internal/store -run "Test.*Tenant.*Schema|Test.*Tenant.*Unique" -count=1`  
Expected: PASS

**Step 5: Commit**

```bash
git add apps/api-go/internal/store/webhook_store.go apps/api-go/internal/store/webhook_store_mysql.go apps/api-go/internal/store/webhook_store_mysql_test.go
git commit -m "feat(store): add tenant-aware schema and unique constraints"
```

---

### Task 4: Store 查询/写入租户隔离

**Files:**
- Modify: `apps/api-go/internal/store/webhook_store.go`
- Modify: `apps/api-go/internal/store/webhook_store_mysql.go`
- Modify: `apps/api-go/internal/http/handlers/events_test.go`
- Modify: `apps/api-go/internal/http/handlers/alerts_test.go`
- Modify: `apps/api-go/internal/http/handlers/rules_test.go`
- Modify: `apps/api-go/internal/http/handlers/webhook_test.go`

**Step 1: Write the failing test**

- Handler 层新增断言：调用 store 时 request context 中存在 `tenant_id`。
- Store 层新增断言/测试：列表查询自动附带租户过滤条件（`WHERE tenant_id = ?`）。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/http/handlers -run "Test(Events|Alerts|Rules|Webhook).*Tenant" -count=1`  
Expected: FAIL（上下文未透传租户）

**Step 3: Write minimal implementation**

- 所有写入 SQL 增加 `tenant_id` 列值（从 `tenantctx.FromContext(ctx)` 获取）。
- 所有列表/过滤 options/统计 SQL 增加租户过滤。
- webhook 入库路径在进入 handler 时补充租户（先支持 header `X-MF-Tenant-ID`，默认 `default`）。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/http/handlers -run "Test(Events|Alerts|Rules|Webhook).*Tenant" -count=1`  
Expected: PASS

**Step 5: Commit**

```bash
git add apps/api-go/internal/store/webhook_store.go apps/api-go/internal/store/webhook_store_mysql.go apps/api-go/internal/http/handlers/events_test.go apps/api-go/internal/http/handlers/alerts_test.go apps/api-go/internal/http/handlers/rules_test.go apps/api-go/internal/http/handlers/webhook_test.go
git commit -m "feat(api): enforce tenant scoping in store reads and writes"
```

---

### Task 5: 租户管理 API（最小集合）

**Files:**
- Create: `apps/api-go/internal/http/handlers/tenants.go`
- Create: `apps/api-go/internal/http/handlers/tenants_test.go`
- Modify: `apps/api-go/internal/store/webhook_store.go`
- Modify: `apps/api-go/internal/store/webhook_store_mysql.go`
- Modify: `apps/api-go/cmd/server/main.go`

**Step 1: Write the failing test**

- `GET /api/tenants` 返回可用租户列表。
- `POST /api/tenants` 创建租户（ID/名称唯一）。
- `PATCH /api/tenants/:id/active` 可启停租户。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/http/handlers -run "TestTenants" -count=1`  
Expected: FAIL（路由/handler/store 方法缺失）

**Step 3: Write minimal implementation**

- 新增租户存储结构与 store 方法：
1. `ListTenants`
2. `CreateTenant`
3. `UpdateTenantActive`
- 在 `main.go` 注册受保护路由：
1. `GET /api/tenants`
2. `POST /api/tenants`
3. `PATCH /api/tenants/:id/active`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/http/handlers -run "TestTenants" -count=1`  
Expected: PASS

**Step 5: Commit**

```bash
git add apps/api-go/internal/http/handlers/tenants.go apps/api-go/internal/http/handlers/tenants_test.go apps/api-go/internal/store/webhook_store.go apps/api-go/internal/store/webhook_store_mysql.go apps/api-go/cmd/server/main.go
git commit -m "feat(api): add tenant management endpoints"
```

---

### Task 6: 前端登录与租户上下文最小接入

**Files:**
- Modify: `apps/web-react/src/auth.ts`
- Modify: `apps/web-react/src/api.ts`
- Modify: `apps/web-react/src/pages/LoginPage.tsx`
- Modify: `apps/web-react/src/i18n.ts` (如需新增文案 key)

**Step 1: Write the failing behavior check**

- 手工检查：
1. 登录请求无法传递 `tenant_id`。
2. 刷新页面后无法读取当前租户上下文。

**Step 2: Implement minimal code**

- 登录页新增租户输入（默认 `default`）。
- 登录成功后保存 `tenant_id` 到 localStorage。
- 请求层对未认证登录接口携带 `tenant_id` body；认证后主要依赖 token claim，无需额外 header。

**Step 3: Verify frontend build**

Run: `npm run build` (in `apps/web-react`)  
Expected: PASS

**Step 4: Commit**

```bash
git add apps/web-react/src/auth.ts apps/web-react/src/api.ts apps/web-react/src/pages/LoginPage.tsx apps/web-react/src/i18n.ts
git commit -m "feat(web): support tenant-aware login context"
```

---

### Task 7: 验证与回归门禁

**Files:**
- Modify: `README.md`
- Modify: `docs/requirements.md`
- Modify: `docs/design.md`
- Modify: `docs/handover.md`

**Step 1: Backend verification**

Run:
- `go test ./... -count=1` (in `apps/api-go`)
- `go build -buildvcs=false ./...` (in `apps/api-go`)

Expected: 全部通过

**Step 2: Frontend verification**

Run:
- `npm run build` (in `apps/web-react`)

Expected: PASS

**Step 3: Manual tenant isolation smoke**

1. 用 tenant `default` 登录并创建规则。  
2. 用 tenant `team-b` 登录，确认看不到 `default` 规则。  
3. 分别触发 webhook，确认事件各自隔离。  

**Step 4: Update docs and commit**

```bash
git add README.md docs/requirements.md docs/design.md docs/handover.md
git commit -m "docs: add tenant boundary architecture and runbook"
```

---

## Risks / Rollback

1. **风险：** 旧唯一索引未正确迁移，导致多租户仍冲突。  
   **回滚：** 保留迁移前快照；分步骤执行 DDL；出错后回退到仅 `default` 模式。

2. **风险：** 某些 SQL 漏加租户过滤导致越权读取。  
   **回滚：** 用 `Test*Tenant` 套件做阻断门禁；上线前做租户 A/B 交叉验收。

3. **风险：** 历史 token 无 `tenant_id` 导致认证失败。  
   **回滚：** 中间件保留 `default` fallback 一个发布周期。

---

## Done Criteria

1. 关键业务表写入与查询均绑定租户。  
2. `admin_users` 与 `webhook_events` 唯一性升级为租户内唯一。  
3. 登录 token 含 `tenant_id`，受保护接口可解析并透传。  
4. 同一实例内两个租户互不可见数据。  
5. `go test ./...` 与 `npm run build` 通过。  
