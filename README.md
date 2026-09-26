# Jota

<p align="center">
  <img src="assets/banner.png" alt="Jota — your music, your rules.">
</p>

> Your music, your rules.

Developed by [salvadorsru](https://github.com/salvadorsru) — a hobby project,
not affiliated with Spotify, Google or any music label.

A self-hosted music streaming app for desktop and mobile. Browse Spotify, play
the audio through YouTube, and start a room over a relay you own.

**No cloud. No tracking. No subscriptions.**

[![CI](https://github.com/Jota-Music/jota/actions/workflows/ci.yml/badge.svg)](https://github.com/Jota-Music/jota/actions/workflows/ci.yml)
[![License: GPL-3.0](https://img.shields.io/badge/license-GPL--3.0-4ea94b.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26-00add8.svg)](go.mod)

## Features

- **Two sources, one player.** Browse Spotify and YouTube side by side and
  play either from the same queue and search bar. Spotify is optional — the
  YouTube side works standalone.
- **Browse Spotify.** Log in to browse playlists, albums, artists, tracks and
  other users' public playlists, with search across all of them — type a song,
  an artist or paste a link.
- **YouTube first-class.** Search videos, playlists and channels, save them to
  your home shelf, or paste a link to jump straight in. Tracks resolve to a
  YouTube stream at play time by saved ID or title/artist match — fix a wrong
  match by hand.
- **Song radio.** Turn any Spotify track into a station of similar songs and
  play it like any other playlist.
- **Full player.** Play/pause, seek, volume, shuffle and repeat, plus a queue
  you can add to, reorder and prune. Upcoming tracks preload so playback stays
  smooth, and Jota picks up where you left off after a reload or a restart.
- **Native media controls.** Media keys, lock-screen and notification controls
  work on desktop and mobile.
- **Rooms.** Save a room, see who is listening, and sync queue, play/pause,
  position, shuffle and repeat across devices over your own WebSocket relay
  ([`Jota-Music/relay`](https://github.com/Jota-Music/relay)).
- **English and Spanish.** Switch the interface language from settings.
- **Cross-platform.** Built with Wails 3 — native desktop on Linux, Windows
  and macOS, plus Android, from one codebase.
- **Private by default.** Data stays local. No accounts, no telemetry, no
  third-party servers — save the app log from settings when reporting a bug.
- **Custom playlists.** Create your own local playlists and mix Spotify and
  YouTube tracks in them. Reorder by drag, see a mosaic cover, and broken
  tracks are flagged instead of breaking the list.
- **Following.** A shelf merged from your Spotify follows, YouTube channels and
  local profiles, shown alongside Playlists and Rooms on the home tab. Follow
  artists, channels or profiles to jump back easily.
- **Liked Songs** surfaced in the home shelf when browsing Spotify.
- **Discord rich presence.** Optional, off by default: show what you are
  listening to in your Discord status.
- **Auto-update.** Jota checks GitHub releases on launch and installs itself
  in place (AppImage, macOS `.app`, Windows per-user, Android APK) with
  SHA256SUMS verification and progress; other channels fall back to the release
  page.

## Screenshots

<p align="center">
  <img src="assets/screenshot-home.png" width="640" alt="Home — saved playlists from Spotify and YouTube">
</p>

<p align="center">
  <img src="assets/screenshot-search.png" width="640" alt="Search across Spotify and YouTube">
</p>

<p align="center">
  <img src="assets/screenshot-player.png" width="640" alt="Full player">
</p>

## Get started

Requirements: Go 1.26+, Bun and `wails3`; on Linux also `webkit2gtk-4.1` and
`gtk+-3.0`. Android builds need JDK ≤ 24, macOS needs Xcode CLT.

```bash
WEBKIT_DISABLE_DMABUF_RENDERER=1 wails3 dev    # live reload (vite on :9245)
wails3 task build                              # binary → bin/jota
wails3 task package                            # .deb + .rpm + archlinux → bin/
wails3 task android                            # release APK → bin/jota.apk
```

## Download

Tagged releases ship:

| Platform | Artifacts |
|----------|-----------|
| Linux | `.deb`, `.rpm`, `.AppImage`, `.flatpak`, `PKGBUILD` `(AUR)` |
| Android | `.apk` |
| Windows | NSIS `.exe` |
| macOS | universal `.dmg` |

On AppImage, macOS `.app`, Windows per-user and Android, Jota checks for
updates on launch and installs them in place; the download above is
also where those channels fetch their next version.

> [!WARNING]
> The Windows `.exe` and macOS `.dmg` are unsigned and may be flagged or blocked
> by SmartScreen and Gatekeeper.

## Cloudflare WARP

Running Jota behind Cloudflare WARP can make playback and API calls sluggish:
the free client routes all traffic through Cloudflare's edge (MTU 1300 and extra
RTT per request), and audio streams ride the tunnel too. On this repo's Linux
setup, `apresolve.spotify.com` connected in ~6s through the tunnel vs ~0.03s
direct.

On a personal (non-managed) WARP client, exclude the music domains so their
traffic goes direct:

```bash
warp-cli tunnel host add "*.spotify.com"
warp-cli tunnel host add "*.scdn.co"
warp-cli tunnel host add "*.youtube.com"
warp-cli tunnel host add "*.googlevideo.com"
warp-cli tunnel host add "*.ytimg.com"
warp-cli disconnect          # re-negotiate the tunnel
warp-cli connect
warp-cli tunnel host list    # confirm the exclusions are stored
```

Host wildcards cover the rotating IPs underneath (Spotify access points and
`googlevideo` hosts), so re-runs are not needed. The local relay (`127.0.0.1`)
never goes through the tunnel. On Zero Trust-managed clients, split tunnels are
controlled by org policy (dashboard/MDM), not `warp-cli`.

## Roadmap

- **More sources** beyond Spotify and YouTube.

## Disclaimer

Jota is an independent, unofficial client. It is not affiliated with,
endorsed or sponsored by Spotify AB, Google LLC or any music label; the
names "Spotify" and "YouTube" are used solely to describe interoperability
and remain the property of their respective owners.

Jota signs into Spotify through the platform's OAuth flow, then reads
metadata over its private Connect protocol and plays audio through
YouTube's undocumented client API. It does not bypass Spotify's access
controls or play its audio streams; it never stores, redistributes or
monetizes content — everything is fetched at runtime from the user's own
session and streamed audio belongs to its rights holders.

By using Jota you are responsible for complying with the terms of service
and applicable laws of the platforms you reach through it. Spotify and
Google may restrict, modify or revoke access to their services at any
time; Jota provides no guarantee that any feature will keep working and
is provided as-is under the GPL-3.0 license.

## License

[GPL-3.0](LICENSE) · Copyright (C) 2026 salvadorsru
