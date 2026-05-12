FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o ./build/email-service \
    ./cmd/email/main.go

# ─────────────────────────────────────────────
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/build/email-service .

EXPOSE 8082 50052

ENTRYPOINT ["./email-service", "--config", "configs/email/config.yaml"]
