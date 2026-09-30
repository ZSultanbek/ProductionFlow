.DEFAULT_GOAL := help
.PHONY: help dev run migrate migrate-down test lint fmt tidy up down logs psql stack

# The single compose file lives at the repo root, so the backend dev loop and the
# full stack cannot drift apart.
COMPOSE := docker compose -f ../docker-compose.yml

# Host port for the compose Postgres. Override if 5432 is already taken on your
# machine — a native Postgres on 127.0.0.1:5432 shadows the container's mapping,
# and the symptom is a confusing `role "productionflow" does not exist`.
POSTGRES_PORT ?= 5432
export POSTGRES_PORT

# Overridable on its own: make migrate DATABASE_URL=...
DATABASE_URL ?= postgres://productionflow:productionflow@localhost:$(POSTGRES_PORT)/productionflow?sslmode=disable

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILT_AT ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.builtAt=$(BUILT_AT)

help: ## Show this help
	@grep -hE '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN{FS=":.*?## "};{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

dev: up migrate run ## Start Postgres, migrate, run the API

run: ## Run the API against DATABASE_URL
	DATABASE_URL="$(DATABASE_URL)" go run -ldflags "$(LDFLAGS)" ./cmd/api

migrate: ## Apply all up migrations
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Roll back the last migration
	migrate -path migrations -database "$(DATABASE_URL)" down 1

test: ## Run the unit tests
	go test ./...

lint: ## go vet plus a gofmt check
	go vet ./...
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	@echo "lint ok"

fmt: ## Format everything
	gofmt -w .

tidy: ## Sync go.mod / go.sum
	go mod tidy

up: ## Start Postgres in the background
	$(COMPOSE) up -d postgres

stack: ## Build and start the whole project (api + web + db + migrations)
	$(COMPOSE) up --build

down: ## Stop the compose stack
	$(COMPOSE) down

logs: ## Tail the compose logs
	$(COMPOSE) logs -f

psql: ## Open a psql shell on the dev database
	$(COMPOSE) exec postgres psql -U productionflow -d productionflow
