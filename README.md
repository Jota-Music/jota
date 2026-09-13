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

## Documentación

Arquitectura completa, auth de Spotify, bindings y convenciones en `AGENTS.md`.