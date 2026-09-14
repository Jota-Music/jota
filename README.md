# Jota

Your music, your rules. Jota is a self-hosted music streaming app for desktop
and mobile: browse Spotify, stream the audio through YouTube, and soon build
your own library without depending on either.

No cloud, no tracking, no subscriptions. Your playlists, your queue, your
listening — all on devices you control.

## Features

- **Spotify browsing.** Connect your Spotify account and browse playlists,
  albums, artists and tracks.
- **YouTube audio.** Every track is resolved to a YouTube stream and played
  right in the app.
- **Cross-platform.** Built with Wails 3 — native desktop on Linux, Windows and
  macOS, plus Android (and iOS) from the same codebase.
- **Listen together.** Sync playback (queue, play/pause, position) across
  devices through your own WebSocket relay. One person hosts the room, shares
  the short code, the rest tune in.
- **Self-hosted by design.** The relay is a small Go binary you can run
  anywhere — Docker, a VPS, your home server.
- **Private by default.** Data stays local. No accounts, no telemetry, no
  third-party servers.

## Requirements

- Go 1.26+
- Bun 1.x (frontend)
- `webkit2gtk-4.1` dev libraries (Linux)
- `wails3` CLI at `~/go/bin/wails3`

## Usage

```bash
WEBKIT_DISABLE_DMABUF_RENDERER=1 wails3 dev    # live reload + vite dev server (:9245)
wails3 task build                              # binary -> bin/jota
wails3 task package                            # .deb + .rpm + archlinux -> bin/
```

## Listen together

The app can sync playback across multiple devices. It's optional: it needs a
WebSocket relay that each user hosts wherever they want.

- One person hosts the room and shares the code; the rest join and listen to
  the same thing (queue, play/pause and position).
- The relay is `github.com/Jota-Music/relay`: a small Go binary, self-hostable
  behind TLS (`wss://`) or with `docker run -p 8080:8080 ghcr.io/jota-music/relay`.
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