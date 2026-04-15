.PHONY: build test api run

APP_NAME = smail
BUILD_DIR = build
CMD_PATH = ./cmd/main.go

PROJECT_ROOT := $(shell pwd)
export PROJECT_ROOT

COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html
COVERAGE_THRESHOLD=60

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_PATH)

run:
	go run $(CMD_PATH)

test:
	go test ./...

test-coverage:
	@echo "Generating coverage profile (excluding mocks)..."
	@go test -coverprofile=coverage.out $(shell go list ./... | grep -v /mocks)
	@echo "\n==> Общее покрытие кода тестами:"
	@go tool cover -func=coverage.out | grep total

test-race:
	go test ./... -race -coverprofile=$(COVERAGE_FILE)

generate:
	go generate ./...

bench:
	go test ./... -bench=. -benchmem

api:
	swag init -g main.go --output docs --dir ./cmd,./internal/app/handler,./internal/app/models,./internal/pkg/response,./internal/pkg/middleware,./internal/pkg/utils

docker-build:
	docker-compose --env-file .env build

docker-up:
	docker-compose --env-file .env up -d

docker-down:
	docker-compose down

clean:
	rm -rf $(BUILD_DIR)

ci: generate test-coverage test-race
