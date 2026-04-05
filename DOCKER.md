# Docker Deployment Guide with Traefik

This guide explains how to deploy Receipt Manager using Docker, Docker Compose, and Traefik as the reverse proxy.

## Why Traefik?

- **🐳 Native Docker Integration**: Auto-discovers containers via labels
- **🔒 Automatic HTTPS**: Let's Encrypt certificates with zero config
- **📊 Built-in Dashboard**: Real-time monitoring of routes, services, and middleware
- **⚡ Dynamic Configuration**: No restarts needed when adding new services
- **🛡️ Rich Middleware**: Rate limiting, circuit breakers, compression, auth

## Architecture

```
                    ┌─────────────────────────────────────┐
                    │           Traefik v3.0              │
                    │    (Reverse Proxy + Load Balancer) │
                    └──────────────┬──────────────────────┘
                                   │
          ┌────────────────────────┼────────────────────────┐
          │                        │                        │
    ┌─────▼─────┐           ┌──────▼──────┐          ┌──────▼──────┐
    │   API     │           │   Worker    │          │  Temporal   │
    │  (x2)     │           │   (x3)      │          │     UI      │
    └───────────┘           └─────────────┘          └─────────────┘
          │                        │                        │
          └────────────────────────┼────────────────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │      Docker Network         │
                    └──────────────┬──────────────┘
                                   │
          ┌────────────────────────┼────────────────────────┐
          │                        │                        │
    ┌─────▼─────┐           ┌──────▼──────┐          ┌──────▼──────┐
    │  PostgreSQL│           │    Redis    │          │   RustFS    │
    │    (16)    │           │    (7)      │          │   (S3)      │
    └─────────────┘           └─────────────┘          └─────────────┘
```

## Quick Start (Development)

1. **Clone and navigate:**
   ```bash
   git clone <repo-url>
   cd receipt-manager
   ```

2. **Set up environment:**
   ```bash
   cp backend/.env.example .env
   # Edit .env with your actual values
   ```

3. **Start with Traefik:**
   ```bash
   make up
   # or: docker compose up -d
   ```

4. **Access services:**
   | Service | URL | Notes |
   |---------|-----|-------|
   | API | http://localhost/api | All API endpoints |
   | Health | http://localhost/health | Health check |
   | Webhooks | http://localhost/webhooks | Bot webhooks |
   | Traefik Dashboard | http://localhost:8080 | Routes, services, middleware |
   | Temporal UI | http://localhost/temporal | Workflow monitoring |
   | RustFS | http://localhost:9001 | Object storage console |

## Traefik Dashboard

The dashboard provides real-time visibility:

```bash
# Open dashboard
make dashboard

# Or manually open: http://localhost:8080
```

**Dashboard Sections:**
- **HTTP Routers**: All configured routes (`/api`, `/webhooks`, etc.)
- **HTTP Services**: Backend services with health status
- **Middlewares**: Rate limiting, CORS, compression
- **TLS**: Certificate status (in production)

## Production Deployment

### 1. Configure Domain

Add your domain to `.env`:
```bash
DOMAIN=your-domain.com
ACME_EMAIL=your-email@example.com  # For Let's Encrypt
```

### 2. Set Strong Secrets
```bash
# Generate JWT secret
JWT_SECRET=$(openssl rand -base64 32)

# Generate database password
POSTGRES_PASSWORD=$(openssl rand -base64 16)

# Generate RustFS password
RUSTFS_ROOT_PASSWORD=$(openssl rand -base64 16)

# Set LLM API key
LLM_API_KEY=sk-...
```

### 3. Deploy
```bash
make prod
# or: docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

Traefik will automatically:
- Obtain Let's Encrypt certificates
- Redirect HTTP → HTTPS
- Apply security headers
- Enable rate limiting

### 4. Update Basic Auth (Optional)
```bash
# Generate password hash for Temporal UI
make htpasswd

# Then add to traefik/dynamic/middlewares.yml
```

## Configuration

### Traefik Static Config (`traefik.yml`)

Global settings, entrypoints, and providers:
- Entrypoints: `:80`, `:443`, `:8082` (ping)
- Certificate resolvers (Let's Encrypt)
- TLS options (modern cipher suites)

### Traefik Dynamic Config (`traefik/dynamic/middlewares.yml`)

Reusable middleware components:

| Middleware | Purpose | Usage |
|------------|---------|-------|
| `security-headers` | Security headers (HSTS, CSP, etc.) | All routes |
| `cors` | Cross-origin requests | API routes |
| `rate-limit-api` | 100 req/min | API endpoints |
| `rate-limit-webhooks` | 60 req/min | Bot webhooks |
| `rate-limit-strict` | 10 req/min | Login, sensitive |
| `compression` | Gzip compression | All routes |
| `circuit-breaker` | Failover protection | API routes |
| `basic-auth` | Basic authentication | Admin endpoints |
| `ip-whitelist` | IP restrictions | Admin access |

### Service Labels (Docker Compose)

Traefik discovers services via labels:

```yaml
labels:
  # Enable Traefik for this container
  - "traefik.enable=true"
  
  # Define router
  - "traefik.http.routers.api.rule=PathPrefix(`/api`)"
  - "traefik.http.routers.api.entrypoints=web"
  - "traefik.http.routers.api.service=api"
  
  # Apply middleware chain
  - "traefik.http.routers.api.middlewares=cors,rate-limit-api,compression"
  
  # Define service
  - "traefik.http.services.api.loadbalancer.server.port=8080"
  - "traefik.http.services.api.loadbalancer.healthcheck.path=/health"
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make up` | Start all services with Traefik |
| `make down` | Stop all services |
| `make dashboard` | Open Traefik dashboard |
| `make traefik-logs` | View Traefik logs |
| `make traefik-debug` | Debug Traefik config |
| `make prod` | Deploy to production with HTTPS |
| `make htpasswd` | Generate basic auth password |
| `make health` | Check service health |

## Docker Compose Commands

```bash
# Development
docker compose up -d                    # Start all
docker compose logs -f traefik          # Follow Traefik logs
docker compose ps                       # List containers

# Production
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Scale workers
docker compose up -d --scale worker=5

# Restart Traefik only
docker compose restart traefik
```

## Troubleshooting

### View Traefik Logs
```bash
docker compose logs -f traefik
```

### Debug Configuration
```bash
# View loaded configuration
curl http://localhost:8080/api/rawdata

# Check specific router
curl http://localhost:8080/api/http/routers/api@docker

# Check services health
curl http://localhost:8080/api/http/services
```

### Common Issues

**Services not appearing in dashboard:**
```bash
# 1. Check if traefik.enable=true is set
docker inspect receipt-manager-api | grep traefik

# 2. Verify network membership
docker network inspect receipt-manager

# 3. Check Traefik can reach Docker socket
docker compose exec traefik ls -la /var/run/docker.sock
```

**HTTPS not working (production):**
```bash
# Check Let's Encrypt logs
docker compose logs traefik | grep -i acme

# Verify domain DNS
dig your-domain.com

# Check certificate status
curl -v https://your-domain.com 2>&1 | grep -i certificate
```

**502 Bad Gateway:**
```bash
# Check backend health
curl http://localhost:8080/api/http/services

# Verify service is running
docker compose ps api

# Check backend logs
docker compose logs api
```

### Reset Everything
```bash
# ⚠️ DELETES ALL DATA
make clean

# Restart fresh
make up
```

## Advanced Configuration

### Custom Middleware Chain

Edit `traefik/dynamic/middlewares.yml`:

```yaml
http:
  middlewares:
    my-custom-chain:
      chain:
        middlewares:
          - security-headers
          - rate-limit-api
          - compression
```

Apply to a route:
```yaml
labels:
  - "traefik.http.routers.myapi.middlewares=my-custom-chain@file"
```

### Add New Route

```yaml
labels:
  - "traefik.http.routers.new-service.rule=Host(`new-domain.com`)"
  - "traefik.http.routers.new-service.tls=true"
  - "traefik.http.routers.new-service.tls.certresolver=letsencrypt"
```

No restart required - Traefik picks it up automatically!

### IP Whitelisting (Production Admin Access)

```yaml
labels:
  - "traefik.http.routers.admin.middlewares=ip-whitelist,basic-auth"
```

Update `traefik/dynamic/middlewares.yml` with your IPs.

## Comparison: Traefik vs Caddy

| Feature | Traefik | Caddy |
|---------|---------|-------|
| Docker Integration | ⭐⭐⭐ Native | ⭐⭐ Via labels |
| Dashboard | ⭐⭐⭐ Built-in | ⭐ Basic |
| Auto HTTPS | ⭐⭐⭐ Yes | ⭐⭐⭐ Yes |
| Config Style | Labels + YAML | Caddyfile |
| Middleware | ⭐⭐⭐ Rich | ⭐⭐ Good |
| Learning Curve | Medium | Low |
| Community | Cloud-native | General |

**Choose Traefik when:**
- You need the dashboard for monitoring
- You're in a Docker environment
- You want dynamic configuration
- You need advanced middleware (circuit breakers, etc.)

**Choose Caddy when:**
- You prefer simplicity
- You're not using Docker
- You want the easiest HTTPS setup

## Security Features

✅ **Automatic HTTPS** - Let's Encrypt certificates  
✅ **Security Headers** - HSTS, CSP, X-Frame-Options  
✅ **Rate Limiting** - Per-endpoint rate limits  
✅ **Basic Auth** - Protected admin endpoints  
✅ **IP Whitelisting** - Restrict access by IP  
✅ **Circuit Breaker** - Failover protection  
✅ **TLS 1.2+** - Modern cipher suites only  

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `DOMAIN` | Prod | Your domain name (e.g., app.example.com) |
| `ACME_EMAIL` | Prod | Email for Let's Encrypt notifications |
| `JWT_SECRET` | Yes | JWT signing secret |
| `LLM_API_KEY` | Yes | LLM provider API key |
| `POSTGRES_PASSWORD` | Prod | PostgreSQL password |
| `RUSTFS_ROOT_PASSWORD` | Prod | RustFS admin password |

## Migration from Caddy

Already have Caddy running? Switching is easy:

1. **Stop Caddy:**
   ```bash
   docker compose stop caddy
   ```

2. **Start Traefik:**
   ```bash
   make up
   ```

3. **Update DNS** if needed (Traefik uses same ports 80/443)

4. **Remove Caddy** when satisfied:
   ```bash
   docker compose rm caddy
   rm Caddyfile
   ```

## Support

- **Traefik Docs**: https://doc.traefik.io/traefik/
- **Docker Docs**: https://docs.docker.com/compose/
- **Issues**: Check logs with `make traefik-logs`
