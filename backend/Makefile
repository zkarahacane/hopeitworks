.PHONY: build run docker-up docker-down docker-logs clean help generate lint lint-api install-tools seed reset-db _check-not-devcontainer

# Database connection defaults (match docker-compose.yml)
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_NAME ?= hopeitworks_dev
DB_USER ?= hopeitworks
DB_PASS ?= hopeitworks_dev_password
DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

generate: ## Generate code from OpenAPI spec and SQL queries
	@echo "Generating API handlers and types from OpenAPI spec..."
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config .oapi-codegen.yaml ../api/openapi.yaml
	@echo "Generating database query functions from SQL..."
	@if [ -n "$$(find queries -name '*.sql' 2>/dev/null)" ]; then \
		if command -v sqlc >/dev/null 2>&1; then \
			sqlc generate; \
		else \
			echo "sqlc not found, skipping SQL generation (files should be committed)"; \
		fi; \
	else \
		echo "No SQL queries found, skipping sqlc generation"; \
	fi
	@echo "Code generation complete."

lint-api: ## Validate OpenAPI spec
	@echo "Validating OpenAPI spec..."
	npx --yes @redocly/cli lint ../api/openapi.yaml

lint: ## Run golangci-lint
	golangci-lint run ./...

install-tools: ## Install code generation tool dependencies
	@echo "Installing code generation tools..."
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed."

build: ## Build the API binary
	go build -o bin/api ./cmd/api

run: build ## Run the API binary locally
	./bin/api

_check-not-devcontainer:
	@if [ "$$HOPEITWORKS_ENV" = "devcontainer" ]; then \
		echo "\033[0;31m[✗] Docker targets are blocked inside the devcontainer.\033[0m"; \
		echo "    The devcontainer is code-only — run docker-compose from the host."; \
		exit 1; \
	fi

docker-up: _check-not-devcontainer ## Start the docker-compose development stack
	docker compose -f ../deploy/docker-compose.yml up --build -d

docker-down: _check-not-devcontainer ## Stop the docker-compose development stack
	docker compose -f ../deploy/docker-compose.yml down

docker-logs: _check-not-devcontainer ## Follow docker-compose logs
	docker compose -f ../deploy/docker-compose.yml logs -f

clean: ## Remove build artifacts and clean cache
	rm -rf bin/
	rm -f internal/api/handler/gen_*.go
	rm -f internal/adapter/postgres/gen_*.go
	go clean -cache -testcache

seed: ## Seed the local database with dev fixtures
	@echo "Running migrations..."
	@migrate -path migrations/ -database "$(DATABASE_URL)" up
	@echo "Seeding dev data..."
	@PGPASSWORD="$(DB_PASS)" psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f testdata/seed.sql
	@echo "Seed complete. Credentials:"
	@echo "  admin@hopeitworks.dev / admin1234 (admin)"
	@echo "  dev@hopeitworks.dev   / user1234  (user)"
	@echo "  alice@hopeitworks.dev / user1234  (user)"

reset-db: ## Drop and recreate dev database, run migrations, and seed
	@echo "Dropping database $(DB_NAME)..."
	@PGPASSWORD="$(DB_PASS)" dropdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) --if-exists $(DB_NAME)
	@echo "Creating database $(DB_NAME)..."
	@PGPASSWORD="$(DB_PASS)" createdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME)
	@$(MAKE) seed
