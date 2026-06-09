# ---- Build Stage ----
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Install templ CLI
RUN go install github.com/a-h/templ/cmd/templ@latest

# Cache dependencies before copying source
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate templ files and build a static binary
RUN templ generate && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server .

# ---- Runtime Stage ----
FROM alpine:3.21

WORKDIR /app

# ca-certificates for TLS (WebSocket connection to ESP32)
RUN apk add --no-cache ca-certificates

# Copy binary and static assets from builder
COPY --from=builder /app/server .
COPY --from=builder /app/static ./static

# Create data directory for SQLite database
RUN mkdir -p /app/data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./server"]
