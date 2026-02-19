# Makefile

# Auto-load .env if it exists
ifneq (,$(wildcard .env))
	include .env
	export
endif

# Variables (fallback defaults)
API_DB_DSN ?=

# If `API_DB_DSN` is not set, try to build it from `API_DATABASE.*` entries in `.env`.
ifeq ($(strip $(API_DB_DSN)),)
ifneq (,$(wildcard .env))
API_DB_USER := $(shell sed -n -e 's/^API_DATABASE\.USER[[:space:]]*=\(.*\)/\1/p' .env | tr -d '"')
API_DB_PASS := $(shell sed -n -e 's/^API_DATABASE\.PASSWORD[[:space:]]*=\(.*\)/\1/p' .env | tr -d '"')
API_DB_HOST := $(shell sed -n -e 's/^API_DATABASE\.HOST[[:space:]]*=\(.*\)/\1/p' .env | tr -d '"')
API_DB_PORT := $(shell sed -n -e 's/^API_DATABASE\.PORT[[:space:]]*=\(.*\)/\1/p' .env | tr -d '"')
API_DB_NAME := $(shell sed -n -e 's/^API_DATABASE\.NAME[[:space:]]*=\(.*\)/\1/p' .env | tr -d '"')
API_DB_SSL  := $(shell sed -n -e 's/^API_DATABASE\.SSL_MODE[[:space:]]*=\(.*\)/\1/p' .env | tr -d '"')
API_DB_DSN := $(shell printf 'postgres://%s:%s@%s:%s/%s?sslmode=%s' '$(API_DB_USER)' '$(API_DB_PASS)' '$(API_DB_HOST)' '$(API_DB_PORT)' '$(API_DB_NAME)' '$(API_DB_SSL)')
endif
endif

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  help              Print this help message"
	@echo "  run               Run the cmd/api application"
	@echo "  migrate-new    Create a new database migration (NAME=...)"
	@echo "  migrate-down   Roll back the last database migration"
	@echo "  migrate-up     Apply all up database migrations"
	@echo "  migrate-fresh  Drop all DB objects then reapply migrations"
	@echo "  tidy              Format Go files and tidy dependencies"

# Internal confirmation target
.PHONY: confirm
confirm:
	@printf "Are you sure? [y/N] " && read ans && [ "$${ans:-N}" = "y" ]

# Run application
.PHONY: run
run:
	go run ./cmd/api

.PHONY: seed
seed:
	@echo "Seeding database..."
	go run ./cmd/seed
	@echo "Database seeded."
	
# Create new migration
.PHONY: migrate-new
migrate-new:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME parameter is required"; \
		echo "Usage: make migrate-new NAME=migration_name"; \
		exit 1; \
	fi
	@echo "Creating migration file for $(NAME)..."
	migrate create -ext sql -dir ./internal/database/migrations -seq $(NAME)
	@echo "Migration file created for $(NAME)."

# Roll back last migration
.PHONY: migrate-down
migrate-down: confirm
	@echo "Rolling back last migration..."
	migrate -database "$(API_DB_DSN)" -path ./internal/database/migrations down 1
	@echo "Last migration rolled back."

# Apply migrations
.PHONY: migrate-up
migrate-up: confirm
	@echo "Running up migrations..."
	migrate -database "$(API_DB_DSN)" -path ./internal/database/migrations up
	@echo "Migrations applied."

# Drop everything managed by migrate and reapply migrations
.PHONY: migrate-fresh
migrate-fresh: confirm
	@echo "Dropping all database objects (migrate drop) and reapplying migrations..."
	@migrate -database "$(API_DB_DSN)" -path ./internal/database/migrations drop -f || true
	@migrate -database "$(API_DB_DSN)" -path ./internal/database/migrations up
	@echo "Database refreshed."

# Go maintenance
.PHONY: tidy
tidy:
	@echo "Formatting .go files..."
	go fmt ./...
	@echo "Tidying module dependencies..."
	go mod tidy
	@echo "Verifying dependencies..."
	go mod verify
