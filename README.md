# shurufa233

Cross-platform Chinese input method prototype using a three-layer isolation model:

- Pure Go input engine and C ABI
- Thin native platform glue for Windows TSF and macOS IMKit
- Go daemon for configuration, dictionaries, IPC, and background tasks
- Vite 8 + React settings UI, ready to be hosted by Wails v3
- GitHub Releases based dictionary hot updates with curated Rime/OpenCC source presets, one-command upstream sync, and configurable China-region mirror/CDN URL templates
- CLI candidate actions for preview paging, selection, candidate pinning/hiding, first/last-character commit, schema switching, Rime `custom.yaml` patch import, app-context behavior checks, pinyin separator previews, post-commit association candidates, Rime-style reverse lookup, user phrases, profile bundle export/import, profile sync export/import, emoji/symbol catalog lookup, and agent workflows
- Local dynamic candidates for date/time triggers and length-limited arithmetic expressions such as `1+2*3` or `8/2`

The Windows production line runs the Go engine in-process behind a thin native TSF layer, with daemon IPC reserved for management/fallback and the settings UI. Building the native layer under `native/windows/tsf` requires Visual Studio Build Tools with the Windows SDK.

Windows native support targets Windows 11 only for the first production line.
See [docs/windows.md](docs/windows.md) for x64/x86/arm64 build, local install, CLI, and agent input details. See [docs/dictionaries.md](docs/dictionaries.md) for importing Rime/Luna/Rime Ice dictionaries instead of maintaining all words from scratch.

Product stance: clean local input method, WeChat-like typing comfort, no telemetry, no ads, no account requirement, and no company-cloud input pipeline.

## Quick Start

```powershell
go test ./...
go run ./cmd/imecli
go run ./cmd/daemon
cd apps/settings
npm install
npm run dev
```

Open the Vite URL and keep `go run ./cmd/daemon` running. The UI talks to `http://127.0.0.1:23333`.

Default mode is simplified Chinese full pinyin (`zh-CN`, `zh`) with `wechat-pinyin`, a horizontal Win11 strip, hidden annotations, seven candidates per page, and `Microsoft YaHei UI` at 12 px. The composition engine preserves typed apostrophe separators, ranks exact dictionary words ahead of generated character combinations, composes uncovered long input from whole words, and limits synthetic fallbacks so they cannot flood the first page. The compact production dictionary protects daily base vocabulary and common single-syllable alternatives before lower-value extension data.

The production path is simplified-Chinese-only. Legacy `script=traditional` and `simplification` values are accepted for profile migration but normalized to `simplified`; settings expose this as a fixed value. Shift light-tap commits an active pinyin spelling as English and switches mode, `-` / `=` page backward/forward, and `;` / `'` quick-select the second/third candidate under the WeChat key profile. Learning, pin, hide, fixed phrases, associations, punctuation, app rules, schema migration, emoji/symbol catalogs, local profile backup, dictionary updates, and the shared Go/C ABI remain available without putting daemon HTTP on the per-key hot path.

Windows TSF candidate windows also expose a native right-click menu for commit, pin, hide, first-character commit, and last-character commit, backed by the same Go candidate-action ABI as the CLI and settings UI.
Rime/Weasel migration covers both fixed `custom_phrase.txt` rows and synced `<schema>.userdb.txt` learning rows; the latter import into shurufa233's stable `userScores` wordbook format.

## Architecture

```text
core/engine      Pure Go input engine: pinyin buffer, candidates, ranking, user learning
core/abi         C ABI export for native TSF/IMKit glue
cmd/daemon       Local HTTP IPC service and config persistence
cmd/imecli       Terminal smoke-test client for the engine
cmd/dictimport   Rime .dict.yaml to shurufa233 JSON dictionary converter
cmd/dictmanifest GitHub/mirror dictionary update manifest generator
cmd/dictsync     Git/Rime source sync tool that clones upstream dictionaries, converts them, and writes a hot-update manifest
apps/settings    Vite 8 + React settings UI for Wails v3 hosting
native/windows   Windows TSF C++ glue skeleton
native/macos     macOS IMKit placeholder
scripts          Build and install helpers
```

## Windows Native Status

`scripts/install-windows.ps1` installs the current-machine daemon, CLI, dictionary release tools, and native TSF DLL, then registers and enables the TSF profile for the current user.
