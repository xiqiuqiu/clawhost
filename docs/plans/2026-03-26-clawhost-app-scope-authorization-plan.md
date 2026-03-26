# ClawHost App Scope Authorization v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow non-platform admins to be granted either full-platform access or access to a selected set of apps, and enforce that scope consistently across reads, writes, and admin management UI.

**Architecture:** Reuse the existing `admin_memberships` table as the single grant source. A non-platform admin remains single-role, but may have either one platform-scoped membership or multiple app-scoped memberships for the same role. Access resolution, app/bot handlers, and admin management payloads all consume the same resolved scope summary so filtering and enforcement stay aligned.

**Tech Stack:** Go, Echo, GORM, PostgreSQL, Next.js, TypeScript, shadcn/ui, existing admin audit service.

---

## File Map

### Backend model and resolution

- Modify: `model/admin_membership.go`
  - Add helpers for validating and replacing scoped memberships.
  - Add a reusable resolver for scope mode and allowed app IDs.
- Modify: `model/role.go`
  - Extend `AdminMembershipSummary` / `AdminAccessSummary` with resolved scope information.
- Test: `model/role_test.go`
- Test: `model/admin_role_test.go`

### Backend handlers and authorization

- Modify: `handler/api/v1/admin_users.go`
  - Accept and return `scope_mode` / `app_scope_ids`.
  - Enforce single-role, app-scope validation, and audit metadata.
- Modify: `handler/api/v1/app.go`
  - Filter list and protect reads/writes with resolved scope.
- Modify: `handler/api/v1/admin_bot.go`
  - Filter bot list and enforce app scope on bot actions.
- Modify: `handler/api/v1/bot_upgrade.go`
  - Enforce app scope on upgrade actions and define behavior for bulk upgrade.
- Modify: `middleware/authorize.go`
  - Only if needed to centralize app-scope decisions cleanly.
- Test: `handler/api/v1/admin_users_test.go`
- Test: `handler/api/v1/admin_audit_test.go`
- Test: `handler/api/v1/app_test.go` (create if missing)
- Test: `handler/api/v1/admin_bot_test.go` (create if missing)
- Test: `middleware/authorize_test.go`

### Admin UI

- Modify: `web/admin/src/lib/api.ts`
  - Extend managed admin types and request payloads.
  - Add app list typing if needed by the admin page.
- Modify: `web/admin/src/app/admins/page.tsx`
  - Add scope mode selector and selected-app UI.
  - Show current scope in the admin list.
- Optional create: `web/admin/src/components/admin-scope-selector.tsx`
  - Only if the page becomes too dense; keep inline if simple.

### Verification

- Run: `go test ./...`
- Run: `npm run build`
- Run local server on `http://localhost:28080/admin`

## Task 1: Extend Access Resolution With Scope Summary

**Files:**
- Modify: `model/admin_membership.go`
- Modify: `model/role.go`
- Test: `model/role_test.go`
- Test: `model/admin_role_test.go`

- [ ] **Step 1: Write the failing scope-resolution tests**

Add tests that express:

- one platform membership resolves `scope_mode = platform`
- multiple app memberships with the same role resolve `scope_mode = selected_apps`
- mixed roles across memberships are rejected
- duplicate app IDs are deduplicated in the resolved summary

- [ ] **Step 2: Run the targeted model tests and verify they fail**

Run: `go test ./model -run "TestResolveAdminAccess|TestAdminMembershipScope" -v`

Expected: FAIL because scope mode, selected app IDs, and mixed-role validation are not implemented yet.

- [ ] **Step 3: Add minimal scope-resolution implementation**

Implement in `model/admin_membership.go` / `model/role.go`:

- a helper that loads memberships and validates that all memberships for one admin share the same role
- a helper that returns:
  - effective role
  - `scope_mode` (`platform` or `selected_apps`)
  - sorted/deduplicated `app_scope_ids`
- `ResolveAdminAccess` should expose this summary for handlers and UI

- [ ] **Step 4: Re-run targeted model tests and make them pass**

Run: `go test ./model -run "TestResolveAdminAccess|TestAdminMembershipScope" -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add model/admin_membership.go model/role.go model/role_test.go model/admin_role_test.go
git commit -m "feat: resolve admin app scope summary"
```

## Task 2: Add Scoped Membership Create/Update Support

**Files:**
- Modify: `handler/api/v1/admin_users.go`
- Modify: `model/admin_membership.go`
- Test: `handler/api/v1/admin_users_test.go`
- Test: `handler/api/v1/admin_audit_test.go`

- [ ] **Step 1: Write failing admin management tests**

Add tests that cover:

- creating a non-platform admin with `scope_mode = selected_apps` and multiple app IDs
- updating an existing admin from platform scope to selected-app scope
- rejecting selected-app scope with no apps
- rejecting `platform_admin` with selected-app scope
- rejecting non-existent app IDs
- rejecting mixed-role memberships after update
- audit entry includes before/after scope data

- [ ] **Step 2: Run the targeted admin handler tests and verify they fail**

Run: `go test ./handler/api/v1 -run "Test(CreateAdminUser|UpdateAdminUser|RolePermissionChangesAreAudited|AuditListEndpoint)" -v`

Expected: FAIL because request payloads and audit metadata do not yet support scope mode or app IDs.

- [ ] **Step 3: Extend admin create/update request handling**

In `handler/api/v1/admin_users.go`:

- extend request structs with:
  - `scope_mode`
  - `app_scope_ids`
- validate:
  - `platform_admin` must stay platform-scoped
  - selected-app mode requires at least one valid app ID
  - all created memberships use the same role
- replace direct single-membership writes with a reusable membership replacement helper
- include scope summary in response payloads

- [ ] **Step 4: Extend audit metadata**

Ensure create/update audit entries clearly capture:

- role before/after
- scope mode before/after
- app scope IDs before/after

- [ ] **Step 5: Re-run targeted admin handler tests and make them pass**

Run: `go test ./handler/api/v1 -run "Test(CreateAdminUser|UpdateAdminUser|RolePermissionChangesAreAudited|AuditListEndpoint)" -v`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add handler/api/v1/admin_users.go handler/api/v1/admin_users_test.go handler/api/v1/admin_audit_test.go model/admin_membership.go
git commit -m "feat: support scoped admin memberships"
```

## Task 3: Enforce Scope On App Reads And Writes

**Files:**
- Modify: `handler/api/v1/app.go`
- Modify: `middleware/authorize.go` (only if reuse improves clarity)
- Test: `handler/api/v1/app_test.go`
- Test: `middleware/authorize_test.go`

- [ ] **Step 1: Write failing app-scope tests**

Cover:

- app-scoped viewer only sees granted apps in `ListApps`
- app-scoped admin gets `403` for out-of-scope `GetApp`
- app-scoped admin gets `403` for out-of-scope `UpdateApp`
- app-scoped admin gets `403` for out-of-scope `DeleteApp`
- app-scoped admin gets `403` for out-of-scope `ResetAppToken`

- [ ] **Step 2: Run the targeted app tests and verify they fail**

Run: `go test ./handler/api/v1 -run "Test(ListApps|GetApp|UpdateApp|DeleteApp|ResetAppToken)" -v`

Expected: FAIL because app handlers currently rely on role permission only and do not filter by granted app IDs for lists.

- [ ] **Step 3: Implement minimal app-scope enforcement**

Implement one reusable path for:

- listing only granted apps when scope mode is `selected_apps`
- rejecting direct app operations when the target app is outside the allowed set

Prefer handler-level filtering over ad hoc SQL in multiple places if a shared helper can keep it small and readable.

- [ ] **Step 4: Re-run targeted app tests and make them pass**

Run: `go test ./handler/api/v1 -run "Test(ListApps|GetApp|UpdateApp|DeleteApp|ResetAppToken)" -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add handler/api/v1/app.go handler/api/v1/app_test.go middleware/authorize.go middleware/authorize_test.go
git commit -m "feat: enforce app scope on app operations"
```

## Task 4: Enforce Scope On Bot Reads And Writes

**Files:**
- Modify: `handler/api/v1/admin_bot.go`
- Modify: `handler/api/v1/bot_upgrade.go`
- Test: `handler/api/v1/admin_bot_test.go`

- [ ] **Step 1: Write failing bot-scope tests**

Cover:

- bot list only returns bots under granted apps
- out-of-scope bot start is rejected
- out-of-scope bot stop is rejected
- out-of-scope bot delete is rejected
- out-of-scope bot upgrade is rejected
- define and test the expected behavior of bulk upgrade:
  - recommended: only operate on in-scope bots for non-platform admins

- [ ] **Step 2: Run the targeted bot tests and verify they fail**

Run: `go test ./handler/api/v1 -run "Test(AdminListBots|AdminStartBot|AdminStopBot|AdminDeleteBot|UpgradeBot|UpgradeAllBots)" -v`

Expected: FAIL because bot list and lifecycle actions do not yet use resolved app scope consistently.

- [ ] **Step 3: Implement minimal bot-scope enforcement**

Implement:

- bot list filtering by allowed app IDs
- direct bot operation scope checks against the bot's `app_id`
- explicit behavior for bulk upgrade/restart under app scope

- [ ] **Step 4: Re-run targeted bot tests and make them pass**

Run: `go test ./handler/api/v1 -run "Test(AdminListBots|AdminStartBot|AdminStopBot|AdminDeleteBot|UpgradeBot|UpgradeAllBots)" -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add handler/api/v1/admin_bot.go handler/api/v1/bot_upgrade.go handler/api/v1/admin_bot_test.go
git commit -m "feat: enforce app scope on bot operations"
```

## Task 5: Add Scope Controls To The Admin UI

**Files:**
- Modify: `web/admin/src/lib/api.ts`
- Modify: `web/admin/src/app/admins/page.tsx`
- Optional create: `web/admin/src/components/admin-scope-selector.tsx`

- [ ] **Step 1: Write the UI behavior checklist before editing code**

Document expected behavior in the task notes or comments:

- non-platform roles can choose `platform` or `selected_apps`
- `platform_admin` forces `platform`
- selected-app mode requires one or more apps
- current scope is visible in the list and edit dialog

- [ ] **Step 2: Implement API type updates**

Extend managed admin payloads and request types with:

- `scope_mode`
- `app_scope_ids`
- optional app labels for display

- [ ] **Step 3: Implement the form controls**

In `web/admin/src/app/admins/page.tsx`:

- load app list for selection
- add scope mode selector
- add multi-select app chooser for selected-app mode
- disable invalid states
- show current scope in the admins table/cards

- [ ] **Step 4: Build the admin UI and fix any errors**

Run: `npm run build`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/api.ts web/admin/src/app/admins/page.tsx web/admin/src/components/admin-scope-selector.tsx
git commit -m "feat: add admin app scope controls"
```

Note: If no separate component is created, omit that file from the commit.

## Task 6: Full Verification On Local Admin UI

**Files:**
- No new production files unless fixes are needed

- [ ] **Step 1: Run the full backend suite**

Run: `go test ./...`

Expected: PASS

- [ ] **Step 2: Run the production admin build**

Run: `npm run build`

Expected: PASS

- [ ] **Step 3: Restart local dependencies and service**

Run the local DB port-forward if needed, then restart the app on `http://localhost:28080`.

Expected:

- health check returns `{"status":"ok"}`
- admin UI loads on `/admin`

- [ ] **Step 4: Verify real user flows**

Use real accounts and UI:

- platform admin creates or edits a non-platform admin with selected-app scope
- scoped admin only sees granted apps
- scoped admin only sees bots under granted apps
- scoped admin cannot operate on out-of-scope app or bot resources
- audit shows the scope change clearly

- [ ] **Step 5: Commit any verification fixes**

```bash
git add <files you changed>
git commit -m "fix: polish app scope authorization"
```

Skip this commit if no fixes were required.

## Final Notes

- Keep the first slice strict: one effective role per admin, even if represented by multiple app-scoped membership rows.
- Prefer extending the existing payloads and UI rather than adding separate scope-specific endpoints or pages.
- Do not introduce a new grants table in this slice.
- If a reusable helper keeps filtering and write enforcement consistent, prefer that over duplicating ad hoc checks in each handler.

