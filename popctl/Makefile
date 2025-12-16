.PHONY: all build install clean test lint fmt help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary name
BINARY_NAME=popctl

# Version (can be overridden: make build VERSION=1.0.0)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-X github.com/Bidon15/popsigner/popctl/cmd.Version=$(VERSION)"

all: build

## Build the CLI binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

## Build with release optimizations
build-release:
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -o $(BINARY_NAME) .

## Install to GOPATH/bin
install:
	$(GOINSTALL) $(LDFLAGS) .

## Clean build artifacts
clean:
	rm -f $(BINARY_NAME)

## Run tests
test:
	$(GOTEST) -v -race ./...

## Run linter
lint:
	$(GOLINT) run ./...

## Format code
fmt:
	$(GOFMT) -s -w .

## Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## Build for all platforms
build-all: build-linux build-darwin build-windows

## Build for Linux (amd64 and arm64)
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-linux-arm64 .

## Build for macOS (amd64 and arm64)
build-darwin:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-darwin-arm64 .

## Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-windows-amd64.exe .

## Create distribution archives
dist: build-all
	cd dist && tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	cd dist && tar -czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	cd dist && tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	cd dist && tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	cd dist && zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe

## Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"

## Show help
help:
	@echo "popctl - POPSigner CLI"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

