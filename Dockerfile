FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o ./build/smail ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/build/smail .

EXPOSE 8080
