# syntax=docker/dockerfile:1

# --- Frontend Build ---
FROM cgr.dev/chainguard/node:latest-dev AS frontend-builder

USER root
WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci

COPY frontend/ ./
RUN rm -rf dist && npm run build

# --- Backend Build ---
FROM cgr.dev/chainguard/go:latest-dev AS go-builder

ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/home/nonroot/.cache/go-build,uid=65532,gid=65532 \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o rashnu ./cmd/server

# --- Runtime ---
FROM cgr.dev/chainguard/static:latest

WORKDIR /app

COPY --from=go-builder --chown=nonroot:nonroot /app/rashnu ./rashnu
COPY --from=go-builder --chown=nonroot:nonroot /app/migrations ./migrations
COPY --from=frontend-builder --chown=nonroot:nonroot /app/frontend/dist ./frontend/dist

EXPOSE 8080

ENTRYPOINT ["./rashnu"]
