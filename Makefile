# NeuroBoost v0.4.0 — Project Makefile
# Usage: make <target>    Run 'make help' for all targets.

SHELL := /bin/bash
.DEFAULT_GOAL := help

# ─── Config ────────────────────────────────────────────────────
SERVER       := 62.76.228.106
SERVER_USER  := root
SERVER_DIR   := /root/neuroboost
PROD_URL     := https://neuroboost.website
BRANCH_MAIN  := main
BRANCH_DEV   := develop

# ─── Help ──────────────────────────────────────────────────────
.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ═══════════════════════════════════════════════════════════════
# LOCAL DEVELOPMENT
# ═══════════════════════════════════════════════════════════════

.PHONY: dev dev-api dev-web install

dev: ## Start full stack locally (docker)
	docker-compose up -d
	@echo "✓ Stack running — API :8080, Web :5173, DB :5432"

dev-api: ## Run Go API locally (no docker)
	cd api-go && go run ./cmd/api

dev-web: ## Run React dev server
	cd web && pnpm dev

install: ## Install all dependencies
	cd api-go && go mod download
	cd web && pnpm install

# ═══════════════════════════════════════════════════════════════
# BUILD & CHECK
# ═══════════════════════════════════════════════════════════════

.PHONY: build build-api build-web check check-api check-web typecheck lint

build: build-api build-web ## Build both API and web

build-api: ## Build Go API binary
	cd api-go && go build -o bin/api ./cmd/api
	cd api-go && go build -o bin/healthcheck ./cmd/healthcheck
	@echo "✓ API binaries built"

build-web: ## Build React production bundle
	cd web && pnpm build
	@echo "✓ Web bundle built"

check: check-api check-web ## Run all checks (build + typecheck + tests)
	@echo "✓ All checks passed"

check-api: ## Build and test Go API
	cd api-go && go build ./...
	cd api-go && go vet ./...
	@echo "✓ API build + vet passed"

check-web: typecheck build-web ## Typecheck and build web
	@echo "✓ Web typecheck + build passed"

typecheck: ## TypeScript type checking only
	cd web && pnpm typecheck

test: ## Run Go tests
	cd api-go && go test -v ./...

# ═══════════════════════════════════════════════════════════════
# DOCKER
# ═══════════════════════════════════════════════════════════════

.PHONY: up down restart logs logs-api logs-web logs-db ps clean

up: ## Start docker stack
	docker-compose up -d

down: ## Stop docker stack
	docker-compose down

restart: ## Rebuild and restart docker stack
	docker-compose build
	docker-compose down
	docker-compose up -d
	@echo "✓ Stack restarted"

logs: ## Tail all container logs
	docker-compose logs -f

logs-api: ## Tail API logs only
	docker-compose logs -f api

logs-web: ## Tail web logs only
	docker-compose logs -f web

logs-db: ## Tail database logs only
	docker-compose logs -f db

ps: ## Show container status
	docker-compose ps

clean: ## Stop stack and remove volumes
	docker-compose down -v
	rm -rf web/dist
	@echo "✓ Cleaned up"

# ═══════════════════════════════════════════════════════════════
# DATABASE
# ═══════════════════════════════════════════════════════════════

.PHONY: migrate migrate-down migrate-status db-shell db-dump

migrate: ## Run all pending migrations (in docker)
	docker-compose exec api sh -c 'migrate -path /srv/migrations -database "$$DATABASE_URL" up'

migrate-down: ## Roll back last migration (in docker)
	docker-compose exec api sh -c 'migrate -path /srv/migrations -database "$$DATABASE_URL" down 1'

migrate-status: ## Show migration version (in docker)
	docker-compose exec api sh -c 'migrate -path /srv/migrations -database "$$DATABASE_URL" version'

db-shell: ## Open psql shell to local database
	docker-compose exec db psql -U neuroboost -d neuroboost

db-dump: ## Dump database to backup file (raw SQL)
	@mkdir -p backups
	docker-compose exec db pg_dump -U neuroboost neuroboost > backups/neuroboost_$$(date +%Y%m%d_%H%M%S).sql
	@echo "✓ Database dumped to backups/"

backup: ## Create gzipped database backup
	@mkdir -p backups
	docker-compose exec -T db pg_dump -U neuroboost neuroboost | gzip > backups/neuroboost_$$(date +%Y%m%d_%H%M%S).sql.gz
	@echo "✓ Backup created in backups/"

# ═══════════════════════════════════════════════════════════════
# DEPLOY (SERVER)
# ═══════════════════════════════════════════════════════════════

.PHONY: deploy deploy-check ssh

deploy: ## Deploy to production server (pull + rebuild + restart)
	ssh $(SERVER_USER)@$(SERVER) 'cd $(SERVER_DIR) && git pull && docker-compose build --pull && docker-compose down && docker-compose up -d && docker-compose ps'
	@echo "✓ Deployed — verifying..."
	@sleep 10
	@curl -fsS $(PROD_URL)/api/health && echo " ✓ Health check passed" || echo " ✗ Health check failed"

deploy-check: ## Check production health
	@echo "Container status:"
	@ssh $(SERVER_USER)@$(SERVER) 'cd $(SERVER_DIR) && docker-compose ps'
	@echo ""
	@echo "API health (local):"
	@ssh $(SERVER_USER)@$(SERVER) 'curl -fsS http://127.0.0.1:8080/api/health' && echo " ✓" || echo " ✗"
	@echo ""
	@echo "API health (public):"
	@curl -fsS $(PROD_URL)/api/health && echo " ✓" || echo " ✗"

ssh: ## SSH into production server
	ssh $(SERVER_USER)@$(SERVER)

# ═══════════════════════════════════════════════════════════════
# GIT & GITHUB
# ═══════════════════════════════════════════════════════════════

.PHONY: status diff pr pr-list ci-status

status: ## Git status + recent commits
	@git status -sb
	@echo ""
	@git log --oneline -10

diff: ## Show unstaged + staged changes
	@git diff
	@git diff --cached

pr: ## Create PR from current branch to main
	@BRANCH=$$(git branch --show-current); \
	if [ "$$BRANCH" = "$(BRANCH_MAIN)" ]; then echo "Already on main"; exit 1; fi; \
	git push -u origin $$BRANCH; \
	gh pr create --base $(BRANCH_MAIN) --head $$BRANCH --fill

pr-list: ## List open pull requests
	gh pr list

ci-status: ## Show CI status for current branch
	@BRANCH=$$(git branch --show-current); \
	echo "CI status for $$BRANCH:"; \
	gh run list --branch $$BRANCH --limit 5

# ═══════════════════════════════════════════════════════════════
# HEALTH & DIAGNOSTICS
# ═══════════════════════════════════════════════════════════════

.PHONY: health health-local health-prod info

health: health-local ## Check local stack health

health-local: ## Check local API health
	@curl -fsS http://127.0.0.1:8080/api/health && echo " ✓ Local API healthy" || echo " ✗ Local API unreachable"

health-prod: ## Check production API health
	@curl -fsS $(PROD_URL)/api/health && echo " ✓ Production API healthy" || echo " ✗ Production API unreachable"

info: ## Show project info and versions
	@echo "=== NeuroBoost v0.4.0 ==="
	@echo "Go:     $$(cd api-go && go version 2>/dev/null || echo 'not installed')"
	@echo "Node:   $$(node --version 2>/dev/null || echo 'not installed')"
	@echo "pnpm:   $$(pnpm --version 2>/dev/null || echo 'not installed')"
	@echo "Docker: $$(docker --version 2>/dev/null || echo 'not installed')"
	@echo "Git:    $$(git branch --show-current) ($$(git rev-parse --short HEAD))"
	@echo "Migrations: $$(ls api-go/migrations/*.up.sql 2>/dev/null | wc -l) files"
