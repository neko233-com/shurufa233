# shurufa233

Cross-platform Chinese input method prototype using a three-layer isolation model:

- Pure Go input engine and C ABI
- Thin native platform glue for Windows TSF and macOS IMKit
- Go daemon for configuration, dictionaries, IPC, and background tasks
- Vite 8 + React settings UI, ready to be hosted by Wails v3
- GitHub Releases based dictionary hot updates with configurable mirror/CDN URLs

The current local MVP runs the Go engine, daemon IPC, and settings UI on Windows. Native Windows TSF glue is scaffolded under `native/windows/tsf`; building and registering it requires Visual Studio Build Tools with the Windows SDK.

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

Default mode is Chinese pinyin (`zh-CN`, `zh`). The settings UI supports Chinese/English mode switching, skin colors, candidate size, dictionary update source configuration, and update checks.

## Architecture

```text
core/engine      Pure Go input engine: pinyin buffer, candidates, ranking, user learning
core/abi         C ABI export for native TSF/IMKit glue
cmd/daemon       Local HTTP IPC service and config persistence
cmd/imecli       Terminal smoke-test client for the engine
apps/settings    Vite 8 + React settings UI for Wails v3 hosting
native/windows   Windows TSF C++ glue skeleton
native/macos     macOS IMKit placeholder
scripts          Build and install helpers
```

## Windows Native Status

`scripts/install-windows.ps1` checks for Visual Studio C++ build tools, builds the Go C ABI DLL, and prepares the native TSF build. This machine currently has Go and Node installed, but no `cmake` or `cl`, so system-wide TSF registration is blocked until Visual Studio Build Tools are installed.
