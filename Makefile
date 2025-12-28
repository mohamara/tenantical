.PHONY: build run test clean docker-build docker-run help

BINARY_NAME=tenant-router
DOCKER_IMAGE=tenant-router:latest

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building..."
	go build -o bin/$(BINARY_NAME) ./cmd/server

run: ## Run the application
	@echo "Running..."
	go run ./cmd/server

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f tenants.db tenants.db-shm tenants.db-wal

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 \
		-e BACKEND_URL=http://host.docker.internal:3000 \
		-e DB_PATH=/data/tenants.db \
		-v $(PWD)/data:/data \
		$(DOCKER_IMAGE)

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

vendor: ## Vendor dependencies
	@echo "Vendoring dependencies..."
	go mod vendor

