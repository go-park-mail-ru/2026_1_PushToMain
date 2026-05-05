.PHONY: \
	build build-user build-email build-folder \
	run-user run-email run-folder \
	test test-coverage test-race \
	generate bench api \
	docker-build docker-up docker-down docker-logs docker-ps \
	docker-build-user docker-build-email docker-build-folder \
	lint clean help

# ─────────────────────────────────────────────────────────────────
# Config
# ─────────────────────────────────────────────────────────────────
APP_NAME        := smail
BUILD_DIR       := build
PROJECT_ROOT    := $(shell pwd)
export PROJECT_ROOT

COVERAGE_FILE      := coverage.out
COVERAGE_HTML      := coverage.html
COVERAGE_THRESHOLD := 60

SERVICES := user email folder

# ─────────────────────────────────────────────────────────────────
# Build
# ─────────────────────────────────────────────────────────────────

## build: compile all three microservices
build: build-user build-email build-folder

build-user:
	@echo "  >> Building user-service..."
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/user-service ./cmd/user/main.go

build-email:
	@echo "  >> Building email-service..."
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/email-service ./cmd/email/main.go

build-folder:
	@echo "  >> Building folder-service..."
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/folder-service ./cmd/folder/main.go

# ─────────────────────────────────────────────────────────────────
# Run (local, without Docker)
# ─────────────────────────────────────────────────────────────────

## run-user: run user service locally
run-user:
	go run ./cmd/user/main.go --config configs/user/config.yaml

## run-email: run email service locally
run-email:
	go run ./cmd/email/main.go --config configs/email/config.yaml

## run-folder: run folder service locally
run-folder:
	go run ./cmd/folder/main.go --config configs/folder/config.yaml

# ─────────────────────────────────────────────────────────────────
# Tests
# ─────────────────────────────────────────────────────────────────

## test: run all tests
test:
	go test ./...

## test-coverage: generate coverage report (excludes mocks)
test-coverage:
	@echo "Generating coverage profile (excluding mocks)..."
	@go test -coverprofile=$(COVERAGE_FILE) $(shell go list ./... | grep -v /mocks)
	@echo ""
	@echo "==> Total coverage:"
	@go tool cover -func=$(COVERAGE_FILE) | grep total
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "==> HTML report: $(COVERAGE_HTML)"

## test-race: run tests with race detector
test-race:
	go test -race -coverprofile=$(COVERAGE_FILE) ./...

## bench: run benchmarks
bench:
	go test ./... -bench=. -benchmem

# ─────────────────────────────────────────────────────────────────
# Code generation
# ─────────────────────────────────────────────────────────────────

## generate: run go generate
generate:
	go generate ./...

## api: regenerate Swagger docs
api:
	swag init -g cmd/user/main.go --output docs \
		--dir ./cmd/user,./cmd/email,./cmd/folder \
		,./microservices/user/delivery/http \
		,./microservices/email/delivery/http \
		,./microservices/folder/delivery/http \
		,./microservices/user/models \
		,./microservices/email/models \
		,./microservices/folder/models \
		,./internal/pkg/response \
		,./internal/pkg/middleware \
		,./internal/pkg/utils

## proto: regenerate protobuf stubs
proto:
	protoc --go_out=. --go-grpc_out=. proto/user/user.proto
	protoc --go_out=. --go-grpc_out=. proto/email/email.proto

# ─────────────────────────────────────────────────────────────────
# Docker
# ─────────────────────────────────────────────────────────────────

## docker-build: build all service images
docker-build:
	docker compose --env-file .env build

## docker-build-user: build only user-service image
docker-build-user:
	docker compose --env-file .env build user-service

## docker-build-email: build only email-service image
docker-build-email:
	docker compose --env-file .env build email-service

## docker-build-folder: build only folder-service image
docker-build-folder:
	docker compose --env-file .env build folder-service

## docker-up: start all services in detached mode
docker-up:
	docker compose --env-file .env up -d

## docker-down: stop and remove containers
docker-down:
	docker compose down

## docker-logs: follow logs for all services (or pass SERVICE=<name>)
docker-logs:
	docker compose logs -f $(SERVICE)

## docker-ps: show running containers
docker-ps:
	docker compose ps

# ─────────────────────────────────────────────────────────────────
# Lint / Clean
# ─────────────────────────────────────────────────────────────────

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## clean: remove build artefacts and coverage files
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)

# ─────────────────────────────────────────────────────────────────
# CI
# ─────────────────────────────────────────────────────────────────

## ci: full CI pipeline (generate → coverage → race)
ci: generate test-coverage test-race

# ─────────────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────────────

## help: print this help message
help:
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | column -t -s ':'
