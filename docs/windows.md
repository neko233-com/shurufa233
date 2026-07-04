# Windows Build and Install

Windows is the production target for the first line. The native input method layer targets Windows 11 and builds as a TSF text service DLL. The product direction is a clean local input method: no telemetry, no ads, no account requirement, and no company-operated cloud features in the input path.

## Artifacts

```text
build/windows/go-amd64/shurufa-daemon.exe
build/windows/go-arm64/shurufa-daemon.exe
build/windows/go-386/shurufa-daemon.exe
build/windows/x64/Shurufa233Tsf.dll
build/windows/x86/Shurufa233Tsf.dll
build/windows/arm64/Shurufa233Tsf.dll
```

## Build Tools

Install the full Windows native toolchain:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows-buildtools.ps1
```

The script installs or modifies Visual Studio Build Tools and installs `MartinStorsjo.LLVM-MinGW.UCRT` for cross-architecture Go `c-shared` builds.

Required Visual Studio components:

- `Microsoft.VisualStudio.Component.VC.Tools.x86.x64`
- `Microsoft.VisualStudio.Component.VC.14.44.17.14.ARM64`
- `Microsoft.VisualStudio.Component.VC.Tools.ARM64`
- `Microsoft.VisualStudio.Component.VC.Tools.ARM64EC`
- `Microsoft.VisualStudio.Component.Windows11SDK.26100`

For the in-process Go core DLL, `build-windows.ps1` also needs a matching MinGW-w64 cross compiler:

- `x64` / `amd64`: `x86_64-w64-mingw32-gcc.exe`
- `x86` / `386`: `i686-w64-mingw32-gcc.exe`
- `arm64`: `aarch64-w64-mingw32-gcc.exe`

If the matching compiler is missing, daemon and CLI artifacts still build, but `shurufa_core.dll` is skipped for that architecture. That package will run through daemon IPC fallback instead of the fastest in-process core path.

## Build

```powershell
.\scripts\build-windows.ps1 -GoArch @('amd64','arm64','386') -NativeArch @('x64','arm64','x86')
```

If ARM64 native linking fails with `msvcprt.lib`, the ARM64 VC tools are not installed yet.

## Package

Create installable Windows zip packages from existing build artifacts:

```powershell
.\scripts\package-windows.ps1 -Arch @('x64','arm64')
```

Each package contains the `build/windows` artifacts, install/uninstall scripts, docs, and a `manifest.json` with SHA-256 hashes. The manifest includes `performanceMode`; production-quality performance should be `in-process-core`, not `daemon-ipc-fallback`. Install from the package root with:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1 -SkipBuild
```

Use `-SkipMissingArch` when packaging on a machine that has not installed every target toolchain yet.

## Install Current Machine

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1
```

The installer:

- copies daemon, CLI, and TSF DLL to `%LOCALAPPDATA%\Programs\shurufa233`
- copies `Shurufa233ProfileCtl.exe` for current-session enable/activate/probe operations
- copies `Shurufa233SmokeEdit.exe` for native TSF and typing performance validation
- copies the React settings UI bundle and serves it from `http://127.0.0.1:23333/settings/`
- creates Start Menu shortcuts under `shurufa233` for the settings panel and input performance lab
- starts the daemon and verifies `http://127.0.0.1:23333/health`
- registers the TSF DLL through UAC because TSF profiles live under HKLM
- enables the profile for the current user
- adds the input method tip to `zh-Hans-CN`
- starts `ctfmon.exe`
- registers a versioned TSF DLL path so loaded DLLs do not block updates
- installs a matching versioned `shurufa_core-<arch>-<stamp>.dll` beside the TSF DLL so the in-process core can update even when the legacy `shurufa_core.dll` is locked
- removes stale versioned native artifacts when possible and schedules locked stale DLLs for deletion on reboot
- keeps the existing Microsoft/Windows input method as the default input method
- saves the pre-install input method list to `%APPDATA%\shurufa233\input-method-backup.json` if no backup exists yet

The installer does not steal the default input method. Use Windows' normal input method switcher, such as `Ctrl+Shift` or the system language switch shortcut configured on the machine, to move between Microsoft IME and shurufa233. Inside shurufa233, a light tap on `Shift` toggles Chinese/English mode; `Ctrl+Shift` and other Ctrl/Alt combinations are passed back to Windows and applications.

During composition, shurufa233 follows the Microsoft IME-style two-line shape: the upper preedit line shows the current English/pinyin spelling, and the lower candidate strip shows Chinese candidates. Skin settings are scoped mainly to the lower candidate strip; the upper preedit line keeps a neutral system look for readability.

The daemon writes local startup and update diagnostics to `%LOCALAPPDATA%\shurufa233-daemon.log`. If install-time health verification fails, the installer prints the most recent daemon log lines before stopping.

Open the installed settings panel at:

```text
http://127.0.0.1:23333/settings/
```

The settings panel is served by the local daemon from the installed static bundle, so it does not require a Vite development server after installation.
After installation, it is also available from Start Menu > `shurufa233` > `Settings`.

For focused development testing only, pass `-ActivateProfile`:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1 -ActivateProfile
```

## Uninstall Current Machine

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\uninstall-windows.ps1
```

The uninstaller:

- unregisters the TSF DLL through UAC when needed
- removes the daemon from `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
- stops the daemon process
- restores `%APPDATA%\shurufa233\input-method-backup.json` when present
- removes `%LOCALAPPDATA%\Programs\shurufa233` and `C:\Program Files\shurufa233`
- removes the Start Menu `shurufa233` shortcuts

Pass `-RemoveUserData` to also remove `%APPDATA%\shurufa233`.

Current input method tip:

```text
0804:{3D7B8D06-9872-4C31-B77D-3B87327CBF64}{B68911A2-4478-491C-A624-978441648E20}
```

## CLI

```powershell
shurufa-imecli status
shurufa-imecli preview nihao
shurufa-imecli update-check
shurufa-imecli update-apply
shurufa-imecli agent "/rewrite hello world"
```

## Native Profile Tool

```powershell
Shurufa233ProfileCtl.exe enable
Shurufa233ProfileCtl.exe activate
Shurufa233ProfileCtl.exe current
Shurufa233ProfileCtl.exe probe
```

Running `Shurufa233ProfileCtl.exe` without arguments only enables the profile. Use `activate` explicitly when you need to switch the current session to shurufa233.
`current` prints the active keyboard TIP, which is useful when verifying `Ctrl+Shift` coexistence with Microsoft IME.

`probe` creates the TSF COM object directly and is useful when checking whether Windows is loading the registered DLL.

Local TSF diagnostics are written to:

```text
%LOCALAPPDATA%\shurufa233-tsf.log
%LOCALAPPDATA%\shurufa233-daemon.log
```

## Input Performance Lab

Native packages include `Shurufa233SmokeEdit.exe`. It is a polished Win32 EDIT-based performance lab rather than a React surface, because it validates the real TSF path used by native Windows apps and latency-sensitive games. It tracks WPM, key events per second, average key-to-text-change latency, IME composition activity, and committed character count.

The React/Vite app remains the cross-platform settings surface. Browser/React typing tests can be added later as a separate compatibility target, but they do not replace the native SmokeEdit path.

## Agent Input Mode

The daemon exposes:

```text
POST /agent/compose
```

Request:

```json
{ "input": "/rewrite hello", "context": "optional active text context" }
```

This is intentionally provider-neutral. The same protocol can later call local models, cloud models, or a user-configured agent endpoint without changing the TSF glue.

## Candidate Window

The TSF layer renders a native Win32 candidate window and reads skin data from the daemon:

```text
GET /ime/skin
GET /ime/candidates
```

Current skin fields come from the settings UI:

- font family
- font size
- accent color
- surface color
- text color
- muted text color
- border color
- highlight text color
- theme mode

The candidate window is local-only. It does not fetch remote UI assets or send input text to a cloud service.

Candidate interaction:

- `Space` / `Enter`: commit the highlighted candidate
- `1`-`9`: commit candidate by number
- mouse click: commit the clicked candidate in the native candidate strip
- `Right` / `Down` / `Tab`: move highlight to the next candidate
- `Left` / `Up`: move highlight to the previous candidate
- `Esc`: clear the active composition and hide the candidate window
