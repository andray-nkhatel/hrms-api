# Quick Start: Docker Deployment

## 🚀 One-Command Start

```bash
cd hrms-api
./docker-start.sh
```

Or manually:

```bash
cd hrms-api

# 1. Create .env file
cat > .env << EOF
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=hrms_db
JWT_SECRET=$(openssl rand -base64 32)
JWT_EXPIRATION_HOURS=24
PORT=8070
EOF

# 2. Start services
docker-compose -f docker-compose.prod.yml up -d

# 3. Access at http://localhost:8070
```

## 📋 Files Created

- `docker-compose.prod.yml` - Production Docker Compose configuration
- `Dockerfile` - Multi-stage build (already existed, updated)
- `.dockerignore` - Files to exclude from Docker build
- `docker-start.sh` - Convenient startup script
- `README_DOCKER.md` - Complete documentation
- `QUICK_START_DOCKER.md` - This file

## 🎯 What Gets Built

1. **Vue.js Frontend** - Built and served as static files
2. **Go Backend API** - Compiled binary
3. **PostgreSQL Database** - Persistent data storage

All in one Docker Compose setup!

## 🔧 Common Commands

```bash
# Start
make docker-up-prod
# or
docker-compose -f docker-compose.prod.yml up -d

# Stop
make docker-down-prod

# View logs
make docker-logs

# Rebuild after code changes
make docker-rebuild

# Check status
make docker-ps
```

## 🌐 Access Points

- **Application**: http://localhost:8070
- **API Swagger**: http://localhost:8070/swagger/index.html
- **Health Check**: http://localhost:8070/health

## 🔐 Default Test Accounts

- **Employee**: NRC=`123456/78/9`, Password=`password123`
- **Manager**: NRC=`987654/32/1`, Password=`password123`
- **Admin**: Username=`admin`, Password=`password123`

⚠️ **Change these in production!**

## 📚 Full Documentation

See `README_DOCKER.md` for complete setup guide, troubleshooting, and production deployment tips.

