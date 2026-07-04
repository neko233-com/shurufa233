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

Candidate `text` is returned in the configured output script. `script` is read
from the shared config and normalized to `simplified` or `traditional`; readings,
weights, kinds, sources, and comments remain tied to the original dictionary
rows. Native renderers should display and commit the returned `text` directly
instead of doing their own simplified/traditional conversion.

`ShurufaPreview` and `ShurufaInputKey` preserve apostrophe pinyin separators in
the returned `buffer`. The Go core collapses separators for dictionary lookup
and emits an explicit `kind=phrase`, `source=separator` candidate when the
separated syllables resolve cleanly, for example `xi'an` -> `西安`. Platform
glue can therefore display the upper English/pinyin preedit exactly as typed
while sharing one cross-platform segmentation result.

They also preserve Rime-style slash symbol prefixes such as `/fs`. The slash is
kept in the returned `buffer`, while lookup uses the imported symbol code
without the slash and filters the result to symbol, emoji, kaomoji, and agent
candidates.

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
char* ShurufaRejectCandidate(uint64_t session, int index);
```

`ShurufaCandidatePayload` returns up to `limit` UTF-8 rows separated by `\n` from the first candidate.
`ShurufaCandidatePayloadRange` returns a paged slice beginning at `start`; Windows uses it for Microsoft IME-style candidate paging.
`ShurufaCommitCandidateChar` commits `side=first` or `side=last` from the selected candidate and clears the active composition. It is reserved for Rime-style first/last-character actions without baking that behavior into platform glue.
`ShurufaRejectCandidate` hides the selected candidate, removes any learned score for the same `reading|text`, persists `user-rejects.json`, and returns JSON with the rejected entry plus the refreshed state.
Each row is:

```text
display_index<TAB>text<TAB>reading<TAB>score<TAB>kind<TAB>source<TAB>comment
```

`kind`, `source`, and `comment` are optional extension fields. Current kinds include ordinary word candidates plus `emoji`, `kaomoji`, `symbol`, `phrase`, `dynamic`, and `agent`; renderers must tolerate older four-column or six-column payloads. Built-in examples include `zan` -> `👍` (`emoji`, comment `赞`), `kaixin` -> `ヽ(・∀・)ﾉ` (`kaomoji`), `shengluehao` -> `……` (`symbol`), `rq` -> today's date (`dynamic`, comment `动态`), and `rewrite` -> `/rewrite ` (`agent`, comment `润色`).

The Windows glue calls `ShurufaFree` after copying the returned payload. Older per-candidate getters remain available as a compatibility fallback.

The in-process core reads the same local config file as the daemon (`%APPDATA%\shurufa233\config.json`, or `SHURUFA233_CONFIG` when set) when creating a session. This keeps TSF hot-path behavior such as fuzzy initials, simplified/traditional `script`, and the selected `doublePinyinScheme` aligned with the settings UI and daemon IPC.
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
char* ShurufaAssociate(uint64_t session, const char* json);
char* ShurufaCandidateAction(uint64_t session, const char* json);
char* ShurufaCatalogJSON(uint64_t session, const char* json);
char* ShurufaConfigJSON(void);
char* ShurufaApplyConfigJSON(char* json);
char* ShurufaSchemaPresetsJSON(void);
char* ShurufaApplySchemaJSON(char* json);
char* ShurufaReloadConfig(void);
char* ShurufaReloadDictionaries(void);
char* ShurufaDictionaryManifestJSON(void);
char* ShurufaUserScoresJSON(uint64_t session);
char* ShurufaImportUserScoresJSON(uint64_t session, char* json);
char* ShurufaUserPhrasesJSON(uint64_t session);
char* ShurufaImportUserPhrasesJSON(uint64_t session, char* json);
char* ShurufaUserRejectsJSON(uint64_t session);
char* ShurufaImportUserRejectsJSON(uint64_t session, char* json);
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
It also includes `schema`, a stable Rime-style scheme id such as
`wechat-pinyin`, `rime-luna-pinyin`, `rime-ice-pinyin`,
`double-pinyin-xiaohe`, or `double-pinyin-microsoft`. `ShurufaSchemaPresetsJSON`
lists the built-in scheme table, while `ShurufaApplySchemaJSON({"id":"..."})`
expands a preset into the shared config fields that the native TSF layer already
understands (`doublePinyin`, `doublePinyinScheme`, `candidateLayout`,
`showCandidateComments`, punctuation, fuzzy initials, and dictionary source
preference). This keeps new schemes in Go/config instead of requiring another
C++ export on developer machines that only consume packaged builds.

`ShurufaCapabilities` advertises feature flags such as
`candidate-payload-v2`, `config-json`, `reload-dictionaries`,
`dictionary-source-presets`, `schema-presets-json`, `apply-schema-json`, `user-scores-json`, `user-phrases-json`, `user-rejects-json`, `commit-text`, `agent-compose`,
`rime-compatible-dictionaries`, `gzip-dictionaries`,
`abbreviation-candidates`, `pinyin-separators`, `rime-symbol-prefix`,
`emoji-kaomoji-symbol-candidates`, `catalog-json`, and
`dynamic-datetime-candidates`, `candidate-char-commit`, and
`candidate-comments`, `association-candidates`, `candidate-action-json`, and
`extension-command-json`.

`ShurufaAssociate(session, {"context":"你好","limit":7})` returns a normal state
object with post-commit association candidates. The same behavior is available
through `ShurufaExecuteCommand(session, "associate", ...)` and through
`candidate-action` with `{"action":"associate","context":"微信"}`. Candidate text
is already script-converted, and rows are tagged `kind=association`.

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
associate               {"context":"你好","limit":7}
catalog-json            {"kind":"emoji","query":"zan","limit":20}
candidate-action        {"action":"next-page","start":0,"limit":7}
candidate-action        {"action":"associate","context":"微信","limit":7}
candidate-action        {"action":"forget","index":0}
select                  {"index":0}
select-candidate-char   {"index":0,"side":"first"}
config-json
apply-config-json       { ...engine.Config } or {"config":{...}}
schema-presets-json
apply-schema-json       {"id":"double-pinyin-microsoft"}
reload-config
reload-dictionaries
dictionary-manifest-json
dictionary-sources-json
user-scores-json
import-user-scores-json {"userScores":{"nihao|你好":25}}
user-phrases-json
import-user-phrases-json {"entries":[{"reading":"msd","text":"马上到！"}],"merge":true}
delete-user-phrase      {"reading":"msd","text":"马上到！"}
user-rejects-json
import-user-rejects-json {"entries":[{"reading":"ceshi","text":"错词"}],"merge":true}
delete-user-reject      {"reading":"ceshi","text":"错词"}
commit-text             {"reading":"nihao","text":"你好"}
agent-compose           {"input":"/rewrite","context":"optional text"}
```

`candidate-action` and `ShurufaCandidateAction` reserve a richer Rime/WeChat-style
candidate event surface without requiring new C++ callbacks. Supported actions
currently include `view`, `next-page`, `prev-page`, `first-page`, `last-page`,
`select`, `forget`, `first-char`, `last-char`, and `select-char`. Selection accepts either
an absolute `index` or a page-relative `displayIndex` plus `start`; paging
returns the same rich candidate payload used by `candidate-payload-v2`.
`forget`/`reject`/`delete-candidate` persists a hidden candidate row in
`user-rejects.json`, removes any learned score for that candidate, keeps the
composition buffer active, and returns a `rejected` object for UI feedback.

`catalog-json` and `ShurufaCatalogJSON` reserve the shared emoji, kaomoji,
symbol, and agent resource surface for future native panels. The payload accepts
`kind=all|emoji|kaomoji|symbol|agent`, `query` or `input`, and `limit`; slash
queries such as `/fs` are normalized to the stored Rime code. The response is:

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

Imported Rime `symbols.yaml`, `symbols.custom.yaml`, and OpenCC emoji resources
use the same `kind`/`source` metadata, so Windows and macOS glue can draw a
WeChat-style symbol/emoji panel from this API without knowing where a row came
from.

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
reserve the learned user-wordbook surface. Import accepts either:

```json
{ "userScores": { "nihao|你好": 25 } }
```

or a raw score map:

```json
{ "nihao|你好": 25 }
```

`ShurufaUserPhrasesJSON` and `ShurufaImportUserPhrasesJSON` reserve fixed
Rime-style user phrases. These are persisted separately from learned scores in
`user-phrases.json` or `SHURUFA233_USER_PHRASES`, loaded as `kind=phrase` and
`source=user-phrase`, and can be replaced or merged without changing C++ glue:

```json
{ "entries": [{ "reading": "msd", "text": "马上到！", "weight": 60000 }], "merge": true }
```

`ShurufaUserRejectsJSON`, `ShurufaImportUserRejectsJSON`, and
`ShurufaRejectCandidate` reserve the bad-candidate hide/restore surface. These
rows are persisted in `user-rejects.json` or `SHURUFA233_USER_REJECTS`, and the
Go core filters them before ranking:

```json
{ "entries": [{ "reading": "ceshi", "text": "错词", "comment": "已屏蔽" }], "merge": true }
```

`ShurufaAgentCompose` is the native bridge for agent-style input actions. It
returns built-in prompt candidates today and keeps the ABI stable for later
local/cloud model routing in the Go daemon.
