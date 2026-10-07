# Settings come from .env when it exists (copy .env.example to .env).
-include .env
export

DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.DEFAULT_GOAL := help

.PHONY: help run build test vet fmt tidy check swagger postman \
	mig-up mig-down mig-force mig-version mig-create \
	up down logs reset

help: ## Show this list
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-12s %s\n", $$1, $$2}'

# --- development

run: ## Run the API on this machine
	go run ./cmd/main.go

build: ## Build the binary into bin/lms
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/lms ./cmd/main.go

test: ## Run all tests (needs Postgres and Redis from .env; tests without them are skipped)
	go test -count=1 ./...

vet: ## go vet
	go vet ./...

fmt: ## gofmt
	gofmt -w cmd internal pkg

tidy: ## go mod tidy
	go mod tidy

check: fmt vet test ## fmt + vet + test

# --- API documentation (needs swag: go install github.com/swaggo/swag/cmd/swag@v1.16.4)

swagger: ## Generate docs/swagger from the annotations of the handlers
	swag init -g main.go -d ./cmd,./internal,./pkg -o docs/swagger --outputTypes go,json,yaml --parseInternal

postman: swagger ## Generate the Postman collection and environment in docs/postman
	python3 scripts/postman.py

# --- migrations (needs the migrate CLI: https://github.com/golang-migrate/migrate)

mig-up: ## Apply all migrations
	migrate -path ./migrations -database '$(DB_URL)' up

mig-down: ## Roll back migrations: make mig-down n=1 (default 1)
	migrate -path ./migrations -database '$(DB_URL)' down $(or $(n),1)

mig-version: ## Show the current migration version
	migrate -path ./migrations -database '$(DB_URL)' version

mig-force: ## Set the version after a failed migration: make mig-force v=3
	@test -n "$(v)" || (echo "usage: make mig-force v=<version>"; exit 1)
	migrate -path ./migrations -database '$(DB_URL)' force $(v)

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
