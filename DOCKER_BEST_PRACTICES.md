# Docker Best Practices for Masaar CRM

Your current approach is **correct and follows best practices**. Let me explain why and what else you should consider.

---

## ✅ Your Current Approach (Best Practice)

### Multiple Dockerfiles (Correct)

You have:
```
docker/Dockerfile           → Go API backend
docker/Dockerfile.web       → Next.js frontend
docker-compose.yml          → Orchestrates all services
```

### Why This Is Right

✅ **Separation of Concerns**
- Each service has its own build process
- Different base images (golang vs node)
- Different build requirements
- Easier to maintain and update independently

✅ **Scalability**
- Can deploy API and frontend independently
- Can scale one without the other
- Different resource requirements

✅ **Build Efficiency**
- Go image rebuilt only when Go code changes
- Node image rebuilt only when frontend changes
- Docker layer caching works optimally

✅ **Multi-Stage Builds** (You're already doing this!)
```dockerfile
FROM golang:1.22-alpine AS builder  # Build stage
FROM alpine:3.19                     # Runtime stage (smaller)
```
This reduces final image size from 800MB to ~50MB.

---

## 🎯 Docker Best Practices Checklist

### 1. Image Security ✅ (Partially Done)

**What you're doing right:**
- Using official base images (golang, node, alpine)
- Alpine for runtime (smallest surface area)

**What to add:**

```dockerfile
# Use specific version tags (not 'latest')
FROM golang:1.22-alpine    # ✅ Good
FROM golang:alpine         # ❌ Bad (uses latest)

# Add security scanning
# In CI/CD pipeline:
docker scan masaar-api
trivy image masaar-api
```

Create `.gitlab-ci.yml` or GitHub Actions:
```yaml
- docker scan masaar-api --severity high
```

### 2. Environment Variables ✅ (Done)

You're correctly using:
- `.env` file for local development
- Environment variables in docker-compose
- Docker environment declarations

**Tip:** Keep secrets out of Dockerfile, use:
- Docker secrets (Swarm)
- Environment variables (Compose)
- .env.example (documentation)

### 3. Health Checks ✅ (Partially Done)

You have `healthcheck` in docker-compose:
```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U masaar"]
  interval: 10s
  timeout: 5s
  retries: 5
```

**Add to all services:**
```yaml
# API
api:
  healthcheck:
    test: ["CMD", "wget", "--quiet", "--spider", "http://localhost:8080/api/v1/stats"]
    interval: 30s
    timeout: 10s
    retries: 3
    start_period: 40s

# Frontend
web:
  healthcheck:
    test: ["CMD", "wget", "--quiet", "--spider", "http://localhost:3000"]
    interval: 30s
    timeout: 10s
    retries: 3
```

### 4. Resource Limits 🔴 (Missing)

Add to docker-compose.yml:

```yaml
services:
  postgres:
    deploy:
      limits:
        cpus: '2'
        memory: 8G
      reservations:
        cpus: '1'
        memory: 4G

  redis:
    deploy:
      limits:
        cpus: '1'
        memory: 2G
      reservations:
        cpus: '0.5'
        memory: 1G

  ollama:
    deploy:
      limits:
        cpus: '4'
        memory: 8G
      reservations:
        cpus: '2'
        memory: 4G

  api:
    deploy:
      limits:
        cpus: '2'
        memory: 2G
      reservations:
        cpus: '1'
        memory: 1G

  web:
    deploy:
      limits:
        cpus: '1'
        memory: 1G
      reservations:
        cpus: '0.5'
        memory: 512M
```

Why? Prevents one service from consuming all resources.

### 5. Logging ✅ (Done)

You have:
```yaml
logging:
  driver: "json-file"
  options:
    max-size: "20m"
    max-file: "5"
```

**Add log rotation cron job:**
```bash
# /opt/masaar-crm/rotate-logs.sh
#!/bin/bash
find /var/lib/docker/containers -name "*.json" -mtime +7 -delete
```

Schedule: `0 0 * * * /opt/masaar-crm/rotate-logs.sh`

### 6. .dockerignore Files ✅ (Check if Present)

**For Go backend** (`docker/.dockerignore` or `.dockerignore`):
```
*.md
.git
.gitignore
.env
.env.local
.DS_Store
node_modules
web/
docs/
```

**For Next.js frontend** (`web/.dockerignore`):
```
*.md
.git
.gitignore
.env.local
.DS_Store
.next
node_modules
.env
.env.*.local
cmd/
internal/
migrations/
docker/
```

### 7. Non-Root User 🔴 (Missing in API)

In `docker/Dockerfile`:

```dockerfile
# Add non-root user
RUN addgroup -S masaar && adduser -S masaar -G masaar

# Set working directory with permissions
WORKDIR /app
RUN chown -R masaar:masaar /app

# Copy with correct ownership
COPY --from=builder --chown=masaar:masaar /masaar .

# Switch to non-root user
USER masaar

ENTRYPOINT ["/app/masaar"]
```

**Why?** Security: if container is compromised, attacker doesn't have root access.

### 8. Layer Caching Optimization 🔴 (Can improve)

Current (OK):
```dockerfile
COPY . .                 # Copies all files
RUN go build
```

Better:
```dockerfile
COPY go.mod go.sum ./
RUN go mod download      # Cache layer - only rebuilds if go.mod changes

COPY . .                 # Now copy rest
RUN go build             # Only rebuild if code changed
```

### 9. Secrets Management 🟡 (Partial)

Current approach (OK for MVP):
- .env file (local development)
- Environment variables (production)

**Better for production:**
```bash
# Use Docker secrets (Swarm mode)
docker secret create jwt_secret -
docker secret create db_password -

# Or use .env file with restricted permissions
chmod 600 /opt/masaar-crm/.env
```

### 10. Image Registry 🔴 (Consider for scalability)

Once you scale, push images to registry:

```bash
# Build with tag
docker build -t masaar-crm/api:1.0.0 .

# Push to registry (DockerHub, GitHub Container Registry, etc)
docker push masaar-crm/api:1.0.0

# Pull on production server
docker pull masaar-crm/api:1.0.0
```

This enables:
- Version control for images
- Easy rollbacks
- Sharing across multiple servers
- CI/CD automation

### 11. Docker Compose Overrides ✅ (You're doing this!)

You have:
```
docker-compose.yml              (dev)
docker-compose.prod.override.yml (prod)
```

Usage:
```bash
# Development
docker compose up

# Production
docker compose -f docker-compose.yml -f docker-compose.prod.override.yml up
```

This is **best practice** ✅

### 12. Volume Management ✅ (Good)

You have named volumes:
```yaml
volumes:
  postgres_data:
  redis_data:
  ollama_data:
```

**Add cleanup:**
```bash
# Regular backups
docker exec masaar-postgres pg_dump ... > backup.sql

# Clean old volumes (monthly)
docker volume prune

# List volumes
docker volume ls
```

---

## 🔧 Recommended Configuration Updates

Create `docker/docker-compose.best-practices.yml`:

```yaml
version: "3.9"

services:
  postgres:
    image: postgres:16
    container_name: masaar-postgres
    restart: always
    environment:
      POSTGRES_USER: masaar
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: masaar
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U masaar"]
      interval: 10s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 8G
        reservations:
          cpus: '1'
          memory: 4G
    logging:
      driver: "json-file"
      options:
        max-size: "50m"
        max-file: "10"
        labels: "service=postgres"

  redis:
    image: redis:7-alpine
    container_name: masaar-redis
    restart: always
    ports:
      - "6380:6380"
    command: redis-server --requirepass ${REDIS_PASSWORD} --port 6380
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-p", "6380", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 1G
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "5"

  api:
    build:
      context: .
      dockerfile: docker/Dockerfile
    container_name: masaar-api
    restart: always
    environment:
      PORT: 8080
      APP_ENV: production
      DATABASE_URL: ${DATABASE_URL}
      REDIS_URL: ${REDIS_URL}
      JWT_SECRET: ${JWT_SECRET}
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--spider", "http://localhost:8080/api/v1/stats"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "10"
        labels: "service=api"

  web:
    build:
      context: .
      dockerfile: docker/Dockerfile.web
    container_name: masaar-web
    restart: always
    environment:
      NEXT_PUBLIC_API_URL: ${NEXT_PUBLIC_API_URL}
    ports:
      - "3000:3000"
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--spider", "http://localhost:3000"]
      interval: 30s
      timeout: 10s
      retries: 3
    depends_on:
      - api
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "5"

volumes:
  postgres_data:
  redis_data:
  ollama_data:
```

---

## 📋 Implementation Priority

| Priority | Task | Impact | Effort |
|----------|------|--------|--------|
| **HIGH** | Add health checks to all services | Prevents zombie containers | 10 min |
| **HIGH** | Add resource limits | Prevents crashes | 15 min |
| **HIGH** | Create .dockerignore files | Faster builds | 5 min |
| **MEDIUM** | Add non-root users | Better security | 20 min |
| **MEDIUM** | Image scanning in CI/CD | Security | 30 min |
| **MEDIUM** | Improve layer caching | Faster builds | 20 min |
| **LOW** | Setup image registry | Easier deployments | Depends on choice |
| **LOW** | Advanced secrets management | Enterprise | Later |

---

## 🔒 Security Checklist

- [ ] All images use specific versions (not `latest`)
- [ ] Alpine base images (smaller = fewer vulnerabilities)
- [ ] Non-root users in Dockerfile
- [ ] .dockerignore files created
- [ ] Health checks on all services
- [ ] Resource limits set
- [ ] Secrets not in Dockerfile
- [ ] Regular security scanning (trivy, snyk)
- [ ] Log rotation configured
- [ ] Regular vulnerability updates

---

## 🚀 Single vs Multiple Dockerfiles

### Your Current Approach (CORRECT)

```
✅ Separate Dockerfile for each service
✅ docker-compose orchestrates
✅ Multi-stage builds for optimization
✅ Each service can be updated independently
```

### Why NOT Use Single Dockerfile

❌ **Single Monolithic Dockerfile:**
```dockerfile
FROM golang:1.22-alpine
RUN apk add nodejs npm
# ... install everything in one
```

Problems:
- Can't use optimal base images
- Huge final image
- Rebuilds everything on any change
- Hard to maintain
- Coupling between services

### When You MIGHT Use Combined File

**Only for tightly coupled services:**
```dockerfile
# Example: Frontend that needs build-time API integration
FROM golang:1.22 AS api-builder
COPY internal internal
RUN go build -o /api ./cmd/server

FROM node:20 AS web-builder
COPY web web
COPY --from=api-builder /api /app/api
RUN npm build

FROM node:20-alpine
COPY --from=web-builder /app/dist ./dist
```

**Not recommended** unless absolutely necessary.

---

## ✨ Your Current Setup Analysis

**What You're Doing Right:**
✅ Multiple Dockerfiles (correct approach)
✅ Multi-stage builds (optimized)
✅ docker-compose for orchestration
✅ Health checks (mostly)
✅ Volume management
✅ Logging configuration
✅ .env for secrets
✅ Production overrides

**What To Improve:**
🔴 Non-root users in API Dockerfile
🔴 Resource limits in docker-compose
🔴 Full health checks on all services
🔴 .dockerignore files
🔴 Image security scanning

---

## 📝 Quick Action Items

### Create .dockerignore files

**`docker/.dockerignore`:**
```
*.md
*.log
.git
.gitignore
.env
.DS_Store
node_modules
web/
```

**`web/.dockerignore`:**
```
*.md
.git
.gitignore
.env.local
.DS_Store
node_modules
cmd/
internal/
migrations/
docker/
```

### Update Dockerfile (Add Non-Root User)

In `docker/Dockerfile`, before `COPY`:
```dockerfile
RUN addgroup -S masaar && adduser -S masaar -G masaar
WORKDIR /app
RUN chown -R masaar:masaar /app
COPY --from=builder --chown=masaar:masaar /masaar .
USER masaar
```

### Add Resource Limits

In `docker-compose.prod.override.yml`:
```yaml
api:
  deploy:
    resources:
      limits:
        cpus: '2'
        memory: 2G
```

---

## 🎯 Summary

### Your Approach: ✅ CORRECT

**Multiple Dockerfiles + docker-compose is the right choice** for:
- Different language/framework requirements
- Independent scaling
- Optimal build efficiency
- Better maintainability

### Next Steps (Priority Order)

1. **Add resource limits** (15 min)
2. **Add health checks** to all services (10 min)
3. **Create .dockerignore files** (5 min)
4. **Add non-root users** (20 min)
5. **Setup security scanning** (30 min)

Everything else is enhancement for maturity, not required for MVP.

---

**Your Docker setup is production-ready. Keep doing what you're doing!** ✅
