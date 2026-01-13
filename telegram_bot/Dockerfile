# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go_backend/go.mod go_backend/go.sum ./go_backend/
COPY telegram_bot/go.mod telegram_bot/go.sum ./telegram_bot/

# Download dependencies
RUN cd telegram_bot && go mod download

# Copy source code
COPY go_backend/ ./go_backend/
COPY telegram_bot/ ./telegram_bot/

# Build the application
RUN cd telegram_bot && CGO_ENABLED=0 GOOS=linux go build -o /spotiflac-bot .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary
COPY --from=builder /spotiflac-bot .

# Create download directory
RUN mkdir -p /tmp/spotiflac_downloads

# Set environment variables
ENV API_PORT=8080
ENV DOWNLOAD_DIR=/tmp/spotiflac_downloads

# Expose API port
EXPOSE 8080

# Run the application
CMD ["./spotiflac-bot"]
