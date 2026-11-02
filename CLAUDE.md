# CLAUDE.md — ProductionFlow

Guidance for Claude Code when working in this repository.

## What this is

A coordination tool for small film crews (5–15 people). A film crew currently
runs a shoot through Telegram + Google Sheets + Drive, so schedule changes get
lost, people work from outdated script versions, and when a shooting day moves
nobody can say which scenes, people and materials are affected.

**The one mechanic that defines this product:** everything is attached to a
shooting day. When someone changes a day or a scene, the system shows *who and
what this affects* before the change is confirmed, then notifies only those
people. Every feature either feeds that mechanic or is out of scope.

This is a university course project (KBTU, IT Project Management, Fall 2026).
Team of three: Zhanibek (backend/DevOps), Almas (Flutter client), Zhaksylyk (PM).
Both developers are learning parts of this stack as they go — **favour obvious
code over clever code**.

## Stack

- **Backend:** Go 1.23+, standard `net/http` with `http.ServeMux` routing (no
  web framework), `pgx/v5` for Postgres (no ORM), `golang-migrate` for schema.
- **DB:** PostgreSQL 16.
- **Client:** Flutter (Dart 3), one codebase → Android APK + web build.
  Riverpod for state, `dio` for HTTP, `json_serializable` for models.
- **Deploy:** Docker Compose locally; free-tier hosting for the demo.
- **iOS is out of scope.** Do not add iOS-specific code or dependencies.

## Layout

```
backend/
  cmd/api/            main.go — wiring only
  internal/
    domain/           entities + business rules (no SQL, no HTTP)
    store/            pgx queries, one file per aggregate
    http/             handlers, middleware, request/response DTOs
    auth/             JWT, password hashing
  migrations/         golang-migrate .up.sql / .down.sql pairs
client/
  lib/
    features/         one folder per feature (shooting_day/, change_feed/, ...)
    core/             api client, models, theme, routing
    mock/             fixture data for offline development
docs/
  api.md              the API contract — source of truth, update it first
  architecture.md     how it all fits: layers, schema, auth, impact flow, hosting
```

Read `docs/architecture.md` before starting a feature. Its decisions table (§8)
records why things are the way they are.

## Domain model

Five tables carry the whole product:

- `project` — one film production.
- `shooting_day` — date, call time, location, project_id. **The central entity.**
- `scene` — number, description, shooting_day_id (nullable — unscheduled scenes exist).
- `participant` — a person in a project with a role (`producer`, `crew`, `cast`).
- `scene_participant` — who is needed in which scene. This join table is what
  makes the impact view possible.

Plus:

- `change_event` — append-only log: actor, entity type, entity id, field diffs,
  created_at. Never updated, never deleted.
- `material` / `material_version` — script and references; the current version is
  a pointer on `material`, old versions are kept.
- `task` — assignee, due date, optional shooting_day_id.

Added for the MVP (see `docs/architecture.md` §3):

- `app_user` — an account. Separate from `participant`: one person is in many
  projects, and a participant may have no account at all (cast added by name).
- `shooting_day_participant` — people on set for the whole day, not tied to a
  scene. The "assigned directly to the day" half of the impact rule.
- `notification` — in-app notice to one participant about one change event.

`material`, `material_version` and `task` are **not** in the MVP schema.

**Rules:**

- All IDs are UUID v4.
- All timestamps stored UTC (`timestamptz`). The crew is in Almaty — convert to
  `Asia/Almaty` for display only, never in the database.
- Call times are wall-clock times on a date, not instants. Store as `date` +
  `time`, not a single timestamp.
- A shooting day's date, call time and location change **only** through
  `/api/changes/apply`. Never add a PATCH for them — it would bypass the preview.

## The impact view — how it works

This is the part to get right. It is a two-step flow, never a single write:

1. `POST /api/changes/preview` — body describes the intended change (e.g. move
   shooting day X to a new date). Returns `affected`: scenes, participants,
   materials, tasks. Writes nothing.
2. `POST /api/changes/apply` — same body plus the preview token. Applies the
   change in a transaction, writes a `change_event`, queues notifications for
   the affected participants only.

Affected participants = participants on any scene of that day ∪ participants
assigned directly to the day. Affected scenes = scenes attached to the day.
Keep this computation in `internal/domain` as a pure function over loaded data,
so it can be unit-tested without a database. **Do not put it in a SQL query and
do not put it in a handler.**

## API conventions

- `docs/api.md` is the contract and is updated *before* the code. The Flutter
  client is developed against mocks derived from it, so a breaking change there
  blocks another person — say so in the PR.
- JSON keys `snake_case`. Timestamps RFC 3339 UTC.
- Errors: `{"error": {"code": "day_not_found", "message": "..."}}` — machine
  code plus human message. Never return raw Go errors.
- Auth: `Authorization: Bearer <jwt>`. Access token only; no refresh flow in MVP.
- List endpoints: cursor pagination (`?cursor=&limit=`), never offset.

## Scope — do not build these

The MVP boundary is a graded course decision. Do not add, and push back if asked:

- Budgeting, invoicing, payments.
- A crew marketplace or public profiles.
- AI script breakdown or any LLM feature.
- Video/media hosting. Materials are documents and images, ≤10 MB.
- Real-time collaborative editing. The change feed is append-only, not a CRDT.
- iOS builds, desktop builds, browser extensions.

If a change requires a new table, say so explicitly before writing it.

## Working style

- **Small PRs.** One feature or one fix. `main` is protected; branches are
  `feat/`, `fix/`, `chore/` + short slug.
- **Tests where the logic is.** The impact computation, the change-event diff
  and auth get unit tests. Handlers get one happy-path and one error test.
  Don't chase coverage on glue code.
- Backend: `go test ./...` must pass before a PR. `go vet` clean.
- Client: `flutter analyze` clean. Widget tests for screens that have state logic.
- Migrations are forward-only in practice — never edit a migration that has run;
  add a new one.
- Commit messages: imperative, one line, mention the area (`backend: add impact
  preview endpoint`).

## When helping in this repo

- Prefer explicit code over abstraction. If a helper is used once, inline it.
- No new dependencies without saying why; the team has to be able to read them.
- When a task touches the API contract, update `docs/api.md` in the same change.
- When something is ambiguous about the domain (how a crew actually works), ask
  rather than invent — two people on this team have produced student films and
  can answer.
- Keep explanations short. The team is learning; comments explaining *why*
  beat comments explaining *what*.
