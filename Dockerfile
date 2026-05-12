# syntax=docker/dockerfile:1.7

# =========================
# Frontend (Bun build)
# =========================
FROM oven/bun:1 AS frontend

WORKDIR /app/client

COPY client/package.json client/bun.lock ./

RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile

COPY client .

RUN bun run build


# =========================
# Backend (Go build static)
# =========================
FROM golang:1.26-alpine AS backend

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY main.go .
COPY internal/ ./internal/

COPY --from=frontend /app/client/dist ./client/dist

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o server .


# =========================
# Runtime
# =========================
FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont \
    ca-certificates \
    ffmpeg \
    python3 \
    py3-pip

RUN wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/local/bin/yt-dlp \
    && chmod +x /usr/local/bin/yt-dlp

ENV ROD_BROWSER=/usr/bin/chromium-browser
ENV CHROME_BIN=/usr/bin/chromium-browser

COPY --from=backend /app/server .

EXPOSE 3001

CMD ["./server"]
