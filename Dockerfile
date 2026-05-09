# =========================
# Frontend (Bun build)
# =========================
FROM oven/bun:1 AS frontend

WORKDIR /app/client

COPY client/package.json client/bun.lockb* ./
RUN bun install --frozen-lockfile

COPY client .
RUN bun run build


# =========================
# Backend (Go build - static)
# =========================
FROM golang:1.26-alpine AS backend

WORKDIR /app

# Mejor cache de módulos
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copiar frontend build
COPY --from=frontend /app/client/dist ./client/dist

# 🔥 Build totalmente estático
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -extldflags '-static'" -o server .


# =========================
# Runtime ultra ligero
# =========================
FROM alpine:3.20

WORKDIR /app

# Solo runtime necesario
RUN apk add --no-cache \
    ca-certificates \
    ffmpeg \
    curl

# yt-dlp sin pip (más limpio y rápido si está disponible en repo)
RUN apk add --no-cache yt-dlp || pip3 install --no-cache-dir yt-dlp

COPY --from=backend /app/server .

EXPOSE 3000

CMD ["./server"]