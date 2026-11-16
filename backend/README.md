# ProductionFlow — backend

Go HTTP API for ProductionFlow. This is the skeleton: routes, config, database
pool and migrations are in place, the endpoints are not implemented yet.

## Requirements

- Go 1.25+ (`go.mod` says 1.25.0 — `pgx/v5` requires it)
- Docker + Docker Compose (for Postgres 16)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI:
  `brew install golang-migrate`

## Run it

```sh
make dev        # start Postgres, apply migrations, run the API on :8080
```

Or step by step:

```sh
make up         # docker compose up -d postgres
make migrate    # apply migrations/*.up.sql
make run        # go run ./cmd/api
```

### Already running a Postgres on 5432?

A native Postgres bound to `127.0.0.1:5432` silently wins over the container's
port mapping, and the symptom is a confusing
`role "productionflow" does not exist` from `make migrate`. Pick another host
port instead:

```sh
POSTGRES_PORT=5433 make dev
```

`POSTGRES_PORT` feeds both the compose mapping and the default `DATABASE_URL`,
so nothing else needs changing.

Check it is alive:

```sh
curl -s localhost:8080/api/health | jq
```

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

The API starts even when Postgres is down — health returns `503` with
`"status": "degraded"` instead of refusing to boot, so the client team can run
the server on its own.

Everything else returns `501`:

```sh
curl -si localhost:8080/api/projects | tail -1
# {"error":{"code":"not_implemented","message":"this endpoint is not implemented yet"}}
```

### Whole stack in Docker

The compose file lives at the **repo root** and covers all four services —
Postgres, migrations, the API and the web client — so this half does not define
its own Postgres and the two cannot drift apart. From the repo root:

```sh
docker compose up --build       # then http://localhost:3000
```

`make stack` does the same thing from in here. Migrations run automatically as a
one-shot service; the API waits for them to exit 0.

### No golang-migrate installed?

`make migrate` needs the CLI, but the same image compose uses will do it:

```sh
docker run --rm -v "$PWD/migrations:/migrations" --network host \
  migrate/migrate:v4.19.0 -path /migrations \
  -database "postgres://productionflow:productionflow@localhost:5432/productionflow?sslmode=disable" up
```

### The image

`Dockerfile` is a two-stage cross-compiling build:

- Go module download and the build cache are BuildKit cache mounts, so a rebuild
  after a one-line change takes seconds.
- `--platform=$BUILDPLATFORM` plus `GOARCH=$TARGETARCH` cross-compiles instead of
  running an emulator, which matters if the demo host is not arm64.
- The runtime stage is `distroless/static` running as `nonroot` — **16.9 MB
  total**, no shell, no package manager.

That last point has one consequence worth knowing: compose cannot health-check
the API container, because there is no `sh`, `curl` or `wget` inside it to run a
check with. Probe `/api/health` from outside instead.

## Make targets

| Target | What it does |
| --- | --- |
| `dev` | `up` + `migrate` + `run` |
| `run` | Run the API with build info stamped in |
| `migrate` / `migrate-down` | Apply / roll back one migration |
| `test` | `go test ./...` |
| `lint` | `go vet ./...` plus a `gofmt -l` check |
| `fmt`, `tidy` | `gofmt -w .`, `go mod tidy` |
| `stack` | Build and start the whole project via the root compose file |
| `up`, `down`, `logs`, `psql` | Compose helpers (they all use `-f ../docker-compose.yml`) |

## Config

All optional — the defaults target the compose Postgres.

| Variable | Default | Notes |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://productionflow:productionflow@localhost:5432/productionflow?sslmode=disable` | libpq connection string |
| `POSTGRES_PORT` | `5432` | Make/compose only, not read by the API. Host port for the compose Postgres. |
| `API_ADDR` | `:8080` | Listen address |
| `APP_ENV` | `dev` | `dev` logs text, anything else logs JSON and requires `JWT_SECRET` |
| `SHUTDOWN_TIMEOUT_SECONDS` | `10` | Grace period for in-flight requests |
| `JWT_SECRET` | a public dev-only value | Signs access tokens. **Required outside dev**, at least 32 bytes: `openssl rand -base64 48` |
| `JWT_TTL_HOURS` | `168` | Access token lifetime (7 days). No refresh flow in the MVP |
| `PUBLIC_WEB_URL` | `http://localhost:3000` | Where the web app lives; invite links are `{PUBLIC_WEB_URL}/join/{code}` |

## Layout

```
cmd/api/main.go      wiring only — config, pool, router, graceful shutdown
internal/config/     environment parsing with defaults
internal/domain/     entities + rules, including the impact computation. No SQL, no HTTP.
internal/store/      pgx pool and queries, one file per aggregate. No rules.
internal/http/       routes, handlers, middleware, DTOs. No rules, no SQL.
internal/auth/       bcrypt password hashing + HS256 access tokens
migrations/          golang-migrate pairs, forward-only in practice
```

Each package has a `doc.go` spelling out what belongs in it. Read that before
adding a file.

## Conventions worth knowing before you add code

- Errors always go through `writeError`, so every failure has the same envelope:
  `{"error":{"code":"day_not_found","message":"..."}}`. Never return a raw Go
  error string — the client switches on `code`.
- The impact computation stays a pure function in `internal/domain`. Not in SQL,
  not in a handler.
- Timestamps are `timestamptz` in UTC. Call times are `date` + `time`, never a
  single instant — a call time is wall-clock, not a point on the timeline.
- Never edit a migration that has run; add a new pair.
- `docs/api.md` is the contract and changes *before* the code.

## Tests

```sh
make test
```

`internal/http` covers the error envelope, the 501 stubs and both health paths
(database up → 200, down → 503 with the driver error not leaked). Once the
impact computation exists it gets unit tests in `internal/domain` — that is
where the real logic will be.
