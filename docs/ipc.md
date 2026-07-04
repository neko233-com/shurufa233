# Daemon IPC

The background daemon listens on `127.0.0.1:23333`.

## Endpoints

- `GET /health`
- `GET /config`
- `PUT /config`
- `POST /engine/preview`
- `POST /engine/associate`
- `GET /wordbook`
- `PUT /wordbook`
- `DELETE /wordbook`
- `GET /phrases`
- `PUT /phrases`
- `DELETE /phrases`
- `GET /rejects`
- `PUT /rejects`
- `DELETE /rejects`
- `GET /pins`
- `PUT /pins`
- `DELETE /pins`
- `GET /profile`
- `PUT /profile`
- `GET /catalog`
- `GET /symbols`
- `GET /updates/check`
- `POST /updates/apply`
- `GET /updates/sources`
- `POST /updates/source`
- `POST /rime/custom`
- `GET /switches`
- `POST /switches/apply`
- `GET /app-rules`
- `PUT /app-rules`
- `POST /app-context/resolve`
- `GET /ime/mode`
- `POST /ime/mode`
- `POST /ime/select-char`
- `POST /ime/candidate-action`
- `GET /ime/candidates`
- `POST /agent/compose`

`POST /engine/preview` body:

```json
{ "input": "nihao" }
```

The preview path accepts apostrophe-separated pinyin such as `xi'an`. The
buffer keeps the visible apostrophe for Microsoft IME-style preedit display,
while lookup collapses it to dictionary readings and adds a high-priority
explicit segmentation candidate when every separated syllable can be resolved.
The same behavior is used by the React/Wails preview strip and
`shurufa-imecli preview "xi'an"`.

It also accepts Rime-style slash symbol prefixes such as `/fs` and `/xh`. The
buffer keeps the slash for preedit display, lookup resolves the imported Rime
symbol code without the slash, and the candidate list is filtered to symbol,
emoji, kaomoji, and agent-style entries so ordinary pinyin words do not leak
into slash-prefixed symbol mode.

The settings UI uses this IPC directly in development. A Wails v3 shell can host the same React bundle and call the same daemon API or proxy these methods through its Go backend.

`POST /engine/associate` body:

```json
{ "context": "你好", "limit": 7 }
```

It clears active composition and returns next-word association candidates in the
normal `State` shape. Rows use `kind=association`, carry a `source`, and are
already converted through the configured `script`. The same association path is
returned after `select` when the committed text has follow-up suggestions, so a
native candidate strip can show WeChat-style post-commit choices without adding
platform-specific prediction logic.

`GET /config` and `PUT /config` include `punctuation`, which is normalized to
`full` or `half`. `full` is the default Chinese punctuation mode; `half` keeps
ASCII punctuation while preserving candidate-first commit behavior during active
composition.

They also include `script`, normalized to `simplified` or `traditional`.
Candidate text returned by preview, paging, selection, and candidate-action
endpoints is already converted for display/commit, while readings and dictionary
metadata stay unchanged. The current converter is a compact built-in mapping so
full OpenCC data can later be hot-updated or swapped in without changing daemon,
CLI, Wails, or native TSF IPC contracts.

They also include `candidatePageSize`, which controls the visible candidates per
page in the native strip and React previews. The default is `7`; values are
clamped to `3..9` so number-key selection remains predictable. `candidateLayout`
controls the candidate strip direction: `horizontal` is the default Microsoft
IME/WeChat-style strip, while `vertical` matches common Rime-style candidate
lists. `showCandidateComments` controls whether native and React candidate
strips display muted annotation text; the candidate payload still carries
comments so dictionary metadata remains available to richer clients.
`associations` controls whether post-commit association candidates are shown.
It defaults to `true`; disabling it keeps selection behavior traditional and
returns an empty candidate page after commit.

They also include `doublePinyin` and `doublePinyinScheme`. The scheme is
normalized to `xiaohe` or `microsoft`; old configs with only
`"doublePinyin": true` continue to use Xiaohe. The Microsoft scheme is kept
explicit because its `;` key is an `ing` final, so native glue must treat it as
input code instead of the second-candidate shortcut while that scheme is active.

They also include key behavior fields: `keyProfile`, `shiftToggleMode`,
`semicolonQuickSelect`, `quoteQuickSelect`, `bracketPageKeys`,
`minusEqualPageKeys`, and `commaPeriodPageKeys`. `keyProfile=wechat` and
`keyProfile=microsoft` keep the default WeChat/Microsoft feel: Shift toggles
Chinese/English, `;` and `'` select the second and third candidates, and `[]`
plus `-=` page candidates. `keyProfile=rime` keeps Shift and bracket/minus-equal
paging while enabling `,`/`.` Rime-style paging and disabling semicolon/quote
quick select. `keyProfile=custom` honors each boolean switch directly. The
Windows TSF layer reads these values from the shared config file, so shortcut
changes do not require new C++ builds.

`POST /rime/custom` imports a Rime `*.custom.yaml` patch and maps common Rime
settings into the shared config. It accepts either raw YAML or JSON:

```json
{ "yaml": "patch:\n  schema_list:\n    - schema: double_pinyin_flypy\n  menu/page_size: 8\n" }
```

Supported patch fields include `schema_list`, `schema/schema_id`,
`menu/page_size`, `style/horizontal`, `style/vertical`, `switches`,
`translator/enable_sentence`, `punctuator/import_preset`,
`punctuator/half_shape`, `key_binder/import_preset`, `key_binder/bindings`, and
`ascii_composer/switch_key/Shift_L|Shift_R`. The response contains the applied
config plus `applied` and `warnings` arrays so GUI, CLI, Wails, and native
debugging tools can show which Rime fields were accepted without adding
platform-specific glue.

`GET /wordbook` returns learned user word scores. `PUT /wordbook` accepts
`{"userScores":{"reading|text":1000},"merge":true}` for JSON import or
replacement when `merge` is false. `DELETE /wordbook?key=reading%7Ctext` removes
one learned row; `DELETE /wordbook` clears all learned user words and persists
the empty wordbook.

`GET /phrases` returns fixed user phrases stored in `user-phrases.json`. These
entries are separate from learned word scores and are loaded as
`kind=phrase`, `source=user-phrase` candidates with high default weight, matching
Rime `custom_phrase.txt` expectations. `PUT /phrases` accepts
`{"entries":[{"reading":"msd","text":"马上到！","weight":60000}],"merge":true}`
or a single `reading`/`text` pair. `DELETE /phrases?key=msd%7C马上到！` deletes
one phrase; `DELETE /phrases` clears all fixed user phrases. The CLI mirrors
this through `shurufa-imecli phrases list|add|import|export|delete|clear`.

`GET /rejects` returns hidden candidate rows stored in `user-rejects.json`.
These rows use the same `reading`/`text` shape as dictionary entries and are
filtered out by the Go core before ranking candidates. `PUT /rejects` accepts
`{"entries":[{"reading":"ceshi","text":"错词"}],"merge":true}` or a single
`reading`/`text` pair. `DELETE /rejects?key=ceshi%7C错词` restores one hidden
candidate; `DELETE /rejects` restores all hidden candidates. The CLI mirrors
this through `shurufa-imecli rejects list|add|import|export|delete|clear`.

`GET /pins` returns pinned candidate rows stored in `user-pins.json`. These rows
use the same `reading`/`text` shape and receive a large ranking bonus in the Go
core, so a user-preferred candidate stays near the top without rewriting the
dictionary. Pinning a candidate removes any matching reject row. `PUT /pins`
accepts `{"entries":[{"reading":"nihao","text":"你好"}],"merge":true}` or a
single `reading`/`text` pair. `DELETE /pins?key=nihao%7C你好` cancels one pin;
`DELETE /pins` cancels all pins. The CLI mirrors this through
`shurufa-imecli pins list|add|import|export|delete|clear`.

`GET /profile` exports a migration bundle containing shared config, learned
`userScores`, fixed `phrases`, hidden `rejects`, pinned `pins`, and per-section
`counts`. `PUT /profile` accepts the same bundle and applies it with
`merge=true` by default; set `merge=false` for a full local replacement. This is
the daemon-level contract for Rime-style user data backup, cross-device restore,
and later cloud sync. The CLI mirrors it through
`shurufa-imecli profile export [profile.json]` and
`shurufa-imecli profile import profile.json [--replace]`.

`GET /catalog` returns the shared emoji, kaomoji, symbol, and agent resource
catalog. Query parameters are `kind=all|emoji|kaomoji|symbol|agent`,
`q=<search>`, and `limit=<n>`. `/symbols` is an alias for the same handler.
Imported Rime `symbols.yaml`, `symbols.custom.yaml`, and OpenCC emoji rows are
ordinary dictionary entries tagged with `kind`, so they appear here without
special per-platform code. Slash-prefixed queries such as `q=/fs` are normalized
to the stored Rime symbol code:

```json
{
  "kind": "symbol",
  "query": "fs",
  "count": 3,
  "entries": [
    {
      "reading": "fs",
      "text": "℃",
      "kind": "symbol",
      "source": "builtin-symbols",
      "comment": "符号",
      "weight": 6200
    }
  ],
  "updatedAt": "2026-07-05T00:00:00Z"
}
```

The CLI mirrors this through
`shurufa-imecli symbols [all|emoji|kaomoji|symbol|agent] [query] [--limit N]`.

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

`GET /switches` returns Rime-style runtime switches derived from the shared
config. `POST /switches/apply` accepts:

```json
{ "id": "ascii_mode", "value": true }
```

Omit `value` or pass `"action":"toggle"` to invert the current switch. Current
switch ids are `ascii_mode`, `ascii_punct`, `simplification`,
`candidate_comments`, `associations`, and `vertical_candidates`. The daemon
persists the resulting config and refreshes active sessions, giving Wails/React,
CLI, and future native switch panels the same behavior without platform glue.

`GET /app-rules` returns the app-aware behavior rules from `config.appRules`.
`PUT /app-rules` accepts `{"rules":[...]}` and persists normalized rules. The
default rules cover password fields, terminal/command-line apps, game/esports
contexts, and IDE/code editors.

`POST /app-context/resolve` accepts the focus context that native TSF/IMKit glue
can gather on focus changes:

```json
{
  "processName": "WeGame.exe",
  "exePath": "D:\\App\\WeGame\\wegame.exe",
  "windowTitle": "WeGame",
  "windowClass": "",
  "gameMode": true
}
```

The response contains the matched rule, a derived config, and direct hot-path
fields such as `mode`, `punctuation`, `candidateLayout`, `disableCandidates`,
and `disableLearning`. This lets games, terminals, password boxes, and agent
automation use English/half-width/no-candidate behavior without putting app
lists or Rime-style recognizer logic inside C++.

`POST /ime/select-char?index=0&side=first` commits the first character of a
candidate, while `side=last` commits the last character. This mirrors Rime's
common first/last-character candidate action without forcing the Windows TSF
layer to sacrifice its current bracket paging shortcut.

`POST /ime/candidate-action` accepts the same JSON action shape as the native
ABI command bus:

```json
{ "action": "next-page", "start": 0, "limit": 7 }
```

Supported actions include `view`, `next-page`, `prev-page`, `first-page`,
`last-page`, `select`, `associate`, `pin`, `forget`, `first-char`, `last-char`, and `select-char`. Selection
can use either an absolute `index` or page-relative `displayIndex` plus `start`.
The response includes `state`, optional `committed`, and a rich `candidates`
page with metadata. This keeps React/Wails previews, daemon fallback clients,
and native C++ glue aligned on the same candidate event model.
`pin`/`pin-candidate` writes `user-pins.json`, returns `pinned`, marks matching
candidate rows with `pinned=true`, and leaves the active composition buffer
visible for continued editing.
`forget`/`reject`/`delete-candidate` hides the selected candidate, writes
`user-rejects.json`, removes any learned score for the same `reading|text`, and
returns `rejected` plus the refreshed candidate page.
The CLI mirrors this endpoint through `shurufa-imecli candidates <input>
[action]`, which first previews the input and then posts the action payload.
It can therefore be used for separator and paging smoke checks, for example
`shurufa-imecli candidates "xi'an" view`.
For association-only checks, use `shurufa-imecli associate "你好"` or
`shurufa-imecli candidates associate --context "微信"`.

`GET /ime/candidates?start=0&limit=7` returns tab-separated candidate rows:
`display_index`, `text`, `reading`, `score`, `kind`, `source`, `comment`, and
`pinned`. The final four fields are optional metadata; older six-column or
seven-column rows remain valid for clients that have not adopted candidate
comments or pin markers yet.

`GET /ime/skin` returns a compact pipe-separated native-renderer payload:
`fontFamily|fontSize|accent|surface|text|mutedText|border|highlightText|theme|candidatePageSize|candidateLayout|showCandidateComments`.
Older nine-field, ten-field, or eleven-field payloads are still treated as the
default seven-candidate horizontal strip with comments shown.

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

## Schema Presets

`GET /schemas` returns the built-in Rime/WeChat/Microsoft-style scheme catalog
and the selected config id:

```json
{
  "selected": "wechat-pinyin",
  "schemas": [
    {
      "id": "double-pinyin-microsoft",
      "name": "微软双拼",
      "kind": "double-pinyin",
      "doublePinyin": true,
      "doublePinyinScheme": "microsoft"
    }
  ],
  "config": { "schema": "wechat-pinyin" }
}
```

`POST /schemas/apply` accepts `{"id":"rime-ice-pinyin"}` and expands the preset
into the ordinary shared config fields, then saves `config.json` and refreshes
active sessions. Native code only needs to keep reading the same normalized
`doublePinyin`, `doublePinyinScheme`, `candidateLayout`, punctuation, and skin
fields; new schemes can be added in Go without changing the TSF DLL.

## Reverse Lookup

`GET /engine/reverse?q=你好&limit=20` or `POST /engine/reverse` with
`{"query":"你好","limit":20}` returns dictionary readings for Chinese text:

```json
{
  "query": "你好",
  "count": 1,
  "entries": [
    {
      "reading": "nihao",
      "text": "你好",
      "kind": "reverse",
      "source": "builtin",
      "comment": "反查",
      "weight": 15000
    }
  ]
}
```

The lookup scans the shared Go dictionary index, including hot-updated Rime
imports and user phrases. Results keep original source/comment metadata when it
exists, which makes Luna/Rime Ice provenance visible in CLI, settings UI, and
future native reverse-lookup panels without adding C++ parsing logic.

## Dictionary Hot Updates

The default source is GitHub Releases:

```text
https://github.com/neko233-com/shurufa233/releases/latest/download/dictionary-manifest.json
```

`GET /updates/sources` returns the built-in dictionary source catalog. It
separates directly installable shurufa233 manifests from upstream Rime/OpenCC
source repositories that must be converted first:

```json
{
  "selected": "shurufa233-github",
  "sources": [
    {
      "id": "rime-ice-source",
      "name": "雾凇拼音 Rime Ice",
      "kind": "rime-source",
      "installable": false,
      "homepage": "https://github.com/iDvel/rime-ice",
      "rawSources": [
        { "label": "rime_ice.dict.yaml", "role": "entry-dictionary" }
      ],
      "convertCommand": "shurufa-dictimport ..."
    }
  ]
}
```

`POST /updates/source` accepts `{"id":"shurufa233-github"}` and applies the
manifest/mirror settings from an installable source to `config.json`. Source-only
Rime presets intentionally return a 400 until a generated shurufa233 manifest is
provided, keeping upstream YAML conversion explicit and license-auditable.

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
  "source": {
    "preset": "rime-ice-source",
    "url": "https://github.com/iDvel/rime-ice",
    "commit": "upstream commit or tag",
    "license": "GPL-3.0",
    "convertCommand": "shurufa-dictimport ..."
  },
  "dictionaries": [
    {
      "language": "zh-CN",
      "version": "2026.07.04",
      "url": "https://github.com/neko233-com/shurufa233/releases/latest/download/zh-CN.2026.07.04.json",
      "sha256": "optional downloaded artifact lowercase hex",
      "compression": "gzip",
      "contentSha256": "optional decompressed JSON lowercase hex",
      "source": { "preset": "rime-ice-source" }
    }
  ]
}
```

Recommended mirror choices:

- GitHub Releases as source of truth
- GitHub Pages or Cloudflare Pages for public static mirrors
- A China-friendly object storage/CDN bucket that syncs release artifacts
- Optional enterprise mirror URL configured by the user, not hardcoded into the client
