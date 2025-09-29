# syntax=docker/dockerfile:1

# ----------- Frontend build stage -----------
FROM node:20-bullseye AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# ----------- Backend build stage -----------
FROM golang:1.25-bookworm AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go env -w GOTOOLCHAIN=auto \
 && CGO_ENABLED=0 GOOS=linux go build -o nebula_manager

# ----------- Runtime stage -----------
FROM debian:bookworm-slim AS runtime
WORKDIR /app
ARG NEBULA_BINARY_VERSION=1.9.3
ARG NEBULA_BINARY_BASE="https://github.com/slackhq/nebula/releases/download"
ARG NEBULA_BINARY_PROXY_PREFIX=""
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
RUN set -eux; \
    base_url="${NEBULA_BINARY_BASE%/}/v${NEBULA_BINARY_VERSION}/nebula-linux-amd64.tar.gz"; \
    if [ -n "$NEBULA_BINARY_PROXY_PREFIX" ]; then \
      url="${NEBULA_BINARY_PROXY_PREFIX%/}/v${NEBULA_BINARY_VERSION}/nebula-linux-amd64.tar.gz"; \
    else \
      url="$base_url"; \
    fi; \
    curl -fsSL "$url" -o /tmp/nebula.tar.gz; \
    tar -xzf /tmp/nebula.tar.gz -C /usr/local/bin nebula-cert nebula; \
    rm /tmp/nebula.tar.gz
COPY --from=backend-builder /app/nebula_manager /usr/local/bin/nebula_manager
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
ENV NEBULA_DATA_DIR=/data \
    NEBULA_SERVER_PORT=8080
VOLUME ["/data"]
EXPOSE 8080
CMD ["nebula_manager"]
