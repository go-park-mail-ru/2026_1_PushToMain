FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o ./build/user-service \
    ./cmd/user/main.go

# ─────────────────────────────────────────────
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/build/user-service .

EXPOSE 8081 50051

ENTRYPOINT ["./user-service", "--config", "configs/user/config.yaml"]
