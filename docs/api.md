# ProductionFlow API

The contract between the Go backend and the Flutter client. **This file changes
before the code.** The client is built against mocks derived from it
(`client/lib/core/mock/`), so a breaking change here blocks someone — say so in
the PR.

Base URL: `http://localhost:8080` in development.

## Conventions

- JSON keys are `snake_case`.
- Instants are RFC 3339 in UTC: `2026-10-04T03:00:00Z`. Display conversion to
  `Asia/Almaty` happens in the client.
- A call time is wall-clock, so a shooting day carries `shoot_date` (`2026-10-04`)
  and `call_time` (`07:30`) as separate fields — never one timestamp.
- Auth: `Authorization: Bearer <jwt>`. Access token only, no refresh in the MVP.
- List endpoints use cursor pagination: `?cursor=<opaque>&limit=<1..100>`,
  never offset. Responses carry `next_cursor` (`null` on the last page).
- Errors are always:

  ```json
  { "error": { "code": "day_not_found", "message": "no shooting day with that id" } }
  ```

  Switch on `code`; `message` is for humans. Current codes: `not_implemented`,
  `not_found`, `internal`.

## Status

| Endpoint | Status |
| --- | --- |
| `GET /api/health` | implemented |
| `GET /api/projects` | `501` |
| `POST /api/projects` | `501` |
| `GET /api/days` | `501` |
| `POST /api/days` | `501` |
| `POST /api/changes/preview` | `501` |
| `POST /api/changes/apply` | `501` |

---

## GET /api/health

No auth. Build info plus a database ping. `200` when the database answers,
`503` with `"status": "degraded"` when it does not — the body shape is identical
either way.

```json
{
  "status": "ok",
  "version": "dev",
  "commit": "f816519",
  "built_at": "2026-09-30T07:15:02Z",
  "go_version": "go1.26.1",
  "database": { "status": "up", "latency_ms": 1.42 }
}
```

## GET /api/projects

Projects the caller participates in.

```json
{
  "data": [
    { "id": "…uuid…", "name": "Dead Air", "created_at": "2026-09-01T09:00:00Z" }
  ],
  "next_cursor": null
}
```

## POST /api/projects

```json
{ "name": "Dead Air" }
```

`201` with the created project.

## GET /api/days

`?project_id=<uuid>` required. Ordered by `shoot_date` ascending.

```json
{
  "data": [
    {
      "id": "…uuid…",
      "project_id": "…uuid…",
      "shoot_date": "2026-10-04",
      "call_time": "07:30",
      "location": "Kok-Tobe, upper lot",
      "scene_count": 3
    }
  ],
  "next_cursor": null
}
```

## POST /api/days

```json
{
  "project_id": "…uuid…",
  "shoot_date": "2026-10-04",
  "call_time": "07:30",
  "location": "Kok-Tobe, upper lot"
}
```

`201` with the created day.

---

## The impact view

Two steps, never a single write. This is the mechanic the product exists for.

### POST /api/changes/preview

Describes an intended change. **Writes nothing.** Returns what it would affect
plus a `preview_token` that `apply` must echo back.

```json
{
  "kind": "move_shooting_day",
  "shooting_day_id": "…uuid…",
  "to": { "shoot_date": "2026-10-06", "call_time": "08:00" }
}
```

```json
{
  "preview_token": "…opaque…",
  "expires_at": "2026-09-30T07:20:00Z",
  "change": {
    "kind": "move_shooting_day",
    "diff": {
      "shoot_date": { "from": "2026-10-04", "to": "2026-10-06" },
      "call_time":  { "from": "07:30",      "to": "08:00" }
    }
  },
  "affected": {
    "scenes": [
      { "id": "…uuid…", "number": "12A", "description": "Rooftop argument" }
    ],
    "participants": [
      { "id": "…uuid…", "display_name": "Aisha", "role": "cast", "reason": "scene_12A" }
    ],
    "materials": [],
    "tasks": []
  }
}
```

Affected participants = participants on any scene of that day ∪ participants
assigned directly to the day. Affected scenes = scenes attached to the day.

### POST /api/changes/apply

The same body plus `preview_token`. Applies the change in one transaction,
writes a `change_event`, and queues notifications for the affected participants
only.

```json
{
  "kind": "move_shooting_day",
  "shooting_day_id": "…uuid…",
  "to": { "shoot_date": "2026-10-06", "call_time": "08:00" },
  "preview_token": "…opaque…"
}
```

`200` with the updated entity and the written `change_event` id. A stale or
mismatched token is `409 preview_stale` — the client must re-preview, because
the impact may have changed underneath it.

---

## Open questions

- **Day-level participant assignment.** The impact rule unions "participants on
  any scene of that day" with "participants assigned directly to the day", but
  the five-table domain model has no place to record the second half —
  `scene_participant` only links people to scenes. Recording it needs a new
  table (`shooting_day_participant`), which is a decision to make explicitly
  rather than in passing. Until then only the scene-derived half is computable.
- **`preview_token`**: signed payload (stateless) or a stored row with an
  expiry? Stateless is less machinery; a stored row makes `preview_stale`
  detection exact.
