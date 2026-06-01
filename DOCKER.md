# Docker Deployment Guide

**Masaar CRM** can be deployed entirely using Docker and Docker Compose, making self-hosting simple and reliable.

## Quick Start (3 steps)

### 1. Clone and Configure

```bash
git clone https://github.com/maidulcu/masaar-crm.git
cd masaar-crm
cp .env.example .env
```

### 2. Edit Environment Variables

```bash
# Generate a secure JWT secret
JWT_SECRET=$(openssl rand -hex 32)

# Update .env with your settings
nano .env  # or your editor
```

**Minimum required changes in `.env`:**
- `DB_PASSWORD` - Set a strong database password
- `JWT_SECRET` - Use the generated value above
- `WA_VERIFY_TOKEN` - Keep or change (used for WhatsApp webhook)

### 3. Start Services

```bash
docker-compose up -d

# Wait for services to start (~30s)
sleep 30

# Verify health
docker-compose ps

# Check logs
docker-compose logs -f masaar
```

✅ **Access the application:**
- Dashboard: http://localhost:3000
- API: http://localhost:8080
- Swagger API Docs: http://localhost:8080/docs
- Default login: `admin@masaar.local` / `changeme`

---

## Architecture

```
┌─────────────┐
│  Browser   │ (port 3000)
└──────┬──────┘
       │
       ▼
┌──────────────┐     ┌──────────────┐
│ Next.js      │────→│ Fiber API    │ (port 8080)
│ Frontend     │     │ Backend      │
└──────────────┘     └──────┬───────┘
                            │
       ┌────────────────────┼────────────────────┐
       │                    │                    │
       ▼                    ▼                    ▼
   ┌───────────┐      ┌──────────┐        ┌──────────┐
   │PostgreSQL │      │ Redis    │        │ Ollama   │
   │ Database  │      │ Cache    │        │ LLM      │
   └───────────┘      └──────────┘        └──────────┘
```

### Services

| Service | Port | Purpose |
|---------|------|---------|
| **masaar** (Fiber API) | 8080 | REST API backend |
| **postgres** | 5432 | Rental/payment/lead database |
| **redis** | 6379 | Session cache, message queue |
| **ollama** | 11434 | Local AI model (thread summaries) |
| **frontend** | 3000 | Next.js dashboard UI |

---

## Configuration

### Environment Variables

All configuration is in `.env`. Common settings:

```bash
# Server
PORT=8080
APP_ENV=production

# Database (change password!)
DB_PASSWORD=very-strong-password

# JWT (generate with: openssl rand -hex 32)
JWT_SECRET=your-random-hex-string

# WhatsApp (optional)
WA_PHONE_NUMBER_ID=123456789
WA_ACCESS_TOKEN=your-token-here

# SMTP Email (optional)
SMTP_HOST=smtp.gmail.com
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password

# Ollama LLM
OLLAMA_BASE_URL=http://ollama:11434
OLLAMA_MODEL=llama3
```

---

## Common Commands

```bash
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# View logs
docker-compose logs -f masaar      # Backend logs
docker-compose logs -f frontend    # Frontend logs
docker-compose logs -f postgres    # Database logs

# Restart a specific service
docker-compose restart masaar

# View status
docker-compose ps

# Shell into container
docker-compose exec masaar sh
docker-compose exec postgres psql -U masaar -d masaar

# Remove all data (⚠️ destructive!)
docker-compose down -v
```

---

## Production Deployment

For production deployments, consider:

### 1. **Reverse Proxy (Nginx)**

```bash
# Use a reverse proxy for SSL/TLS
# Example Nginx config at docker/nginx.conf
```

### 2. **Database Backups**

```bash
# Backup database
docker-compose exec postgres pg_dump \
  -U masaar -d masaar > backup.sql

# Restore from backup
docker-compose exec -T postgres psql \
  -U masaar -d masaar < backup.sql
```

### 3. **Environment Security**

```bash
# Don't commit .env to git
echo ".env" >> .gitignore

# Use strong passwords
# Generate: openssl rand -hex 16

# Rotate JWT secret every 90 days
JWT_SECRET=$(openssl rand -hex 32)
```

### 4. **Volume Persistence**

Docker volumes are named and persist across restarts:
- `postgres_data` - Database files
- `redis_data` - Cache data
- `ollama_data` - AI model files

Backup volumes regularly:

```bash
docker run --rm -v masaar-crm_postgres_data:/data \
  -v /backup:/backup \
  alpine tar czf /backup/postgres-$(date +%s).tar.gz -C /data .
```

### 5. **Resource Limits**

Add to `docker-compose.yml` for each service:

```yaml
services:
  masaar:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

### 6. **Monitoring**

```bash
# Monitor resource usage
docker stats

# View error logs
docker-compose logs -f --tail 100 masaar
```

---

## Troubleshooting

### Database Connection Fails

```bash
# Check if postgres is running
docker-compose ps postgres

# Check postgres logs
docker-compose logs postgres

# Verify environment variables
docker-compose exec masaar env | grep DATABASE

# Test connection
docker-compose exec postgres \
  psql -U masaar -d masaar -c "SELECT NOW();"
```

### Redis Connection Fails

```bash
# Check redis
docker-compose ps redis

# Test redis connection
docker-compose exec redis redis-cli ping
```

### Ollama Not Responding

```bash
# Check ollama
docker-compose ps ollama

# View ollama logs (may take time on first run)
docker-compose logs ollama

# Test ollama API
curl http://localhost:11434/api/tags
```

### Migrations Fail

```bash
# Check database connection
docker-compose logs masaar | grep "migration"

# Verify database exists
docker-compose exec postgres \
  psql -U masaar -l | grep masaar
```

---

## Updates

### Update to Latest Version

```bash
# Pull latest code
git pull origin main

# Rebuild images
docker-compose build --no-cache

# Restart services
docker-compose up -d

# View logs to confirm
docker-compose logs -f masaar
```

---

## Self-Hosted vs SaaS

| Aspect | Self-Hosted | Masaar Cloud |
|--------|-------------|--------------|
| **Cost** | Free (open-source) | AED 300–3000+/month |
| **Data** | Your servers | Hosted in UAE |
| **Updates** | Manual | Automatic |
| **Support** | Community | Premium |
| **Customization** | Full | Limited |
| **Compliance** | You manage | PDPL certified |

Both use the **same code** — only deployment differs.

---

## Support

- **Documentation:** https://github.com/maidulcu/masaar-crm
- **Issues:** https://github.com/maidulcu/masaar-crm/issues
- **Community:** Discussions on GitHub

For SaaS hosting and support:
- Contact: support@masaar.cloud
- Website: https://masaar.cloud
