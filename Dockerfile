# Use a lightweight Go base image for ARM
FROM golang:1.21-bullseye AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the Go binary for ARM
RUN GOOS=linux GOARCH=arm GOARM=7 go build -o myapp

# Use a minimal base image for the final container
FROM arm32v7/debian:bullseye-slim

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/myapp .

# Expose port (if your app uses one, e.g., 8080)
EXPOSE 8080

# Run the binary
CMD ["./myapp"]