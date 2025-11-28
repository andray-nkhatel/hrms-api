# Multi-stage build for HRMS API and Client
# Build context should be the parent directory (Sources)

# Stage 1: Build Vue client
FROM node:20-alpine AS client-builder

WORKDIR /app/client

# Copy client package files
COPY hrmsclient/package*.json ./

# Install dependencies
RUN npm ci

# Copy client source
COPY hrmsclient/ ./

# Build client (production mode - will use relative URLs for API)
# Set mode to production explicitly
ENV NODE_ENV=production
RUN npm run build

# Stage 2: Build Go API
FROM golang:1.21-alpine AS api-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY hrms-api/go.mod hrms-api/go.sum ./

# Download dependencies (will automatically download required Go version if needed)
RUN go mod download

# Copy source code
COPY hrms-api/ ./

# Copy built client files to static directory
COPY --from=client-builder /app/client/dist ./static

# Build Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hrms-api main.go

# Stage 3: Final image
FROM alpine:latest

WORKDIR /app

# Install ca-certificates and wget for health checks
RUN apk --no-cache add ca-certificates tzdata wget

# Copy binary and static files from builder
COPY --from=api-builder /app/hrms-api .
COPY --from=api-builder /app/static ./static

# Expose port
EXPOSE 8070

# Set environment variable for port
ENV PORT=8070

# Run the application
CMD ["./hrms-api"]

