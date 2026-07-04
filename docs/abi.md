# Go Core C ABI

Native platform glue calls the Go input engine through a compact C ABI.

## Session

```c
uint64_t ShurufaCreateSession(void);
void ShurufaDestroySession(uint64_t session);
```

## Editing

```c
char* ShurufaInputKey(uint64_t session, char key);
char* ShurufaPreview(uint64_t session, const char* input);
char* ShurufaBackspace(uint64_t session);
char* ShurufaClear(uint64_t session);
char* ShurufaSetMode(uint64_t session, const char* mode);
char* ShurufaToggleMode(uint64_t session);
char* ShurufaMode(uint64_t session);
char* ShurufaSelect(uint64_t session, int index);
void ShurufaFree(char* value);
```

Every function returning `char*` returns UTF-8 JSON. Call `ShurufaFree` after copying the JSON.

Example response:

```json
{
  "buffer": "nihao",
  "mode": "zh",
  "candidates": [
    { "text": "你好", "reading": "nihao", "weight": 15000, "userScore": 0 }
  ],
  "updatedAt": "2026-07-04T00:00:00Z"
}
```

`mode` is session-scoped and normalized to `zh` or `en`. Native glue can use
`ShurufaToggleMode` for a light Shift tap and `ShurufaSetMode` for an explicit UI
choice. Switching modes clears the active composition buffer, matching Microsoft
IME-style Chinese/English toggling.

## Hot Path

Windows TSF uses compact non-JSON calls on the per-key path:

```c
int ShurufaInputKeyFast(uint64_t session, char key);
int ShurufaBackspaceFast(uint64_t session);
int ShurufaCandidateCount(uint64_t session);
char* ShurufaCandidatePayload(uint64_t session, int limit);
char* ShurufaCandidatePayloadRange(uint64_t session, int start, int limit);
char* ShurufaCommitCandidate(uint64_t session, int index);
char* ShurufaCommitCandidateChar(uint64_t session, int index, const char* side);
```

`ShurufaCandidatePayload` returns up to `limit` UTF-8 rows separated by `\n` from the first candidate.
`ShurufaCandidatePayloadRange` returns a paged slice beginning at `start`; Windows uses it for Microsoft IME-style candidate paging.
`ShurufaCommitCandidateChar` commits `side=first` or `side=last` from the selected candidate and clears the active composition. It is reserved for Rime-style first/last-character actions without baking that behavior into platform glue.
Each row is:

```text
display_index<TAB>text<TAB>reading<TAB>score<TAB>kind<TAB>source<TAB>comment
```

`kind`, `source`, and `comment` are optional extension fields. Current kinds include ordinary word candidates plus `emoji`, `kaomoji`, `symbol`, `phrase`, `dynamic`, and `agent`; renderers must tolerate older four-column or six-column payloads. Built-in examples include `zan` -> `👍` (`emoji`, comment `赞`), `kaixin` -> `ヽ(・∀・)ﾉ` (`kaomoji`), `shengluehao` -> `……` (`symbol`), `rq` -> today's date (`dynamic`, comment `动态`), and `rewrite` -> `/rewrite ` (`agent`, comment `润色`).

The Windows glue calls `ShurufaFree` after copying the returned payload. Older per-candidate getters remain available as a compatibility fallback.

The in-process core reads the same local config file as the daemon (`%APPDATA%\shurufa233\config.json`, or `SHURUFA233_CONFIG` when set) when creating a session. This keeps TSF hot-path behavior such as fuzzy initials and the selected `doublePinyinScheme` aligned with the settings UI and daemon IPC.
The Windows TSF layer also reads the same config file for `punctuation` (`full` or `half`) so punctuation-mode changes from the settings UI can take effect without routing every key through the daemon.

## Reserved Extension ABI

The Windows TSF DLL dynamically probes the following optional exports when it
loads the in-process Go core. They are not required for the per-key hot path, so
older cores and daemon IPC fallback remain compatible. New platform features
should first try to use these JSON APIs before adding more C++ glue.

```c
char* ShurufaAbiVersion(void);
char* ShurufaCapabilities(void);
char* ShurufaState(uint64_t session);
char* ShurufaCandidatePayloadV2(uint64_t session, int start, int limit);
char* ShurufaConfigJSON(void);
char* ShurufaApplyConfigJSON(char* json);
char* ShurufaReloadConfig(void);
char* ShurufaReloadDictionaries(void);
char* ShurufaDictionaryManifestJSON(void);
char* ShurufaUserScoresJSON(uint64_t session);
char* ShurufaImportUserScoresJSON(uint64_t session, char* json);
char* ShurufaCommitText(uint64_t session, char* reading, char* text);
char* ShurufaAgentCompose(char* input, char* context);
char* ShurufaSelectCandidateChar(uint64_t session, int index, const char* side);
char* ShurufaExecuteCommand(uint64_t session, const char* command, const char* json);
```

All returned strings are UTF-8 and must be released with `ShurufaFree`.
`ShurufaAbiVersion` returns a plain version string; every other extension export
returns JSON with an `ok` field and `updatedAt`.

The shared `config-json` payload includes display-only fields such as
`candidatePageSize`, `candidateLayout`, and `showCandidateComments`; candidate
comment text remains part of candidate payloads even when the UI hides it.

`ShurufaCapabilities` advertises feature flags such as
`candidate-payload-v2`, `config-json`, `reload-dictionaries`,
`user-scores-json`, `commit-text`, `agent-compose`,
`rime-compatible-dictionaries`, `gzip-dictionaries`,
`abbreviation-candidates`, `emoji-kaomoji-symbol-candidates`, and
`dynamic-datetime-candidates`, `candidate-char-commit`, and
`candidate-comments`, and `extension-command-json`.

`ShurufaExecuteCommand` is the reserved forward-compatible command bus for
future native glue. The first argument is the session id, the second is a stable
command name, and the third is a UTF-8 JSON payload. It returns UTF-8 JSON and
must be released with `ShurufaFree`. Windows TSF loads this optional export once
but keeps the current per-key hot path on compact APIs. New platform features
should prefer adding a Go-side command here before adding another C++ callback.

Current commands include:

```text
state
preview                 {"input":"zan"}
input-key               {"input":"z"}
backspace
clear
mode
set-mode                {"mode":"en"} or {"toggle":true}
toggle-mode
candidate-payload-v2    {"start":0,"limit":7}
select                  {"index":0}
select-candidate-char   {"index":0,"side":"first"}
config-json
apply-config-json       { ...engine.Config } or {"config":{...}}
reload-config
reload-dictionaries
dictionary-manifest-json
user-scores-json
import-user-scores-json {"userScores":{"nihao|你好":25}}
commit-text             {"reading":"nihao","text":"你好"}
agent-compose           {"input":"/rewrite","context":"optional text"}
```

`ShurufaCandidatePayloadV2` is the future rich candidate contract for native
renderers, React/Wails diagnostics, esports typing labs, and mouse/skin
experiments:

```json
{
  "ok": true,
  "start": 0,
  "limit": 7,
  "total": 42,
  "items": [
    {
      "index": 0,
      "displayIndex": 1,
      "text": "你好",
      "reading": "nihao",
      "kind": "phrase",
      "source": "segmenter",
      "comment": "整句",
      "weight": 15000,
      "userScore": 25,
      "score": 15025
    }
  ],
  "updatedAt": "2026-07-05T00:00:00Z"
}
```

`ShurufaApplyConfigJSON` applies an `engine.Config` JSON object to all active
in-process sessions. `ShurufaReloadConfig` reloads `%APPDATA%\shurufa233\config.json`
or `SHURUFA233_CONFIG` and applies it to active sessions. Use these for skin,
candidate count, visible candidate page size, candidate layout, fuzzy initials,
double pinyin scheme, punctuation, update-channel, and mode changes that should
be visible without reinstalling the TSF DLL.

`ShurufaReloadDictionaries` reloads local `.json` and `.json.gz` dictionary
files from `%APPDATA%\shurufa233\dictionaries` or
`SHURUFA233_DICTIONARY_DIR`, then adds or updates entries in active sessions.
`ShurufaDictionaryManifestJSON` returns the installed manifest when
`manifest.json` or `dictionary-manifest.json` exists.

`ShurufaUserScoresJSON`, `ShurufaImportUserScoresJSON`, and `ShurufaCommitText`
reserve the user-wordbook surface. Import accepts either:

```json
{ "userScores": { "nihao|你好": 25 } }
```

or a raw score map:

```json
{ "nihao|你好": 25 }
```

`ShurufaAgentCompose` is the native bridge for agent-style input actions. It
returns built-in prompt candidates today and keeps the ABI stable for later
local/cloud model routing in the Go daemon.
