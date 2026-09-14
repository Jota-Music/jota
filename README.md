# Jota

Your music, your rules. Jota is a self-hosted music streaming app for desktop
and mobile: browse Spotify, stream the audio through YouTube, and soon build
your own library without depending on either.

No cloud, no tracking, no subscriptions. Your playlists, your queue, your
listening — all on devices you control.

## Features

- **Two sources, one app.** Browse Spotify and YouTube side by side and play
  either from the same player, queue and search bar. Spotify is optional — the
  YouTube side works on its own.
- **Spotify browsing.** Connect your account and browse playlists, albums,
  artists, tracks and other users' public playlists, with search across all of
  them.
- **YouTube as a first-class source.** Search videos and playlists, save
  playlists to your home shelf, and paste a YouTube link or ID to jump straight
  to it. Playlists behave like any other: open, filter, order and play them.
- **YouTube audio.** Every track is resolved to a YouTube stream at play time —
  using its saved YouTube ID when it has one, or matching by title and artist.
  When a match is wrong you can link the correct YouTube ID by hand.
- **Full player.** Play/pause, seek, volume, shuffle and repeat (off/all/one),
  plus a queue you can add to, reorder, move after the current track and remove.
  Upcoming tracks are preloaded so playback stays smooth.
- **Native media controls.** Playback integrates with the OS media session, so
  media keys, lock-screen and notification controls work on desktop and mobile.
- **Listen together.** Sync playback across devices through your own WebSocket
  relay: queue, play/pause, position, shuffle and repeat. One person hosts the
  room and shares the code, the rest tune in.
- **Cross-platform.** Built with Wails 3 — native desktop on Linux, Windows and
  macOS, plus Android (and iOS) from the same codebase. The window is frameless
  with custom controls, remembers its size and position, and can stay on top.
- **Self-hosted by design.** The relay is a small Go binary you can run
  anywhere — Docker, a VPS, your home server.
- **Private by default.** Data stays local. No accounts, no telemetry, no
  third-party servers.

## Requirements

- Go 1.26+
- Bun 1.x (frontend)
- Linux dev libraries: `webkit2gtk-4.1`, `gtk+-3.0`, `vorbis`, `flac`, `mpg123`, `alsa`
- `wails3` CLI at `~/go/bin/wails3`
- JDK ≤ 24 for Android builds (Gradle 9)

## Usage

```bash
WEBKIT_DISABLE_DMABUF_RENDERER=1 wails3 dev    # live reload + vite dev server (:9245)
wails3 task build                              # binary -> bin/jota
wails3 task package                            # .deb + .rpm + archlinux -> bin/
```

## Listen together

The app can sync playback across multiple devices. It's optional: it needs a
WebSocket relay that each user hosts wherever they want.

- One person hosts the room and shares the code; the rest join and listen to the
  same thing — queue, play/pause, position, shuffle and repeat. Late joiners and
  reconnects catch up from the last cached state.
- Audio stays aligned across the room as devices keep their clocks in sync.
- Rooms can be protected with an optional password, and the relay itself can
  require a shared auth token.
- The relay is `github.com/Jota-Music/relay`: a small Go binary, self-hostable
  behind TLS (`wss://`) or with `docker run -e PORT=8080 -p 8080:8080 ghcr.io/jota-music/relay`.
- In the app, open "Listen together", paste the relay URL and the room code.

## Roadmap

- **Internal playlists.** Search and create playlists stored locally, without
  depending on Spotify or YouTube.
- **More sources.** Additional music sources beyond Spotify and YouTube.

## Disclaimer

Jota is provided "as-is" for personal and educational use. This software
interfaces with Spotify's public OAuth API and YouTube's undocumented innertube
API for audio playback.

- **Spotify**: uses the standard OAuth flow via
  [go-librespot](https://github.com/Jota-Music/go-librespot), a fork of
  [devgianlu/go-librespot](https://github.com/devgianlu/go-librespot). Users
  should have a valid Spotify account. Jota does not bypass Spotify's access
  controls.
- **YouTube**: audio streams are fetched via YouTube's undocumented innertube
  API. This usage is not endorsed by YouTube/Google and may violate their Terms
  of Service. Users are responsible for ensuring their use complies with
  applicable laws and platform policies. Respect the rights of content owners
  and use music streaming features in accordance with your own subscriptions
  and regional laws.

Jota does not distribute, cache or monetize any copyrighted content. All music
metadata and audio streams are fetched at runtime and belong to their
respective rights holders. The developers assume no liability for misuse of
this software. Use at your own discretion.

## Documentation

Full architecture, Spotify auth, bindings and conventions in `AGENTS.md`.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.