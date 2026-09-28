include .env 

CURRENT_DIR :=$(shell pwd)

DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

mig-up:
	@echo "Running database migrations..."
	migrate -path ./migrations -database '$(DB_URL)' up

mig-down:
	@echo "Rolling back database migrations..."
	migrate -path ./migrations -database '$(DB_URL)' -verbose down

mig-force:
	@echo "Forcing database migrations..."
	migrate -path ./migrations -database '$(DB_URL)' -verbose force 1

mig-create:
	@echo "Creating a new migration..."
	migrate create -ext sql -dir ./migrations -seq $(name)