# Full Dataset Filter Options Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add full-dataset filter options APIs for Events/Alerts/Rules and wire frontend dropdowns to stable backend-provided options.

**Architecture:** Keep existing list endpoints unchanged and add three dedicated protected endpoints (`/events/filter-options`, `/alerts/filter-options`, `/rules/filter-options`). Backend computes distinct non-empty values from DB; frontend loads once per page and uses returned options for dropdowns. This keeps concerns separated (options vs list rows) and avoids pagination-coupled option drift.

**Tech Stack:** Go (Gin + store layer for PostgreSQL/MySQL), React + TypeScript + i18next, existing JWT auth middleware.

---

### Task 1: Events filter options API (TDD)

**Files:**
- Modify: `apps/api-go/internal/http/handlers/events_test.go`
- Modify: `apps/api-go/internal/http/handlers/events.go`
- Modify: `apps/api-go/internal/store/webhook_store.go`
- Modify: `apps/api-go/internal/store/webhook_store_mysql.go`
- Test: `apps/api-go/internal/http/handlers/events_test.go`

**Step 1: Write failing tests**
- Add handler test for `GET /events/filter-options`:
  - returns `200` + `ok=true`
  - contains `options.event_types/actions/repositories/senders`
  - store call error returns `500`

**Step 2: Run test to verify it fails**
- Run: `go test ./internal/http/handlers -run TestEventsFilterOptions -count=1`
- Expected: FAIL (handler/method not implemented)

**Step 3: Write minimal implementation**
- Extend events store interface with distinct query method.
- Implement query in PG and MySQL stores.
- Add `EventsHandler.FilterOptions` and register route in `main.go`.

**Step 4: Run test to verify it passes**
- Run: `go test ./internal/http/handlers -run TestEventsFilterOptions -count=1`
- Expected: PASS

**Step 5: Commit**
```bash
git add apps/api-go/internal/http/handlers/events_test.go apps/api-go/internal/http/handlers/events.go apps/api-go/internal/store/webhook_store.go apps/api-go/internal/store/webhook_store_mysql.go apps/api-go/cmd/server/main.go
git commit -m "feat(api): add events full filter-options endpoint"
```

### Task 2: Alerts filter options API (TDD)

**Files:**
- Modify: `apps/api-go/internal/http/handlers/alerts_test.go`
- Modify: `apps/api-go/internal/http/handlers/alerts.go`
- Modify: `apps/api-go/internal/store/webhook_store.go`
- Modify: `apps/api-go/internal/store/webhook_store_mysql.go`

**Step 1: Write failing tests**
- Add `GET /alerts/filter-options` tests for success and store error.

**Step 2: Run failing tests**
- Run: `go test ./internal/http/handlers -run TestAlertsFilterOptions -count=1`
- Expected: FAIL

**Step 3: Implement minimal code**
- Add store distinct query for alerts fields.
- Add handler method and route wiring.

**Step 4: Verify pass**
- Run: `go test ./internal/http/handlers -run TestAlertsFilterOptions -count=1`
- Expected: PASS

**Step 5: Commit**
```bash
git add apps/api-go/internal/http/handlers/alerts_test.go apps/api-go/internal/http/handlers/alerts.go apps/api-go/internal/store/webhook_store.go apps/api-go/internal/store/webhook_store_mysql.go apps/api-go/cmd/server/main.go
git commit -m "feat(api): add alerts full filter-options endpoint"
```

### Task 3: Rules filter options API (TDD)

**Files:**
- Modify: `apps/api-go/internal/http/handlers/rules_test.go`
- Modify: `apps/api-go/internal/http/handlers/rules.go`
- Modify: `apps/api-go/internal/store/webhook_store.go`
- Modify: `apps/api-go/internal/store/webhook_store_mysql.go`

**Step 1: Write failing tests**
- Add `GET /rules/filter-options` tests:
  - includes `event_types/suggestion_types/active_states`
  - error path returns `500`

**Step 2: Run failing tests**
- Run: `go test ./internal/http/handlers -run TestRulesFilterOptions -count=1`
- Expected: FAIL

**Step 3: Implement minimal code**
- Add store distinct query for rules fields.
- Handler + route.
- `active_states` synthesized as `active/inactive` when seen in dataset.

**Step 4: Verify pass**
- Run: `go test ./internal/http/handlers -run TestRulesFilterOptions -count=1`
- Expected: PASS

**Step 5: Commit**
```bash
git add apps/api-go/internal/http/handlers/rules_test.go apps/api-go/internal/http/handlers/rules.go apps/api-go/internal/store/webhook_store.go apps/api-go/internal/store/webhook_store_mysql.go apps/api-go/cmd/server/main.go
git commit -m "feat(api): add rules full filter-options endpoint"
```

### Task 4: Frontend wire-up (Events/Alerts/Rules)

**Files:**
- Modify: `apps/web-react/src/pages/EventsPage.tsx`
- Modify: `apps/web-react/src/pages/AlertsPage.tsx`
- Modify: `apps/web-react/src/pages/RulesPage.tsx`

**Step 1: Write failing behavior check (manual quick check)**
- Confirm current dropdown options still page-coupled.

**Step 2: Implement minimal frontend changes**
- Add `filterOptions` state in each page.
- On mount, call corresponding `/api/*/filter-options` endpoint.
- Use backend options as primary source; keep selected value fallback.
- Keep existing list query and filter submission behavior unchanged.

**Step 3: Verify frontend build**
- Run: `npm run build` in `apps/web-react`
- Expected: PASS

**Step 4: Commit**
```bash
git add apps/web-react/src/pages/EventsPage.tsx apps/web-react/src/pages/AlertsPage.tsx apps/web-react/src/pages/RulesPage.tsx
git commit -m "feat(web): use backend full filter-options for events alerts rules"
```

### Task 5: Documentation + full verification + push

**Files:**
- Modify: `README.md`
- Modify: `docs/requirements.md`
- Modify: `docs/design.md`
- Modify: `docs/handover.md`

**Step 1: Update docs**
- Add new filter-options APIs and mention full-dataset dropdown source.
- Keep endpoint examples concise.

**Step 2: Run full verification**
- `go test ./... -count=1` (`apps/api-go`)
- `go build ./...` (`apps/api-go`)
- `npm run build` (`apps/web-react`)

**Step 3: Commit docs**
```bash
git add README.md docs/requirements.md docs/design.md docs/handover.md
git commit -m "docs: add full filter-options APIs and frontend usage"
```

**Step 4: Push in batches**
```bash
git push origin main
```
