# Build frontend
FROM node:20-alpine AS frontend

WORKDIR /build

# Copy package files
COPY web/package*.json ./

# Install dependencies
RUN npm ci

# Copy source
COPY web/ ./

# Build static files (outputs to /build/../static = /static)
RUN npm run build

# Build backend
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend from the correct path
COPY --from=frontend /static ./static

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o spotiflac-bot .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary
COPY --from=builder /app/spotiflac-bot .

# Copy static files
COPY --from=builder /app/static ./static

# Create download directory
RUN mkdir -p /tmp/spotiflac_downloads

# Expose API port
EXPOSE 8080

# Run the application
CMD ["./spotiflac-bot"]
