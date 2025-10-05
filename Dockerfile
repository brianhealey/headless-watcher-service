# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 go build -o sensecap-server ./cmd/server

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    sqlite \
    sqlite-libs

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/sensecap-server .

# Create data directory for SQLite
RUN mkdir -p /app/data

# Expose port
EXPOSE 8834

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8834/health || exit 1

# Run the server
CMD ["./sensecap-server"]
