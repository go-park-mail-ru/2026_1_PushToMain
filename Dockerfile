FROM golang:alpine

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o ./build/smail ./cmd/main.go

EXPOSE 8080
CMD ["./build/smail"]
