# AGENTS.md

Self-hosted music streaming app — Go + Wails v3 backend, Preact frontend.

## Stack

- Backend: Go + Wails v3 + BadgerDB (`internal/`)
- Frontend: Preact + Vite + Bun + TypeScript (`frontend/`)
- Sources: Spotify (go-librespot) + YouTube (innertube)

## Commands

Prereqs: Go, `wails3` (`~/go/bin/wails3`), `bun`. Linux also needs `webkit2gtk-4.1`/`gtk+-3.0`.

```bash
WEBKIT_DISABLE_DMABUF_RENDERER=1 wails3 dev   # live reload (vite on :9245)
wails3 task build                             # binary -> bin/jota
wails3 task package                           # .deb/.rpm/archlinux -> bin/
wails3 task darwin:package:dmg                # universal .app + .dmg -> bin/ (macOS only)
wails3 task android                           # release APK -> bin/jota.apk
wails3 generate bindings -ts -i -clean=true   # regenerate frontend/bindings/
```

Platform tasks live in `build/<os>/Taskfile.yml`. Android needs JDK ≤ 24; the Linux
build uses the `gtk3` tag (webkit2gtk-4.1). macOS needs Xcode command line tools
(the go-librespot audio driver links `AudioToolbox`/`CoreAudio`).

## Releasing

**If** you change anything under `build/linux/**` or `.github/workflows/release.yml`:

1. **Build and run the artifacts locally first.** Never push or tag an AppImage/Flatpak
   change you have not executed yourself:

   ```bash
   wails3 task test:appimage   # builds the AppImage, then smokes it on debian:12, ubuntu:24.04, fedora:41
   wails3 task test:flatpak    # builds and smokes the flatpak (needs flatpak-builder)
   ```

2. Only then commit, push, and tag.

**Do not iterate through GitHub Actions** — a `v*` tag spends release minutes and every
`main` push costs a `ci` run. Local first, remote once.

macOS artifacts cannot be produced on Linux, so `build/darwin/**` and the `macos` job
are validated only by the release workflow (or on a Mac). The `.dmg` is ad-hoc signed
but not notarized; notarization needs an Apple Developer account.

`test:appimage` mirrors the workflow smoke tests (static runtime, GTK/WebKit init,
missing-library message) via podman. `act` is installed but the workflow runs containers inside the runner
(docker-in-docker), so the direct scripts are the reliable path.

## Linting

```bash
go fmt ./...
cd frontend && bun run typecheck && bun run lint
```

## Env

| Variable | Default | Description |
|----------|---------|-------------|
| `SPOTIFY_CLIENT_ID` | librespot default | Custom Spotify OAuth client ID. |
| `YOUTUBE_API_KEY` | public InnerTube key | Override the YouTube InnerTube API key. |
| `RELAY_API_URL` | _(empty)_ | Dev-only: pin the relay URL, overriding the one saved in the UI. |
| `RELAY_API_TOKEN` | _(empty)_ | Dev-only: relay auth token to pair with `RELAY_API_URL`. |
| `WEBKIT_DISABLE_DMABUF_RENDERER` | `1` | Wayland/DMA-BUF workaround (set in `main.go`). |

## Layout

- Backend: `internal/app` (Go↔JS bindings), `internal/services/{spotify,youtube,sync}`,
  `internal/music` (domain), `internal/{kv,env}`.
- Frontend: features in `frontend/src/lib/{auth,music,sync,shared}`.

## Conventions

- KISS: simplest thing that works; stdlib/native before dependencies, no speculative abstraction.
- DDD: domain model in `internal/music` (types + `Source` port); services are adapters; dependencies point inward.
- Go: stdlib first; return errors as values.
- Preact + signals; `@tanstack/preact-query` for data fetching.
- Frontend state: `useSignal` for component-local state, `signal()` only at module scope.
  For layout that must be right before first paint, use `useLayoutEffect`, not `useSignalEffect`
  (the latter runs after paint, on the next animation frame).
- Add Go↔JS methods to `App` (`internal/app/app.go`); regenerate `frontend/bindings/`,
  never edit it by hand.
- Keep Go `music.*` JSON tags in sync with `frontend/src/lib/music/model`.
- Commits: Conventional Commits in English, imperative (`feat:`, `fix:`, `refactor:`, `docs:`, ...).
