# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mbflow-server ./cmd/server

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 mbflow && \
    adduser -D -u 1000 -G mbflow mbflow

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/mbflow-server .

# Change ownership
RUN chown -R mbflow:mbflow /app

# Use non-root user
USER mbflow

# Expose port
EXPOSE 8585

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8585/health || exit 1

# Run the application
CMD ["./mbflow-server"]
