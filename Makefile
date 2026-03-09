.PHONY: build test api run

APP_NAME = smail
BUILD_DIR = build
CMD_PATH = ./cmd/main.go

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_PATH)

run:
	go run $(CMD_PATH)

test:
	go test ./...

api:
	swag init -g main.go --output docs --dir ./cmd,./internal/app/handler,./internal/app/models,./internal/pkg/response,./internal/pkg/middleware,./internal/pkg/utils

clean:
	rm -rf $(BUILD_DIR)