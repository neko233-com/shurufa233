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

They also preserve Rime-style special-resource prefixes such as `/fs` and the
Rime Ice v-mode alias `vfs`. The prefix is kept in the returned `buffer`, while
lookup uses the imported symbol code without the prefix and filters the result
to symbol, emoji, kaomoji, and agent candidates.

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
char* ShurufaPinCandidate(uint64_t session, int index);
```

`ShurufaCandidatePayload` returns up to `limit` UTF-8 rows separated by `\n` from the first candidate.
`ShurufaCandidatePayloadRange` returns a paged slice beginning at `start`; Windows uses it for Microsoft IME-style candidate paging.
`ShurufaCommitCandidateChar` commits `side=first` or `side=last` from the selected candidate and clears the active composition. It is reserved for Rime-style first/last-character actions without baking that behavior into platform glue.
`ShurufaRejectCandidate` hides the selected candidate, removes any learned score for the same `reading|text`, persists `user-rejects.json`, and returns JSON with the rejected entry plus the refreshed state.
`ShurufaPinCandidate` pins the selected candidate into `user-pins.json`, removes any matching reject row, and returns JSON with the pinned entry plus the refreshed state. Windows probes it as a native-menu fast path while keeping `candidate-action` as the forward-compatible command bus.
Each row is:

```text
display_index<TAB>text<TAB>reading<TAB>score<TAB>kind<TAB>source<TAB>comment<TAB>pinned
```

`kind`, `source`, `comment`, and `pinned` are optional extension fields. Current kinds include ordinary word candidates plus `emoji`, `kaomoji`, `symbol`, `phrase`, `dynamic`, and `agent`; renderers must tolerate older four-column, six-column, or seven-column payloads. Built-in examples include `zan` -> `👍` (`emoji`, comment `赞`), `kaixin` -> `ヽ(・∀・)ﾉ` (`kaomoji`), `shengluehao` -> `……` (`symbol`), `rq` -> today's date (`dynamic`, source `builtin-datetime`, comment `动态`), `1+2*3` -> `7` (`dynamic`, source `builtin-calculator`, comment `计算`), and `rewrite` -> `/rewrite ` (`agent`, comment `润色`). `pinned=true` marks candidates promoted by `user-pins.json`.

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
char* ShurufaKeyEventJSON(uint64_t session, const char* json);
char* ShurufaCatalogJSON(uint64_t session, const char* json);
char* ShurufaReverseLookupJSON(uint64_t session, const char* json);
char* ShurufaDictionarySourcesJSON(void);
char* ShurufaApplyDictionarySourceJSON(char* json);
char* ShurufaConfigJSON(void);
char* ShurufaApplyConfigJSON(char* json);
char* ShurufaSchemaPresetsJSON(void);
char* ShurufaApplySchemaJSON(char* json);
char* ShurufaSkinPresetsJSON(void);
char* ShurufaApplySkinPresetJSON(char* json);
char* ShurufaApplyRimeCustomJSON(uint64_t session, char* json);
char* ShurufaImportRimeProfileJSON(uint64_t session, char* json);
char* ShurufaRecognizerPatternsJSON(uint64_t session);
char* ShurufaSwitchesJSON(uint64_t session);
char* ShurufaApplySwitchJSON(uint64_t session, char* json);
char* ShurufaAppRulesJSON(uint64_t session);
char* ShurufaApplyAppRulesJSON(uint64_t session, char* json);
char* ShurufaResolveAppContextJSON(uint64_t session, char* json);
char* ShurufaReloadConfig(void);
char* ShurufaReloadDictionaries(void);
char* ShurufaDictionaryManifestJSON(void);
char* ShurufaDictionaryUpdatePlanJSON(char* json);
char* ShurufaDictionaryUpdateCheckJSON(char* json);
char* ShurufaDictionaryUpdateApplyJSON(char* json);
char* ShurufaUserScoresJSON(uint64_t session);
char* ShurufaImportUserScoresJSON(uint64_t session, char* json);
char* ShurufaUserPhrasesJSON(uint64_t session);
char* ShurufaImportUserPhrasesJSON(uint64_t session, char* json);
char* ShurufaDeleteUserPhraseJSON(uint64_t session, char* json);
char* ShurufaUserRejectsJSON(uint64_t session);
char* ShurufaImportUserRejectsJSON(uint64_t session, char* json);
char* ShurufaDeleteUserRejectJSON(uint64_t session, char* json);
char* ShurufaUserPinsJSON(uint64_t session);
char* ShurufaImportUserPinsJSON(uint64_t session, char* json);
char* ShurufaDeleteUserPinJSON(uint64_t session, char* json);
char* ShurufaProfileJSON(uint64_t session);
char* ShurufaImportProfileJSON(uint64_t session, char* json);
char* ShurufaCommitText(uint64_t session, char* reading, char* text);
char* ShurufaAgentConfigJSON(void);
char* ShurufaApplyAgentConfigJSON(char* json);
char* ShurufaSyncConfigJSON(void);
char* ShurufaApplySyncConfigJSON(char* json);
char* ShurufaExportProfileSyncJSON(uint64_t session, char* json);
char* ShurufaImportProfileSyncJSON(uint64_t session, char* json);
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
`double-pinyin-xiaohe`, `double-pinyin-ziranma`, or
`double-pinyin-microsoft`. `ShurufaSchemaPresetsJSON`
lists the built-in scheme table, while `ShurufaApplySchemaJSON({"id":"..."})`
expands a preset into the shared config fields that the native TSF layer already
understands (`doublePinyin`, `doublePinyinScheme`, `candidateLayout`,
`showCandidateComments`, punctuation, fuzzy initials, key behavior profile, and dictionary source
preference). This keeps new schemes in Go/config instead of requiring another
C++ export on developer machines that only consume packaged builds.
`ShurufaSkinPresetsJSON` and `ShurufaApplySkinPresetJSON` provide the same
direct surface for WeChat/Rime-style candidate strip skins, so future native
skin menus can list and apply presets without adding another TSF loader pass.
Skin presets include colors plus native renderer metrics (`cornerRadius`,
`paddingX`, `paddingY`, `rowGap`, `shadow`, and `opacity`), letting packaged C++
glue consume new WeChat/Microsoft/Rime-like candidate-window spacing and depth
without adding another C ABI entry point.

`ShurufaCapabilities` advertises feature flags such as
`candidate-payload-v2`, `config-json`, `reload-dictionaries`,
`dictionary-source-presets`, `dictionary-update-plan-json`,
`schema-presets-json`, `apply-schema-json`, `skin-presets-json`, `apply-skin-preset-json`, `rime-custom-yaml`, `reverse-lookup-json`, `user-scores-json`, `rime-userdb-text`, `user-phrases-json`, `rime-custom-phrase-text`, `user-rejects-json`, `user-pins-json`, `profile-bundle-json`, `profile-sync-json`, `apply-sync-config-json`, `commit-text`, `agent-compose`, `agent-config-json`, `apply-agent-config-json`,
`rime-compatible-dictionaries`, `gzip-dictionaries`,
`abbreviation-candidates`, `pinyin-separators`, `rime-symbol-prefix`,
`rime-ice-v-symbol-prefix`,
`emoji-kaomoji-symbol-candidates`, `catalog-json`, and
`dynamic-datetime-candidates`, `calculator-candidates`, `candidate-char-commit`, and
`candidate-comments`, `association-candidates`, `candidate-action-json`, and
`extension-command-json`, `key-behavior-config`, `rime-switches-json`,
`app-context-rules-json`, `apply-app-rules-json`, `user-data-delete-json`, and
`key-event-json`.

`ShurufaAssociate(session, {"context":"你好","limit":7})` returns a normal state
object with post-commit or local context association candidates. The same
behavior is available through `ShurufaExecuteCommand(session, "associate", ...)`
and through `candidate-action` with `{"action":"associate","context":"微信"}`.
Candidate text is already script-converted, and rows are tagged
`kind=association` with sources such as `builtin-association` or
`context-association`.

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
dictionary-update-plan-json {"language":"all","mirrorBaseUrls":["https://gh-proxy.com/{url}"]}
input-key               {"input":"z"}
backspace
clear
mode
set-mode                {"mode":"en"} or {"toggle":true}
toggle-mode
candidate-payload-v2    {"start":0,"limit":7}
associate               {"context":"你好","limit":7}
catalog-json            {"kind":"emoji","query":"zan","limit":20}
reverse-lookup-json     {"query":"你好","limit":20}
candidate-action        {"action":"next-page","start":0,"limit":7}
candidate-action        {"action":"associate","context":"微信","limit":7}
candidate-action        {"action":"pin","index":0}
candidate-action        {"action":"forget","index":0}
key-event-json          {"key":"n","character":"n"}
key-event-json          {"key":"space","index":0}
key-event-json          {"key":"n","character":"n","appContext":{"processName":"WeGame.exe"}}
recognizer-decision-json {"input":"www.example.com"}
select                  {"index":0}
select-candidate-char   {"index":0,"side":"first"}
config-json
apply-config-json       { ...engine.Config } or {"config":{...}}
agent-config-json
apply-agent-config-json {"agent":{"provider":"local","model":"qwen","endpoint":"http://127.0.0.1:8787"}}
sync-config-json
apply-sync-config-json {"sync":{"enabled":true,"provider":"local-directory","directory":"D:/Sync/shurufa233"}}
sync-export             {"directory":"D:/Sync/shurufa233"}
sync-import             {"directory":"D:/Sync/shurufa233","merge":true}
schema-presets-json
apply-schema-json       {"id":"double-pinyin-ziranma"}
skin-presets-json
apply-skin-preset-json  {"id":"wechat-clean"}
rime-custom-json        {"yaml":"patch:\n  schema_list:\n    - schema: double_pinyin_flypy\n"}
rime-profile-import-json {"rimeUserDBText":"cha jian 插件 c=4 d=0.5 t=8\n","rimeCustomPhraseText":"马上到！\tmsd\t1\n","customYaml":"patch:\n  menu/page_size: 8\n"}
switches-json
apply-switch-json       {"id":"ascii_mode","value":true}
toggle-switch           {"id":"ascii_punct"}
app-rules-json
apply-app-rules-json    {"rules":[{"id":"game","name":"Game","processNames":["WeGame.exe"],"mode":"en"}]}
resolve-app-context-json {"appContext":{"processName":"WeGame.exe","gameMode":true}}
reload-config
reload-dictionaries
dictionary-manifest-json
dictionary-sources-json
apply-dictionary-source-json {"id":"shurufa233-github-cn","mirrorBaseUrls":["https://gh-proxy.com/{url}"]}
dictionary-update-check-json {"manifestUrls":["https://example.com/dictionary-manifest.json"]}
dictionary-update-apply-json {"language":"zh-CN","manifestUrls":["https://example.com/dictionary-manifest.json"],"force":true}
user-scores-json
import-user-scores-json {"userScores":{"nihao|你好":25}}
import-user-scores-json {"format":"rime-userdb","data":"cha jian 插件 c=4 d=0.5 t=8\n"}
user-phrases-json
import-user-phrases-json {"entries":[{"reading":"msd","text":"马上到！"}],"merge":true}
rime-custom-phrase-text
import-rime-custom-phrases {"data":"马上到！\tmsd\t1\n","merge":true}
delete-user-phrase      {"reading":"msd","text":"马上到！"}
user-rejects-json
import-user-rejects-json {"entries":[{"reading":"ceshi","text":"错词"}],"merge":true}
delete-user-reject      {"reading":"ceshi","text":"错词"}
user-pins-json
import-user-pins-json   {"entries":[{"reading":"nihao","text":"你好"}],"merge":true}
delete-user-pin         {"reading":"nihao","text":"你好"}
profile-json
import-profile-json     {"config":{...},"userScores":{...},"phrases":[...],"merge":true}
commit-text             {"reading":"nihao","text":"你好"}
agent-compose           {"input":"/rewrite","context":"optional text"}
```

`candidate-action`, `ShurufaCandidateAction`, `ShurufaRejectCandidate`, and
`ShurufaPinCandidate` reserve a richer Rime/WeChat-style candidate event surface
without requiring new C++ callbacks. The Windows TSF candidate strip uses this
surface for its native right-click menu: commit, pin, hide, first-character
commit, and last-character commit. Supported actions
currently include `view`, `next-page`, `prev-page`, `first-page`, `last-page`,
`select`, `pin`, `forget`, `first-char`, `last-char`, and `select-char`. Selection accepts either
an absolute `index` or a page-relative `displayIndex` plus `start`; paging
returns the same rich candidate payload used by `candidate-payload-v2`.
`pin`/`pin-candidate` persists a pinned candidate row in `user-pins.json`,
removes any matching reject, keeps the composition buffer active, returns a
`pinned` object for UI feedback, and marks pinned rows in candidate payloads.
`forget`/`reject`/`delete-candidate` persists a hidden candidate row in
`user-rejects.json`, removes any learned score for that candidate, keeps the
composition buffer active, and returns a `rejected` object for UI feedback.

`key-event-json` and `ShurufaKeyEventJSON` reserve the next-level keyboard event
surface for TSF, IMKit, and game validation tools. Payloads may include `key`,
`character`, `code`, `ctrl`, `alt`, `shift`, `meta`, `modifiers`, `index`,
`start`, `limit`, and `appContext`. The result tells native glue whether the key
was `handled`, which `committed` text should be inserted, which text should be
`passThrough` to the host app, the refreshed `state`, and the current candidate
page. It already covers composing characters, Backspace, Escape, Space/Enter
candidate commit, number-key candidate selection, semicolon/quote quick select,
candidate selection movement, configured candidate paging keys, Shift tap mode
toggle, and game/password/terminal app-context pass-through. It also resolves
full-width and half-width punctuation through the shared config, including Rime
`punctuator/full_shape` and `punctuator/half_shape` overrides, and follows the
candidate-first punctuation flow used by Microsoft/WeChat-style IMEs. This keeps
future WeChat/Rime-style key behavior in Go/config first, instead of requiring
another round of platform C++ or IMKit changes.

`switches-json`, `apply-switch-json`, `toggle-switch`, `ShurufaSwitchesJSON`,
and `ShurufaApplySwitchJSON` reserve a Rime-style runtime switch surface.
Current switches map directly onto shared config fields: `ascii_mode` (`mode`),
`ascii_punct` (`punctuation`), `simplification` (`script`),
`candidate_comments` (`showCandidateComments`), `associations`, and
`vertical_candidates` (`candidateLayout`). This lets native glue send one JSON
switch event for Weasel/Squirrel-style UI behavior while Go owns the actual
field mapping and future switch expansion.

`rime-custom-json`, `rime-custom-yaml`, `apply-rime-custom-json`, and
`ShurufaApplyRimeCustomJSON` reserve the Rime `*.custom.yaml` patch import
surface. Payloads may contain `{"yaml":"patch: ..."}` or raw YAML text. The Go
core maps common Rime patch fields such as `schema_list`, `menu/page_size`,
`speller/algebra`, `switches`, `style/horizontal`, `style/candidate_list_layout`, `style/font_face`,
`style/font_point`, `style/color_scheme`, `preset_color_schemes`, `punctuator/import_preset`,
`recognizer/import_preset`, `recognizer/patterns`, `key_binder/import_preset`,
`key_binder/bindings`, `app_options`, and
`ascii_composer/switch_key` into the shared config, persists it, and returns
`applied` plus `warnings`. Native glue should prefer this JSON command instead
of learning individual Rime YAML concepts in C++. The raw spelling algebra is
kept as `config.spellerAlgebra`, and common fuzzy `derive` rules are folded
into active `fuzzyInitials` pairs so native callers get behavior changes through
the existing config reload path. Weasel/Squirrel `candidate_list_layout` aliases
such as `linear`/`inline` and `stacked` normalize to `candidateLayout`
horizontal/vertical, so native callers do not need a Weasel-specific layout
branch. Weasel/Squirrel frontend style colors are
stored in `config.skin` after converting common Rime `0xBBGGRR` values to
standard `#rrggbb`.
Rime `punctuator/full_shape` and
`punctuator/half_shape` maps are persisted as `config.punctuationFullShape` and
`config.punctuationHalfShape`; Windows TSF reads those maps from the same local
config file before using its default punctuation table. Rime recognizer patterns
are exposed as `config.recognizerPatterns` and through
`rime-recognizer-patterns-json`, so native glue can query or reload URL/email,
uppercase, reverse-lookup, and future recognizer rules through the command bus.
`recognizer-decision-json` lets native callers ask whether the current buffer is
a literal passthrough recognizer match. `key-event-json` uses the same Go-side
decision before punctuation handling, so URL/email/uppercase buffers keep ASCII
characters such as `.`, `@`, `-`, and ending `,` instead of being interrupted by
Chinese punctuation or candidate selection in platform glue.
Rime `app_options/<app>` patches are converted into shared `appRules`, including
app-scoped `ascii_mode`, `ascii_punct`, disable-candidate, and disable-learning
behavior for games, terminals, IDEs, and macOS bundle identifiers.

`app-rules-json`, `resolve-app-context-json`, `ShurufaAppRulesJSON`, and
`ShurufaResolveAppContextJSON` reserve the app-aware behavior surface that
native TSF/IMKit glue needs for WeChat-like scene switching. Rules live in the
shared config as `appRules`, sorted by priority. Built-in rules cover password
fields, terminals, games/esports contexts, and code editors. A context payload
can include `processName`, `exePath`, `windowTitle`, `windowClass`,
`passwordField`, `terminal`, and `gameMode`; the decision returns a derived
config plus `mode`, `punctuation`, `candidateLayout`, `disableCandidates`, and
`disableLearning`. Platform glue should call this before composition starts or
when focus changes, then use the returned mode/candidate flags without hard
coding game or password behavior in C++.

`catalog-json` and `ShurufaCatalogJSON` reserve the shared emoji, kaomoji,
symbol, and agent resource surface for future native panels. The payload accepts
`kind=all|emoji|kaomoji|symbol|agent`, `query` or `input`, and `limit`; slash
queries such as `/fs` and Rime Ice v-mode queries such as `vfs` are normalized
to the stored Rime code. The response is:

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

`reverse-lookup-json` and `ShurufaReverseLookupJSON` reserve the shared Rime-style
reverse lookup surface. The payload accepts `query`, `text`, or `input` plus
`limit`, and returns matching dictionary entries with readings, weights, source,
kind, and comment metadata. Native panels can use this to show pinyin for a
selected Chinese word without parsing dictionaries in C++.

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
double pinyin scheme, punctuation, key behavior profile, update-channel, and mode changes that should
be visible without reinstalling the TSF DLL.

`skin-presets-json` and `apply-skin-preset-json` expose Go-owned
candidate-window presets for native renderers and settings surfaces. Presets
include WeChat-like light/dark horizontal strips, a Microsoft-light strip, and a
Rime-style vertical strip; applying one updates colors, page size, layout, and
candidate comment visibility through shared config while preserving the user's
font family.

`ShurufaReloadDictionaries` reloads local `.json` and `.json.gz` dictionary
files from `%APPDATA%\shurufa233\dictionaries` or
`SHURUFA233_DICTIONARY_DIR`, then adds or updates entries in active sessions.
`ShurufaDictionaryManifestJSON` returns the installed manifest when
`manifest.json` or `dictionary-manifest.json` exists.
`ShurufaDictionarySourcesJSON` returns built-in hot-update presets, including
the default GitHub Release source, the China mirror-ready source, and raw Rime
source references. `ShurufaApplyDictionarySourceJSON` accepts
`{"id":"shurufa233-github-cn"}` plus optional `manifestUrls` and
`mirrorBaseUrls`; passing an empty `mirrorBaseUrls` array explicitly disables
mirrors while preserving the selected preset.
`ShurufaDictionaryUpdateCheckJSON` and `ShurufaDictionaryUpdateApplyJSON` expose
the daemon hot-update path directly to native callers: they read the configured
manifest URLs, try mirror templates before canonical URLs, verify artifact and
content SHA-256 when provided, decompress gzip dictionaries, persist the
installed manifest/dictionaries, hot-load active sessions, and update
`installedVersion` in shared config. Payloads may override `manifestUrls`,
`mirrorBaseUrls`, and `force` without changing C++ glue.

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

The generic command bus also accepts and exports Rime/Weasel/Squirrel
`custom_phrase.txt` text directly. Use `rime-custom-phrase-text` or
`user-phrases-rime-text` to export, and `import-rime-custom-phrases` or
`import-user-phrases-json` with `{"format":"rime-custom-phrase","data":"..."}`
to import at runtime:

```json
{ "format": "rime-custom-phrase", "data": "马上到！\tmsd\t1\n", "merge": true }
```

For whole-profile migration from Weasel/Squirrel/Rime, use
`ShurufaImportRimeProfileJSON` or the command alias
`rime-profile-import-json`. The payload may include `rimeUserDBText`,
`rimeCustomPhraseText`, and `customYaml`; shurufa233 parses those source formats
and stores the result as the stable native profile: `userScores`, `userPhrases`,
and shared config. Imports merge by default to protect existing local learning;
pass `{"replace":true}` or `{"action":"replace"}` to replace migrated userdb and
custom phrase rows.

`ShurufaUserRejectsJSON`, `ShurufaImportUserRejectsJSON`,
`ShurufaRejectCandidate`, `ShurufaUserPinsJSON`, `ShurufaImportUserPinsJSON`,
and `ShurufaPinCandidate` reserve the bad-candidate hide/restore and
good-candidate pin/restore surfaces. Hidden rows are persisted in
`user-rejects.json` or `SHURUFA233_USER_REJECTS`, pinned rows are persisted in
`user-pins.json` or `SHURUFA233_USER_PINS`, and the Go core applies them before
ranking:

```json
{ "entries": [{ "reading": "ceshi", "text": "错词", "comment": "已屏蔽" }], "merge": true }
```

`ShurufaProfileJSON`, `ShurufaImportProfileJSON`, `profile-json`, and
`import-profile-json` reserve a single user-profile migration surface. The
bundle contains normalized config, learned `userScores`, fixed `phrases`, hidden
`rejects`, pinned `pins`, and `counts`. Import merges by default and replaces
local profile sections when `merge=false`:

```json
{
  "version": 1,
  "product": "shurufa233",
  "config": { "...": "engine.Config" },
  "userScores": { "nihao|你好": 25 },
  "phrases": [{ "reading": "msd", "text": "马上到！", "weight": 60000 }],
  "rejects": [{ "reading": "ceshi", "text": "错词" }],
  "pins": [{ "reading": "nihao", "text": "你好" }],
  "merge": true
}
```

`ShurufaSyncConfigJSON`, `ShurufaApplySyncConfigJSON`,
`ShurufaExportProfileSyncJSON`, `ShurufaImportProfileSyncJSON`, and the
`sync-*` command aliases reserve the Rime-style user-data sync surface. The
shared config carries `sync.enabled`, `provider`, `directory`, optional
`remoteUrl`, `mirrorBaseUrls`, `autoExport`, `autoImport`, and
`conflictPolicy`. The implemented path writes or reads
`shurufa233-profile.json` from a local sync directory; remote GitHub/WebDAV
fields are protocol metadata for future authenticated runners and are not used
by the TSF hot path.

`ShurufaAgentCompose` is the native bridge for agent-style input actions. It
returns built-in prompt candidates today and keeps the ABI stable for later
local/cloud model routing in the Go daemon.
`ShurufaAgentConfigJSON`, `ShurufaApplyAgentConfigJSON`, `agent-config-json`,
and `apply-agent-config-json` expose the shared provider-neutral agent config to
native menus and future TSF candidate actions. The payload carries `enabled`,
`provider`, optional `endpoint`, `model`, `systemPrompt`, `triggers`, `actions`,
and `timeoutMs`; applying it normalizes defaults, persists `config.json`, and
refreshes active Go sessions without requiring another Windows C++ export.
