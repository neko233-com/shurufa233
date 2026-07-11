# Windows Build and Install

Windows is the production target for the first line. The native input method layer targets Windows 11 and builds as a TSF text service DLL. The product direction is a clean local input method: no telemetry, no ads, no account requirement, and no company-operated cloud features in the input path.

## Artifacts

```text
build/windows/go-amd64/shurufa-daemon.exe
build/windows/go-amd64/shurufa-imecli.exe
build/windows/go-amd64/shurufa-dictimport.exe
build/windows/go-amd64/shurufa-dictmanifest.exe
build/windows/go-amd64/shurufa-dictsync.exe
build/windows/go-arm64/shurufa-daemon.exe
build/windows/go-arm64/shurufa-imecli.exe
build/windows/go-arm64/shurufa-dictimport.exe
build/windows/go-arm64/shurufa-dictmanifest.exe
build/windows/go-arm64/shurufa-dictsync.exe
build/windows/go-386/shurufa-daemon.exe
build/windows/go-386/shurufa-imecli.exe
build/windows/go-386/shurufa-dictimport.exe
build/windows/go-386/shurufa-dictmanifest.exe
build/windows/go-386/shurufa-dictsync.exe
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

If the matching compiler is missing, daemon, CLI, and dictionary release artifacts still build, but `shurufa_core.dll` is skipped for that architecture. That package will run through daemon IPC fallback instead of the fastest in-process core path.

## Build

```powershell
.\scripts\build-windows.ps1 -GoArch @('amd64','arm64','386') -NativeArch @('x64','arm64','x86')
```

If ARM64 native linking fails with `msvcprt.lib`, the ARM64 VC tools are not installed yet.

## Package

Create installable Windows zip packages from existing build artifacts:

```powershell
.\scripts\package-windows.ps1
```

By default this produces `x64`, `arm64`, and `x86` packages. Each package contains the `build/windows` artifacts, install/uninstall scripts, docs, and a `manifest.json` with SHA-256 hashes. The manifest includes `performanceMode`; production-quality performance should be `in-process-core`, not `daemon-ipc-fallback`. Install from the package root with:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1 -SkipBuild
```

Use `-SkipMissingArch` when packaging on a machine that has not installed every target toolchain yet.

## Install Current Machine

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1
```

The installer:

- verifies `manifest.json` when present, including platform, real OS architecture, `in-process-core` mode, required production artifact roles, required files, and SHA-256 hashes before changing system state
- copies daemon, CLI, dictionary release tools, and TSF DLL to `%LOCALAPPDATA%\Programs\shurufa233`
- detects the real Windows OS architecture, including 32-bit PowerShell running under WOW64, before choosing x64, arm64, or x86 artifacts
- copies `Shurufa233ProfileCtl.exe` for current-session enable/activate/probe operations
- copies `Shurufa233SmokeEdit.exe` for native TSF and typing performance validation
- stops fixed and versioned `Shurufa233SmokeEdit*` validation labs before copying
- if `Shurufa233SmokeEdit.exe` is still locked by Windows, installs the latest lab as `Shurufa233SmokeEdit-<stamp>.exe` and points the Start Menu shortcut to that fresh binary
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

The installer does not steal the default input method. Use Windows' normal input method switcher, such as `Ctrl+Shift` or the system language switch shortcut configured on the machine, to move between Microsoft IME and shurufa233. Inside shurufa233, a light tap on `Shift` toggles Chinese/English mode; Ctrl/Alt combinations are passed back to Windows and applications unless the user explicitly binds that key to a shurufa233 IME action.

During composition, shurufa233 follows the Microsoft IME-style two-line shape: the upper preedit line shows the current English/pinyin spelling, and the lower candidate strip shows Chinese candidates. Skin settings are scoped mainly to the lower candidate strip; the upper preedit line keeps a neutral system look for readability.

The daemon writes local startup and update diagnostics to `%LOCALAPPDATA%\shurufa233-daemon.log`. If install-time health verification fails, the installer prints the most recent daemon log lines before stopping.

Open the installed settings panel at:

```text
http://127.0.0.1:23333/settings/
```

The settings panel is served by the local daemon from the installed static bundle, so it does not require a Vite development server after installation.
After installation, it is also available from Start Menu > `shurufa233` > `Settings`.
The settings panel can check configured GitHub/mirror dictionary manifests and apply dictionary updates through the daemon without leaving the UI.
It can also manage learned user words: refresh, export JSON, merge-import JSON, delete individual rows, or clear the local learning store. Fixed phrase management accepts either JSON or pasted Rime `custom_phrase.txt`, and can export the current fixed phrases back to `custom_phrase.txt`.

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
shurufa-imecli associate "你好"
shurufa-imecli candidates nihao
shurufa-imecli candidates nihao next-page --limit 7
shurufa-imecli candidates nihao select --display-index 1
shurufa-imecli wordbook list
shurufa-imecli wordbook import .\user-wordbook.json
shurufa-imecli wordbook import .\luna_pinyin.userdb.txt
shurufa-imecli phrases add msd "马上到！" 60000
shurufa-imecli phrases import .\user-phrases.json
shurufa-imecli phrases import .\custom_phrase.txt --replace
shurufa-imecli phrases export-rime > .\custom_phrase.txt
shurufa-imecli rejects add ceshi "错词"
shurufa-imecli candidates ceshi forget --index 0
shurufa-imecli symbols emoji zan --limit 12
shurufa-imecli symbols symbol /fs
shurufa-imecli reverse "你好"
shurufa-imecli schemas
shurufa-imecli schema apply double-pinyin-ziranma
shurufa-imecli skin list
shurufa-imecli skin apply wechat-clean
shurufa-imecli rime import .\default.custom.yaml
shurufa-imecli switches
shurufa-imecli switch ascii_mode on
shurufa-imecli switch ascii_punct toggle
shurufa-imecli app-rules
shurufa-imecli app-context resolve --process WeGame.exe --game
shurufa-imecli app-context resolve --process pwsh.exe --terminal
shurufa-imecli profile export .\shurufa233-profile.json
shurufa-imecli profile import .\shurufa233-profile.json
shurufa-imecli profile import .\shurufa233-profile.json --replace
shurufa-imecli sync status
shurufa-imecli sync config --enable --dir D:\Sync\shurufa233 --policy merge-newer
shurufa-imecli sync export --dir D:\Sync\shurufa233
shurufa-imecli sync import --dir D:\Sync\shurufa233
shurufa-imecli update-sources
shurufa-imecli update-source shurufa233-github
shurufa-imecli update-plan --language all
shurufa-imecli update-check
shurufa-imecli update-apply --language zh-CN
shurufa-imecli agent "/rewrite hello world"
```

Dictionary source conversion:

```powershell
shurufa-dictimport -language zh-CN -version rime-import -source rime-luna-pinyin -out .\data\dictionaries\zh-CN.rime.json path\to\luna_pinyin.dict.yaml
```

Rime entry dictionaries that declare `import_tables`, such as Rime Ice, are
resolved recursively by default:

```powershell
shurufa-dictimport -language zh-CN -version rime-ice -source rime-ice -missing-imports=warn -out .\data\dictionaries\zh-CN.rime-ice.json.gz path\to\rime_ice.dict.yaml
```

Dictionary manifest generation:

```powershell
shurufa-dictmanifest -version rime-ice -base-url https://github.com/neko233-com/shurufa233/releases/latest/download -out .\data\dictionaries\dictionary-manifest.json .\data\dictionaries\zh-CN.rime-ice.json.gz
```

One-command upstream sync for release dictionaries:

```powershell
shurufa-dictsync -preset rime-ice-source -version rime-ice-2026.07.05 -base-url https://github.com/neko233-com/shurufa233/releases/latest/download
```

For China-region GitHub access, pass one or more clone mirrors before the
canonical GitHub URL fallback:

```powershell
shurufa-dictsync -preset rime-ice-source -mirror-url "https://ghproxy.example/{url}" -mirror-url "https://git.example/{repo}.git"
```

For end-user hot updates, the installable `shurufa233-github-cn` source keeps
GitHub Releases as the canonical manifest while trying configured
`mirrorBaseUrls` first. Mirror entries can be plain CDN bases or proxy templates:

```powershell
shurufa-imecli update-source shurufa233-github-cn --mirror "https://gh-proxy.com/{url}"
```

## Native Profile Tool

```powershell
Shurufa233ProfileCtl.exe enable
Shurufa233ProfileCtl.exe activate
Shurufa233ProfileCtl.exe activate-microsoft
Shurufa233ProfileCtl.exe activate-tip 0x0804 "{81D4E9C9-1D3B-41BC-9E6C-4B40BF79E35E}" "{FA550B04-5AD7-411F-A5AC-CA038EC515D7}"
Shurufa233ProfileCtl.exe current
Shurufa233ProfileCtl.exe probe
```

Running `Shurufa233ProfileCtl.exe` without arguments only enables the profile. Use `activate` explicitly when you need to switch the current session to shurufa233.
`current` prints the active keyboard TIP, which is useful when verifying `Ctrl+Shift` coexistence with Microsoft IME.
`activate-microsoft` switches the current session back to Windows Microsoft Pinyin for safe local validation. `activate-tip` is the generic restore path: save `current` output before a test, then pass its `langid`, `clsid`, and `profile` back to restore that exact TIP.

`probe` creates the TSF COM object directly and is useful when checking whether Windows is loading the registered DLL.

## Native Typing Behavior

The Windows TSF layer keeps Microsoft IME-style session behavior:

- `Shift` toggles Chinese/English mode only when no composition is active
- the Go session state exposes the normalized `zh`/`en` mode, the daemon mirrors it through `GET/POST /ime/mode`, and the Windows TSF Shift toggle synchronizes that session state before showing `EN` or `中`
- `Ctrl`/`Alt` shortcuts are passed through to the host app unless they are configured as shurufa233 IME action shortcuts
- key behavior comes from `keyProfile` and related shared config switches; `wechat`/`microsoft` keeps Shift toggle, semicolon/quote quick select, and `[]`/`-=` paging, while `rime` enables `,`/`.` paging and disables semicolon/quote quick select without rebuilding the TSF DLL
- per-action shortcuts come from shared `keyBindings`, are editable through the settings panel and daemon `/shortcuts` endpoints, and are rejected with conflict metadata when the same key is bound to multiple enabled IME actions; the Windows TSF layer reloads the same config and only intercepts configured Ctrl/Alt keys
- configured fuzzy initials such as `zh=z`, `ch=c`, and `sh=s` are handled in the Go core with exact pinyin candidates kept ahead of fuzzy matches
- Rime-style schema presets are exposed through daemon `/schemas`, CLI `schema`, and the C ABI `schema-presets-json` / `apply-schema-json`; applying `wechat-pinyin`, `rime-luna-pinyin`, `rime-ice-pinyin`, `double-pinyin-xiaohe`, `double-pinyin-ziranma`, or `double-pinyin-microsoft` only updates the shared config and active Go sessions, so the Windows TSF glue keeps using the same reserved APIs
- Rime `*.custom.yaml` patches are exposed through daemon `/rime/custom`, CLI `rime import`, C ABI `ShurufaApplyRimeCustomJSON`, and the generic `rime-custom-json` command; common `schema_list`, `menu/page_size`, `speller/algebra`, `switches`, `style`, `punctuator`, `recognizer`, `key_binder`, `app_options`, and `ascii_composer` fields are mapped to shared config without teaching the TSF DLL YAML parsing, common fuzzy `derive` spelling rules become active `fuzzyInitials` pairs, custom `punctuator/full_shape` / `half_shape` maps are read by the TSF punctuation hot path, `recognizer/patterns` is loaded into shared config plus TSF's reserved config cache, Rime `app_options/<app>` entries become shared `appRules` for app-scoped Chinese/English mode, full/half punctuation, candidate suppression, and learning suppression, and Weasel/Squirrel frontend style fields such as `style/font_face`, `style/font_point`, `style/color_scheme`, `style/candidate_list_layout`, and `preset_color_schemes` update the shared candidate skin/layout after converting Rime `0xBBGGRR` colors to normal `#rrggbb`
- Rime-style runtime switches are exposed through daemon `/switches`, CLI `switches` / `switch`, C ABI `ShurufaSwitchesJSON` / `ShurufaApplySwitchJSON`, and the generic `ShurufaExecuteCommand(..., "apply-switch-json", ...)`; `ascii_mode`, `ascii_punct`, `simplification`, `candidate_comments`, `associations`, and `vertical_candidates` all map to shared config fields without changing the TSF DLL
- application-aware behavior rules are exposed through daemon `/app-rules` and `/app-context/resolve`, CLI `app-rules` / `app-context resolve`, C ABI `ShurufaAppRulesJSON` / `ShurufaResolveAppContextJSON`, Rime `app_options`, and the generic `resolve-app-context-json` command; built-in rules make password fields, terminals, and game/esports contexts prefer English/half-width/no-candidate behavior without taking over the Windows default input method
- when double pinyin is enabled, the Go core decodes the configured `doublePinyinScheme` (`xiaohe`, `ziranma`, or `microsoft`) while keeping full-pinyin fallback available; old configs with only `doublePinyin=true` use Xiaohe
- in Microsoft double pinyin mode, the native TSF layer treats `;` as the `ing` final and sends it to the Go core instead of using it as the second-candidate shortcut
- short initial input such as `nh`, `wx`, `srf`, and `zgr` is handled by the Go core abbreviation index, with exact full-pinyin candidates still kept ahead
- apostrophe-separated pinyin such as `xi'an` is preserved in the Go preedit buffer and uses the separator to force syllable boundaries, so shared daemon, CLI, Wails preview, and future native glue can rank `西安` ahead of plain `xian` exact candidates without duplicating segmentation logic
- Rime-style recognizer patterns are stored in `recognizerPatterns`; default email, URL, uppercase, and reverse-lookup rules keep literal buffers intact or route backtick-prefixed lookup through the Go candidate engine while more complex custom regex rules remain available through the ABI/config surface for later native handling; `recognizer-decision-json` exposes that match result to native glue, and `key-event-json` now protects literal URL/email/uppercase buffers so ASCII punctuation keeps extending or ending the literal instead of being converted to Chinese punctuation
- Rime-style special-resource prefixes such as `/fs`, `/xh`, `vfs`, and `vxh` are preserved by the Go core, daemon IPC, CLI, C ABI, and Windows TSF. In Chinese mode, pressing an unshifted `/` from an empty composition starts slash-prefixed symbol composition; Rime Ice v-mode prefixes travel through the normal TSF letter path. Lookup uses imported Rime symbol codes without the prefix, and ordinary word candidates are filtered out. Shift+`/` still follows the configured question-mark punctuation path, and ASCII mode continues to pass literal slash through.
- Rime-style reverse lookup is exposed through daemon `/engine/reverse`, CLI `reverse`, settings UI, and C ABI `reverse-lookup-json`; it scans the shared Go dictionaries and keeps source/comment metadata so imported Luna/Rime Ice entries can be audited without platform-specific code
- emoji, kaomoji, symbol, and agent resources are exposed through the shared catalog API (`GET /catalog`, `shurufa-imecli symbols`, and `catalog-json` in the C ABI), so Rime `symbols.yaml` and OpenCC emoji imports can power future native symbol panels without adding more Windows C++ glue
- Rime-style fixed user phrases are managed through the Go core, daemon IPC, CLI, settings panel, and reserved C ABI as `kind=phrase`, `source=user-phrase`, stored in `user-phrases.json` separately from learned word scores; JSON import/export and direct runtime `custom_phrase.txt` import/export are both supported so personal rows from Weasel/Squirrel can outrank ordinary dictionary candidates without rebuilding a release dictionary
- Rime/Weasel/Squirrel synced user dictionaries such as `luna_pinyin.userdb.txt` or `rime_ice.userdb.txt` can be imported through `shurufa-imecli wordbook import`, daemon `/wordbook` with `format=rime-userdb`, or C ABI `import-user-scores`; rows like `cha jian 插件 c=4 d=0.5 t=8` become stable shurufa233 `userScores` entries such as `chajian|插件`, preserving learned frequency without adopting Rime's internal storage format
- whole Rime/Weasel/Squirrel profile migration is also reserved as one C ABI and command-bus call, `ShurufaImportRimeProfileJSON` / `rime-profile-import-json`; it accepts `rimeUserDBText`, `rimeCustomPhraseText`, and `customYaml` together, merges by default, and stores the converted result in shurufa233 user scores, fixed phrases, and shared config instead of keeping Rime's runtime file layout
- wrong candidates can be hidden through the Windows native candidate right-click menu, `candidate-action=forget`, `ShurufaRejectCandidate`, `ShurufaExecuteCommand(..., "candidate-action", ...)`, daemon `/rejects`, CLI `rejects`, and the React settings panel; records are stored in `user-rejects.json` and remove matching learned scores so bad candidates do not keep returning after user learning
- preferred candidates can be pinned through the Windows native candidate right-click menu, `candidate-action=pin`, direct `ShurufaPinCandidate`, `ShurufaCandidateAction`, `ShurufaExecuteCommand(..., "candidate-action", ...)`, daemon `/pins`, CLI `pins`, and the React settings panel; records are stored in `user-pins.json`, receive a large ranking bonus, and remove any matching hidden-candidate row
- full user profile bundles are exposed through daemon `/profile`, CLI `profile export|import`, `ShurufaProfileJSON`, `ShurufaImportProfileJSON`, and the generic `profile-json` command; a bundle carries config, learned scores, fixed phrases, hidden candidates, and pinned candidates for backup, migration, or later cloud sync without changing the Windows default input method
- Rime-style profile sync is exposed through daemon `/sync`, CLI `sync status|config|export|import`, C ABI `ShurufaSyncConfigJSON` / `ShurufaApplySyncConfigJSON` / `ShurufaExportProfileSyncJSON` / `ShurufaImportProfileSyncJSON`, and the generic `sync-*` command aliases; the current implementation writes a local `shurufa233-profile.json` bundle to a configured sync directory, while GitHub/WebDAV URL and mirror fields are pre-reserved metadata and never auto-upload private user data
- dynamic Rime-style utility triggers such as `rq`, `sj`, `xq`, `dt`, and `ts` produce local date, time, weekday, datetime, and Unix timestamp candidates; English aliases `date`, `time`, `week`, `datetime`, and `timestamp` are available for double-pinyin and agent workflows; local arithmetic expressions such as `1+2*3`, `(1+2)*3=`, and `8/2` produce offline `builtin-calculator` result candidates without evaluating arbitrary code. The Windows TSF path starts a provisional numeric expression with a digit or `(` and accepts main-keyboard and numeric-keypad arithmetic operators, so this behavior is available in native input controls rather than only in preview APIs.
- Rime-style first/last-character candidate actions are exposed by the Go core, daemon IPC, C ABI, and configurable key bindings as `side=first`/`side=last`; the Windows TSF layer keeps bracket paging as the default while allowing a non-conflicting shortcut to be chosen in settings
- the optional `ShurufaExecuteCommand(session, command, json)` C ABI is loaded by the Windows TSF DLL as a forward-compatible JSON command bus, so future features such as richer candidate actions, agent hooks, wordbook tools, and config reloads can be added in Go before requiring any new C++ glue
- `candidate-action` / `ShurufaCandidateAction` reserve page, select, candidate pinning/hiding, first/last-character, and future Rime-style candidate events behind that JSON command bus, keeping the hot TSF layer thin for machines that only consume packaged builds; the Windows TSF strip now exposes the same actions through a native right-click menu with commit, pin, hide, first-character commit, and last-character commit
- `key-event-json` / `ShurufaKeyEventJSON` is preloaded by the Windows TSF DLL as a higher-level keyboard event API for future native migration; it accepts key labels, printable characters, modifiers, selected candidate index, page window, and app context, then returns `handled`, `committed`, `passThrough`, refreshed state, candidate payload, optional recognizer decision metadata, and optional navigation deltas; the Go side already resolves semicolon/quote quick select, selection movement, configured paging keys, calculator-expression input, full/half punctuation, Rime punctuation shape overrides, literal URL/email punctuation, and candidate-first punctuation commits so WeChat/Rime-like key behavior can move into Go/config before another C++ rebuild is needed
- management surfaces used by future native menus are now explicit optional exports as well: catalog/symbol lookup, schema presets, skin presets, Rime custom import and whole-profile migration, switches, app-rule resolve/apply, profile import/export, dictionary sources/manifests/update check/update apply, agent config/apply/compose, sync import/export, and user score/phrase/reject/pin import/delete are all probed by the TSF DLL while remaining available through `ShurufaExecuteCommand`; this keeps candidate context menus, Rime migration panels, skin pickers, dictionary update panels, agent provider settings, and app-aware status UI in Go/config instead of requiring another C++ loader pass
- visible candidates per page come from `candidatePageSize` in the shared config, defaulting to 7 and clamped to 3..9 so the native strip, HTTP fallback, and React previews stay aligned
- candidate layout comes from `candidateLayout`, defaulting to Microsoft IME/WeChat-style horizontal rows while allowing a Rime-style vertical list without changing the input engine; Weasel `candidate_list_layout=linear|inline|stacked` is accepted only as an import alias and persists as `horizontal` or `vertical`
- candidate comment hints are controlled by `showCandidateComments`; the Microsoft/WeChat product profiles default it off, while explicit Rime profiles may enable annotations without dropping comment data from the engine or ABI
- simplified/traditional output comes from `script`, defaulting to `simplified`; the Go core converts candidate display/commit text before it reaches daemon, CLI, ABI, or the native strip, so Windows TSF does not need a platform-specific conversion layer
- post-commit and local context association candidates are controlled by `associations`, defaulting on; selection state can now carry `kind=association` follow-up candidates from direct suffix matches or offline context keywords, and the Windows TSF candidate strip immediately renders those follow-ups after a normal, mouse, or first/last-character commit for continuous one-tap selection. Entering new pinyin replaces that list, while punctuation is treated as a hard boundary and dismisses it before inserting punctuation.
- full-sentence input can fall back to a scored segmenter that chooses the best dictionary path, including user-learned word scores, instead of the first greedy split, while strong exact phrases still stay ahead
- space, enter, main-row or numpad number keys, semicolon outside Microsoft double pinyin, apostrophe, brackets, page up/down, home/end, `-`, and `=` operate the visible candidate page in the current Windows TSF glue; the apostrophe separator behavior is already available through the shared Go core and preview/IPC paths while native key mapping stays compatible with the existing candidate strip and paired quote behavior
- Chinese punctuation commits the selected candidate first, then inserts the punctuation; default `punctuation=full` maps common shifted punctuation such as `!`, `^`, `(`, `)`, and `-` to `！`, `……`, `（`, `）`, and `——`, while quote keys alternate paired Chinese quotes `“”` and `‘’`
- when `punctuation=half`, punctuation keys use ASCII punctuation; if a candidate or raw pinyin buffer is active, TSF still commits that text first and then appends the half-width punctuation
- if a raw letter buffer has no candidates, space, enter, or Chinese punctuation commits the raw letters instead of dropping the buffer

Local TSF diagnostics are written to:

```text
%LOCALAPPDATA%\shurufa233-tsf.log
%LOCALAPPDATA%\shurufa233-daemon.log
```

TSF keeps hot-path success logs off by default for typing latency. Set
`SHURUFA233_TSF_DEBUG=1` before starting the text service host when you need
verbose lifecycle and commit diagnostics.

## Input Performance Lab

Native packages include `Shurufa233SmokeEdit.exe`. It is a polished Win32 EDIT-based esports typing and performance lab rather than a React surface, because it validates the real TSF path used by native Windows apps and latency-sensitive games. It tracks WPM, key events per second, average key-to-text-change latency, P95 latency, one-second burst peak keys/s, IME composition cycles, committed character count, text-change events, a recent-key trail, and a latency sparkline for burst typing review.

SmokeEdit is single-instance guarded. Launching it again focuses the existing lab window when one is present, which avoids duplicate hidden labs holding the installed executable open during updates.

Inside SmokeEdit, press `F6` to cancel any stale IME composition/candidate state, hide stale Windows Input Experience candidate residue, activate the shurufa233 TSF profile for the current test session, and immediately refocus the native edit control. This is intended for local validation and does not change the Windows default input method. Press `F5` to cancel active composition/candidate state, hide stale system candidate residue, clear the test buffer, and reset metrics.

The React/Vite settings app also includes a fixed phrase manager for Rime-style `custom_phrase` rows, with add, preview, JSON import/export, direct `custom_phrase.txt` paste import/export, delete, and clear operations backed by the daemon `/phrases` endpoint. It includes candidate hide/restore and pin/unpin managers backed by `/rejects` and `/pins`; right-clicking a candidate in the preview strip hides it immediately, double-clicking pins it, and the managers can restore/cancel, import, export, or clear rows. A Rime recipe panel imports `default.custom.yaml` / `<schema>.custom.yaml` patch snippets and applies supported schema, switch, speller algebra, key, punctuation, and candidate layout fields through `/rime/custom`. A profile sync panel now manages sync provider, local sync directory, future GitHub/WebDAV URL metadata, mirror URLs, conflict policy, manual export, and manual import, while still exposing the raw JSON profile bundle containing config, learned scores, fixed phrases, hidden candidates, and pinned candidates. It includes a compact emoji/symbol catalog panel backed by `/catalog`, with kind filtering and search across built-in and imported Rime/OpenCC resources; clicking a resource sends its special-resource code to the same candidate preview strip used by the native contract. Input-rule controls include simplified/traditional output, Rime switches, app-aware behavior rule previews, key behavior profiles, a per-action shortcut editor with conflict detection, a post-commit association toggle, and Agent provider/model/endpoint/trigger settings backed by `/agent/config`. It also includes an esports-style typing lab for the browser/settings-panel path. The lab tracks WPM, CPM, input event rate, average key-to-input latency when key events are available, P95 latency, one-second burst peak keys/s, accuracy, IME composition activity, and prompt completion progress. The lab probes the daemon preview API from the current trailing pinyin token and shows the live candidate strip, candidate metadata kinds, pin markers, and recent-key trail, then can export a JSON test report for regression records. This React lab is useful for UI and Wails-hosted settings validation, while `Shurufa233SmokeEdit.exe` remains the authoritative native TSF validation target. Use prompts such as `zan`, `kaixin`, `wuyu`, `shengqi`, `shengluehao`, `/fs`, `/xh`, `vfs`, `vxh`, `rq`, `sj`, `xq`, `dt`, and `ts` to verify emoji, kaomoji, symbol-prefix, and dynamic datetime candidates.

## Agent Input Mode

The daemon exposes:

```text
GET /agent/config
PUT /agent/config
POST /agent/compose
```

`/agent/config` stores the provider-neutral agent config in the same shared
`config.json` used by the TSF layer, daemon, CLI, and React settings panel. The
current fields are `enabled`, `provider`, `endpoint`, `model`, `systemPrompt`,
`triggers`, `actions`, and `timeoutMs`; empty or unsafe values are normalized
back to the built-in prompt-router defaults. Windows TSF probes the matching
optional C ABI exports (`ShurufaAgentConfigJSON` and
`ShurufaApplyAgentConfigJSON`) but keeps the typing hot path independent from
agent provider calls.

Request:

```json
{ "input": "/rewrite hello", "context": "optional active text context" }
```

Response items include `text`, `intent`, `action`, `source`, and optional
`context` fields, while the legacy `candidates` string array is still emitted for
simple CLI clients. The bundled intents currently cover rewrite, translate, ask,
and generic compose prompts; these structured rows are the handoff point for a
future TSF candidate-row integration or an external local/cloud agent provider.

This is intentionally provider-neutral. The same protocol can later call local models, cloud models, or a user-configured agent endpoint without changing the TSF glue.
The Go core also exposes built-in agent command candidates in the ordinary
candidate list. Triggers such as `ai`, `agent`, `ask`, `rewrite`, `runse`,
`translate`, and `fanyi` surface `/ask `, `/rewrite `, or `/translate ` rows with
`kind=agent` and `source=builtin-agent`; the native candidate strip renders these
with a compact `AI` badge.

## Candidate Window

The TSF layer renders a native Win32 candidate window and reads skin data from the daemon:

```text
GET /ime/skin
GET /ime/candidates?start=0&limit=7
POST /ime/candidate-action
```

`/ime/candidates` returns the same tab-separated payload shape as the C ABI:
`display_index`, `text`, `reading`, `score`, `kind`, `source`, `comment`, and `pinned`.
The `start` and `limit` query parameters let the native window request only the
visible page, so HTTP fallback keeps the same paging, emoji, kaomoji, symbol
badge, and candidate comment behavior as the in-process core path.
`/ime/candidate-action` mirrors the generic ABI action bus for page navigation,
candidate commit, pin/hide, and first/last-character actions, giving React/Wails and
daemon fallback clients the same event contract as packaged native builds.
Candidate type badges are intentionally short localized labels (`表情`, `颜`, `符`,
`短`, `时`, `算`, `AI`) so emoji, kaomoji, symbol, phrase, dynamic datetime, dynamic calculator, and agent candidates stay readable without
making the native strip feel like a debug surface. Candidate comments are drawn
as muted inline hints after the candidate text, preserving Rime/OpenCC annotations
without turning the lower candidate strip into a table. Set
`showCandidateComments=false` in the shared config to hide those hints in both
the native strip and the React preview while keeping them available in candidate
payloads.

The production `candidateWindowMode=win11` path uses a single 46-DIP horizontal
strip anchored below the caret, seven visible candidates, a selected rounded
pill with a left accent edge, and compact trailing paging/utility controls. The
typed spelling is maintained as a real TSF composition in the application at
the caret (including apostrophe separators) instead of consuming a second row
inside the floating window. The configured candidate font is rendered at its
actual size; the shipped default is `Microsoft YaHei UI` at **12 px** with no
hidden `+3 px` inflation. `candidateWindowMode=full` retains the explicit
two-line preedit/candidate presentation for users who choose it.

Current skin fields come from the settings UI:

- font family
- font size (default `12` px before DPI scaling)
- visible candidates per page
- horizontal or vertical candidate layout
- accent color
- surface color
- text color
- muted text color
- border color
- highlight text color
- theme mode
- corner radius
- horizontal and vertical padding
- row gap
- shadow strength
- opacity

Shared skin presets are now owned by the Go core and exposed through daemon,
CLI, ABI, and the React settings panel. Built-in preset IDs include
`wechat-clean`, `wechat-dark`, `microsoft-light`, and `rime-vertical`; aliases
such as `wechat`, `microsoft`, and `rime` normalize to those presets. Applying a
preset updates colors plus candidate page size, horizontal/vertical layout,
comment visibility, radius, padding, row spacing, shadow, and opacity while
preserving the user's font family. This keeps the native C++ renderer thin:
future WeChat/Microsoft/Rime-like visual presets can ship through Go/config
without requiring another TSF DLL rebuild.

The daemon normalizes skin colors before saving config. Invalid color strings fall back to defaults, and low-contrast candidate text, muted text, or highlighted text is automatically corrected to a readable black/white value. This keeps custom skins from producing an unreadable candidate strip during live typing.
The native Windows candidate renderer also detects dark skins from the configured surface color, so custom dark themes do not need a special theme id to get dark-mode derived borders, idle candidate backgrounds, and preedit chrome.
The TSF renderer caches the local config path and checks its timestamp on a short local poll interval so skin and punctuation mode changes can take effect quickly, while daemon HTTP skin fallback remains throttled to avoid network waits in the typing hot path.
Candidate font metrics, spacing, radius, and page controls are scaled from the current Windows DPI so the strip stays readable on 125%/150% displays.

The candidate window is local-only. It does not fetch remote UI assets or send input text to a cloud service.

Candidate interaction:

- `Space` / `Enter`: commit the highlighted candidate
- `1`-`9` or `Numpad 1`-`Numpad 9`: commit candidate by number
- `;` / `'`: commit the second / third candidate when that candidate exists
- `,` / `.`: page candidates when `keyProfile=rime` or `commaPeriodPageKeys=true`
- mouse click: commit the clicked candidate in the native candidate strip
- mouse hover: move the highlighted candidate under the pointer
- mouse wheel over the candidate strip: page candidates
- mouse click on the right-side page arrows: page candidates
- `Right` / `Down` / `Tab`: move highlight to the next candidate
- `Left` / `Up` / `Shift+Tab`: move highlight to the previous candidate
- `Home` / `End`: jump to the first / last candidate
- `PageDown` / `PageUp`, `=` / `-`, or `]` / `[`: page candidates
- `Esc`: clear the active composition and hide the candidate window
