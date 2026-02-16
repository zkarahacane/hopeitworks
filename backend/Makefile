BINARY_NAME := api
BUILD_DIR := bin
CMD_DIR := ./cmd/api
COMPOSE_FILE := ../deploy/docker-compose.yml

.PHONY: build run test docker-up docker-down docker-logs clean help

## build: Build the API binary
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

## run: Run the API binary locally
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

## test: Run all tests
test:
	go test ./... -v

## docker-up: Start docker compose development stack
docker-up:
	docker compose -f $(COMPOSE_FILE) up -d

## docker-down: Stop docker compose development stack
docker-down:
	docker compose -f $(COMPOSE_FILE) down

## docker-logs: Follow docker compose logs
docker-logs:
	docker compose -f $(COMPOSE_FILE) logs -f

## clean: Remove binary and clean build cache
clean:
	rm -rf $(BUILD_DIR)
	go clean -cache

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
