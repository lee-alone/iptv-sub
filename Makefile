.PHONY: help build build-windows build-debian clean test test-race coverage fmt lint deps all

BINARY_NAME=iptv-aggregator
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d %H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

WINDOWS_BINARY=$(BINARY_NAME).exe
WINDOWS_OUTPUT=build/windows/$(WINDOWS_BINARY)

DEBIAN_BINARY=$(BINARY_NAME)
DEBIAN_OUTPUT=build/debian/$(DEBIAN_BINARY)

LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

help:
	@echo "IPTV M3U Aggregator - Build Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build              - Build for current platform"
	@echo "  make build-windows      - Build for Windows x64"
	@echo "  make build-debian       - Build for Debian/Linux"
	@echo "  make build-all          - Build for all platforms"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make test               - Run tests"
	@echo "  make test-race          - Run tests with race detector"
	@echo "  make coverage           - Generate coverage report"
	@echo "  make fmt                - Format code"
	@echo "  make lint               - Run linter"
	@echo "  make deps               - Update dependencies"
	@echo "  make all                - Clean, test, and build all"
	@echo ""

build:
	@echo "Building for current platform..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

build-windows:
	@echo "Building for Windows x64..."
	@mkdir -p build/windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(WINDOWS_OUTPUT) .
	@echo "✓ Windows binary: $(WINDOWS_OUTPUT)"

build-debian:
	@echo "Building for Debian/Linux x64..."
	@mkdir -p build/debian
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DEBIAN_OUTPUT) .
	@chmod +x $(DEBIAN_OUTPUT)
	@echo "✓ Debian binary: $(DEBIAN_OUTPUT)"

build-all: build-windows build-debian
	@echo "✓ All platforms built successfully"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf build/
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	@echo "✓ Clean complete"

test:
	@echo "Running tests..."
	go test -v ./...

test-race:
	@echo "Running tests with race detector..."
	go test -race -v ./...

coverage:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Format complete"

lint:
	@echo "Running linter..."
	go vet ./...
	@echo "✓ Lint complete"

deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✓ Dependencies updated"

all: clean test build-all
	@echo "✓ Complete build successful"
