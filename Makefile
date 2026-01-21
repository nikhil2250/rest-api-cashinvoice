.PHONY: help build run docker-build docker-up docker-down docker-logs clean

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the Go application
	go build -o main .

run: ## Run the application locally
	go run .

deps: ## Download Go dependencies
	go mod download
	go mod tidy

docker-build: ## Build Docker images
	docker-compose build

docker-up: ## Start all services with Docker Compose
	docker-compose up -d

docker-down: ## Stop all services
	docker-compose down

docker-logs: ## View logs from all services
	docker-compose logs -f

docker-restart: ## Restart all services
	docker-compose restart

clean: ## Clean build artifacts
	rm -f main
	docker-compose down -v

test: ## Run tests (when implemented)
	go test -v ./...

lint: ## Run linter (requires golangci-lint)
	golangci-lint run