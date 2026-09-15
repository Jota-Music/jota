# AGENTS.md

Self-hosted music streaming app — Go + Wails v3 backend, Preact frontend.

## Stack

- Backend: Go + Wails v3 + BadgerDB (`internal/`)
- Frontend: Preact + Vite + Bun + TypeScript (`frontend/`)
- Sources: Spotify (go-librespot) + YouTube (innertube)

## Commands

Prereqs: Go, `wails3` (`~/go/bin/wails3`), `bun`. Linux also needs `webkit2gtk-4.1`/`gtk+-3.0`
and the audio dev libs (`vorbis`, `flac`, `mpg123`, `alsa`) for go-librespot's CGO build.

```bash
WEBKIT_DISABLE_DMABUF_RENDERER=1 wails3 dev   # live reload (vite on :9245)
wails3 task build                             # binary -> bin/jota
wails3 task package                           # .deb/.rpm/archlinux -> bin/
wails3 task android                           # release APK -> bin/jota.apk
wails3 generate bindings -ts -i -clean=true   # regenerate frontend/bindings/
```

Platform tasks live in `build/<os>/Taskfile.yml`. Android needs JDK ≤ 24; the Linux
build uses the `gtk3` tag (webkit2gtk-4.1).

## Releasing

**If** you change anything under `build/linux/**` or `.github/workflows/release.yml`,
**do not iterate through GitHub Actions** — a `v*` tag spends release minutes. Run the
same checks locally first and only tag once they pass:

```bash
wails3 task test:appimage   # build + smoke on debian:12, ubuntu:24.04, fedora:41
wails3 task test:flatpak    # build + smoke the flatpak (needs flatpak-builder)
```

`test:appimage` mirrors the workflow smoke tests (static runtime, GTK/WebKit init,
missing-library message) via podman. Since the AppImage bundles the build host's audio
codecs, an image built on a newer host can legitimately fail on older distros — CI
builds on Ubuntu 24.04. `act` is installed but the workflow runs containers inside the
runner (docker-in-docker), so the direct scripts are the reliable path.

## Linting

```bash
go fmt ./...
cd frontend && bun run typecheck && bun run lint
```

## Env

| Variable | Default | Description |
|----------|---------|-------------|
| `SPOTIFY_CLIENT_ID` | librespot default | Custom Spotify OAuth client ID. |
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
- Add Go↔JS methods to `App` (`internal/app/app.go`); regenerate `frontend/bindings/`,
  never edit it by hand.
- Keep Go `music.*` JSON tags in sync with `frontend/src/lib/music/model`.
- Commits: Conventional Commits in English, imperative (`feat:`, `fix:`, `refactor:`, `docs:`, ...).
