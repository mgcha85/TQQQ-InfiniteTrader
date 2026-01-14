# ============================================
# Stage 1: Builder
# ============================================
FROM golang:1.24 AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y git build-essential

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# ============================================
# Stage 2: Production Runtime (minimal)
# ============================================
FROM debian:trixie-slim

# Install minimal runtime dependencies
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy only the binary from builder
COPY --from=builder /app/server /app/server

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Run the binary
CMD ["/app/server"]
