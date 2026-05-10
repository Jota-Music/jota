# =========================
# Frontend (Bun build)
# =========================
FROM oven/bun:1 AS frontend

WORKDIR /app/client

COPY client/package.json client/bun.lockb* ./
RUN bun install --frozen-lockfile

COPY client .

# ❗ VITE_* vendrán de Dokploy en build time
RUN bun run build


# =========================
# Backend (Go build - static)
# =========================
FROM golang:1.26-alpine AS backend

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

COPY --from=frontend /app/client/dist ./client/dist

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -extldflags '-static'" -o server .


# =========================
# Runtime
# =========================
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache \
    ca-certificates \
    ffmpeg \
    curl \
    python3 \
    py3-pip

RUN apk add --no-cache yt-dlp || pip3 install --break-system-packages --no-cache-dir yt-dlp

COPY --from=backend /app/server .

EXPOSE 3000

CMD ["./server"]