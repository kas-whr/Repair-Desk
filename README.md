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

## Kubernetes (minikube)
Manifests are in [k8s/](k8s/): namespace, PostgreSQL (StatefulSet + PVC), migrations (Job),
api (ConfigMap + Deployment with 2 replicas + Service), frontend (Deployment + NodePort Service).
DB credentials go to the Secret `postgres-credentials`, created from `.env`.

```sh
minikube start --driver=docker
make k8s-build      # build images inside minikube's Docker (eval $(minikube docker-env))
make k8s-deploy     # secret + kubectl apply -f k8s/
make k8s-status
make k8s-url        # frontend URL; on Windows keep this terminal open
```

Images are tagged with `VERSION` from the Makefile; bump it together with the tags in `k8s/*.yaml`.

### Logs (VictoriaLogs)
All services log JSON lines to stdout (`timestamp`, `level`, `message`, `service.name`, `request_id`, …).
A Vector DaemonSet ships logs of the `repair-desk` namespace to VictoriaLogs (manifests in [k8s/logging/](k8s/logging/)).

```sh
make logging-deploy   # VictoriaLogs + Vector in namespace logging
make logging-ui       # then open http://localhost:9428/select/vmui/
make alerts-ui        # vmalert (alert on error logs): http://localhost:8880/vmalert/alerts
```

### Metrics (VictoriaMetrics)
The API exposes Prometheus/OpenMetrics metrics on port `9090` (`/metrics`): Go runtime, HTTP and business metrics
(`internal/metrics`). PostgreSQL metrics come from a `postgres-exporter` sidecar. VictoriaMetrics single (Helm) scrapes
pods annotated with `prometheus.io/scrape: "true"`.

```sh
make monitoring-deploy   # helm upgrade --install vm-single vm/victoria-metrics-single -n monitoring
make monitoring-ui       # then open http://localhost:8428/vmui/ or /targets
```

A prober (`cmd/prober`, [k8s/50-prober.yaml](k8s/50-prober.yaml)) opens the tickets page through the frontend every 10s
and exports `repairdesk_probe_success` (user-facing availability SLI).

For SLO demos the API supports fault injection, disabled by default and limited to `/api/*`:
```sh
kubectl -n repair-desk set env deployment/api FAULT_ERROR_RATE=0.3   # 30% of requests -> 500
kubectl -n repair-desk set env deployment/api FAULT_DELAY_MS=400     # +400 ms per request
kubectl -n repair-desk set env deployment/api FAULT_ERROR_RATE- FAULT_DELAY_MS-   # back to normal
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
