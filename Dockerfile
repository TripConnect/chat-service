# Stage 1: Build the Go app
FROM golang:1.24.4 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o chat-service .

# Stage 2: Run the Go app in Alpine
FROM alpine:3.22.0

WORKDIR /root/

COPY --from=builder /app/chat-service .

RUN chmod +x chat-service

EXPOSE 31073

CMD ["./chat-service"]
