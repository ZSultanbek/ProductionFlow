# ProductionFlow API

The contract between the Go backend and the Flutter client. **This file changes
before the code.** The client is built against mocks derived from it
(`client/lib/core/mock/`), so a breaking change here blocks someone — say so in
the PR.

The reasoning behind the shapes below is in [`architecture.md`](architecture.md).

| Environment | Base URL |
| --- | --- |
| Local, through the web container | same origin (`/api/...`) |
| Local, API directly | `http://localhost:8080` (`http://10.0.2.2:8080` from the Android emulator) |
| Production | `https://productionflow-api.onrender.com` (placeholder until created) |

---

## Conventions

- JSON keys are `snake_case`. Request bodies are `application/json`.
- IDs are UUID strings. A path ID that is not a valid UUID is treated as one that
  does not exist: `404`, not `422`.
- Instants are RFC 3339 in UTC: `2026-10-04T03:00:00Z`.
- **A call time is wall-clock.** `shoot_date` is `YYYY-MM-DD`, `call_time` is
  `HH:MM` (24 h). They are never combined, never time-zone converted — on the
  server or in the client.
- **Auth:** `Authorization: Bearer <token>` on every endpoint except those marked
  *public*. Missing, malformed or expired → `401 unauthorized`.
- **Lists** use cursor pagination: `?cursor=<opaque>&limit=<1..100>` (default 50).
  The response is always `{"data": [...], "next_cursor": "…" | null}`. Treat the
  cursor as an opaque string — do not parse it, do not build it.
- **Errors** are always:

  ```json
  { "error": { "code": "day_not_found", "message": "no shooting day with that id" } }
  ```

  Validation errors add a `fields` object, one human message per bad field:

  ```json
  {
    "error": {
      "code": "validation_failed",
      "message": "some fields are invalid",
      "fields": { "call_time": "must be HH:MM, 24-hour" }
    }
  }
  ```

  Switch on `code`; `message` is for humans and may change.
- **Membership is private.** If you are not a participant of a project, its
  endpoints answer `404 project_not_found` (or `day_not_found`,
  `scene_not_found` for nested resources) — never `403`. `403 forbidden` means
  "you are in this project, but your role cannot do this".

### Error codes

| Code | Status | When |
| --- | --- | --- |
| `validation_failed` | 422 | Body or query fails the rules in the endpoint's section; see `fields` |
| `no_change` | 422 | A change preview whose `to` equals the current values |
| `unauthorized` | 401 | No token, bad token, expired token |
| `invalid_credentials` | 401 | Login with an unknown email **or** a wrong password — deliberately the same |
| `forbidden` | 403 | Member of the project, but the role is not allowed (only producers write) |
| `not_found` | 404 | No such route |
| `project_not_found` | 404 | No such project, or you are not in it |
| `day_not_found` | 404 | No such shooting day, or not in your project |
| `scene_not_found` | 404 | No such scene, or not in your project |
| `invite_not_found` | 404 | No project with that invite code |
| `notification_not_found` | 404 | No such notification for you |
| `email_taken` | 409 | Register with an email that already has an account |
| `participant_email_taken` | 409 | Adding a participant whose email is already in the project, or joining when your email belongs to someone else there |
| `scene_number_taken` | 409 | Scene number already used in this project |
| `preview_stale` | 409 | Apply after the day or its affected people changed — preview again |
| `preview_expired` | 409 | Apply with a token older than 10 minutes — preview again |
| `not_implemented` | 501 | Contracted here, not built yet |
| `internal` | 500 | A bug. The server logs the details; the client gets nothing useful |

### Field rules

Used by every endpoint that accepts these fields.

| Field | Rule |
| --- | --- |
| `email` | Trimmed, contains `@`, at most 254 characters. Compared case-insensitively |
| `password` | 8 to 72 bytes |
| `display_name` | Trimmed, 1 to 80 characters |
| `name` (project) | Trimmed, 1 to 120 characters |
| `role` (when joining or adding) | `crew` or `cast`. `producer` is never accepted from a client |
| `shoot_date` | `YYYY-MM-DD`, a real calendar date. Past dates are allowed |
| `call_time` | `HH:MM`, `00:00`–`23:59` |
| `location` | Trimmed, 0 to 200 characters |
| `number` (scene) | Trimmed, 1 to 20 characters — `12A`, `7pt2` are fine |
| `description` (scene) | 0 to 500 characters |

---

## Status

Updated in the same PR that implements an endpoint.

| Endpoint | Story | Status |
| --- | --- | --- |
| `GET /api/health` | — | implemented |
| `POST /api/auth/register` | US-01 | implemented |
| `POST /api/auth/login` | US-01 | implemented |
| `GET /api/me` | US-01 | implemented |
| `GET /api/projects` | US-02 | implemented |
| `POST /api/projects` | US-02 | implemented |
| `GET /api/projects/{project_id}` | US-02 | implemented |
| `GET /api/invites/{code}` | US-03 | implemented |
| `POST /api/invites/{code}/join` | US-03 | implemented |
| `GET /api/projects/{project_id}/participants` | US-06 | `501` |
| `POST /api/projects/{project_id}/participants` | US-06 | `501` |
| `GET /api/projects/{project_id}/days` | US-04 | implemented |
| `POST /api/projects/{project_id}/days` | US-04 | implemented |
| `GET /api/days/{day_id}` | US-05 | implemented |
| `PUT /api/days/{day_id}/participants` | US-06 | `501` |
| `GET /api/projects/{project_id}/scenes` | US-06 | `501` |
| `POST /api/projects/{project_id}/scenes` | US-06 | `501` |
| `PATCH /api/scenes/{scene_id}` | US-06 | `501` |
| `PUT /api/scenes/{scene_id}/participants` | US-06 | `501` |
| `POST /api/changes/preview` | US-07 | `501` |
| `POST /api/changes/apply` | US-08 | `501` |
| `GET /api/projects/{project_id}/changes` | US-08 | `501` |
| `GET /api/me/today` | US-05 | implemented |
| `GET /api/me/schedule` | US-10 | `501` |
| `GET /api/me/notifications` | US-09 | `501` |
| `POST /api/me/notifications/{notification_id}/read` | US-09 | `501` |

---

## Shared shapes

Defined once here; endpoints below refer to them by name. Where a JSON example
contains a bare shape name such as `ChangeEvent` or `[ AgendaDay ]`, it means
"an object (or list) of that shape" — it is not literal JSON.

**User**

```json
{
  "id": "…uuid…",
  "email": "madina@example.kz",
  "display_name": "Madina",
  "created_at": "2026-11-02T09:00:00Z"
}
```

**Project** — `invite_code` and `invite_url` are `null` unless `my_role` is `producer`.

```json
{
  "id": "…uuid…",
  "name": "Dead Air",
  "my_role": "producer",
  "my_participant_id": "…uuid…",
  "invite_code": "3f9a0c2e7b1d4e8fa6c5b2d1e0f9a8b7",
  "invite_url": "https://productionflow.pages.dev/join/3f9a0c2e7b1d4e8fa6c5b2d1e0f9a8b7",
  "created_at": "2026-09-01T09:00:00Z"
}
```

**Participant** — `has_account` is `false` for people the producer added by name
who never registered (they appear in impact lists but get no notifications).

```json
{
  "id": "…uuid…",
  "project_id": "…uuid…",
  "display_name": "Aisha",
  "email": "aisha@example.kz",
  "role": "cast",
  "has_account": true
}
```

**ParticipantRef** — the short form used inside other objects.

```json
{ "id": "…uuid…", "display_name": "Aisha", "role": "cast" }
```

**ShootingDay**

```json
{
  "id": "…uuid…",
  "project_id": "…uuid…",
  "shoot_date": "2026-10-04",
  "call_time": "07:30",
  "location": "Kok-Tobe, upper lot",
  "scene_count": 3,
  "updated_at": "2026-10-01T12:00:00Z"
}
```

**Scene** — `shooting_day_id` is `null` for an unscheduled scene.

```json
{
  "id": "…uuid…",
  "project_id": "…uuid…",
  "shooting_day_id": "…uuid…",
  "number": "12A",
  "description": "Rooftop argument, dusk",
  "participants": [ { "id": "…uuid…", "display_name": "Aisha", "role": "cast" } ]
}
```

**ChangeEvent** — `actor` is `null` if the person was later removed from the
project; the event stays. `diff` holds only the fields that changed.

```json
{
  "id": "…uuid…",
  "project_id": "…uuid…",
  "actor": { "participant_id": "…uuid…", "display_name": "Madina" },
  "entity_type": "shooting_day",
  "entity_id": "…uuid…",
  "diff": {
    "shoot_date": { "from": "2026-10-04", "to": "2026-10-06" },
    "call_time":  { "from": "07:30",      "to": "08:00" }
  },
  "created_at": "2026-10-01T12:03:11Z"
}
```

`entity_type` is `shooting_day` (an applied change) or `scene` (a scene edit via
`PATCH /api/scenes/{scene_id}`). Linking people to scenes or days does not write
events in the MVP.

**AgendaDay** — a day as seen by one person, used by `/me/today` and `/me/schedule`.

```json
{
  "id": "…uuid…",
  "project": { "id": "…uuid…", "name": "Dead Air" },
  "shoot_date": "2026-11-02",
  "call_time": "07:30",
  "location": "Kok-Tobe, upper lot",
  "my_scenes": [ { "id": "…uuid…", "number": "12A", "description": "Rooftop argument, dusk" } ],
  "i_am_on_set_all_day": false
}
```

---

## Health

### GET /api/health

*Public.* Build info plus a database ping. `200` when the database answers,
`503` with `"status": "degraded"` when it does not — the body shape is identical
either way. Also the endpoint to open before a demo, to wake the server.

```json
{
  "status": "ok",
  "version": "dev",
  "commit": "f816519",
  "built_at": "2026-09-30T07:15:02Z",
  "go_version": "go1.25.1",
  "database": { "status": "up", "latency_ms": 1.42 }
}
```

---

## Auth — US-01

### POST /api/auth/register

*Public.*

```json
{ "email": "madina@example.kz", "password": "correct horse", "display_name": "Madina" }
```

`201`:

```json
{
  "token": "eyJhbGciOi…",
  "expires_at": "2026-11-09T09:00:00Z",
  "user": { "id": "…uuid…", "email": "madina@example.kz", "display_name": "Madina", "created_at": "2026-11-02T09:00:00Z" }
}
```

Errors: `422 validation_failed`, `409 email_taken`.

### POST /api/auth/login

*Public.*

```json
{ "email": "madina@example.kz", "password": "correct horse" }
```

`200` with the same body as register. Errors: `422 validation_failed`,
`401 invalid_credentials`.

The token lives 7 days. There is no refresh: on any `401` the client forgets the
token and shows the login screen.

### GET /api/me

`200`: `{ "user": User }`. Used by the client at startup to check that a stored
token is still good.

---

## Projects — US-02

### GET /api/projects

Projects the caller is a participant of, newest first. `200`:
`{ "data": [Project], "next_cursor": null }`.

### POST /api/projects

```json
{ "name": "Dead Air" }
```

`201`: the new **Project**. The caller becomes its `producer` — a participant row
is created with their account's `display_name` and `email`. Errors:
`422 validation_failed`.

### GET /api/projects/{project_id}

`200`: **Project**. Errors: `404 project_not_found`.

---

## Invites — US-03

One reusable invite code per project, shown to producers on the **Project**
object. Anyone holding the link can join, any number of times; there is no
expiry and no revoking in the MVP (grooming decision).

### GET /api/invites/{code}

*Public*, so the join page can say what you are joining before you log in.

`200`:

```json
{ "project_id": "…uuid…", "project_name": "Dead Air" }
```

Errors: `404 invite_not_found`.

### POST /api/invites/{code}/join

```json
{ "role": "crew" }
```

- Not yet in the project → a participant is created with the chosen role, the
  account's `display_name` and `email`. `201`: **Participant**.
- Already in the project → nothing changes, `role` in the body is ignored.
  `200`: the existing **Participant**. Safe to call twice.
- The producer already added you by name **with your email**, without an account
  → that participant is linked to your account instead of creating a duplicate,
  and keeps the role the producer gave it. `200`: **Participant**.

Errors: `404 invite_not_found`, `422 validation_failed` (role missing or
`producer`), `409 participant_email_taken` (someone else in the project is
already linked to an account with your email — should not happen while emails
are unique per account, but the server refuses rather than guesses).

---

## Participants — US-06

### GET /api/projects/{project_id}/participants

Any member. Ordered by role (producer, crew, cast), then name.
`200`: `{ "data": [Participant], "next_cursor": null }`.

### POST /api/projects/{project_id}/participants

*Producer only.* Adds a person who may never install the app — they still count
in impact previews.

```json
{ "display_name": "Ruslan", "role": "crew", "email": null }
```

`201`: **Participant** with `has_account: false`. If they register later with the
same email and follow the invite link, they are linked to this row.

Errors: `422 validation_failed`, `403 forbidden`, `409 participant_email_taken`.

---

## Shooting days — US-04, US-05

The date, call time and location of an existing day **cannot be edited here**.
They change only through `/api/changes/apply`, so every such change goes through
the impact preview.

### GET /api/projects/{project_id}/days

Any member. Ordered by `shoot_date`, then `call_time`. Optional
`?from=YYYY-MM-DD` to skip past days.

`200`: `{ "data": [ShootingDay], "next_cursor": null }`.

Errors: `422 validation_failed` (`from`, `limit` or `cursor`), `404 project_not_found`.

### POST /api/projects/{project_id}/days

*Producer only.*

```json
{ "shoot_date": "2026-10-04", "call_time": "07:30", "location": "Kok-Tobe, upper lot" }
```

`201`: **ShootingDay** with `scene_count: 0`. Two days on the same date are
allowed — a production can run two units.

Errors: `422 validation_failed`, `403 forbidden`, `404 project_not_found`.
Permission is checked before the body: a crew member gets `403` whatever they
send.

### GET /api/days/{day_id}

The call sheet. Any member.

`200`:

```json
{
  "id": "…uuid…",
  "project": { "id": "…uuid…", "name": "Dead Air", "my_role": "crew" },
  "shoot_date": "2026-10-04",
  "call_time": "07:30",
  "location": "Kok-Tobe, upper lot",
  "updated_at": "2026-10-01T12:00:00Z",
  "on_set_all_day": [
    { "id": "…uuid…", "display_name": "Daniyar", "role": "crew" }
  ],
  "scenes": [
    {
      "id": "…uuid…",
      "number": "12A",
      "description": "Rooftop argument, dusk",
      "participants": [ { "id": "…uuid…", "display_name": "Aisha", "role": "cast" } ]
    }
  ]
}
```

Until the US-06 endpoints exist there is no way to add scenes or put people on
set through the API, so `scenes` and `on_set_all_day` are `[]` for every day
created so far. They are never `null`.

Errors: `404 day_not_found`.

### PUT /api/days/{day_id}/participants

*Producer only.* Replaces the set of people on set for the whole day (not tied to
a scene — the gaffer, the sound recordist). Send the full list; an empty list
clears it.

```json
{ "participant_ids": [ "…uuid…", "…uuid…" ] }
```

`200`: `{ "on_set_all_day": [ParticipantRef] }`.

Errors: `422 validation_failed` (an id that is not a participant of this
project — `fields.participant_ids` says which), `403 forbidden`,
`404 day_not_found`.

---

## Scenes — US-06

### GET /api/projects/{project_id}/scenes

Any member. Filters, both optional: `?shooting_day_id=<uuid>` or
`?unscheduled=true`. Ordered by `number` as text (`12A` sorts before `2`; the
client may sort naturally for display).

`200`: `{ "data": [Scene], "next_cursor": null }`.

### POST /api/projects/{project_id}/scenes

*Producer only.*

```json
{ "number": "14", "description": "Kitchen, morning", "shooting_day_id": null }
```

`201`: **Scene** with an empty `participants` list.

Errors: `422 validation_failed` (including a `shooting_day_id` from another
project), `403 forbidden`, `409 scene_number_taken`.

### PATCH /api/scenes/{scene_id}

*Producer only.* Send only the fields to change. `shooting_day_id: null`
unschedules the scene. `number` cannot be changed.

```json
{ "shooting_day_id": "…uuid…", "description": "Kitchen, morning — rewritten" }
```

`200`: **Scene**. Writes one **ChangeEvent** (`entity_type: "scene"`) with the
fields that actually changed; no event if nothing did. Moving a scene does **not**
go through the impact preview and sends no notifications (grooming decision).

Errors: `422 validation_failed`, `403 forbidden`, `404 scene_not_found`.

### PUT /api/scenes/{scene_id}/participants

*Producer only.* Replaces the set of people needed in this scene.

```json
{ "participant_ids": [ "…uuid…", "…uuid…" ] }
```

`200`: **Scene**.

Errors: `422 validation_failed`, `403 forbidden`, `404 scene_not_found`.

---

## The impact view — US-07, US-08

Two steps, never a single write. This is the mechanic the product exists for.

In the MVP there is exactly one kind of change, `move_shooting_day`: a new date,
call time and/or location for an existing day. The preview lists affected
**scenes and participants only** — materials, tasks and equipment are out of
scope (grooming decision).

*Producer only*, both steps.

### POST /api/changes/preview

**Writes nothing.**

```json
{
  "kind": "move_shooting_day",
  "shooting_day_id": "…uuid…",
  "to": { "shoot_date": "2026-10-06", "call_time": "08:00" }
}
```

`to` must contain at least one of `shoot_date`, `call_time`, `location`. Fields
left out stay as they are.

`200`:

```json
{
  "preview_token": "…opaque…",
  "expires_at": "2026-10-01T12:10:00Z",
  "change": {
    "kind": "move_shooting_day",
    "shooting_day_id": "…uuid…",
    "diff": {
      "shoot_date": { "from": "2026-10-04", "to": "2026-10-06" },
      "call_time":  { "from": "07:30",      "to": "08:00" }
    }
  },
  "affected": {
    "scenes": [
      { "id": "…uuid…", "number": "12A", "description": "Rooftop argument, dusk" },
      { "id": "…uuid…", "number": "13",  "description": "Insert: the note on the table" }
    ],
    "participants": [
      {
        "id": "…uuid…", "display_name": "Daniyar", "role": "crew", "has_account": true,
        "via": { "day": true, "scene_numbers": [] }
      },
      {
        "id": "…uuid…", "display_name": "Aisha", "role": "cast", "has_account": true,
        "via": { "day": false, "scene_numbers": ["12A", "13"] }
      },
      {
        "id": "…uuid…", "display_name": "Ruslan", "role": "crew", "has_account": false,
        "via": { "day": false, "scene_numbers": ["13"] }
      }
    ]
  },
  "will_notify_count": 2
}
```

How `affected` is computed:

- **Scenes**: every scene scheduled on this day. Unscheduled scenes never count.
- **Participants**: everyone in any of those scenes, plus everyone on set for the
  whole day. **Each person appears once**; `via` lists every reason.
- Ordered: scenes by `number`; participants by role (producer, crew, cast), then
  `display_name`.
- `will_notify_count` = affected participants with an account, minus the
  producer making the change.
- A day with nobody attached returns empty lists and can still be applied.

Errors: `422 validation_failed`, `422 no_change` (every field in `to` equals the
current value), `403 forbidden`, `404 day_not_found`.

### POST /api/changes/apply

The same body plus the token from the preview:

```json
{
  "kind": "move_shooting_day",
  "shooting_day_id": "…uuid…",
  "to": { "shoot_date": "2026-10-06", "call_time": "08:00" },
  "preview_token": "…opaque…"
}
```

In one transaction: re-checks that the day and its affected people are exactly
what the preview showed, updates the day, writes one **ChangeEvent**, and creates
a notification for each affected participant with an account (except the actor).

`200`:

```json
{
  "shooting_day": { "id": "…uuid…", "project_id": "…uuid…", "shoot_date": "2026-10-06", "call_time": "08:00", "location": "Kok-Tobe, upper lot", "scene_count": 2, "updated_at": "2026-10-01T12:03:11Z" },
  "change_event": ChangeEvent,
  "notified_count": 2
}
```

Errors:

- `409 preview_stale` — the request does not match the token, **or** the day was
  changed by someone else, **or** the set of affected scenes or people changed
  since the preview (for example, someone was added to a scene on that day).
  Nothing was written. The client should preview again and show the new list.
- `409 preview_expired` — the token is older than 10 minutes. Preview again.
- `422 validation_failed`, `403 forbidden`, `404 day_not_found`.

### GET /api/projects/{project_id}/changes

The change feed. Any member. Newest first.

`200`: `{ "data": [ChangeEvent], "next_cursor": "…" }`.

---

## Me — US-05, US-09, US-10

Everything about the caller across all their projects.

### GET /api/me/today?date=YYYY-MM-DD

`date` is **required** and is the phone's local date — the server never guesses
the caller's time zone.

Returns every day on `date` in any of the caller's projects. If there are none,
it returns all days on the **next** date that has any; if there is nothing
upcoming at all, `date` is `null` and `days` is empty.

`200`:

```json
{
  "date": "2026-11-02",
  "is_today": true,
  "days": [ AgendaDay ]
}
```

`days` includes every day of the caller's projects on that date, even ones where
`my_scenes` is empty and `i_am_on_set_all_day` is false — a crew member still wants to
see that their project is shooting today. Ordered by `call_time`.

Errors: `422 validation_failed` (missing or invalid `date`).

### GET /api/me/schedule?from=YYYY-MM-DD

Only the days the caller is **personally** attached to — through a scene or on
set for the whole day. `from` is required (the phone's local date). Ordered by
`shoot_date`, then `call_time`.

`200`: `{ "data": [AgendaDay], "next_cursor": null }`.

### GET /api/me/notifications

Optional `?unread_only=true`. Newest first.

`200`:

```json
{
  "data": [
    {
      "id": "…uuid…",
      "project": { "id": "…uuid…", "name": "Dead Air" },
      "change_event": ChangeEvent,
      "created_at": "2026-10-01T12:03:11Z",
      "read_at": null
    }
  ],
  "next_cursor": null,
  "unread_count": 1
}
```

### POST /api/me/notifications/{notification_id}/read

`204`, no body. Marking an already-read notification is fine and changes nothing.

Errors: `404 notification_not_found`.

---

## Changes to this contract

### 2 November 2026 — full MVP contract

Breaking for anyone building against the previous version (the client mocks):

- Routes are nested under projects: `GET /api/days?project_id=…` is now
  `GET /api/projects/{project_id}/days`; same for `POST`.
- Impact preview: each participant's `reason` string is replaced by
  `via: { "day": bool, "scene_numbers": [...] }`, so a person in two scenes is
  listed once with both reasons. `affected.materials` and `affected.tasks` are
  removed. `change.shooting_day_id`, `has_account` and `will_notify_count` are new.
- **ShootingDay** gains `updated_at`. **Participant** gains `has_account`.
  **Project** gains `my_role`, `my_participant_id`, `invite_code`, `invite_url`.

The fixtures in `client/lib/core/mock/fixtures/` were brought in line with this
on 16 November 2026, together with the first client screens.

Resolved open questions from the previous version:

- **Day-level participants** → new table `shooting_day_participant` and
  `PUT /api/days/{day_id}/participants`.
- **`preview_token`** → signed and stateless, re-verified inside the apply
  transaction. See architecture.md §4.4.
