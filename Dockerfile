# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy go mod file and download dependencies
COPY go.mod ./
RUN go mod download || true

# Copy source code
COPY . .

# Generate go.sum and build
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o spa-server ./cmd/spa-server

# Production stage - using distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Copy binary
COPY --from=builder /build/spa-server /usr/local/bin/spa-server

# Set working directory for serving files
WORKDIR /app

# Expose default port
EXPOSE 8080

# Run as non-root user (distroless:nonroot runs as uid 65532)
USER nonroot:nonroot

# Default command
ENTRYPOINT ["/usr/local/bin/spa-server"]
CMD ["--directory", "/app"]
