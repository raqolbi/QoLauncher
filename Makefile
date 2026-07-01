.PHONY: build test lint docker-build clean help

BINARY   := launcher
VERSION  ?= 0.1.0-dev
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILDDATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILDDATE)"
GO       := go
DOCKER   := docker

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build launcher binary to bin/
	@mkdir -p bin
	$(GO) build $(LDFLAGS) -o bin/$(BINARY) ./cmd/launcher

test: ## Run all tests
	$(GO) test ./...

lint: ## Run linters (golangci-lint or go vet fallback)
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || $(GO) vet ./...

docker-build: ## Build Docker image (requires Docker)
	$(DOCKER) build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILDDATE=$(BUILDDATE) \
		-t qolauncher:$(VERSION) -t qolauncher:latest .

build-examples: ## Build example app binaries into apps/ (static Linux, for Alpine Docker)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o apps/http-server/server ./apps/http-server
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o apps/hello/hello ./apps/hello

compose-up: build-examples docker-build ## Build apps + image, run all via launcher.sh
	cp -n .env.example .env 2>/dev/null || true
	./launcher.sh --run-all

compose-down: ## Stop all launcher containers
	./launcher.sh --stop

compose-logs: ## Follow all compose service logs
	$(DOCKER) compose logs -f

clean: ## Remove build artifacts
	rm -rf bin/
