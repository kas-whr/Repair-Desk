-include .env
export

# Git Bash on Windows rewrites args like /migrations into C:/Program Files/Git/...
export MSYS_NO_PATHCONV := 1

COMPOSE := docker compose
MIGRATE := $(COMPOSE) run --rm migrate
DB_URL  := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=disable

.PHONY: help up down restart logs ps db-up run-api run-frontend test \
        migrate-create migrate-up migrate-down clean

help: ## Show available commands
	@grep -hE '^[a-zA-Z_-]+:.*?## ' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

# --- Full stack in Docker ---------------------------------------------------

up: ## Build and start the whole stack (postgres, migrations, api, frontend)
	$(COMPOSE) up -d --build

down: ## Stop the stack (data is kept)
	$(COMPOSE) down

restart: down up ## Restart the stack

logs: ## Follow logs (optionally s=api)
	$(COMPOSE) logs -f $(s)

ps: ## Show container status
	$(COMPOSE) ps -a

clean: ## Stop the stack and DELETE the database volume
	@read -p "Delete all database data? [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		$(COMPOSE) down -v && echo "Database volume removed"; \
	else \
		echo "Canceled"; \
	fi

# --- Local development ------------------------------------------------------

db-up: ## Start only postgres and apply migrations (for running api locally)
	$(COMPOSE) up -d migrate

run-api: ## Run backend locally (needs make db-up)
	go run ./cmd/api

run-frontend: ## Run Vite dev server locally (proxies /api to localhost:$(HTTP_PORT))
	cd frontend && npm run dev

test: ## Run backend unit tests
	go test ./...

# --- Migrations ---------------------------------------------------------------

migrate-create: ## Create a migration: make migrate-create name=init
	$(if $(name),,$(error name is required, e.g. make migrate-create name=init))
	$(MIGRATE) create -ext sql -dir /migrations -seq $(name)

migrate-up: ## Apply all pending migrations
	@$(MIGRATE) -path=/migrations -database="$(DB_URL)" up

migrate-down: ## Roll back migrations: make migrate-down n=1 (default 1)
	@$(MIGRATE) -path=/migrations -database="$(DB_URL)" down $(or $(n),1)
