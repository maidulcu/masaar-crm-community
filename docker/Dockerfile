# ---- Build stage ----
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependencies separately for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /masaar ./cmd/server

# ---- Final stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user for security
RUN addgroup -S masaar && adduser -S masaar -G masaar

WORKDIR /app

# Copy binary and migrations from builder
COPY --from=builder --chown=masaar:masaar /masaar .
COPY --from=builder --chown=masaar:masaar /app/migrations ./migrations
# Copy .env if it exists (create empty one if not)
RUN if [ -f /app/.env ]; then cp --chown=masaar:masaar /app/.env .env; fi

# Set permissions
RUN chmod +x /app/masaar

# Run as non-root user
USER masaar

EXPOSE 8080

# Health check for container orchestration
HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=40s \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/masaar"]
