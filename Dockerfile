# Multi-stage Dockerfile for AI Health Coach application
# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

# Build argument for version
ARG VERSION=dev

# Install necessary packages for building
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 creates a statically linked binary
# -ldflags="-s -w" strips debug information to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o bin/health-coach \
    cmd/server/main.go

# Stage 2: Create the runtime image
FROM alpine:3.18

# Install ca-certificates for HTTPS requests and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/bin/health-coach .

# Copy static files and templates
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Set environment variables
ENV ENVIRONMENT=production
ENV PORT=8080

# Command to run the application
CMD ["./health-coach"]

# Labels for metadata
LABEL maintainer="your-email@example.com" \
      version="1.0.0" \
      description="AI Health Coach - Daily health journal with AI insights" \
      org.opencontainers.image.source="https://github.com/your-org/health-coach" \
      org.opencontainers.image.documentation="https://github.com/your-org/health-coach/README.md" \
      org.opencontainers.image.licenses="MIT"