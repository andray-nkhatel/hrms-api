# HRMS Docker Setup Guide

This guide explains how to run the HRMS system using Docker containers.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- At least 2GB of free disk space
- At least 512MB of RAM available

## Quick Start

1. **Clone the repository** (if not already done):
   ```bash
   git clone <repository-url>
   cd Sources
   ```

2. **Create environment file**:
   ```bash
   cd hrms-api
   cp .env.example .env
   # Edit .env and set your JWT_SECRET and database password
   ```

3. **Start all services**:
   ```bash
   docker-compose -f docker-compose.prod.yml up -d
   ```

4. **Access the application**:
   - Frontend + API: http://localhost:8070
   - Swagger Docs: http://localhost:8070/swagger/index.html
   - Health Check: http://localhost:8070/health

## Configuration

### Environment Variables

Edit the `.env` file or set environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_USER` | postgres | PostgreSQL username |
| `DB_PASSWORD` | postgres | PostgreSQL password (change in production!) |
| `DB_NAME` | hrms_db | Database name |
| `DB_PORT` | 5432 | PostgreSQL port (host) |
| `JWT_SECRET` | (required) | Secret key for JWT tokens (use strong random string) |
| `JWT_EXPIRATION_HOURS` | 24 | JWT token expiration time |
| `PORT` | 8070 | API server port |

### Generate JWT Secret

```bash
# Generate a secure random secret
openssl rand -base64 32
```

## Docker Compose Commands

### Start services
```bash
docker compose -f docker-compose.prod.yml up -d
```

### Stop services
```bash
docker compose -f docker-compose.prod.yml down
```

### View logs
```bash
# All services
docker compose -f docker-compose.prod.yml logs -f

# Specific service
docker compose -f docker-compose.prod.yml logs -f hrms-api
docker compose -f docker-compose.prod.yml logs -f postgres
```

### Restart services
```bash
docker compose -f docker-compose.prod.yml restart
```

### Rebuild and restart
```bash
docker compose -f docker-compose.prod.yml up -d --build
```

### Stop and remove volumes (⚠️ deletes database)
```bash
docker compose -f docker-compose.prod.yml down -v
```

## Architecture

The setup includes:

1. **PostgreSQL Database** (`postgres`)
   - Stores all application data
   - Persistent volume for data
   - Health checks enabled

2. **HRMS API + Frontend** (`hrms-api`)
   - Go backend API server
   - Vue.js frontend (built and served as static files)
   - Single container serving both

## Data Persistence

Database data is stored in a Docker volume `postgres_data`. To backup:

```bash
# Backup
docker exec hrms-postgres pg_dump -U postgres hrms_db > backup.sql

# Restore
docker exec -i hrms-postgres psql -U postgres hrms_db < backup.sql
```

## Production Deployment

### Security Checklist

- [ ] Change `DB_PASSWORD` to a strong password
- [ ] Set `JWT_SECRET` to a strong random string (use `openssl rand -base64 32`)
- [ ] Set `GIN_MODE=release` (already set in docker-compose)
- [ ] Use HTTPS (add reverse proxy like nginx)
- [ ] Restrict database port exposure (remove `DB_PORT` mapping in production)
- [ ] Set up proper firewall rules
- [ ] Enable database backups
- [ ] Review and update CORS settings if needed

### Using with Reverse Proxy (Nginx)

Example nginx configuration:

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8070;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Using with SSL/TLS

1. Use a reverse proxy (nginx, traefik, etc.) with Let's Encrypt
2. Or use Docker with SSL certificates mounted as volumes

## Troubleshooting

### Container won't start

```bash
# Check logs
docker-compose -f docker-compose.prod.yml logs

# Check if port is already in use
netstat -tlnp | grep 8070
```

### Database connection errors

```bash
# Check if postgres is healthy
docker-compose -f docker-compose.prod.yml ps

# Check postgres logs
docker-compose -f docker-compose.prod.yml logs postgres
```

### Rebuild from scratch

```bash
# Stop and remove everything
docker-compose -f docker-compose.prod.yml down -v

# Remove images
docker rmi hrms-api_hrms-api

# Rebuild
docker-compose -f docker-compose.prod.yml up -d --build
```

### Access database directly

```bash
docker exec -it hrms-postgres psql -U postgres -d hrms_db
```

## Default Test Accounts

After first startup, the following test accounts are created:

- **Employee**: NRC=`123456/78/9`, Password=`password123`
- **Manager**: NRC=`987654/32/1`, Password=`password123`
- **Admin**: Username=`admin`, Password=`password123`

⚠️ **Change these passwords in production!**

## Monitoring

### Health Checks

Both services have health checks configured. Check status:

```bash
docker-compose -f docker-compose.prod.yml ps
```

### Resource Usage

```bash
docker stats hrms-api hrms-postgres
```

## Updates

To update to a new version:

```bash
# Pull latest code
git pull

# Rebuild and restart
docker-compose -f docker-compose.prod.yml up -d --build
```

## Support

For issues, check:
- Application logs: `docker-compose -f docker-compose.prod.yml logs hrms-api`
- Database logs: `docker-compose -f docker-compose.prod.yml logs postgres`
- Container status: `docker-compose -f docker-compose.prod.yml ps`

