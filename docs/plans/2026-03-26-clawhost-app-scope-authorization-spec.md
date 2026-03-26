# ClawHost App Scope Authorization v1 Spec

**Date:** 2026-03-26

## Summary

ClawHost now has persisted admin roles and persisted role-to-permission policy. The next gap is scope management. Today an admin role can describe what actions someone may perform, but the system still needs a manageable way to describe which apps those actions apply to.

App Scope Authorization v1 adds that missing layer.

The first slice keeps the admin model intentionally simple:

- each admin user still has one effective role
- non-platform roles may be granted full-platform access or a selected set of apps
- app and bot visibility must respect that selected app scope
- mutating actions on apps and bots must be rejected outside the granted scope

This keeps the system understandable while making it suitable for real business team isolation.

## Problem

The current model has one practical weakness:

- roles answer "what can this admin do?"
- the system still needs a clean answer to "which apps can this admin do it on?"

Without managed app scope:

- an `operator` role is too broad once granted
- an `app_admin` role cannot be limited to a business unit's apps
- a `viewer` either sees everything or nothing useful
- platform teams cannot safely delegate day-to-day ownership to app teams

## Goals

- Keep the current admin mental model simple
- Add app scope to non-platform admin grants
- Let platform admins manage app scope from the existing admin management UI
- Filter app and bot reads by granted app scope
- Enforce app scope on mutating app and bot operations
- Audit changes to granted app scope

## Non-Goals

- Multiple roles per admin in this slice
- Department hierarchy
- Custom resource groups
- Approval workflows
- End-user permission changes outside admin management

## Recommended Boundary

App Scope Authorization v1 should use this boundary:

- one admin user has one effective role
- `platform_admin` always remains platform-wide
- `app_admin`, `operator`, and `viewer` may be either:
  - platform-wide
  - app-scoped to one or more apps

This is the smallest model that solves real delegation without making the UI or authorization logic too complex.

## Approaches Considered

### Option A: Single role, multiple app scopes

Each admin keeps one role. Non-platform roles can be granted access to many apps.

Pros:

- simplest evolution from the current system
- easiest to explain in the admin UI
- smallest backend and test surface
- enough for most first enterprise deployments

Cons:

- less expressive than full multi-role policy

### Option B: Multiple roles, each with its own app scope

An admin could be both `viewer` for one set of apps and `operator` for another.

Pros:

- strongest long-term expressiveness

Cons:

- much heavier UI
- more complex audit and validation rules
- harder to reason about effective access

### Option C: Single role, single app scope

Each admin could only be tied to one app.

Pros:

- smallest implementation

Cons:

- too weak for realistic operations teams

## Recommendation

Use **Option A: single role, multiple app scopes**.

It preserves the current admin experience, solves the real enterprise delegation problem, and leaves room for future multi-role expansion without forcing that complexity into this slice.

## Core Design

### Role remains action authority

The role still answers:

- which actions this admin may perform

That logic remains powered by the persisted role-permission system already introduced in Permission Matrix v2.

### Scope answers where actions apply

The scope layer answers:

- whether the admin is platform-wide
- or limited to a selected app set

For non-platform roles, app scope becomes part of the grant itself.

### Effective access model

The system should resolve access in this order:

1. load the admin membership
2. resolve role permissions
3. determine whether the membership is platform-wide or app-scoped
4. if app-scoped, restrict app and bot access to the granted app IDs

## Data Model

### Keep: `admin_memberships`

This table already models:

- `admin_user_id`
- `role`
- `scope_type`
- `scope_id`

For App Scope Authorization v1, it should remain the source of truth for grants.

### Interpretation in v1

- `platform_admin` uses:
  - `scope_type = platform`
  - `scope_id = ""`

- non-platform full-platform grants use:
  - `scope_type = platform`
  - `scope_id = ""`

- non-platform app-scoped grants use one membership record per granted app:
  - `scope_type = app`
  - `scope_id = <app_id>`

This avoids adding a new table in the first slice and fits the current model well.

### Effective role consistency

Because v1 keeps one effective role per admin, all memberships for an admin must use the same role key.

That means the system should reject:

- mixed roles across memberships for one admin

## Authorization Rules

### Platform admin

`platform_admin` remains unchanged:

- can see all apps
- can see all bots
- can operate on all apps and bots
- can manage admin policy

### Non-platform full-platform grant

If a non-platform admin has one platform-scoped membership:

- reads are platform-wide
- writes are platform-wide
- actions are still limited by that role's permissions

### Non-platform app-scoped grant

If a non-platform admin has app-scoped memberships:

- app list returns only granted apps
- bot list returns only bots whose `app_id` is granted
- app detail and update operations require the target app to be granted
- bot lifecycle and delete operations require the target bot's `app_id` to be granted

## Admin UI Design

The existing `Admins` page should be extended rather than creating a new page.

### Create and edit form

Add a new grant section:

- scope mode:
  - `platform`
  - `selected_apps`

If `selected_apps` is chosen:

- show a selectable list of apps
- allow multi-select

### UX rules

- `platform_admin` cannot be switched to `selected_apps`
- if role is not `platform_admin` and scope mode is `selected_apps`, at least one app must be selected
- role changes must keep memberships internally consistent

## API Shape

The first slice should update existing admin management endpoints rather than introducing a separate grant API.

### Admin user payloads should include

- `scope_mode`
- `app_scope_ids`
- `app_scope_apps` (optional display payload for UI convenience)

### Create admin request should support

- `role`
- `scope_mode`
- `app_scope_ids`

### Update admin request should support

- `role`
- `scope_mode`
- `app_scope_ids`

## Backend Behavior

### Reads

For app-scoped admins:

- app listings must be filtered
- bot listings must be filtered
- audit visibility should continue following the current audit rules for now

### Writes

For app-scoped admins:

- app update/delete/token-reset must reject out-of-scope targets
- bot start/stop/delete/upgrade must reject out-of-scope targets

## Audit Requirements

Admin grant changes must be audited.

At minimum, audit should capture:

- actor
- target admin
- role before/after
- scope mode before/after
- app scope IDs before/after

Suggested action names:

- `admin_membership.create`
- `admin_membership.update`
- `admin_membership.delete`

The current `admin.create` and `admin.update` events may keep working if they are extended to include the same membership delta clearly.

## Validation Rules

The system should reject:

- assigning `platform_admin` with app-scoped memberships
- saving a non-platform app-scoped grant with zero selected apps
- saving memberships with mixed roles for one admin
- saving app scope IDs that do not exist

## Migration Strategy

### Phase A: Backend support

- extend admin membership resolution to collect allowed app IDs
- make app and bot list endpoints respect allowed app IDs
- make app and bot mutation endpoints reject out-of-scope targets

### Phase B: Admin management support

- extend admin create/update handlers to manage platform vs selected-app scope
- extend admin payloads with resolved scope information

### Phase C: UI support

- update the `Admins` page to let platform admins choose scope mode and app selections

## Risks

### Risk 1: Existing admins with multiple memberships become ambiguous

Mitigation:

- enforce one effective role per admin in this slice
- add tests for mixed-membership rejection

### Risk 2: Read filtering and write enforcement diverge

Mitigation:

- test list filtering separately from mutation rejection
- keep app scope resolution in one reusable model/service path

### Risk 3: UI makes scope look optional when it is required

Mitigation:

- disable invalid save states
- return explicit API validation errors

## Acceptance Criteria

App Scope Authorization v1 is complete when:

- non-platform admins can be granted platform-wide or selected-app access
- app-scoped admins only see granted apps in the admin UI
- app-scoped admins only see bots under granted apps
- out-of-scope app and bot mutations are rejected
- admin management UI can assign selected apps
- scope changes are clearly audited

