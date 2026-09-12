# AGENTS.md

Self-hosted music streaming desktop app — Go backend with Preact frontend, packaged as a Wails desktop binary.

## Stack

- **Backend**: Go + Wails v3 (desktop app) + BadgerDB (embedded key-value store)
- **Frontend**: Preact + Vite + Bun + TypeScript
- **Music Sources**: Spotify (via go-librespot) + YouTube (via innertube API)
- **Spotify Auth**: Local callback server inside the desktop app + default librespot client
- **Build**: Wails v3 `Taskfile.yml` + `build/` scaffolding; targets Linux (native), Android (via Gradle), Windows/macOS/iOS (template)

## Commands

```bash
# Desktop app (live-reload)
WEBKIT_DISABLE_DMABUF_RENDERER=1 wails3 dev

# Production build (Linux)
wails3 task build            # output: bin/jota
wails3 dev                   # dev build + vite dev server (port 9245)

# Regenerate TypeScript bindings only
wails3 generate bindings -ts -clean=true   # into frontend/bindings/
```

`wails3` lives at `~/go/bin/wails3` (add to PATH). The Linux build hard-codes the `gtk3` tag (webkit2gtk-4.1); the default compiles against GTK4/webkitgtk-6.0 which crashes gcc 16 on this machine.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBKIT_DISABLE_DMABUF_RENDERER` | `1` | Set to `1` by default in `main.go` to avoid Wayland DMA-BUF protocol errors. |
| `SPOTIFY_CLIENT_ID` | `librespot.ClientIdHex` | Custom Spotify OAuth client ID; required if you want to use your own Spotify dev app with a registered redirect URI. |

## Spotify OAuth Flow (Desktop)

1. User clicks "Log in with Spotify" → `loginSpotify()` is called
2. Backend starts a local callback server on a random localhost port and returns the auth URL pointing to `http://127.0.0.1:{port}/login`
3. Frontend calls `BrowserOpenURL(url)` to open the URL in the system browser
4. User completes Spotify login in the browser; Spotify redirects to the local callback server
5. Backend captures the auth code, exchanges it for tokens, persists session to BadgerDB
6. Frontend polls `SpotifyGetStatus()` to detect the new connection

The default `librespot.ClientIdHex` is registered with Spotify for `http://127.0.0.1:{port}/login` (any localhost port). For production deploys with a custom `SPOTIFY_CLIENT_ID`, set `SPOTIFY_CLIENT_ID` to your own Spotify dev app's client ID with `http://127.0.0.1:<port>/login` as a redirect URI.

## Project Structure

```
main.go                          # Wails entry point; embeds frontend/dist
├── internal/
│   ├── app/
│   │   └── app.go                # App struct; bound as a Wails v3 Service
│   ├── services/
│   │   ├── spotify/             # Spotify OAuth, session, music API
│   │   │   ├── service.go       # SpotifyService; callback server, reconnect, OAuth
│   │   │   ├── music.go         # Playlist / track / search / album / artist API
│   │   │   ├── playlists.go
│   │   │   ├── tracks.go
│   │   │   ├── albums.go
│   │   │   ├── artist.go
│   │   │   ├── search.go
│   │   │   ├── id.go
│   │   │   └── metadata.go
│   │   └── youtube/             # innertube API stream URL retrieval
│   ├── music/                    # Domain types + MusicRepository
│   ├── kv/                       # BadgerDB key-value abstraction
│   ├── session/                  # Spotify session persistence (BadgerDB)
│   └── env/                      # env.Load() configuration
├── frontend/                     # Preact SPA
│   ├── src/
│   │   ├── lib/                  # Feature code (auth, music, shared)
│   │   │   ├── auth/            # Spotify login flow (Wails Browser.OpenURL)
│   │   │   ├── music/           # Playlists, player, queue, audio cache
│   │   │   └── shared/          # Layouts, components, hooks, stores
│   │   └── main.tsx
│   └── bindings/                # Generated Wails v3 bindings (@wailsio/runtime)
├── Taskfile.yml                  # Wails v3 tasks (build/dev/package)
└── build/                        # Platform scaffolding + config.yml (icons, metadata)
```

## Wails Bindings (exposed to JS)

| Binding | Returns | Description |
|---------|---------|-------------|
| `SpotifyGetStatus()` | `{ connected, user }` | Current session state |
| `SpotifyLogin()` | `string` (auth URL) | Starts OAuth; opens local callback server |
| `SpotifyLoginAndWait()` | `void` | Blocks until OAuth completes (success or timeout) |
| `SpotifyReconnect()` | `void` | Reconnect using stored credentials |
| `SpotifyDisconnect()` | `void` | Drop session and clear stored credentials |
| `Search(query, type)` | `SearchResult[]` | Spotify search |
| `GetFullPlaylist(id)` | `Playlist` | Full playlist with all tracks (12h cache) |
| `GetPlaylist(id, page, size)` | `Playlist` | Paginated playlist |
| `GetUserPlaylists(user)` | `PlaylistSummary[]` | User's playlists |
| `GetSong(id)` | `Song` | Track details |
| `GetArtist(uri)` | `ArtistInfo` | Artist + top tracks |
| `GetArtistDiscography(uri)` | `ArtistDiscography` | Artist + albums |
| `GetAlbumTracks(uri)` | `Song[]` | Album tracks |
| `GetYouTubeAudio(spotifyId, search)` | `Audio` | YouTube audio stream URL |
| `SetYouTubeId(spotifyId, youtubeId)` | `void` | Manually link a YouTube ID |

## Spotify OAuth Flow (Desktop)

1. User clicks "Log in with Spotify" → `loginSpotify()` is called
2. Backend starts a local callback server on a random port and returns the auth URL
3. Frontend calls `BrowserOpenURL(url)` to open the URL in the system browser
4. User completes Spotify login in the browser; Spotify redirects to the local callback server
5. Backend captures the auth code, exchanges it for tokens, persists session to BadgerDB
6. Frontend polls `SpotifyGetStatus()` to detect the new connection
7. After Spotify redirect, the user is returned to the desktop app

The default `librespot.ClientIdHex` is registered with Spotify for `http://127.0.0.1:{port}/login` (any localhost port). For production deploys with a custom `SPOTIFY_CLIENT_ID`, set it to your own Spotify dev app's client ID with `http://127.0.0.1:<port>/login` as a redirect URI.

## Code Conventions

- Go: Standard library + Wails v3; no heavy OOP frameworks
- Frontend: Preact with signals; uses `@tanstack/preact-query` for data fetching
- Error handling: Return errors as values; log and continue where appropriate
- Bindings: All Go→JS bridge methods live on `internal/app/app.go` (struct `App`)
- Caching: Spotify playlist/track metadata cached for 12h in BadgerDB (`sessionBucket`)
- Types: Keep Go `music.Song`, `music.Playlist`, etc. JSON tags in sync with `frontend/src/lib/music/model/index.ts`

## Linting

```bash
# Go
go fmt ./...

# TypeScript
cd frontend && bun run typecheck
cd frontend && bun run lint
```

## Common Tasks

- **Add a new binding**: Add a method to `App` in `internal/app/app.go` and rebuild with `wails3 task build` to regenerate TypeScript bindings at `frontend/bindings/`
- **Add a new Spotify endpoint**: Add to `internal/services/spotify/`, expose via `Music` (MusicRepository), call from `App` in `internal/app/app.go`
- **Add a new YouTube helper**: Add to `internal/services/youtube/`, expose via `YouTube` in `App`
