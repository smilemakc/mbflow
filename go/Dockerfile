# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependencies separately from source
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/mbflow-server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/mbflow-migrate ./cmd/migrate

# Runtime stage
FROM alpine:3.22

RUN apk --no-cache add ca-certificates tzdata wget && \
    addgroup -g 1000 mbflow && \
    adduser -D -u 1000 -G mbflow mbflow

WORKDIR /app

COPY --from=builder /out/mbflow-server /out/mbflow-migrate ./

# Writable directory for local file storage (MBFLOW_FILE_STORAGE_PATH)
RUN mkdir -p /app/data/storage && chown -R mbflow:mbflow /app/data

USER mbflow

ENV MBFLOW_FILE_STORAGE_PATH=/app/data/storage

EXPOSE 8585

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8585/health || exit 1

ENTRYPOINT ["./mbflow-server"]
