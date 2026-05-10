# =========================
# Frontend (Bun build)
# =========================
FROM oven/bun:1 AS frontend

WORKDIR /app/client

ARG VITE_API_URL
ARG VITE_WS_URL

COPY client/package.json client/bun.lock ./
RUN bun install --frozen-lockfile

COPY client .

RUN VITE_API_URL=${VITE_API_URL:-http://localhost:3000} \
    VITE_WS_URL=${VITE_WS_URL:-ws://localhost:3000/ws} \
    bun run build


# =========================
# Backend (Go build static)
# =========================
FROM golang:1.26-alpine AS backend

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY --from=frontend /app/client/dist ./client/dist
COPY main.go .
COPY internal/ ./internal/

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS:-$(go env GOOS)} \
    GOARCH=${TARGETARCH:-$(go env GOARCH)} \
    go build -trimpath -ldflags="-s -w" -o server .


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