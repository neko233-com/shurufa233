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

Required Visual Studio components:

- `Microsoft.VisualStudio.Component.VC.Tools.x86.x64`
- `Microsoft.VisualStudio.Component.VC.14.44.17.14.ARM64`
- `Microsoft.VisualStudio.Component.VC.Tools.ARM64`
- `Microsoft.VisualStudio.Component.VC.Tools.ARM64EC`
- `Microsoft.VisualStudio.Component.Windows11SDK.26100`

## Build

```powershell
.\scripts\build-windows.ps1 -GoArch @('amd64','arm64','386') -NativeArch @('x64','arm64','x86')
```

If ARM64 native linking fails with `msvcprt.lib`, the ARM64 VC tools are not installed yet.

## Install Current Machine

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1
```

The installer:

- copies daemon, CLI, and TSF DLL to `%LOCALAPPDATA%\Programs\shurufa233`
- copies `Shurufa233ProfileCtl.exe` for current-session enable/activate/probe operations
- registers the TSF DLL through UAC because TSF profiles live under HKLM
- enables the profile for the current user
- adds the input method tip to `zh-Hans-CN`
- sets it as the default input method override
- starts `ctfmon.exe`
- registers a versioned TSF DLL path so loaded DLLs do not block updates
- activates the profile for the current session with `ITfInputProcessorProfileMgr::ActivateProfile`
- saves the pre-install input method list to `%APPDATA%\shurufa233\input-method-backup.json` if no backup exists yet

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
Shurufa233ProfileCtl.exe probe
```

`probe` creates the TSF COM object directly and is useful when checking whether Windows is loading the registered DLL.

Local TSF diagnostics are written to:

```text
%LOCALAPPDATA%\shurufa233-tsf.log
```

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
- `Right` / `Down` / `Tab`: move highlight to the next candidate
- `Left` / `Up`: move highlight to the previous candidate
- `Esc`: clear the active composition and hide the candidate window
