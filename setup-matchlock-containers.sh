#!/bin/bash
# Run this from the ROOT of your matchlock repo clone (the "matchlock/" folder
# that contains backend-go/, frontend-react/, blockchain/, docs/).
set -e

if [ ! -d "backend-go" ] || [ ! -d "frontend-react" ]; then
  echo "ERROR: run this from the repo root (matchlock/), not from inside a subfolder."
  exit 1
fi

mkdir -p deploy

# ---------- root Dockerfile (combined, for Render) ----------
cat > Dockerfile << 'EOF'
# syntax=docker/dockerfile:1

########################################
# Stage 1: Build frontend (React/Vite/pnpm)
########################################
FROM node:20-alpine AS frontend-builder

RUN corepack enable && corepack prepare pnpm@10.20.0 --activate

WORKDIR /app/frontend
COPY frontend-react/package.json frontend-react/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY frontend-react/ ./

# Same-origin deploy: frontend calls /api on its own domain, Nginx proxies it internally.
ARG VITE_API_URL=/api
ENV VITE_API_URL=${VITE_API_URL}

RUN pnpm run build

########################################
# Stage 2: Build backend (Go)
########################################
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app/backend
COPY backend-go/go.mod backend-go/go.sum ./
RUN go mod download

COPY backend-go/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /matchlock-server ./cmd/keeper

########################################
# Stage 3: Runtime (Nginx + Go binary, supervised)
########################################
FROM alpine:3.20

RUN apk add --no-cache nginx supervisor ca-certificates gettext

# Backend binary
COPY --from=backend-builder /matchlock-server /app/matchlock-server

# Frontend static build
COPY --from=frontend-builder /app/frontend/dist /usr/share/nginx/html

# Nginx template (PORT is substituted at container start, since Render assigns it dynamically)
COPY deploy/nginx.conf.template /etc/nginx/templates/default.conf.template

# Supervisor config to run nginx + backend as one process group
COPY deploy/supervisord.conf /etc/supervisord.conf

# Entrypoint substitutes $PORT into the nginx config, then hands off to supervisord
COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 10000
ENTRYPOINT ["/entrypoint.sh"]
EOF

# ---------- root docker-compose.yml (local testing) ----------
cat > docker-compose.yml << 'EOF'
version: "3.9"

services:
  postgres:
    image: postgres:16-alpine
    container_name: matchlock-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: matchlock
      POSTGRES_PASSWORD: matchlock
      POSTGRES_DB: matchlock
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U matchlock"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: matchlock-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - redisdata:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  backend:
    build:
      context: ./backend-go
      dockerfile: Dockerfile
    container_name: matchlock-backend
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    env_file:
      - ./backend-go/.env
    environment:
      DATABASE_URL: postgres://matchlock:matchlock@postgres:5432/matchlock?sslmode=disable
      REDIS_URL: redis://redis:6379/0
      HTTP_ADDR: ":8080"
      FRONTEND_URL: http://localhost:3000
      CORS_ALLOWED_ORIGINS: http://localhost:3000,http://127.0.0.1:3000
    volumes:
      - ./backend-go/keys:/app/keys:ro
    ports:
      - "8080:8080"

  frontend:
    build:
      context: ./frontend-react
      dockerfile: Dockerfile
      args:
        VITE_API_URL: http://localhost:8080
    container_name: matchlock-frontend
    restart: unless-stopped
    depends_on:
      - backend
    ports:
      - "3000:80"

volumes:
  pgdata:
  redisdata:
EOF

# ---------- backend-go/Dockerfile (local compose only) ----------
cat > backend-go/Dockerfile << 'EOF'
# --- Build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /matchlock-server ./cmd/keeper

# --- Runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /matchlock-server .

EXPOSE 8080
ENTRYPOINT ["./matchlock-server"]
EOF

# ---------- frontend-react/Dockerfile (local compose only) ----------
cat > frontend-react/Dockerfile << 'EOF'
# --- Build stage ---
FROM node:20-alpine AS builder

RUN corepack enable && corepack prepare pnpm@10.20.0 --activate

WORKDIR /app

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .

ARG VITE_API_URL
ENV VITE_API_URL=${VITE_API_URL}

RUN pnpm run build

# --- Runtime stage ---
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
EOF

# ---------- frontend-react/nginx.conf (local compose only) ----------
cat > frontend-react/nginx.conf << 'EOF'
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    location /api/ {
        proxy_pass http://backend:8080/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|svg|ico|woff2?)$ {
        expires 30d;
        add_header Cache-Control "public, no-transform";
    }
}
EOF

# ---------- deploy/nginx.conf.template (used by root Dockerfile) ----------
cat > deploy/nginx.conf.template << 'EOF'
server {
    listen ${PORT};
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:8080/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # SSE support (match events / live score streaming)
        proxy_set_header Connection "";
        proxy_buffering off;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|svg|ico|woff2?)$ {
        expires 30d;
        add_header Cache-Control "public, no-transform";
    }
}
EOF

# ---------- deploy/supervisord.conf ----------
cat > deploy/supervisord.conf << 'EOF'
[supervisord]
nodaemon=true
user=root
logfile=/dev/stdout
logfile_maxbytes=0

[program:backend]
command=/app/matchlock-server
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
priority=1

[program:nginx]
command=nginx -g "daemon off;"
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
priority=10
EOF

# ---------- deploy/entrypoint.sh ----------
cat > deploy/entrypoint.sh << 'EOF'
#!/bin/sh
set -e

# Render assigns PORT dynamically (defaults to 10000 if unset)
export PORT="${PORT:-10000}"

envsubst '${PORT}' < /etc/nginx/templates/default.conf.template > /etc/nginx/http.d/default.conf

exec supervisord -c /etc/supervisord.conf
EOF
chmod +x deploy/entrypoint.sh

echo ""
echo "Done. Files created:"
find Dockerfile docker-compose.yml deploy backend-go/Dockerfile frontend-react/Dockerfile frontend-react/nginx.conf -type f
