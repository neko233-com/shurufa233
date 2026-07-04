# Dictionary Sources

shurufa233 should not maintain a Chinese dictionary from zero. The built-in
dictionary can stay small for smoke tests, while production dictionaries should
be generated from reviewed open-source sources and user-provided imports.

## Recommended Upstream Sources

- Rime Luna Pinyin: `https://github.com/rime/rime-luna-pinyin`
- Rime Ice: `https://github.com/iDvel/rime-ice`
- Rime Plum package manager and recipes: `https://github.com/rime/plum`

Check each upstream repository's license before redistributing generated JSON
artifacts. Prefer storing source URL, commit, license, and conversion command in
release notes or dictionary manifests.

## Rime Import

Rime dictionaries usually use `.dict.yaml` files with a YAML header ending in
`...`, followed by tab-separated rows:

```text
词条<TAB>pin yin<TAB>weight
```

Convert one or more Rime dictionaries into the shurufa233 JSON format:

```powershell
go run ./cmd/dictimport `
  -language zh-CN `
  -version rime-luna-2026-07-05 `
  -source rime-luna-pinyin `
  -out data/dictionaries/zh-CN.rime-luna.json `
  path\to\luna_pinyin.dict.yaml
```

By default, `shurufa-dictimport` resolves Rime `import_tables` recursively, so
an entry dictionary such as Rime Ice's `rime_ice.dict.yaml` can pull concrete
tables from folders like `cn_dicts/` automatically:

```powershell
go run ./cmd/dictimport `
  -language zh-CN `
  -version rime-ice-2026-07-05 `
  -source rime-ice `
  -out data/dictionaries/zh-CN.rime-ice.json.gz `
  path\to\rime_ice.dict.yaml
```

Useful import flags:

- `-gzip`: write gzip-compressed JSON. This is also enabled automatically when `-out` ends with `.gz`.
- `-imports=false`: convert only the files named on the command line.
- `-missing-imports=error`: fail when an imported table cannot be found. This is the default and safest release behavior.
- `-missing-imports=warn`: keep converting available tables and print warnings for missing optional imports.
- `-missing-imports=skip`: silently ignore missing imports for quick local experiments.

After conversion, point `data/dictionaries/dictionary-manifest.json` or a GitHub
Release manifest at the generated JSON or JSON gzip artifact so the daemon can
hot-update it. For `.json.gz` releases, set manifest `compression` to `gzip`,
`sha256` to the compressed artifact hash, and optionally `contentSha256` to the
decompressed JSON hash.

Generate that manifest instead of writing hashes by hand:

```powershell
go run ./cmd/dictmanifest `
  -version rime-ice-2026-07-05 `
  -channel stable `
  -base-url https://github.com/neko233-com/shurufa233/releases/latest/download `
  -out data/dictionaries/dictionary-manifest.json `
  data/dictionaries/zh-CN.rime-ice.json.gz
```

`shurufa-dictmanifest` reads each dictionary artifact, validates that it is a
real shurufa233 dictionary, detects gzip automatically, computes both artifact
and decompressed-content SHA-256 hashes, and emits the manifest shape consumed
by `GET /updates/check` and `POST /updates/apply`.
