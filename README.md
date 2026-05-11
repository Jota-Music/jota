# Jota

A collaborative music playback server. Browse Spotify playlists, stream audio via YouTube, and sync playback in real-time with friends.

## Architecture

- **Backend** — Go with [Fiber v3](https://gofiber.io) HTTP framework
- **Frontend** — [Preact](https://preactjs.com) SPA built with [Vite](https://vitejs.dev) and [Bun](https://bun.sh)
- **Database** — [BadgerDB](https://github.com/dgraph-io/badger) (embedded key-value store)
- **Storage** — [yt-dlp](https://github.com/yt-dlp/yt-dlp) for YouTube audio extraction
- **Browser Automation** — [go-rod](https://github.com/go-rod/rod) for Spotify web token acquisition

## Features

- Browse and search Spotify playlists
- Stream audio from YouTube
- Collaborative rooms with real-time queue sync (WebSocket)
- User authentication with session management
- Spotifiy metadata caching

## Prerequisites

- [Go](https://go.dev) 1.26+
- [Bun](https://bun.sh) 1.x (for frontend development)
- [Podman](https://podman.io) or Docker (for containerized deployment)
- [Chromium](https://www.chromium.org) (for rod token extraction — installed automatically in the container)

## Quick Start (Development)

### Backend

```bash
# Start the Go server (with air for live-reload)
go run main.go
```

The server starts on `http://localhost:3001` by default. Set `PORT` in `.env` to change it.

### Frontend

```bash
cd client
bun install
bun run dev
```

The Vite dev server starts on `http://localhost:5173` and proxies API requests to the Go backend.

## Docker

### Build

```bash
podman build -t jota:latest .
```

### Run

```bash
podman run -d \
  --name jota \
  -p 3002:3001 \
  jota:latest
```

The service is then accessible at `http://localhost:3002`.

### Stop & Remove

```bash
podman stop jota
podman rm jota
```

## API Endpoints

### Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/register` | Register a new user |
| POST | `/api/auth/log-in` | Log in, creates a session |
| POST | `/api/auth/log-out` | Log out, clears the session |
| GET | `/api/auth/user/:id` | Get user by ID |
| GET | `/api/auth/me` | Get the current authenticated user |

### Music

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/music/playlist/:id` | Get a paginated playlist (`?page=0&size=20`) |
| GET | `/api/music/playlist/full/:id` | Get a full playlist (cached 12h) |
| GET | `/api/music/playlists/:user` | List a user's playlists (cached 12h) |
| GET | `/api/music/song/:id` | Get song details |
| POST | `/api/music/link-youtube` | Link a Spotify track to a YouTube video |

### YouTube Audio

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/youtube/audio/:id` | Get the audio stream URL for a song |
| POST | `/api/youtube/audio/:id` | Manually set a YouTube ID for a song |

### WebSocket

| Path | Description |
|------|-------------|
| `/ws/:room` | Real-time collaborative room |

WebSocket actions: `join`, `leave`, `toggle`, `snapshot`, `seek`, `set-visibility`

## Project Structure

```
.
├── main.go                  # Entry point, Fiber app setup
├── Dockerfile               # Multi-stage container build
├── go.mod / go.sum          # Go module dependencies
├── client/                  # Preact SPA frontend
│   ├── package.json
│   ├── bun.lock
│   └── ...
├── internal/
│   ├── api/
│   │   ├── handler/         # Route definitions
│   │   └── controllers/     # Request handlers
│   ├── services/
│   │   ├── spotify/         # Spotify API client + rod token extraction
│   │   ├── youtube/         # yt-dlp audio stream retrieval
│   │   └── user/            # User CRUD
│   ├── auth/                # User domain & repository
│   ├── music/               # Music domain & repository
│   ├── room/                # Collaborative room domain
│   ├── session/             # Session management (BadgerDB)
│   ├── kv/                  # BadgerDB key-value abstraction
│   ├── repositories/        # Service wiring
│   ├── utils/               # Shared utilities
│   └── env/                 # Environment configuration
└── .env                     # Environment variables
```
