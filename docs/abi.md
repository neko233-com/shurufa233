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
char* ShurufaSelect(uint64_t session, int index);
void ShurufaFree(char* value);
```

Every function returning `char*` returns UTF-8 JSON. Call `ShurufaFree` after copying the JSON.

Example response:

```json
{
  "buffer": "nihao",
  "candidates": [
    { "text": "你好", "reading": "nihao", "weight": 15000, "userScore": 0 }
  ],
  "updatedAt": "2026-07-04T00:00:00Z"
}
```

## Hot Path

Windows TSF uses compact non-JSON calls on the per-key path:

```c
int ShurufaInputKeyFast(uint64_t session, char key);
int ShurufaBackspaceFast(uint64_t session);
int ShurufaCandidateCount(uint64_t session);
char* ShurufaCandidatePayload(uint64_t session, int limit);
char* ShurufaCandidatePayloadRange(uint64_t session, int start, int limit);
char* ShurufaCommitCandidate(uint64_t session, int index);
```

`ShurufaCandidatePayload` returns up to `limit` UTF-8 rows separated by `\n` from the first candidate.
`ShurufaCandidatePayloadRange` returns a paged slice beginning at `start`; Windows uses it for Microsoft IME-style candidate paging.
Each row is:

```text
display_index<TAB>text<TAB>reading<TAB>score<TAB>kind<TAB>source
```

`kind` and `source` are optional extension fields. Current kinds include ordinary word candidates plus `emoji`, `kaomoji`, `symbol`, and `phrase`; renderers must tolerate older four-column payloads. Built-in examples include `zan` -> `👍` (`emoji`), `kaixin` -> `ヽ(・∀・)ﾉ` (`kaomoji`), and `shengluehao` -> `……` (`symbol`).

The Windows glue calls `ShurufaFree` after copying the returned payload. Older per-candidate getters remain available as a compatibility fallback.

The in-process core reads the same local config file as the daemon (`%APPDATA%\shurufa233\config.json`, or `SHURUFA233_CONFIG` when set) when creating a session. This keeps TSF hot-path behavior such as fuzzy initials aligned with the settings UI and daemon IPC.
