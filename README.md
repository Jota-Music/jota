# Jota

Self-hosted music streaming desktop app. Browse Spotify and stream the audio
through YouTube.

## Requisitos

- Go 1.26+
- Bun 1.x (frontend)
- `webkit2gtk-4.1` dev libraries (Linux)
- `wails3` CLI en `~/go/bin/wails3`

## Uso

```bash
WEBKIT_DISABLE_DMABUF_RENDERER=1 wails3 dev    # live reload + vite dev server (:9245)
wails3 task build                              # binario -> bin/jota
wails3 task package                            # .deb + .rpm + archlinux -> bin/
```

## Escuchar juntos

La app puede sincronizar la reproducción entre varios dispositivos. Es opcional:
necesita un relay WebSocket, que cada usuario hostea donde quiera.

- Una persona hostea la sala y comparte el código; las demás se unen y escuchan
  lo mismo (cola, play/pausa y posición).
- El relay es `github.com/Jota-Music/relay`: un binario Go chico, self-hosteable
  detrás de TLS (`wss://`) o con `docker run -p 8080:8080 ghcr.io/jota-music/relay`.
- En la app, abrí "Listen together", pegá la URL del relay y el código de sala.

## Documentación

Arquitectura completa, auth de Spotify, bindings y convenciones en `AGENTS.md`.