APP_NAME = hxh_rpg_system
GO_CMD = go
GOOSE_CMD = goose
DOCKER_COMPOSE = docker compose
DB_URL = postgres://$(PG_DB_USER):$(PG_DB_PASS)@$(PG_DB_HOST):$(PG_DB_PORT)/$(PG_DB_NAME)?sslmode=$(PG_DB_SSLMODE)

# Load environment variables
include .env
export $(shell sed 's/=.*//' .env)

# Main commands
.PHONY: all
all: build

.PHONY: build
build:
	$(GO_CMD) build -o bin/$(APP_NAME) ./cmd/api/main.go

.PHONY: run
run: db-up db-wait migrate-up build
	./bin/$(APP_NAME)

# Dev com hot reload (air) — reinicia automaticamente ao salvar qualquer .go
# Requer: go install github.com/air-verse/air@latest
.PHONY: run-dev
run-dev: db-up db-wait migrate-up
	@trap 'kill 0' EXIT; \
	air -c .air.api.toml & \
	air -c .air.game.toml; \
	wait

# Targets individuais (sem db setup) — úteis para rodar isolado em terminal próprio
.PHONY: dev-api
dev-api:
	air -c .air.api.toml

.PHONY: dev-game
dev-game:
	air -c .air.game.toml

# Database lifecycle
.PHONY: db-up
db-up:
	$(DOCKER_COMPOSE) up -d

.PHONY: db-down
db-down:
	$(DOCKER_COMPOSE) down

.PHONY: db-wait
db-wait:
	@echo "Waiting for PostgreSQL..."
	@until $(DOCKER_COMPOSE) exec -T db pg_isready -U $(PG_DB_USER) -d $(PG_DB_NAME) > /dev/null 2>&1; do sleep 1; done
	@echo "PostgreSQL ready."

# Migration commands
.PHONY: migrate-up
migrate-up:
	$(GOOSE_CMD) -dir ./migrations postgres "$(DB_URL)" up

.PHONY: migrate-down
migrate-down:
	$(GOOSE_CMD) -dir ./migrations postgres "$(DB_URL)" down

.PHONY: migrate-create
migrate-create:
	$(GOOSE_CMD) -dir ./migrations create $(name) sql

# Test commands
.PHONY: test
test:
	$(GO_CMD) test ./...

.PHONY: test-integration
test-integration:
	$(GO_CMD) test -tags=integration -p 1 ./internal/gateway/pg/...

# Auxiliary commands
.PHONY: env
env:
	@echo "Loaded environment variables:"
	@env | grep PG_DB
