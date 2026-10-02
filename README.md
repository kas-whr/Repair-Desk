# Repair-Desk
Accounting of requests for equipment repairs or troubleshooting.

## Stack
Go (backend) · PostgreSQL 17 · React + TypeScript + Vite (frontend, served by nginx) · Docker Compose

## Quick start
Requirements: Docker, GNU Make.

```sh
cp .env.example .env   # set your own credentials
make up                # postgres -> migrations -> api -> frontend
```

- Frontend: http://localhost:3000 (`/api/*` is proxied to the backend)
- API: http://localhost:8080 (health check: `GET /healthz`)
- PostgreSQL: `localhost:5432`

`make help` lists all commands.

## Local development
```sh
make db-up          # postgres + migrations in Docker
make run-api        # backend via go run, config from .env
make run-frontend   # Vite dev server, proxies /api to the backend
```

## Migrations
```sh
make migrate-create name=add_something
make migrate-up
make migrate-down n=1
```
