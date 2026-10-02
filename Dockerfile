# Build stage
FROM golang:1.27-alpine AS builder
WORKDIR /build

RUN apk add --no-cache git ca-certificates

# Cache dependencies separately for better layer caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary: no CGO dependencies are required
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o masaar ./cmd/server

# Final stage
FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S masaar && adduser -S masaar -G masaar

WORKDIR /app

COPY --from=builder --chown=masaar:masaar /build/masaar .
COPY --from=builder --chown=masaar:masaar /build/migrations ./migrations

# Never run the server as root
USER masaar

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/masaar"]
