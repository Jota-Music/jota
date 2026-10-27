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
| `DISCORD_CLIENT_ID` | Jota's app | Discord application for rich presence. Forks set their own app ID. |
| `YOUTUBE_API_KEY` | public InnerTube key | Override the YouTube InnerTube API key. |
| `RELAY_API_URL` | _(empty)_ | Dev-only: pin the relay URL, overriding the one saved in the UI. |
| `RELAY_API_TOKEN` | _(empty)_ | Dev-only: relay auth token to pair with `RELAY_API_URL`. |
| `WEBKIT_DISABLE_DMABUF_RENDERER` | `1` | Wayland/DMA-BUF workaround (set in `main.go`). |
| `JOTA_PROFILE_DIR` | _(empty)_ | Dev-only: writes `cpu.pprof` on start and `heap.pprof` on exit to this directory. Profile with `go tool pprof <dir>/cpu.pprof`. |

## Layout

- Backend: `internal/app` (Go↔JS bindings), `internal/music` (domain), local contexts
  (`internal/{playlists,follows,ordering,rooms,update}`), external service adapters
  (`internal/services/{spotify,youtube,discord}`), `internal/{kv,env}`.
- Frontend: features in `frontend/src/lib/{auth,music,sync,shared}`.

## YouTube sign-in

The account's cookies let innertube requests sign with `SAPISIDHASH`, which is what
unlocks age-restricted videos, the bot check and the account's own playlists. The
string lives in the `youtube-auth` bucket and leaves a device only through a file
the user exports (encrypted when a pass is set) or through the `sessions` section
of the sync document, which only travels behind that pass (see Playlist sync). The
UI warns before storing it: Google can ban an account used this way.

- **Linux reads the browser's jar** (`internal/services/youtube/cookies_*_linux.go`).
  Chromium's `Cookies` SQLite file is copied (with `-wal`/`-shm`) before reading, and
  every encrypted row is opened with the passwords the desktop keyring holds:
  `secret-tool` (`chrome_libsecret_os_crypt_password_v2`, label `<Browser> Safe
  Storage`), then `kwallet-query`, then the no-keyring constants `peanuts` and `""`.
  Firefox's `moz_cookies` is plaintext and needs none of this.
- **Chrome's app-bound encryption is not readable.** A `v20` row (Chrome 127+) is
  not `v10`/`v11`, so it fails like a locked keyring and the import answers
  `errKeyring`. Firefox still works, and so does pasting a cookies.txt.
- **Android cannot read a browser jar and has no sign-in of its own.** The app is
  one fullscreen WebView that cannot load an external URL, so a session reaches the
  phone through a cookies.txt it imports, or through the sync document (below).
  `cookies_linux.go` is tagged `!android` for that reason, and `cookies_other.go`
  (`!linux || android`) reports the jar import as unsupported instead of offering a
  button that cannot work.
- **A session can be carried between devices as a file.** `ExportYouTubeCookiesFile`
  writes the stored cookies to a file the user picks (on Android it shares it
  instead), encrypting it when a pass is set; `ImportYouTubeCookies` accepts that
  file back, decrypting it with the pass this device has. The export is the way a
  desktop session reaches a phone.
- A session started from the desktop in-app window is read from the window's own
  cookie store, not from `document.cookie`, which never carries the HttpOnly ones.
  `WebviewCookies` reads WebKitGTK's `cookies.sqlite` (libsoup's plaintext
  `moz_cookies`), WebView2's `Network/Cookies` (AES-256-GCM with the DPAPI key in
  `Local State`) and WKWebView's `Cookies.binarycookies`. macOS needs no SQLite at
  all, so `cookies_sqlite.go` is tagged `!android && !darwin` and the driver stays
  out of those binaries. The store is flushed on
  the webview's own schedule, so `youtube_login.go` polls it for ten seconds and
  falls back to the page it already fetched. Windows and macOS still have no
  *browser* jar reader (`cookies_other.go`), only this one.
- **WebKitGTK does not write `cookies.sqlite` on its own.** The sign-in window calls
  `PersistWebviewCookies` before it opens, pointing the default web context's cookie
  manager at the same path `WebviewCookies` reads back; without that the jar stays in
  memory and only the visible half of the session can be handed over.
- A session is verified against YouTube (`Verify`) before it is kept. A partial one
  still signs requests but cannot open private playlists or the account's own lists,
  which is what `SessionFull` reports to the panel and `youtube.Full` reports in Go.
- **Startup checks the session.** `verifyYouTubeSession` runs in the background
  after `ServiceStartup` and drops a stored session YouTube answers with
  `ErrSession`, so the panel stops claiming it is signed in. The same verdict from
  `GetMyYouTubePlaylists` drops it too, and `dropYouTubeSession` tombstones it so
  the removal reaches the other devices instead of the next merge restoring it.
- **Playback asks anonymously first.** `playerRequest` sends the player call as a
guest and only retries it signed when the answer is `LOGIN_REQUIRED` and cookies
are stored, so the account is spent only where it is needed. `audioURL` then
tries the client that last produced a playable stream first (`youtube-client`
bucket) and keeps the other innertube clients as a fallback, so a session does
not walk the whole list on every track.
- **`build/android/**` comes from the Wails template.** `SyncDocuments`, the hooks
  in `WailsBridge`/`MainActivity`/`WailsJSBridge` and the manifest entry are
  additions on top of it. A template regeneration overwrites them, so re-apply them
  afterwards instead of hand-merging.

## Playlist sync

Autosave keeps every device's library converging through a single JSON document,
`jota.backup.json` (`playlists.BackupFile`), in a folder. On desktop it is a real
directory, so any file-sync tool works. On Android it is a `content://` document
URI from the SAF picker, and there the provider may hand back a **cached copy**
instead of the current file.

- The document lays six sections side by side: `playlists`, `follows`, `saved`,
`order`, `settings` and `sessions`. Each belongs to the package that owns its
records (`internal/{playlists,follows,ordering}/section.go`,
`internal/services/discord/saved/section.go`, `internal/app/backup_settings.go`,
`internal/app/sessions_store.go`), and a section the library has nothing for is
omitted from the file.
- The document is written compact. Measured on a real 4.6 MB library, indentation was 40% of the
bytes, so it is dropped; the sections share metadata, which is what the size is really about
(`internal/backup/size_test.go` pins it). gzip measured 4.4x and is out: it would need a binary
bridge on Android, and cloud providers already compress what they store.
- Follows are one list per install (`internal/follows`), not per account: this
app has no accounts of its own. `saved` and the `following` order scope work the
same way, without an owning account.
- Every record carries the moment it was written, so a background sync takes the
newest copy. An explicit import (`Restore`) lets the file win outright and also
undoes a local deletion. Removals still travel as tombstones — `deleted` in
playlists, `removed` in follows and saved — because otherwise the next device
that still holds the record would bring it back. Order has none: a shelf's list
is saved whole, so the latest whole list wins. Settings reconcile per key, using
the `-at` companion key to date each value.
- Order travels as one entry per shelf, `{"scope": ..., "account": ..., "ids":
[...], "at": ...}`, keyed by scope plus the owning account, so the library order
of two accounts — and the rooms shelf, which has no account — never collide. A
record stored before the scope existed still reads as a bare list of ids.
- Only the settings that make sense on another device travel: the compound
`sync` key (its `import` and `interval`) and `relay`. A folder path belongs to the
device, the sync status is this device's own state, and everything else stays out.
`readSyncSettings` still reads the older `sync-import`/`sync-interval` keys as a
fallback, so an upgrade does not silently reset them.
- `sessions` carries the YouTube cookies and the Spotify credentials, one entry per
source (`youtube`, `spotify`), as an encrypted blob that merges exactly like a
setting: `internal/backup/settings.go` aliases `Sessions` to `Settings`, so
`mergeSettings` is the one merge and the `-at` companion still dates it. Both are
staged into `session:<source>` when they change (`syncYouTubeCookies`,
`syncSpotifySession`) and applied on a pass (`applyYouTubeCookies`,
`applySpotifySession`), which reconnects or re-signs the source on the spot.
- **The sessions only travel behind a pass.** `encryptWithPass` derives a 32-byte
key with PBKDF2-SHA256 (100k iterations, random 16-byte salt) and seals the blob
with AES-GCM under a random 12-byte nonce, prefixed `JOTAENC1:` so a plaintext
value still reads back unchanged. The pass is the `session-pass` setting; a value
written before the sessions existed lives in `youtube-cookie-pass`, which is still
read as a fallback. With no pass nothing is staged, an empty value is a tombstone
and a blob that will not open with the local pass is skipped.
- A partial session never overwrites a full one: `applyYouTubeCookies` compares
`youtube.Full` on both sides and leaves the local cookies alone when the incoming
blob cannot open the account's own lists. The local full session is re-staged at
startup, so it wins the next pass instead. The Spotify copy is applied only when
it differs from the stored credentials, so a pass carrying nothing new does not
disconnect a working session.
- **The relay token is written in cleartext in the file**: the `relay` setting
stores the URL and the token together, so anyone who can read the backup can use
it. The settings UI says so next to the autosave controls.
- The playlists section carries the local playlists plus the playlists Spotify
and YouTube own. Those source-owned snapshots are read from and written back to
the file, but they are never restored as local playlists.
- The document is indexed. A playlist lists its songs as `{"id": ...}` refs, and
the top-level `songs`, `albums` and `artists` maps hold each entity once, keyed by
id (`source:id` for artists), however many playlists reference it. Only what cannot
be derived is stored: a song's `url` and `share`, and the album `url`, are rebuilt
from the reference on read. An album or artist without an id is written inline on
the ref instead of gaining a map entry.
- Reading must rebuild exactly what was written: a bare `{"id"}` ref (an unresolved
or broken track) keeps just its id, `youtubeId` is only written when the reference
does not already carry it, and the artist key prefix keeps a YouTube channel id
from colliding with a Spotify one.
- The autosave switch decides whether the folder is reconciled on focus, app
lifecycle and the interval. With autoimport off a pass still writes the union of
every section, so nothing another device holds is lost, but this device's library
is left untouched. A missing autoimport setting means on: that was the only
behaviour before the switch existed.
- The manual save is offered while a store reports news the last pass did not
write. A carried setting counts as news from the moment it is written (`touch`)
until a pass writes the document out; a merge that does not apply leaves the
store's own value and the mark in place, so the change is still waiting.
- Passes are serialized. `syncFolder` holds `syncMu` because two passes would
overwrite the same temp file, and the triggers now arrive from many callers at
once; the frontend's `autosave` also keeps one run in flight and chains a manual
call behind it instead of dropping it.
- On Android every document call is asynchronous: `readSyncText`, `writeSyncText`
and `syncFileInfoAsync` answer through `window.__jotaSyncCallback`, because a
synchronous SAF or Drive read would freeze the WebView for as long as the
provider takes. `SyncDocuments` (the SAF half), the bridge methods that forward
to it and the `window.__jotaSync` call from `onResume`/`onPause` live under
`build/android/**`, which comes from the Wails template: a template update can
erase them, so re-apply them on top.
- SAF/Drive can serve a cached copy. `frontend/src/lib/music/app/backup.ts` compares
  the copy against our **own last write** (`SyncStatus.writeAt` + a FNV-1a hash of
  what we wrote, both in settings). Bytes that are our own, or a provider that
  reports no timestamp, are safe to write: a merge is a union. Only a copy that is
  not ours and whose `lastModified` did not move after our write is treated as
  stale and merged read-only, so a merge still pending on another device is never
  overwritten. A manual save is the user asking for the file now, so it trusts the
  copy it just read even when the timestamp did not move; automatic passes keep the
  conservative rule.
- Only an applied write moves that baseline. A sync that just pulled must not, or
  the phone would consider every later copy stale and stop pushing forever.
- The reading side decides `fresh` and passes it to `SyncDocument`, which records
  the outcome (and the new baseline when it applied) in `SyncStatus`.
- Export uses the same filename, so saving into the sync folder updates the sync
  document instead of creating a second file. A backup found under an older name
  is adopted and renamed to the canonical one.
- The document itself can be created from the Android picker (`SyncDocuments.create()`,
  `ACTION_CREATE_DOCUMENT`, `application/json`), so a fresh install can bootstrap it
  instead of only opening an existing file.
- The reliable path is a local folder synced by Syncthing (a real directory on
  desktop, a local SAF folder on Android), not a Drive document behind the SAF
  cache.

## Spotify

The session is a go-librespot client of Jota's own, restored from the `spotify-global`
bucket on start and retried by a background watcher when it fails. The Connect device
id is generated once per install and kept in `spotify-device`, because a fresh id on
every launch registers a new device and drops the saved playback state.

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
