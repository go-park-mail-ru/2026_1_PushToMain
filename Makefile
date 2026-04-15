.PHONY: build test api run

APP_NAME = smail
BUILD_DIR = build
CMD_PATH = ./cmd/main.go

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
	@echo "Generating coverage profile (correct mode)..."
	go test ./... -coverprofile=$(COVERAGE_FILE) -covermode=atomic

	@echo ""
	go tool cover -func=$(COVERAGE_FILE)

	@echo ""
	@echo "Checking coverage threshold..."
	@COVERAGE=$$(go tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc) -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets threshold $(COVERAGE_THRESHOLD)%"; \
	fi

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
