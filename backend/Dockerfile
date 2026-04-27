# ============================================
# Stage 1: Frontend Builder
# ============================================
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

# Build context is project root, so frontend/ is accessible
COPY frontend/package*.json ./
RUN npm install --legacy-peer-deps

COPY frontend/ .
RUN npm run build

# ============================================
# Stage 2: Backend Builder
# ============================================
FROM golang:1.24 AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y git build-essential

# Copy go mod files first for caching
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source code
COPY backend/ .

# Copy frontend build output into the embed path expected by main.go
COPY --from=frontend-builder /frontend/build ./cmd/server/frontend/build

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

# Copy config directory (required for symbols.json)
COPY --from=builder /app/config /app/config

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Run the binary
CMD ["/app/server"]
