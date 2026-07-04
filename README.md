# shurufa233

Cross-platform Chinese input method prototype using a three-layer isolation model:

- Pure Go input engine and C ABI
- Thin native platform glue for Windows TSF and macOS IMKit
- Go daemon for configuration, dictionaries, IPC, and background tasks
- Vite 8 + React settings UI, ready to be hosted by Wails v3
- GitHub Releases based dictionary hot updates with curated Rime/OpenCC source presets, one-command upstream sync, and configurable mirror/CDN URLs
- CLI candidate actions for preview paging, selection, candidate pinning/hiding, first/last-character commit, schema switching, Rime `custom.yaml` patch import, app-context behavior checks, pinyin separator previews, post-commit association candidates, Rime-style reverse lookup, user phrases, profile bundle export/import, emoji/symbol catalog lookup, and agent workflows

The current local MVP runs the Go engine, daemon IPC, and settings UI on Windows. Native Windows TSF glue is scaffolded under `native/windows/tsf`; building and registering it requires Visual Studio Build Tools with the Windows SDK.

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

Default mode is Chinese pinyin (`zh-CN`, `zh`). The Go core preserves apostrophe pinyin separators in the preedit buffer and uses them to force ambiguous syllable boundaries, so `xi'an` can rank `西安` ahead of plain `xian` candidates. It also supports Rime-style schema presets (`wechat-pinyin`, `rime-luna-pinyin`, `rime-ice-pinyin`, `double-pinyin-xiaohe`, `double-pinyin-microsoft`), Rime-style `custom.yaml` patch import for common `schema_list`, `menu/page_size`, `speller/algebra`, `switches`, `style`, `punctuator`, `key_binder`, and `ascii_composer` fields, Rime-style runtime switches (`ascii_mode`, `ascii_punct`, `simplification`, `candidate_comments`, `associations`, `vertical_candidates`), app-aware behavior rules for password fields, terminals, games, and IDEs, slash symbol prefixes such as `/fs` and `/xh`, reverse lookup from Chinese text back to readings, plus runtime fixed user phrases stored separately from learned user scores, so personal `custom_phrase.txt`-style rows can be managed without rebuilding dictionaries. Wrong candidates can be hidden into `user-rejects.json`, and preferred candidates can be pinned into `user-pins.json`, through the shared candidate-action API, matching the practical “delete bad candidate / fix good candidate near the top” workflow of mature IMEs. Emoji, kaomoji, symbols, and agent rows are exposed through a shared catalog API so imported Rime `symbols.yaml` and OpenCC emoji resources can feed the settings UI, CLI, daemon fallback, and future native panels without new C++ glue. Candidate text can be emitted in simplified or traditional script through the shared `script` config; the current built-in conversion table is intentionally small and OpenCC-ready so a full OpenCC dictionary package can replace it without touching TSF glue. Post-commit association candidates are generated in the Go core and exposed through daemon, CLI, and C ABI APIs, giving WeChat-like next-word suggestions without adding platform-specific C++ logic. The shared config now includes key behavior profiles for WeChat/Microsoft, Rime, custom shortcuts, stored Rime spelling algebra, and Rime full/half-shape punctuation maps; common fuzzy `derive` rules and custom punctuation maps take effect through local config reloads, so Shift mode toggling, semicolon/quote quick select, bracket/minus-equal paging, comma-period Rime paging, punctuation shape, and app-specific ASCII/game behavior can change without recompiling TSF C++. User profile bundles can export/import config, learned scores, fixed phrases, hidden candidates, and pinned candidates as one JSON file for Rime-style backup, cross-device migration, and later cloud sync. The settings UI supports Chinese/English mode switching, Rime switch toggles, Rime custom patch import, app-rule previewing, schema preset switching, simplified/traditional output, post-commit associations, full-width/half-width punctuation mode, key behavior profiles, Xiaohe and Microsoft/Sogou double-pinyin scheme selection, skin colors, candidate size, fixed phrase add/import/export/delete, candidate pin/unpin, candidate hide/restore, full profile export/import, reverse lookup, emoji/symbol catalog search, user wordbook import/export/delete, dictionary update source configuration, update checks, and one-click dictionary update apply.

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
