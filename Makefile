-include .env
export

# Recipes use bash syntax (eval, read, [ ]). From PowerShell/cmd GNU make on Windows
# would fall back to cmd.exe, so point it to Git Bash explicitly.
ifeq ($(OS),Windows_NT)
SHELL := C:/Program Files/Git/bin/bash.exe
endif

# Git Bash on Windows rewrites args like /migrations into C:/Program Files/Git/...
export MSYS_NO_PATHCONV := 1

COMPOSE := docker compose
MIGRATE := $(COMPOSE) run --rm migrate
DB_URL  := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=disable

# Image tag used by docker build and k8s manifests (bump on every release)
VERSION := 0.4.0
K8S_NS  := repair-desk
KUBECTL := kubectl -n $(K8S_NS)

.PHONY: help up down restart logs ps db-up run-api run-frontend test \
        migrate-create migrate-up migrate-down clean \
        k8s-build k8s-secret k8s-deploy k8s-status k8s-url k8s-delete \
        logging-deploy logging-ui alerts-ui monitoring-deploy monitoring-ui

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

# --- Kubernetes (minikube) ----------------------------------------------------

k8s-build: ## Build images inside minikube's Docker daemon
	eval $$(minikube docker-env --shell bash) && \
	docker build -t repair-desk-api:$(VERSION) . && \
	docker build -t repair-desk-frontend:$(VERSION) frontend && \
	docker build -t repair-desk-migrate:$(VERSION) migrations

k8s-secret: ## Create/update Secret postgres-credentials from .env
	@kubectl apply -f k8s/00-namespace.yaml
	@$(KUBECTL) create secret generic postgres-credentials \
		--from-literal=POSTGRES_DB="$(POSTGRES_DB)" \
		--from-literal=POSTGRES_USER="$(POSTGRES_USER)" \
		--from-literal=POSTGRES_PASSWORD="$(POSTGRES_PASSWORD)" \
		--dry-run=client -o yaml | kubectl apply -f -

k8s-deploy: k8s-secret ## Deploy everything (the migrate Job is recreated on each deploy)
	$(KUBECTL) delete job migrate --ignore-not-found
	kubectl apply -f k8s/

k8s-status: ## Show all resources in the namespace
	$(KUBECTL) get all,pvc,configmap,secret

k8s-url: ## Print the frontend URL (keep the command running on Windows/macOS)
	minikube service frontend -n $(K8S_NS) --url

k8s-delete: ## Delete the namespace with ALL data (including the database volume)
	@read -p "Delete namespace $(K8S_NS) with all data? [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then kubectl delete namespace $(K8S_NS); else echo "Canceled"; fi

# --- Logging: VictoriaLogs + Vector ---------------------------------------------

logging-deploy: ## Deploy VictoriaLogs and the Vector DaemonSet (namespace logging)
	kubectl apply -f k8s/logging/
	# Vector and vmalert read their ConfigMaps only at startup
	kubectl -n logging rollout restart daemonset/vector deployment/vmalert

logging-ui: ## Open VictoriaLogs at http://localhost:9428/vmui/ (keep running)
	kubectl -n logging port-forward svc/victorialogs 9428:9428

alerts-ui: ## Open vmalert at http://localhost:8880/vmalert/alerts (keep running)
	kubectl -n logging port-forward svc/vmalert 8880:8880

# --- Metrics: VictoriaMetrics single (Helm) ------------------------------------

monitoring-deploy: ## Install/upgrade VictoriaMetrics single via Helm (namespace monitoring)
	helm repo add vm https://victoriametrics.github.io/helm-charts/ --force-update
	helm upgrade --install vm-single vm/victoria-metrics-single --version 0.49.0 		-n monitoring --create-namespace -f k8s/monitoring/vmsingle-values.yaml

monitoring-ui: ## Open VictoriaMetrics at http://localhost:8428/vmui/ (keep running)
	kubectl -n monitoring port-forward svc/vm-single-victoria-metrics-single-server 8428:8428


# --- Metric alerts + Alertmanager ----------------------------------------------
.PHONY: metric-alerts-deploy metric-alerts-ui alertmanager-ui alert-inbox-ui alerts-test

metric-alerts-deploy: ## Deploy metric rules, vmalert, Alertmanager and local webhook
	kubectl apply -k k8s/monitoring/alerts

metric-alerts-ui: ## Open metric vmalert UI
	kubectl -n monitoring port-forward svc/metrics-vmalert 8881:8880

alertmanager-ui: ## Open Alertmanager
	kubectl -n monitoring port-forward svc/alertmanager 9093:9093

alert-inbox-ui: ## Open received webhook notifications
	kubectl -n monitoring port-forward svc/alert-inbox 8095:8080

alerts-test: ## Check metric rules, unit scenarios and Alertmanager config
	docker run --rm --user 0 --entrypoint /bin/promtool -v "$(CURDIR):/repo:ro" -w /repo/monitoring prom/prometheus:v3.4.1 test rules rules.test.yaml
	docker run --rm --user 0 --entrypoint /bin/amtool -v "$(CURDIR)/k8s/monitoring/alerts:/config:ro" prom/alertmanager:v0.28.1 check-config /config/alertmanager.yaml

.PHONY: grafana-deploy grafana-ui
grafana-deploy: ## Deploy Grafana with VictoriaMetrics and Repair Desk/Postgres dashboards
	kubectl apply -k k8s/monitoring/grafana

grafana-ui: ## Open Grafana
	kubectl -n monitoring port-forward svc/grafana 3000:80