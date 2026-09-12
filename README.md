# Jota

Self-hosted music streaming desktop app. Browse Spotify playlists, stream the
audio through YouTube, and play it with a local decoder.

## Stack

- **Backend** — Go + [Wails v3](https://wails.io) desktop app
- **Frontend** — [Preact](https://preactjs.com) SPA built with [Vite](https://vitejs.dev) and [Bun](https://bun.sh)
- **Database** — [BadgerDB](https://github.com/dgraph-io/badger) (embedded key-value store)
- **Audio** — YouTube stream retrieval via the innertube (`youtubei/v1`) API

## Prerequisites

- [Go](https://go.dev) 1.26+
- [Bun](https://bun.sh) 1.x (frontend)
- Linux desktop with `webkit2gtk-4.1` dev libraries
- [wails3](https://wails.io) CLI at `~/go/bin/wails3`

## Development

```bash
WEBKIT_DISABLE_DMABUF_RENDERER=1 wails3 dev    # live reload + vite dev server (port 9245)
```

## Build

```bash
wails3 task build     # production binary -> bin/jota
wails3 task package   # system packages -> bin/ (AppImage, deb, rpm, ...)
```

## Auth

Log in with Spotify via the system browser. The default librespot client ID
only allows redirects to `http://127.0.0.1:<port>/login`; for your own Spotify
dev app, register that redirect URI and set `SPOTIFY_CLIENT_ID`.

## Project

See `AGENTS.md` for the full architecture, bindings reference, and conventions.