# Daemon IPC

The background daemon listens on `127.0.0.1:23333`.

## Endpoints

- `GET /health`
- `GET /config`
- `PUT /config`
- `POST /engine/preview`
- `GET /wordbook`
- `PUT /wordbook`
- `DELETE /wordbook`
- `GET /updates/check`
- `POST /updates/apply`
- `GET /ime/mode`
- `POST /ime/mode`
- `POST /ime/select-char`
- `GET /ime/candidates`
- `POST /agent/compose`

`POST /engine/preview` body:

```json
{ "input": "nihao" }
```

The settings UI uses this IPC directly in development. A Wails v3 shell can host the same React bundle and call the same daemon API or proxy these methods through its Go backend.

`GET /config` and `PUT /config` include `punctuation`, which is normalized to
`full` or `half`. `full` is the default Chinese punctuation mode; `half` keeps
ASCII punctuation while preserving candidate-first commit behavior during active
composition.

They also include `candidatePageSize`, which controls the visible candidates per
page in the native strip and React previews. The default is `7`; values are
clamped to `3..9` so number-key selection remains predictable.

They also include `doublePinyin` and `doublePinyinScheme`. The scheme is
normalized to `xiaohe` or `microsoft`; old configs with only
`"doublePinyin": true` continue to use Xiaohe. The Microsoft scheme is kept
explicit because its `;` key is an `ing` final, so native glue must treat it as
input code instead of the second-candidate shortcut while that scheme is active.

`GET /wordbook` returns learned user word scores. `PUT /wordbook` accepts
`{"userScores":{"reading|text":1000},"merge":true}` for JSON import or
replacement when `merge` is false. `DELETE /wordbook?key=reading%7Ctext` removes
one learned row; `DELETE /wordbook` clears all learned user words and persists
the empty wordbook.

`GET /updates/check` returns the current and latest dictionary manifest version.
`POST /updates/apply` downloads the matching language dictionary from configured
mirror/CDN URLs first and then GitHub, verifies hashes when provided, loads it
into active IME sessions, persists it locally, and returns the applied language
versions.

`GET /ime/mode` returns the current session state, including `mode`.

`POST /ime/mode` body:

```json
{ "mode": "en" }
```

or:

```json
{ "toggle": true }
```

Mode is session-scoped (`zh` or `en`) and switching mode clears the active
composition buffer. This mirrors the native Shift toggle without rewriting the
saved default input mode in `config.json`.

`POST /ime/select-char?index=0&side=first` commits the first character of a
candidate, while `side=last` commits the last character. This mirrors Rime's
common first/last-character candidate action without forcing the Windows TSF
layer to sacrifice its current bracket paging shortcut.

`GET /ime/candidates?start=0&limit=7` returns tab-separated candidate rows:
`display_index`, `text`, `reading`, `score`, `kind`, `source`, and `comment`.
The final three fields are optional metadata; older six-column rows remain valid
for clients that have not adopted candidate comments yet.

`GET /ime/skin` returns a compact pipe-separated native-renderer payload:
`fontFamily|fontSize|accent|surface|text|mutedText|border|highlightText|theme|candidatePageSize`.
Older nine-field payloads are still treated as the default seven-candidate page.

`POST /agent/compose` body:

```json
{ "input": "/rewrite", "context": "optional selected or nearby text" }
```

Response:

```json
{
  "input": "/rewrite",
  "context": "optional selected or nearby text",
  "candidates": ["请润色这段文字：optional selected or nearby text"],
  "items": [
    {
      "text": "请润色这段文字：optional selected or nearby text",
      "intent": "rewrite",
      "action": "agent.rewrite.polish",
      "source": "builtin-agent",
      "context": "optional selected or nearby text"
    }
  ],
  "actions": ["commit", "copy", "open-settings"]
}
```

`candidates` is kept as a legacy string list for simple clients. New clients should
prefer `items`, because the explicit `intent`, `action`, and `source` fields let a
future TSF candidate row, Wails settings panel, or external agent runner decide
whether to commit text, copy a prompt, or open a richer agent workflow.

## Dictionary Hot Updates

The default source is GitHub Releases:

```text
https://github.com/neko233-com/shurufa233/releases/latest/download/dictionary-manifest.json
```

For China-region acceleration, keep GitHub as the canonical source and publish the same release artifacts to one or more configured mirror/CDN base URLs. The daemon tries mirror base URLs before the original dictionary URL.
When `autoCheck` is enabled, the daemon checks the configured manifest in the background after startup and then periodically. When `autoApply` is also enabled, a newer manifest is downloaded, SHA-256 verified when hashes are present, loaded into all active IME sessions, and persisted under the local dictionary directory without requiring the settings panel to stay open.

Large dictionaries can be published as `.json.gz`. The daemon verifies `sha256`
against the downloaded artifact bytes, decompresses gzip when `compression` is
`gzip` or the URL ends with `.gz`, verifies `contentSha256` against the
decompressed JSON when provided, and then persists the decompressed `.json`
atomically.

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
      "sha256": "optional downloaded artifact lowercase hex",
      "compression": "gzip",
      "contentSha256": "optional decompressed JSON lowercase hex"
    }
  ]
}
```

Recommended mirror choices:

- GitHub Releases as source of truth
- GitHub Pages or Cloudflare Pages for public static mirrors
- A China-friendly object storage/CDN bucket that syncs release artifacts
- Optional enterprise mirror URL configured by the user, not hardcoded into the client
