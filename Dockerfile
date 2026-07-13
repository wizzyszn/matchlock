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
ARG VITE_BACKEND_URL=/api
ARG VITE_SOLANA_RPC_URL
ARG VITE_PROGRAM_ID
ARG VITE_CLUSTER
ARG VITE_USDT_MINT
ENV VITE_API_URL=${VITE_API_URL}
ENV VITE_BACKEND_URL=${VITE_BACKEND_URL}
ENV VITE_SOLANA_RPC_URL=${VITE_SOLANA_RPC_URL}
ENV VITE_PROGRAM_ID=${VITE_PROGRAM_ID}
ENV VITE_CLUSTER=${VITE_CLUSTER}
ENV VITE_USDT_MINT=${VITE_USDT_MINT}

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
