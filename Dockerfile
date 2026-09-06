# syntax=docker/dockerfile:1.7
#
# Wails desktop app build environment. Use `wails build` to produce a binary
# that is fully self-contained (frontend assets are embedded via `//go:embed`).
#
# Build:   docker build -t jota-builder .
# Run:     docker run --rm -v "$PWD/build:/out" jota-builder
#
# Produces:  build/bin/jota   (Linux desktop binary)
#
# Notes:
# - The output binary is a desktop app (uses GTK / webkit2gtk). To run it,
#   copy it to a Linux host with the runtime libraries installed, e.g.
#   Ubuntu 24.04 with `webkit2gtk-4.1` (use `-tags webkit2_41`) or Ubuntu
#   22.04 with `webkit2gtk-4.0` (default).
# - Spotify auth: the desktop app opens the system browser. Inside a
#   container there is no system browser; for Spotify login run the binary
#   on a host desktop.

ARG WAILS_VERSION=2.15.0
ARG GO_VERSION=1.26

# =========================
# Frontend deps (Bun)
# =========================
FROM oven/bun:1 AS frontend
WORKDIR /app/client
COPY client/package.json client/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile
COPY client .
RUN bun run build

# =========================
# Backend (Wails build)
# =========================
FROM golang:${GO_VERSION}-alpine AS backend

ARG WAILS_VERSION

RUN apk add --no-cache \
    gcc \
    musl-dev \
    pkgconf \
    libvorbis-dev \
    flac-dev \
    mpg123-dev \
    alsa-lib-dev \
    gtk+3.0-dev \
    webkit2gtk-4.1-dev \
    git

RUN go install github.com/wailsapp/wails/v2/cmd/wails@v${WAILS_VERSION}

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY main.go .
COPY internal/ ./internal/
COPY wails.json ./

COPY --from=frontend /app/client/dist ./client/dist

RUN wails build -tags webkit2_41 -trimpath \
    -ldflags="-s -w" \
    -o /out/jota .

# =========================
# Runtime hint image
# =========================
FROM alpine:3.22 AS runtime

RUN apk add --no-cache \
    gtk+3.0 \
    webkit2gtk-4.1 \
    libvorbis \
    flac \
    mpg123 \
    alsa-lib \
    ffmpeg \
    python3 \
    py3-pip \
    ca-certificates

RUN python3 -m pip install --break-system-packages yt-dlp && \
    apk del py3-pip

WORKDIR /app
COPY --from=backend /out/jota .
VOLUME ["/app/storage"]
CMD ["./jota"]
