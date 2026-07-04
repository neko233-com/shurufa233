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
