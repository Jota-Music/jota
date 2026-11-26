# AGENTS.md

Self-hosted music streaming app — Go + Wails v3 backend, Preact frontend.

## Stack

- Backend: Go + Wails v3 + a JSON document store (`internal/state`, `internal/cache`)
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
cd frontend && bun run typecheck && bun run lint && bun test
```

## Startup budget

Every launch logs `window ready in <ms>`, measured from the top of `main()`. It is
the number to compare across releases: no profiler needed, and a regression shows
up as a bigger number instead of a feeling.

Measured on a release build, warm cache, 5120x1440 Wayland session (2026-10-29):

| phase | time |
|-------|------|
| `main()` → window created (`app.New` + `application.New` + geometry) | 1-2 ms |
| `Run()` → first frame painted (frontend `performance.now()`) | 60-80 ms |
| `main()` → `window ready` (the logged number) | 187-238 ms |

The logged number lands *after* the paint: the window is shown with its solid
background as soon as it is created (`webview_window_linux.go`), so the content
arriving at 60-80 ms is what the user sees. The rest is WebKit's renderer
processor and the runtime handshake, which is not ours. In the AppImage smoke
test under xvfb the same line reads ~240 ms; `build/linux/test-appimage.sh` fails
the release above 3000 ms.

Rules that keep it there:

- The window geometry lives in `window.json` next to the log (`loadWindowState`),
  so building the window never asks the store, and the state document and the
  cache start in the background at `WindowRuntimeReady`. The page does read the
  library before its first render (`warm()` in `main.tsx`): that is a state read
  from memory, which is why it is allowed there, and it is the only read on the
  launch path.
- Nothing on the launch path may wait on the network. The statuses the page asks
  for before painting are the store's; `syncYouTubeStatus` is not awaited because
  resolving the account's own name is a request the first time, and the panel
  fills in when it lands.
- The sync document is never read at launch: the frontend throttles itself for
  `BOOT_GRACE_MS` and the backend's ticker only starts after `syncTick`.
- A blocking call on the GTK thread is a frozen window, not a slow one. The
  clipboard poll is the known one (`gtk_clipboard_wait_for_text`); on desktop it
  only reads on focus.
- Closing the window must end the process, and nothing slow may run there. The
  state document is written by a writer thread, so the exit flushes what is
  pending (`state.Close()`): one write of a small file, nothing else. The cache
  needs no close, every entry lands through a rename (`shutdownBudget` = 3s only
  guards a loop that will not unwind). `build/linux/test-appimage.sh` clicks the
  close button and fails the release if the process outlives it.
- A launch that dies before its window exists is retried in the software
  renderer (`compatRenderer` in `main.go`, driven by the crash marker). Creating
  the WebKit view can segfault inside cgo — nothing to recover from in-process,
  and the alternative was a crash loop. It lasts one session: a launch that opens
  its window clears the marker.
- Blocking the GTK thread freezes the window rather than slowing it. The
  clipboard is the known one: `gtk_clipboard_wait_for_text` waits on the
  compositor's selection bridge from an XWayland client, so `internal/clipboard`
  asks with `gtk_clipboard_request_text` instead and gives up after 400ms.

## Store

Two things, and they are not the same thing.

**The state** is one document: `<config>/jota/state.json`. It holds a named
section per feature (`playlists`, `follows`, `saved`, `order`, `rooms`,
`settings`, `sessions`, `spotify`, `youtube`), each owned by the package that
knows its shape. The document is written by a single writer thread, so a change
is one write whatever it touched and a read is a lookup in memory. It is
replaced through `tmp` + `fsync` + `rename`, so a crash leaves the old document
or the new one, never half. A section this build does not know is kept as it is,
and a document that will not decode is moved aside as `state.json.bad` instead
of being replaced by the next write: it is the only copy of what the app knew,
so nothing collects it.

**The cache** is what the sources answer again: one file per entity, under a
name that says what it is (`spotify/playlists/5zYozysNnBKny8EhXzgVL7.json`,
`youtube/tracks/dQw4w9WgXcQ.json`). An entry carries the fingerprint of the
fetch that produced it, which is what tells a background refresh whether the
source still matches. The cache is never flushed: an entry a crash empties is
fetched again. Each directory has a ceiling (`cache.Limit`), and the oldest
entries go first.

A write lands through `tmp` + `rename`, so a reader never sees half an entry. A
write that dies between the two leaves the temporary file behind;
`cache.Start` sweeps the ones older than the process, and leaves the ones this
run still owns. The temporary name carries four random bytes, so two writers of
the same entry (a background refresh and the request that asked for it) do not
share one.

An entry's file name carries the name it was written under: the last part is
base64 when it fits, which is how `cache.Walk` reads a folder's names without
opening a single file. A name too long for that is hashed, and the entry then
carries its own name inside, which is the only case `Walk` opens.

`internal/state` and `internal/cache` are the two halves. There are no keys and
no buckets: a feature owns a section and writes it whole, and the cache is
addressed by the entity it holds. A channel's playlist list and a playlist
itself share the `youtube/playlists/` folder: a channel id starts with `UC` and
a playlist id never does, so neither overwrites the other.

A document that cannot be read is not an empty document: `state.load` returns the
error and leaves the store unloaded, so the next read tries again and no write
replaces a file that is still there. Only a document that does not decode is
moved aside as `state.json.bad`, and only when the move itself works.

Because a section is written whole, a feature that changes one field of it reads,
changes and writes back under `editSettings`/`editSessions` (`internal/app`): two
of those interleaved — a sync pass recording its status while the user flips a
switch — would otherwise drop one of the two.

The resolved YouTube streams live in `youtube/streams/<videoId>.json`, one file
each, and they carry the life their URL has left: an entry a crash empties is
resolved again, and the ones whose URL died are collected by age
(`youtube.PurgeStreams`), because no googlevideo URL lives longer than a day.
The video a search settled on lives in `youtube/links/<songId>`, one file per
song with no extension, holding the 11 bytes of the video id read directly: it
is a pick, not a decision, so it is cache and can be re-derived. The pins the
user fixed by hand are the only thing that stays in the `youtube` section of the
state (`links`), and they are never overwritten by a resolve. They are dropped
only when the pinned video is gone and a search found a replacement. The
`linksMigrated` flag in that section is what keeps `youtube.MigrateLinks` from
running twice: the map it reads is the one the pins live in now, so a second run
would turn them into picks. The `youtube/session` file holds the tokens alone
(visitor, version, client), so writing one does not rewrite the streams.

The embedded store this app used to keep (`storage/kv`) is imported on the
first launch after an upgrade: `importLegacyStore` runs from `ServiceStartup`,
off the startup path, and emits `store:ready` so the page reads the library
again. With no store left behind it is two stats, so every later launch pays
nothing for it. `cmd/migrate` does the same thing by hand, for a device whose
launch never gets there.

**Badger is not a dependency.** `internal/legacy` reads the old store by its own
format: the manifest (protobuf), the table index (flatbuffers), the blocks
(zstd) and the memtable log, all parsed by hand. Only the zstd decoder
(`klauspost/compress`) is pulled in. The values that lived in the value log are
not read: Badger only writes one there past a megabyte, so a record that big is
cached content and not the device's state — and the store is kept, not pruned,
when the reader skips any, because it cannot prove what it skipped.
`internal/app/testdata/store` is a real Badger store and
`internal/app/testdata/wal` a real Badger memtable log, so the reader is
checked without Badger in the tree.

The memtable log is not an edge case: the old app never closed Badger, so the
newest writes sit in `00001.mem` until the store is opened. A delete is
written there as an entry with no value, so the log scan keeps going on an
empty value and only stops on a key that is not there.

The reader refuses what it does not understand: a store of another format
version or with a checksum that does not match is an error, never a half
import. The import itself never overwrites what the device already holds: the
state it has is newer than the store, so it wins where both have something and
the store only fills the gaps (the playlists are merged by id and by name,
because a local id is generated, the follows by id, the settings and the
sessions are filled key by key). A record the device removed stays removed: the
store still holds the copy the removal dropped, and restoring it would undo the
removal for good, because the store is deleted after one import (`Deleted` for
the playlists, `Removed` for the follows and the saved playlists, the newest
wins).

The links (the video each song plays, `youtube-source`) merge by song rather
than only filling an empty section: a device that has played a few songs since
the upgrade must still take the rest, and a link the device already holds is the
newer pick. The old store held hundreds of them (632 on a real one), so the
silent skip was invisible: `expect` counts every link the store holds, not the
ones the import decided to carry.

The import runs before the tasks that read it. `importLegacy` is a `sync.OnceFunc`
and `syncYouTubeCookies`, `syncSpotifySession`, `verifyYouTubeSession` and the
startup `Connect` call it first: they all read the very sections the import
fills, and reading a 17 MB store takes longer than a state read, so without the
wait an upgrade would stage no session and connect with no credentials until the
next launch.

A local playlist that reaches the library twice under two ids is collapsed by
`collapse` when the import runs: the newest copy wins under the id the library
already uses, and the copy that loses is tombstoned so the backup does not bring
it back. The old store's index keeps copies the document already dropped, which
is how the duplicate arrived in the first place. Only `local:` playlists are
grouped: a source-owned playlist is its own record, and a local one that shares
its name is not a copy of it.

The old directory is moved aside once the import has been read back and found
complete (`verify`), and removed `keepStore` (14 days) later by `pruneStore`. A
count proves what came across, not that the values are right, and the store is
the only copy of what the previous version knew: a mapping bug found a release
later is fixed and imported again, because the import is repeatable and does not
duplicate. A store that comes short, or that the reader refuses, is kept where
it is so the next launch tries again. Nothing on the launch path touches any of
it — the import is a goroutine started from `ServiceStartup`, so the window
never waits for a 17 MB read.

`window.json` is the app's own file: the import writes it only when it is not
there, so a retried import cannot move a window the user already placed.

Only `jota.backup.json` is shared with jondo. Its `state.json`, `playlists/` and
`library.json` are that app's own device files and speak its own Song/Playlist
shapes, so writing them here would only give jondo something it cannot read; the
document is where both apps meet.

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
  (`internal/services/{spotify,youtube,discord}`), `internal/{state,cache,env}`.
- Frontend: features in `frontend/src/lib/{auth,music,sync,shared}`.

## Shutdown

Closing goes through `Window.Close()`, and nothing asks for a quit of its own. The
window's close path is the whole shutdown: Wails destroys the widget on the GTK
thread, unregisters the window and ends the loop, in that order, and the app is
declared to end with its last window (`ApplicationShouldTerminateAfterLastWindowClosed`
is macOS's default *off*, which is why it is set). `Run()` then returns, `main`
flushes the geometry and exits, and SIGTERM/SIGINT/SIGHUP take the same cleanup (see
the signal handler in `main.go`).

`Application.Quit()` from the `WindowClosing` handler is what this used to do, and it
is the bug that made a close sometimes a crash. Wails runs each listener of an event
in its own goroutine, so the quit and the widget's own destruction race on the main
loop's idle queue; a quit that lands first ends the loop with the window still alive,
and GtkApplication then destroys it while it is finalized, after the loop is gone.
WebKit does not survive being torn down there when it still has work in flight. The
process dies on a native thread, so nothing is logged: the marker stays `ready` and
the next launch only reports "ended unexpectedly". The `gtk_widget_destroy` probe is
what shows it, because the process exits cleanly either way, and so does the release
smoke test.

The watchdog that outlives the window (`shutdownBudget`) guards the opposite failure:
a loop that never unwinds would leave the process alive with no window, holding the
single-instance socket every later launch has to hand off to. Closing is best-effort
and must not wait: `discord.Close` takes a `TryLock` and leaves an in-flight connect
to the process exit for exactly this reason.

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
- **The export can be opened with the app on Android.** The manifest registers
  `ACTION_VIEW` for `application/octet-stream`, which is the MIME the system
  reports for the unknown `.enc` extension; "open with" is bound to a type, so
  there is nothing narrower to use. `MainActivity.handleImportIntent` reads the
  URI off the UI thread and keeps the text in the bridge until the page takes it
  (`WailsBridge.takeIncomingFile`), which covers both a cold start and an app
  already running (the wake is `window.__jotaIncoming`). `watchIncoming`
  (`frontend/src/lib/auth/views/stores/youtube.ts`) imports it with the local
  pass on mount and on resume. A destroyed activity also ends the process
  (`MainActivity.onDestroy`): the bridge shutdown stops the Go runtime and the
  native side cannot attach a second activity to it, so a cached process would
  answer the next launch with a fatal exit. `build/android/**` comes from the
  Wails template, so re-apply these additions after a regeneration.
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
- A session is not verified against YouTube when it is kept: the next signed
  request is the judge, and only the sign-in window logs what `Verify` answered
  (`youtube_login.go`). A partial one still signs requests but cannot open private
  playlists or the account's own lists, which is what `SessionFull` reports to the
  panel and `youtube.Full` reports in Go, and `ErrPartialSession` keeps it from
  replacing a full one.
- **A verdict never costs the session.** The same rules in Jondo, adapted to each
  app's language and to when each one asks the network:
  * A verdict only marks it (`state.YouTube.Expired`, read by `youtube.SignedIn`
    and the panel): the cookies stay, and only the user's clear drops them, a
    deletion that travels (`tombstoneSession`).
  * The evidence is a real signed request: the account's own lists, which load
    once every `refresh.Interval`, or a signed stream. A load that answers clears
    the mark (`MarkExpired(false)`). There is no homepage probe: a transport
    failure proves nothing, and a partial set reads as signed out there.
  * The jar the session came from is remembered (`SessionBrowser`, written only by
    a browser import) and a launch refreshes from it (`refreshYouTubeSession`,
    local, no network). A rejected request asks that jar first
    (`recoverYouTubeSession`, no account check: Google rotates the cookies there),
    then any jar that continues the same account (`youtube.SameAccount`), and
    never another account's.
  * The document is the last net: the merge only applies a section that changed,
    so a launch with no cookies applies the staged blob itself
    (`storeYouTubeSession`).
  * An import is judged locally, never remotely: a set without SAPISID or one
    that would replace a full session with a partial one is refused
    (`ErrPartialSession`).
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
is saved whole, so the latest whole list wins. Settings reconcile per key, and
the moment each value was written travels with it, so the newest copy wins.
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
`mergeSettings` is the one merge and the moment it was written still dates it. Both are
staged into `session:<source>` when they change (`syncYouTubeCookies`,
`syncSpotifySession`) and applied on a pass (`storeYouTubeSession`,
`applySpotifySession`), which reconnects or re-signs the source on the spot.
- **The sessions only travel behind a pass.** `encryptWithPass` derives a 32-byte
key with PBKDF2-SHA256 (100k iterations, random 16-byte salt) and seals the blob
with AES-GCM under a random 12-byte nonce, prefixed `JOTAENC1:` so a plaintext
value still reads back unchanged. The pass is the `session-pass` setting; a value
written before the sessions existed lives in `youtube-cookie-pass`, which is still
read as a fallback. With no pass nothing is staged, an empty value is a tombstone
and a blob that will not open with the local pass is skipped.
- **Staging a session is idempotent.** `syncYouTubeCookies` and
`syncSpotifySession` open the blob already staged and return when its plaintext is
identical to what they were about to stage: re-encrypting the same cookies under a
fresh salt would change the blob, mark the store dirty and send a pass for nothing.
Only a blob this pass can open and that matches counts as proof; a blob it cannot
open, or a tombstone, stages as before, so a pass set later still seals the session.
- A partial session never overwrites a full one: `storeYouTubeSession` compares
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
the file, but they are never restored as local playlists. `snapshots`
(`internal/playlists/section.go`) writes them in the order the caller resolved
them, so the document's order is whatever the source answered; whether the APIs
ever vary it is unproven, and it stays untouched until a real log shows a
reorder worth normalising.
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
- On Android the document moves as a **file, never as a string over the bridge**:
`SyncDocuments.stage` copies the provider's document into `filesDir/sync/incoming.json`
and answers its metadata from that same pass, the backend reconciles it into
`outgoing.json` (the staged names are `syncIncoming`/`syncOutgoing` in
`internal/app/sync_folder.go`), and `commit` streams that back with `"wt"` so a
shorter document cannot keep the old tail. A library is megabytes, and a string
through the WebView costs three copies of it on each side. Off Android
`syncStageDir` is empty and the folder is read and written in place.
- On Android every document call is asynchronous: `readSyncText`, `writeSyncText`
and `syncFileInfoAsync` answer through `window.__jotaSyncCallback`, because a
synchronous SAF or Drive read would freeze the WebView for as long as the
provider takes. `SyncDocuments` (the SAF half), the bridge methods that forward
to it and the `window.__jotaSync` call from `onResume`/`onPause` live under
`build/android/**`, which comes from the Wails template: a template update can
erase them, so re-apply them on top and keep the staged filenames matching the
Go constants. Neither the APK nor the Drive picker can be exercised on desktop,
only on a device.
- SAF/Drive can serve a cached copy. The pass compares the copy against our
  **own last write** (`SyncStatus.writeAt` + a FNV-1a hash of what we wrote, both
  in settings). Bytes that are our own, or a provider that reports no timestamp,
  are safe to write: a merge is a union. A copy that is not ours and whose
  `lastModified` did not move after our write is treated as stale: the page reads
  it once more, and if the provider still serves it, writes the union anyway. The
  refusal exists so a merge another device has not uploaded is never silently
  dropped; the retry exists because refusing *forever* would keep this device's
  records out of the file until the provider felt like refreshing, which on Drive
  never happens. A manual save is the user asking for the file now, so it trusts
  the copy it just read even when the timestamp did not move.
- **The write baseline moves only once the file really holds the document.** The
  folder path knows because `backup.Service.Sync` wrote it before answering; the
  platform path cannot know, because the page commits the staged copy afterwards,
  so `SyncWritten` (after the commit) is what records `WriteAt`/`Hash` and clears
  `Stale`/`Problem`. A commit that failed therefore leaves the baseline where it
  was and the next pass writes again, instead of this device trusting bytes the
  provider never received. A sync that only pulled must not move it either.
- The freshness rule lives in the backend, not in the page: `freshCopy`
  (`internal/app/sync_folder.go`) compares the fingerprint of the document this
  device last wrote (`SyncStatus.WriteAt` + `Hash`) against the copy the provider
  served and its `lastModified`. `SyncDocument(lastModified, trust)` applies it,
  leaves the merged document in `outgoing.json`, records what the pass read in
  `SyncStatus`, and answers whether there is something to write back.
- **The Android pass skips the download only on a settled copy.** `settled`
  (`frontend/src/lib/music/app/backup.ts`) compares the provider's stamp against
  `SyncStatus.ReadStamp`, the stamp of the copy the last pass already merged: when
  they match and nothing else is pending, statting the document is the whole pass
  and `SyncTouched` records it. It is deliberately narrow — a manual save, a forced
  overwrite, `SyncPending`, a due snapshot refresh, `Stale`, `Problem`, another
  URI or a missing stamp all keep the full stage/merge/commit path, because each
  of those is a reason to read the copy again. A settled pass reconciles nothing
  and must not move the write baseline.
- A copy that cannot be parsed is a truncated transfer or a write cut short, not a
  library. Both paths rebuild the document from this device's library instead of
  failing every later pass; a document of a version this build does not write is
  not damage, so it fails the pass and is left untouched. The failure is recorded
  in `SyncStatus.Problem`, which the panel shows and which keeps `SyncPending`
  true so the manual save stays offered.
- "Reload backup" in the panel applies the document the location holds, the file
  winning over local deletions (`SyncReload` → `backup.Service.Restore`), and
  leaves the file untouched: it is the read direction. It goes through the same
  queue as a pass, so it never lands in the middle of one staging or committing.
- Export uses the same filename, so saving into the sync folder updates the sync
  document instead of creating a second file. A backup found under an older name
  is adopted and renamed to the canonical one.
- On Android the button opens a sheet with three sources, because no single system
  picker reaches all of them (`SyncDocuments.folder`/`pick`/`create`):
  - **Phone folder** (`ACTION_OPEN_DOCUMENT_TREE`): the tree picker lists the local
    storage and every provider that announces tree support, and the native side
    answers the `jota.backup.json` inside the chosen folder, creating it
    (`application/json`) when the folder holds none, so a fresh install bootstraps
    the document instead of only opening an existing file. What is stored is the
    **document** URI, rebuilt onto the tree (`buildDocumentUriUsingTree`, also for
    what `createDocument` answers, which may be a plain document URI): the
    persistable grant is held on the folder, and a tree grant does not open a loose
    document URI. **Drive never appears here**: its provider announces no tree
    support, and MIUI additionally hides every non-local root from the tree picker.
  - **Open from the cloud** (`ACTION_OPEN_DOCUMENT`) and **create in the cloud**
    (`ACTION_CREATE_DOCUMENT`, `application/json`): the cloud is reached one
    document at a time. With no document stored yet, `EXTRA_INITIAL_URI` starts the
    picker at the Drive provider's first writable root (found by querying
    `buildRootsUri` and its `FLAG_SUPPORTS_CREATE`), so it opens in Drive instead of
    "Recent"; afterwards it starts where the current document already lives.
  - No picker can be summoned programmatically and no grant can be taken without
    one: `takePersistableUriPermission` only succeeds on the URI a system picker
    just answered, which is why every path goes through `startActivityForResult`
    and the grant is what keeps the document readable across launches.
- The reliable path is a local folder synced by Syncthing (a real directory on
  desktop, a local SAF folder on Android), not a Drive document behind the SAF
  cache.

## Android media notification

`navigator.mediaSession` is disabled in the WebView, so the notification, the
lockscreen and the media keys are a native foreground service
(`MediaPlaybackService`). The page drives it through `window.wails.mediaUpdate`
and gets the transport actions back as `media:action`.

**A paused element keeps the stream URL it resolved, and that URL expires.** A
phone left paused is an idle page, not a dead one: measured on a Redmi that sat
paused for three minutes, the media key reached the session, the page ran its
`media:action` handler, and `play()` was accepted with `paused=false` and
`readyState=4` — buffer full, no error — while the clock stood still and nothing
reached the speakers. The stall watchdogs cannot help (they are stopped while
paused), so `resume()` has to notice, and the two ways it does are both in
`lib/music/views/stores/audio.ts`:

- Every resume tries the element first and is held to it: `playing()` (in
  `lib/music/app/recovery`) only counts the position moving past a blip (0.3 s),
  because a dying stream hands over its last buffered tenth of a second and
  stops. The tick samples once a second and a resume gives it 1.8 s, so a full
  second of movement judged healthy starts dead and rebuilt them right after
  they began. A stream that fails that goes to the recovery ladder, which clears
  the cached entry, re-resolves and resumes at the same position.
- That window is `RESUME_MS` (1.8 s), not the full `START_MS`: the stream is
  usually alive, and an eager rebuild on every pause older than 30 s made the
  commonest interaction (pause, come back a minute later, play) take the resolve
  instead of instant. Measured after the change: 3 of 3 resumes at the samples
  250 ms after the key were already playing. A pause longer than
  `LONG_PAUSE_MS` (5 s) turns the loader on while it proves itself, so the button
  never looks dead.

**The card offers what actually exists.** A transport button that answers with
silence reads as a broken card, so the page says whether there is anywhere to
skip to (`next` in the media payload, wired from `canNext` in `audio.ts` through
`wireSkipNext` — `media-session` keeps no dependency on the queue). When there is
not, the service drops `ACTION_SKIP_TO_NEXT` from the `PlaybackState` and the
Next action from the notification, and the card shows two buttons instead of
three. Previous always stays: with nothing behind the playhead it restarts the
track, which is a real action.

**A timer created while the page is hidden never fires.** Measured: a 6 s timeout
and a 500 ms poll created from a paused app stayed silent for twenty-five minutes,
while an interval created before the app left the foreground kept ticking on
schedule. That is why `playing()` schedules nothing: `audio.ts` starts one ticker
at load (`setInterval(tick, 1000)`) and every wait rides it, and why a `play()`
promise is never awaited on its own — an element in that state can leave it
pending for good.

**Past five minutes hidden, the page stops running altogether**, and none of the
above can help: `fetch` never begins, so the stream cannot even be re-resolved.
(Measured at 5.5 min: the action arrived, the page pushed the state it was
leaving, and nothing else happened.) The host is what gets it back.

`emitMediaAction` (`WailsBridge`) asks two questions at once. First, whether the
page can run anything: the `media:action` handler schedules a zero-delay timer
that calls `mediaTick`, and if it does not arrive in 800 ms the page's task queue
is dead (the injected handler runs, nothing it schedules does) and the reload
starts right there — measured, that took the frozen case from 5.1 s to 1.7 s.
Second, whether the action took effect: a 3 s watchdog accepts it only when the
page reports the state it asked for — `playing` for play, `paused` for pause,
anything for the rest, because the page pushes the state it is leaving and would
otherwise answer itself. A silent action reloads the page either way. The
fresh page then **asks** for where it was (`takeMediaResume` in `WailsJSBridge`,
read by `restorePlayback` in `audio.ts`) instead of being told on a timer: only
the page knows when it is ready, and two timings here are traps — injecting too
early finds no page, and `onPageFinished` is far too late for a `restorePlayback`
that runs while the page loads. Three details keep the card steady:

- The host writes the resume in `revive()`, before the reload, because the page
  asks during its own load.
- A recovery reload does not push the track it restored: the resumed position is
  loaded with `prepareSong(song, seconds)` rather than published at zero.
- A push that reports position 0 inside the revive window is dropped, which is
  what the remaining "prepare the element" push would show.

Measured end to end on the phone (card tapped, app untouched): 5.5 minutes
paused, key pressed, playing at the same position in 1.7 s, no 00:00 in between.
MIUI's card does not draw `STATE_BUFFERING` (it keeps the play triangle), but it
**does** draw the notification's own actions: while a recovery runs the middle
action becomes a loading icon (`ic_popup_sync`), which is the feedback the user
sees in the first 300 ms. The state is published as buffering anyway, and a
15 s timeout takes the loader away if the recovery never lands, so the card
cannot promise sound forever. In the app the same rebuild shows the loader instead of a
dead button. `resumePosition` takes the later of the element's clock and the UI's
progress, because the autoplay prime swaps a silent clip into the shared element
on the first gesture and leaves the element at zero.

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
