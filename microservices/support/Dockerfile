FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o ./build/support-service \
    ./cmd/support/main.go

# ─────────────────────────────────────────────
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/build/support-service .

EXPOSE 8084

ENTRYPOINT ["./support-service", "--config", "configs/support/config.yaml"]
