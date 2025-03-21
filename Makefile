APP_NAME = hxh_rpg_system
GO_CMD = go
GOOSE_CMD = goose
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
run: build
	./bin/$(APP_NAME)

.PHONY: run-dev
run-dev:
	$(GO_CMD) run ./cmd/api/main.go

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

# Auxiliary commands
.PHONY: env
env:
	@echo "Loaded environment variables:"
	@env | grep PG_DB