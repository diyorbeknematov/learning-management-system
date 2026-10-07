# The app and the tests read .env themselves, and docker compose does too, so
# this file does not include it: that would override values given on the
# command line (POSTGRES_PORT=5435 make up). Only the migrations need the
# database settings, and they load .env inside their own shell command.
LOAD_ENV := set -a; [ -f .env ] && . ./.env; set +a;
DB_URL := postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=disable

.DEFAULT_GOAL := help

.PHONY: help run build test vet fmt tidy check swagger postman seed seed-spread \
	mig-up mig-down mig-force mig-version mig-create \
	up down logs reset

help: ## Show this list
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-12s %s\n", $$1, $$2}'

# --- development

run: ## Run the API on this machine
	go run ./cmd/main.go

build: ## Build the binary into bin/lms
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/lms ./cmd/main.go

test: ## Run all tests (they use the services of .env; tests without them are skipped)
	go test -count=1 ./...

vet: ## go vet
	go vet ./...

fmt: ## gofmt
	gofmt -w cmd internal pkg

tidy: ## go mod tidy
	go mod tidy

check: fmt vet test ## fmt + vet + test

# --- demo data

seed: ## Fill the running system with demo data (needs ADMIN_PASSWORD; SEED_API if the API is not on :8080)
	@$(LOAD_ENV) go run ./cmd/seed

seed-spread: ## Spread the demo payments over the last six months, so the charts have something to draw (docker compose only)
	docker compose exec -T postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -v ON_ERROR_STOP=1' < scripts/spread-demo-dates.sql

# --- API documentation (needs swag: go install github.com/swaggo/swag/cmd/swag@v1.16.4)

swagger: ## Generate docs/swagger from the annotations of the handlers
	swag init -g main.go -d ./cmd,./internal,./pkg -o docs/swagger --outputTypes go,json,yaml --parseInternal

postman: swagger ## Generate the Postman collection and environment in docs/postman
	python3 scripts/postman.py

# --- migrations (needs the migrate CLI: https://github.com/golang-migrate/migrate)

mig-up: ## Apply all migrations
	@$(LOAD_ENV) migrate -path ./migrations -database "$(DB_URL)" up

mig-down: ## Roll back migrations: make mig-down n=1 (default 1)
	@$(LOAD_ENV) migrate -path ./migrations -database "$(DB_URL)" down $(or $(n),1)

mig-version: ## Show the current migration version
	@$(LOAD_ENV) migrate -path ./migrations -database "$(DB_URL)" version

mig-force: ## Set the version after a failed migration: make mig-force v=3
	@test -n "$(v)" || (echo "usage: make mig-force v=<version>"; exit 1)
	@$(LOAD_ENV) migrate -path ./migrations -database "$(DB_URL)" force $(v)

mig-create: ## Create a migration: make mig-create name=add_x
	@test -n "$(name)" || (echo "usage: make mig-create name=<name>"; exit 1)
	migrate create -ext sql -dir ./migrations -seq $(name)

# --- docker compose

up: ## Start everything in Docker (API, Postgres, Redis, MinIO, Mailpit)
	docker compose up --build -d

down: ## Stop the containers (data stays)
	docker compose down

logs: ## Follow the logs of the API
	docker compose logs -f app

reset: ## Stop the containers and DELETE all their data
	docker compose down -v
