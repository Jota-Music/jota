# =========================
# Frontend (Bun build)
# =========================
FROM oven/bun:1 AS frontend

WORKDIR /app/client

# Build args (Dokploy los inyecta aquí)
ARG VITE_API_URL
ARG VITE_WS_URL

# Convertimos a ENV para Vite build
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_WS_URL=$VITE_WS_URL

COPY client/package.json client/bun.lockb* ./
RUN bun install --frozen-lockfile

COPY client .

RUN bun run build


# =========================
# Backend (Go build static)
# =========================
FROM golang:1.26-alpine AS backend

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Frontend ya compilado
COPY --from=frontend /app/client/dist ./client/dist

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -extldflags '-static'" -o server .


# =========================
# Runtime
# =========================
FROM alpine:3.20

WORKDIR /app

# Runtime deps
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