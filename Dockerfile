FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum .

RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY docs/ ./docs/

RUN go build -o ./build/smail ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/build/smail .
COPY .env .

EXPOSE 8080
CMD ["./smail"]
