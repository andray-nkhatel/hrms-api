# HRMS Docker Deployment

Complete Docker setup for self-hosting the HRMS system anywhere.

## Quick Start

```bash
# 1. Navigate to the API directory
cd hrms-api

# 2. Create environment file
cp .env.example .env
# Edit .env and set a strong JWT_SECRET (use: openssl rand -base64 32)

# 3. Start all services
docker-compose -f docker-compose.prod.yml up -d

# 4. Access the application
# Open http://localhost:8070 in your browser
```

## What's Included

- **PostgreSQL Database**: Persistent data storage
- **HRMS API + Frontend**: Combined container serving both backend and frontend
- **Health Checks**: Automatic service health monitoring
- **Auto-restart**: Services restart automatically on failure

## Configuration

### Environment Variables

Create a `.env` file in the `hrms-api` directory:

```env
# Database
DB_USER=postgres
DB_PASSWORD=your-secure-password
DB_NAME=hrms_db
DB_PORT=5432

# JWT
JWT_SECRET=your-very-secure-random-secret-key
JWT_EXPIRATION_HOURS=24

# Server
PORT=8070
```

### Generate Secure JWT Secret

```bash
openssl rand -base64 32
```

## Commands

### Start Services
```bash
docker-compose -f docker-compose.prod.yml up -d
```

### Stop Services
```bash
docker-compose -f docker-compose.prod.yml down
```

### View Logs
```bash
# All services
docker-compose -f docker-compose.prod.yml logs -f

# API only
docker-compose -f docker-compose.prod.yml logs -f hrms-api

# Database only
docker-compose -f docker-compose.prod.yml logs -f postgres
```

### Rebuild After Code Changes
```bash
docker-compose -f docker-compose.prod.yml up -d --build
```

### Complete Reset (⚠️ Deletes all data)
```bash
docker-compose -f docker-compose.prod.yml down -v
docker-compose -f docker-compose.prod.yml up -d --build
```

## Architecture

```
┌─────────────────────────────────────┐
│  Docker Network: hrms-network      │
│                                     │
│  ┌──────────────┐  ┌─────────────┐ │
│  │  PostgreSQL  │  │  HRMS API   │ │
│  │  (Port 5432) │◄─┤  (Port 8070)│ │
│  │              │  │  + Frontend │ │
│  └──────────────┘  └─────────────┘ │
│                                     │
└─────────────────────────────────────┘
```

## Data Persistence

Database data is stored in Docker volume `postgres_data` and persists across container restarts.

### Backup Database
```bash
docker exec hrms-postgres pg_dump -U postgres hrms_db > backup.sql
```

### Restore Database
```bash
docker exec -i hrms-postgres psql -U postgres hrms_db < backup.sql
```

## Production Deployment

### Security Checklist

- [ ] Change `DB_PASSWORD` to a strong password
- [ ] Set `JWT_SECRET` to a strong random string
- [ ] Remove database port mapping (only expose API port)
- [ ] Use HTTPS with reverse proxy (nginx, traefik, etc.)
- [ ] Set up regular database backups
- [ ] Configure firewall rules
- [ ] Change default test account passwords

### Using with Nginx Reverse Proxy

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

## Troubleshooting

### Container won't start
```bash
docker-compose -f docker-compose.prod.yml logs
```

### Port already in use
```bash
# Check what's using port 8070
sudo lsof -i :8070
# Or
sudo netstat -tlnp | grep 8070
```

### Database connection issues
```bash
# Check if postgres is running
docker-compose -f docker-compose.prod.yml ps

# Check postgres logs
docker-compose -f docker-compose.prod.yml logs postgres
```

### Access database directly
```bash
docker exec -it hrms-postgres psql -U postgres -d hrms_db
```

## Default Accounts

After first startup:
- **Employee**: NRC=`123456/78/9`, Password=`password123`
- **Manager**: NRC=`987654/32/1`, Password=`password123`
- **Admin**: Username=`admin`, Password=`password123`

⚠️ **Change these in production!**

## Monitoring

### Check Service Status
```bash
docker-compose -f docker-compose.prod.yml ps
```

### Resource Usage
```bash
docker stats hrms-api hrms-postgres
```

## Updates

```bash
# Pull latest code
git pull

# Rebuild and restart
docker-compose -f docker-compose.prod.yml up -d --build
```

