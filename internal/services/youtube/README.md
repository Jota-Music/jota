# YouTube source

How playback survives YouTube's bot check, and the open validation gate.

## Clients

`VISIONOS` preferred, then `ANDROID_VR`, `IOS`, and finally `WEB`. Only `WEB` can
be fed a PO token, matching yt-dlp's `visionos,web` strategy.

## Visitor handling

The visitor token is adopted from innertube responses and persisted, so the
homepage is only touched on a cold start or after a bot check invalidates it. A
failed refresh backs off for `visitorBackoff`. `audioURL` refreshes the visitor
only when no client played, so a healthy fallback never hits the homepage.

## PO token provider

Set `YOUTUBE_POTOKEN_PROVIDER` to a provider base URL (e.g.
`bgutil-ytdlp-pot-provider`). The `WEB` client then calls `POST /get_pot` with a
video-id `content_binding` and sends the token as
`serviceIntegrityDimensions.poToken`.

```bash
npx bgutil-ytdlp-pot-provider
YOUTUBE_POTOKEN_PROVIDER=http://127.0.0.1:4416 wails3 dev
```

## Validation gate

Do not build an in-app token generator (a hidden WebView) until the token is
proven useful on a blocked IP:

```bash
YOUTUBE_POTOKEN_PROVIDER=http://127.0.0.1:4416 \
  go test ./internal/services/youtube/ -run TestWebClientWithProviderIntegration -v
```

Passes → the generator is worth building. Fails → the block is not
PO-token-solvable for these clients and the topic is closed.
