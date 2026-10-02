# Repair-Desk
Accounting of requests for equipment repairs or troubleshooting.

An employee reports a broken laptop, printer, or network problem; the ticket gets a priority,
a deadline from the category SLA, and goes through the statuses NEW → IN_PROGRESS → RESOLVED → CLOSED.

## Stack
Go 1.24 (stdlib `net/http` + pgx) · PostgreSQL 17 · React + TypeScript + Vite (served by nginx) · Docker Compose

## Quick start
Requirements: Docker, GNU Make.

```sh
cp .env.example .env   # set your own credentials
make up                # postgres -> migrations -> api -> frontend
```

- Frontend: http://localhost:3000 (`/api/*` is proxied to the backend)
- API: http://localhost:8080/api/v1 — see [docs/api-docs.md](docs/api-docs.md)
- Health: `GET /healthz` (liveness), `GET /readyz` (database is reachable)
- PostgreSQL: `localhost:5432`

`make help` lists all commands.

## Project structure
```
cmd/api/main.go              entry point: config, DB pool, HTTP server, graceful shutdown
internal/config              configuration from environment variables
internal/domain              entities, validation, status state machine, business errors
internal/service             business rules (SLA, transitions, retired equipment, deletion)
internal/repository/postgres SQL via pgx
internal/httpapi             routes, handlers, unified error format, logging middleware
migrations/                  SQL migrations (golang-migrate)
frontend/                    React SPA + nginx config
docs/                        API docs, ER diagram, status diagram
```

## Local development
```sh
make db-up          # postgres + migrations in Docker
make run-api        # backend via go run, config from .env
make run-frontend   # Vite dev server, proxies /api to the backend
make test           # backend unit tests
```

## Migrations
```sh
make migrate-create name=add_something
make migrate-up
make migrate-down n=1
```

## Twelve-factor
| Factor | How |
|---|---|
| I. Codebase | One repository, one deployable per service (api, frontend) |
| II. Dependencies | `go.mod`/`go.sum`, `package-lock.json`; built inside Docker images |
| III. Config | Only environment variables (`.env` is not committed, see `.env.example`) |
| IV. Backing services | PostgreSQL is attached by host/port/credentials from env |
| V. Build, release, run | Multi-stage Dockerfiles build immutable images; config is applied at run time |
| VI. Processes | API is stateless, all state is in PostgreSQL |
| VII. Port binding | API listens on `HTTP_PORT`, frontend on port 80 of its container |
| VIII. Concurrency | Stateless API can be scaled by running more containers |
| IX. Disposability | Fast start, graceful shutdown on SIGTERM |
| X. Dev/prod parity | Same images and PostgreSQL version locally and in deployment |
| XI. Logs | JSON logs to stdout, one line per request |
| XII. Admin processes | Migrations run as a separate one-off `migrate` container |
