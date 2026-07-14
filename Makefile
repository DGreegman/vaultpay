include .env
export

MIGRATIONS_DIR := migrations
BINARY         := bin/vaultpay

.PHONY: help dev run build test test-race lint vet fmt tidy \
        docker-up docker-down docker-logs docker-reset \
        migrate-up migrate-down migrate-create migrate-version migrate-force \
        setup clean

## help: show available targets
help:
	@grep -hE '^## ' $(MAKEFILE_LIST) | sed 's/## //'

# ---------- app ----------

## dev: run the api with live reload
dev:
	air

## run: run the api once
run:
	go run ./cmd/api

## build: compile the api binary
build:
	go build -o $(BINARY) ./cmd/api

# ---------- quality ----------

## test: run tests
test:
	go test ./...

## test-race: run tests with the race detector
test-race:
	go test -race ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format all code
fmt:
	go fmt ./...

## tidy: tidy go.mod
tidy:
	go mod tidy

## lint: fmt + vet + tidy check
lint: fmt vet tidy

# ---------- infra ----------

## docker-up: start postgres + redis
docker-up:
	docker compose up -d

## docker-down: stop containers (keeps data)
docker-down:
	docker compose down

## docker-logs: tail container logs
docker-logs:
	docker compose logs -f

## docker-reset: DESTROY containers AND data volumes
docker-reset:
	@read -p "This deletes all data. Type 'yes' to confirm: " ans; \
	if [ "$$ans" = "yes" ]; then docker compose down -v; else echo "aborted"; fi

# ---------- migrations ----------

## migrate-up: apply all migrations
migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

## migrate-down: roll back one migration
migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

## migrate-create: create a new migration pair
migrate-create:
	@read -p "migration name: " name; \
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $$name

## migrate-version: show current schema version
migrate-version:
	@migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" version 2>&1 || true

## migrate-force: force schema version (fixes 'dirty' state)
migrate-force:
	@read -p "force to version: " v; \
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" force $$v

# ---------- composite ----------

## setup: bring up infra, wait, migrate
setup: docker-up
	@echo "waiting for postgres..."
	@sleep 5
	@$(MAKE) migrate-up
	@echo "ready. run 'make dev'"

## clean: remove build artifacts
clean:
	rm -rf bin tmp