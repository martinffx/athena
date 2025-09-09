.PHONY: help build test lint fmt clean install dev release-test

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Development commands
build: ## Build the binary
	go build -ldflags="-s -w" -o openrouter-cc main.go

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

lint: ## Run linting
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

fmt: ## Format code
	gofmt -s -w .
	go mod tidy

vet: ## Run go vet
	go vet ./...

check: fmt vet lint test ## Run all checks (format, vet, lint, test)

clean: ## Clean build artifacts
	rm -f openrouter-cc openrouter-cc-*
	rm -rf dist/ build/

dev: build ## Build and run in development mode
	./openrouter-cc -port 11434

install: build ## Install binary to ~/.local/bin
	mkdir -p ~/.local/bin
	cp openrouter-cc ~/.local/bin/
	cp openrouter ~/.local/bin/
	chmod +x ~/.local/bin/openrouter

# Cross-compilation targets
build-linux: ## Build for Linux AMD64
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o openrouter-cc-linux-amd64 main.go

build-darwin: ## Build for macOS AMD64  
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o openrouter-cc-darwin-amd64 main.go

build-windows: ## Build for Windows AMD64
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o openrouter-cc-windows-amd64.exe main.go

build-all: build-linux build-darwin build-windows ## Build for all major platforms

# Release testing
release-test: ## Test the release build process locally
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/openrouter-cc-linux-amd64 main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/openrouter-cc-darwin-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/openrouter-cc-windows-amd64.exe main.go
	@echo "Release builds completed in dist/"

# Setup development environment
setup: ## Set up development environment
	@echo "Setting up development environment..."
	go mod download
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@echo "Development environment ready!"
	@echo "Copy config: cp openrouter.example.yml openrouter.yml"
	@echo "Edit config with your API key, then run: make dev"