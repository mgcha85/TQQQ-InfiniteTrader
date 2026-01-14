# ============================================
# Stage 1: Builder
# ============================================
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO (SQLite)
RUN apk add --no-cache git build-base

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
FROM alpine:3.19

# Install minimal runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

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
