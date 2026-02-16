.PHONY: build run docker-up docker-down docker-logs clean help generate lint-api install-tools

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

generate: ## Generate code from OpenAPI spec
	@echo "Generating API handlers and types from OpenAPI spec..."
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config .oapi-codegen.yaml ../api/openapi.yaml
	@echo "Code generation complete."

lint-api: ## Validate OpenAPI spec
	@echo "Validating OpenAPI spec..."
	npx --yes @redocly/cli lint ../api/openapi.yaml

install-tools: ## Install code generation tool dependencies
	@echo "Installing code generation tools..."
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "Tools installed."

build: ## Build the API binary
	go build -o bin/api ./cmd/api

run: build ## Run the API binary locally
	./bin/api

docker-up: ## Start the docker-compose development stack
	docker compose -f ../deploy/docker-compose.yml up --build -d

docker-down: ## Stop the docker-compose development stack
	docker compose -f ../deploy/docker-compose.yml down

docker-logs: ## Follow docker-compose logs
	docker compose -f ../deploy/docker-compose.yml logs -f

clean: ## Remove build artifacts and clean cache
	rm -rf bin/
	rm -f internal/api/handler/gen_*.go
	rm -f internal/adapter/postgres/gen_*.go
	go clean -cache -testcache
