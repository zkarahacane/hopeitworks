.PHONY: build run docker-up docker-down docker-logs clean help

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

## build: Build the API binary
build:
	go build -o bin/api ./cmd/api

## run: Run the API binary locally
run: build
	./bin/api

## docker-up: Start the docker-compose development stack
docker-up:
	docker compose -f ../deploy/docker-compose.yml --env-file ../deploy/.env up --build -d

## docker-down: Stop the docker-compose development stack
docker-down:
	docker compose -f ../deploy/docker-compose.yml --env-file ../deploy/.env down

## docker-logs: Follow docker-compose logs
docker-logs:
	docker compose -f ../deploy/docker-compose.yml --env-file ../deploy/.env logs -f

## clean: Remove build artifacts and clean cache
clean:
	rm -rf bin/
	go clean -cache -testcache
