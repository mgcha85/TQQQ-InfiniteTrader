FROM golang:1.23-alpine

WORKDIR /app

# Install git and build tools (gcc, musl-dev) for CGO
# Install git and build tools (gcc, musl-dev) for CGO, and tzdata for timezone support
RUN apk add --no-cache git build-base tzdata

ENV CGO_ENABLED=1

COPY go.mod ./
# COPY go.sum ./ # Not created yet, will be copied if exists
RUN go mod download

COPY . .

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["go", "run", "./cmd/server"]
