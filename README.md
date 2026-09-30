# ProductionFlow

Shoot coordination for small film crews (5–15 people).

Everything attaches to a **shooting day**. When someone changes a day or a scene,
ProductionFlow shows *who and what this affects* before the change is confirmed,
then notifies only those people. That mechanic is the product; see
[CLAUDE.md](CLAUDE.md) for the scope boundary around it.

KBTU · IT Project Management · Fall 2026.
Zhanibek (backend/DevOps), Almas (Flutter client), Zhaksylyk (PM).

## Repository

| Path | What |
| --- | --- |
| [`backend/`](backend/README.md) | Go API, Postgres 16, migrations |
| [`client/`](client/README.md) | Flutter app — Android + web |
| [`docker-compose.yml`](docker-compose.yml) | The whole stack. The only compose file in the repo. |
| [`docs/api.md`](docs/api.md) | The API contract. **Changes before the code.** |
| [`CLAUDE.md`](CLAUDE.md) | Domain model, conventions, scope, working style |

## Status

Skeleton. Both halves run; no features are implemented.

- `GET /api/health` works (build info + database ping). Every other endpoint
  returns `501` in the standard error envelope.
- The client has a landing page and an app shell with navigation. It runs with no
  backend, reading nothing — fixtures for that live in `client/lib/core/mock/`.
- Migration `0001_init` creates the six core tables: `project`, `shooting_day`,
  `scene`, `participant`, `scene_participant`, `change_event`.

## Run the whole thing

One command, from the repo root:

```sh
docker compose up --build
```

Then open **http://localhost:3000**.

That starts four things in order: Postgres, a one-shot migration job, the Go API,
and nginx serving the Flutter web build. The API waits for the migrations to exit
successfully, so there is no second step and no race on an unmigrated database.

| Service | Where | Notes |
| --- | --- | --- |
| `web` | http://localhost:3000 | nginx + the Flutter web bundle |
| `api` | http://localhost:8080 | exposed for `curl`; the browser does not use this port |
| `postgres` | `localhost:5432` | `productionflow` / `productionflow` |
| `migrate` | — | runs `migrations/*.up.sql`, then exits 0 |

The browser talks to `/api/...` on port 3000 and nginx proxies it to the API, so
the two are same-origin and there is no CORS configuration anywhere.

`docker compose up` will print `productionflow-migrate-1 exited with code 0`
partway through. That is the migration job finishing, not a failure.

```sh
curl -s localhost:3000/api/health | jq   # through nginx
curl -s localhost:8080/api/health | jq   # straight at the API
docker compose down                      # stop; add -v to drop the database
```

Port already taken? `POSTGRES_PORT` and `WEB_PORT` override the host side:

```sh
POSTGRES_PORT=5433 WEB_PORT=3001 docker compose up --build
```

Compose cannot run `git` to fill in build info, so `/api/health` reports
`version: dev` unless you pass it — worth doing for anything you hand to someone:

```sh
VERSION=$(git describe --tags --always --dirty) \
COMMIT=$(git rev-parse --short HEAD) \
BUILT_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
docker compose up --build
```

## Develop on one half

Docker is for running the stack, not for the edit loop — rebuilding a Flutter
image per keystroke is no way to work.

```sh
# API on :8080 with Postgres in Docker, rebuilt by Go directly
cd backend && make dev

# Client with hot reload, in another terminal
cd client && flutter run -d chrome
```

Details in each half's README.

## Open decisions

Two things are deliberately unresolved and written down in
[`docs/api.md`](docs/api.md#open-questions) rather than guessed at:

1. **Day-level participant assignment.** The impact rule unions "people on the
   day's scenes" with "people assigned directly to the day", but there is no
   table for the second half — that needs a new table, which is a decision to
   make out loud.
2. **`preview_token`**: signed and stateless, or a stored row with an expiry.
