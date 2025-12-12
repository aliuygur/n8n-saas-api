.PHONY: build run dev clean migrate-up migrate-down sqlc test

# Build the application
build:
	@echo "Building application..."
	@go build -o bin/server ./cmd/server

# Run the application
run: build
	@echo "Running application..."
	@./bin/server

# Run with live reload (requires air: go install github.com/air-verse/air@latest)
dev:
	templ generate --watch --proxybind="127.0.0.1" --proxyport="8080" --proxy="http://127.0.0.1:8081"  --cmd="go run cmd/server/main.go"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean

# Generate SQLC code
sqlc:
	@echo "Generating SQLC code..."
	@sqlc generate

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Install development dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/a-h/templ/cmd/templ@latest

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@golangci-lint run

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Generate templ templates
templ:
	@echo "Generating templ templates..."
	@templ generate


# Create .env from example
env:
	@if [ -f .env ]; then \
		echo ".env already exists. Backup and remove it first if you want to recreate."; \
	else \
		cp .env.example .env; \
		echo ".env created from .env.example"; \
		echo "Please edit .env and fill in your actual values"; \
	fi

# Build and submit Docker image to Google Cloud Build
gcloud-build:
	@echo "Building and submitting Docker image with Google Cloud Build..."
	gcloud builds submit --tag us-central1-docker.pkg.dev/instol/n8n-saas/n8n-saas-api:latest .
