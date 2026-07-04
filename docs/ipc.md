# Daemon IPC

The background daemon listens on `127.0.0.1:23333`.

## Endpoints

- `GET /health`
- `GET /config`
- `PUT /config`
- `POST /engine/preview`
- `GET /wordbook`
- `GET /updates/check`
- `POST /updates/apply`

`POST /engine/preview` body:

```json
{ "input": "nihao" }
```

The settings UI uses this IPC directly in development. A Wails v3 shell can host the same React bundle and call the same daemon API or proxy these methods through its Go backend.

## Dictionary Hot Updates

The default source is GitHub Releases:

```text
https://github.com/neko233-com/shurufa233/releases/latest/download/dictionary-manifest.json
```

For China-region acceleration, keep GitHub as the canonical source and publish the same release artifacts to one or more configured mirror/CDN base URLs. The daemon tries mirror base URLs before the original dictionary URL.
When `autoCheck` is enabled, the daemon checks the configured manifest in the background after startup and then periodically. When `autoApply` is also enabled, a newer manifest is downloaded, SHA-256 verified when hashes are present, loaded into all active IME sessions, and persisted under the local dictionary directory without requiring the settings panel to stay open.

Manifest shape:

```json
{
  "version": "2026.07.04",
  "channel": "stable",
  "dictionaries": [
    {
      "language": "zh-CN",
      "version": "2026.07.04",
      "url": "https://github.com/neko233-com/shurufa233/releases/latest/download/zh-CN.2026.07.04.json",
      "sha256": "optional lowercase hex"
    }
  ]
}
```

Recommended mirror choices:

- GitHub Releases as source of truth
- GitHub Pages or Cloudflare Pages for public static mirrors
- A China-friendly object storage/CDN bucket that syncs release artifacts
- Optional enterprise mirror URL configured by the user, not hardcoded into the client
