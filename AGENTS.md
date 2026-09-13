# AGENTS.md

Self-hosted music streaming desktop app — Go backend with Preact frontend, packaged as a Wails desktop binary.

## Stack

- **Backend**: Go + Wails v3 (desktop app) + BadgerDB (embedded key-value store)
- **Frontend**: Preact + Vite + Bun + TypeScript
- **Music Sources**: Spotify (via go-librespot) + YouTube (via innertube API)
- **Spotify Auth**: Local callback server inside the desktop app + default librespot client
- **Build**: Wails v3 `Taskfile.yml` + `build/` scaffolding; targets Linux (native), Android (via Gradle), Windows/macOS/iOS (template)

## Commands

Requisitos generales: Go, `wails3` en `PATH` (`~/go/bin/wails3`), `bun` para la frontend.
`WEBKIT_DISABLE_DMABUF_RENDERER=1` evita errores Wayland/DMA-BUF en Linux.

### Linux (nativo)

```bash
WEBKIT_DISABLE_DMABUF_RENDERER=1 wails3 dev   # live-reload; vite dev server en :9245
wails3 task build                             # binario release -> bin/jota
wails3 task package                           # .deb + .rpm + archlinux -> bin/
wails3 task linux:create:deb                  # solo .deb -> bin/
wails3 task linux:create:appimage             # AppImage -> bin/
wails3 task linux:create:rpm                  # solo .rpm -> bin/
```

### Android

**JDK <= 24 obligatorio** (Gradle 9 rechaza Java 26; uno en `~/.local/share/jota-jdk21`).
Antes de cualquier tarea Android:
`export JAVA_HOME=~/.local/share/jota-jdk21 && export PATH="$JAVA_HOME/bin:$PATH"`.

```bash
wails3 task android:package ARCH=arm64      # APK release arm64 (teléfonos) -> bin/jota.apk
wails3 task android:package ARCH=amd64      # APK release x86_64 (emulador) -> bin/jota.apk
wails3 task android:package:fat ARCH=arm64  # APK con arm64 + x86_64 -> bin/jota.apk
wails3 task android:bundle ARCH=arm64       # AAB para Play Store -> bin/jota.aab
wails3 task android:run                      # build debug en emulador
wails3 task android:run:device               # build debug en teléfono (USB) [ARCH=arm64]
wails3 task android:deploy-device ARCH=arm64 # APK release en teléfono (USB)
```

Notas: `package`/`bundle` fuerzan el build con flags de producción
(`-trimpath -ldflags="-w -s"`); `run`/`deploy-emulator` compilan en debug y no
strippan el `.so`. Los builds de Android **limpian** `build/android/app/src/main/jniLibs`
para no arrastrar ABIs/artefactos de builds anteriores.

### iOS (solo macOS)

```bash
wails3 task ios:package:ipa                  # .ipa (requiere IOS_PLATFORM=device + signing)
wails3 task ios:deploy-device                # instala en dispositivos conectados
wails3 task ios:run                          # simulador
```

### macOS (solo macOS)

```bash
wails3 task darwin:build                     # binario -> bin/jota
wails3 task darwin:package                   # .app bundle -> bin/ (macOS)
wails3 task darwin:package:dmg               # .dmg -> bin/
wails3 task darwin:package:universal         # .app universal (arm64 + x86_64)
```

### Windows (solo Windows; cross-compile Go funciona desde Linux)

```bash
wails3 task windows:build                    # -> bin/jota.exe
wails3 task windows:package                  # instala MSIX + crea instalador
wails3 task windows:create:nsis:installer    # instalador NSIS -> bin/
wails3 task windows:create:msix:package      # paquete MSIX -> bin/
```

### Utilidades

```bash
wails3 generate bindings -ts -clean=true   # regenera frontend/bindings/
wails3 task build GOOS=windows             # dispatcher genérico de build (Taskfile raíz)
```

`wails3 task <target>:<tarea>` llama directamente el Taskfile de la plataforma;
`wails3 task <tarea>` desde la raíz rutea por `GOOS` a la plataforma del host.

`wails3` lives at `~/go/bin/wails3` (add to PATH). The Linux build hard-codes the `gtk3` tag (webkit2gtk-4.1); the default compiles against GTK4/webkitgtk-6.0 which crashes gcc 16 on this machine.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBKIT_DISABLE_DMABUF_RENDERER` | `1` | Set to `1` by default in `main.go` to avoid Wayland DMA-BUF protocol errors. |
| `SPOTIFY_CLIENT_ID` | `librespot.ClientIdHex` | Custom Spotify OAuth client ID; required if you want to use your own Spotify dev app with a registered redirect URI. |

## Spotify OAuth Flow (Desktop & Android)

1. User clicks "Log in with Spotify" → `loginSpotify()`
2. Backend starts a local callback server on a random localhost port and returns the auth URL pointing to `http://127.0.0.1:{port}/login`
3. Desktop: frontend calls `BrowserOpenURL(url)`; Android: the OAuth runs inside the app's own WebView (`window.location.href = url`)
4. User completes Spotify login; Spotify redirects to the local callback server
5. Backend captures the auth code, exchanges it for tokens, persists session to BadgerDB
6. Frontend polls `SpotifyGetStatus()` to detect the new connection

The redirect is a loopback URL in all platforms (loopback is a registered redirect for
`librespot.ClientIdHex`, needs no Spotify app and no native deep-link plumbing). On
Android the callback success page redirects the WebView back to the app origin
(`https://wails.localhost/`). For production deploys with a custom `SPOTIFY_CLIENT_ID`,
set `SPOTIFY_CLIENT_ID` to your own Spotify dev app's client ID with
`http://127.0.0.1:<port>/login` as a redirect URI.

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

## Spotify OAuth Flow

1. User clicks "Log in with Spotify" → `loginSpotify()`
2. Backend starts a local callback server on a random port and returns the auth URL pointing to `http://127.0.0.1:{port}/login`
3. Desktop: frontend calls `BrowserOpenURL(url)`; Android: OAuth runs inside the app's own WebView (`window.location.href = url`)
4. User completes Spotify login; Spotify redirects to the local callback server
5. Backend captures the auth code, exchanges it for tokens, persists session to BadgerDB
6. Frontend polls `SpotifyGetStatus()` to detect the new connection
7. After Spotify redirect, the user is returned to the app

Desktop uses the system browser; Android runs the flow in the app's WebView (the default
`xdg-open`-based `BrowserOpenURL` doesn't work on Android). Both share the same loopback
redirect (`http://127.0.0.1:{port}/login`) and the in-process callback server, so no
Spotify app, custom scheme, or native deep-link plumbing is required. For production
deploys with a custom `SPOTIFY_CLIENT_ID`, set `SPOTIFY_CLIENT_ID` to your own Spotify
dev app's client ID with `http://127.0.0.1:<port>/login` as a redirect URI.

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
