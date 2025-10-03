# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk update && apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimized flags
RUN go build -tags netgo -ldflags '-s -w' -o app .

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy the binary from builder
COPY --from=builder /app/app .

# Expose port (can be overridden by PORT env var)
EXPOSE 8080

# Run the application
CMD ["./app"]
