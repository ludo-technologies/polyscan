# Makefile for jscan

# Variables
BINARY_NAME := jscan
GO_MODULE := github.com/ludo-technologies/polyscan/jscan
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date +%Y-%m-%d)
LDFLAGS := -ldflags "-s -w \
                     -X '$(GO_MODULE)/internal/version.Version=$(VERSION)' \
                     -X '$(GO_MODULE)/internal/version.Commit=$(COMMIT)' \
                     -X '$(GO_MODULE)/internal/version.Date=$(DATE)' \
                     -X '$(GO_MODULE)/internal/version.BuiltBy=make'"

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m # No Color

.PHONY: all build test clean install run version help lint fmt

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## all: Build and test
all: test build

## build: Build the binary
build:
	@printf "$(GREEN)Building $(BINARY_NAME) $(VERSION)...$(NC)\n"
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/jscan

## test: Run tests
test:
	@printf "$(GREEN)Running tests...$(NC)\n"
	go test -v ./...

## test-short: Run short tests only
test-short:
	@printf "$(GREEN)Running short tests...$(NC)\n"
	go test -short ./...

## bench: Run benchmarks
bench:
	@printf "$(GREEN)Running benchmarks...$(NC)\n"
	go test -bench=. -benchmem ./...

## coverage: Generate coverage report
coverage:
	@printf "$(GREEN)Generating coverage report...$(NC)\n"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@printf "$(GREEN)Coverage report generated: coverage.html$(NC)\n"

## lint: Run linters
lint:
	@printf "$(GREEN)Running linters...$(NC)\n"
	golangci-lint run

## fmt: Format code
fmt:
	@printf "$(GREEN)Formatting code...$(NC)\n"
	go fmt ./...

## clean: Clean build artifacts
clean:
	@printf "$(YELLOW)Cleaning...$(NC)\n"
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf dist/
	go clean

## install: Install the binary
install: build
	@printf "$(GREEN)Installing $(BINARY_NAME)...$(NC)\n"
	go install $(LDFLAGS) ./cmd/jscan

## run: Run the binary with sample files
run: build
	@printf "$(GREEN)Running $(BINARY_NAME)...$(NC)\n"
	./$(BINARY_NAME) analyze testdata/javascript/simple/

## version: Show version information
version:
	@echo "$(BINARY_NAME) version $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Date: $(DATE)"

## build-all: Build for all platforms
build-all:
	@printf "$(GREEN)Building for all platforms...$(NC)\n"
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/jscan
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/jscan
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/jscan
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/jscan
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/jscan
	@printf "$(GREEN)Built binaries in dist/$(NC)\n"

## deps: Download dependencies
deps:
	@printf "$(GREEN)Downloading dependencies...$(NC)\n"
	go mod download
	go mod verify

## tidy: Tidy dependencies
tidy:
	@printf "$(GREEN)Tidying dependencies...$(NC)\n"
	go mod tidy
