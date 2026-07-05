# Dictionary Sources

shurufa233 should not maintain a Chinese dictionary from zero. The built-in
dictionary can stay small for smoke tests, while production dictionaries should
be generated from reviewed open-source sources and user-provided imports.

## Recommended Upstream Sources

- Rime Luna Pinyin: `https://github.com/rime/rime-luna-pinyin`
- Rime Ice: `https://github.com/iDvel/rime-ice`
- Rime Emoji: `https://github.com/rime/rime-emoji`
- OpenCC Emoji: `https://github.com/rime-aca/OpenCC_Emoji`
- Rime Plum package manager and recipes: `https://github.com/rime/plum`

Check each upstream repository's license before redistributing generated JSON
artifacts. Prefer storing source URL, commit, license, and conversion command in
release notes or dictionary manifests.

## Built-In Source Catalog

The daemon and CLI expose a small source catalog so production builds do not
depend on hand-maintained URLs scattered through the UI:

```powershell
shurufa-imecli update-sources
shurufa-imecli update-source shurufa233-github
```

`shurufa233-github` is directly installable because it points at a shurufa233
Release manifest. Rime Luna, Rime Ice, and Rime Emoji entries are marked
source-only: they list upstream GitHub raw files and the conversion command, but
they are not applied directly until a generated shurufa233 dictionary manifest is
published. This keeps Rime/OpenCC license review, source commit pinning, and
hash generation explicit while still making the maintained upstreams discoverable
from the settings UI and CLI.

For China-region access, keep the upstream GitHub URLs as provenance and publish
the generated shurufa233 manifest/artifacts to a user-controlled mirror or CDN.
Put that mirror in `mirrorBaseUrls`; the daemon tries mirror manifest and
dictionary URLs before falling back to the canonical GitHub URLs. A mirror can be
a static base such as `https://cdn.example/dicts`, or a proxy template such as
`https://gh-proxy.com/{url}`. The template keeps one source URL in the manifest
while still allowing China-region GitHub release acceleration.

## One-Command Upstream Sync

`shurufa-dictsync` wraps the normal Rime release workflow so dictionary updates
do not depend on manually cloning repositories and stitching commands together.
It is a pure Go tool, ships in the Windows package, and does not require Visual
Studio on machines that only refresh dictionaries.

Default Rime Ice sync:

```powershell
shurufa-dictsync `
  -preset rime-ice-source `
  -version rime-ice-2026.07.05 `
  -base-url https://github.com/neko233-com/shurufa233/releases/latest/download
```

The command clones or updates the upstream repository under
`.cache/dictionaries`, converts the preset's known source files with
`shurufa-dictimport`, records the exact upstream commit, and writes
`data/dictionaries/dictionary-manifest.json` with `shurufa-dictmanifest`.
For Rime Ice, the preset converts `rime_ice.dict.yaml`, `symbols_v.yaml`, and
`opencc/emoji.txt` together, so imported OpenCC emoji rows can reuse readings
from the main dictionary in the same conversion pass.

Useful sync flags:

- `-preset`: `rime-ice-source`, `rime-luna-source`, or `rime-emoji-source`.
- `-ref`: pin a branch, tag, or commit before conversion.
- `-workdir`: choose where upstream Git checkouts are cached.
- `-out-dir`: choose where generated `.json.gz` dictionaries are written.
- `-manifest`: choose the generated manifest path.
- `-mirror-url`: add a clone mirror before canonical GitHub fallback. Repeat it
  for multiple mirrors.
- `-skip-pull`: reuse an existing checkout for offline or audited builds.

Mirror templates can use `{url}` for the canonical clone URL and `{repo}` for
the `owner/repo` slug:

```powershell
shurufa-dictsync `
  -preset rime-ice-source `
  -mirror-url "https://ghproxy.example/{url}" `
  -mirror-url "https://git.example/{repo}.git"
```

The generated manifest still keeps the canonical upstream homepage and commit
as provenance; mirrors are transport accelerators, not source-of-truth metadata.
After checking licenses and publishing the generated `.json.gz` plus
`dictionary-manifest.json` to a GitHub Release or CDN, end users can install the
hot update from the settings UI, daemon endpoints, or:

```powershell
shurufa-imecli update-check
shurufa-imecli update-apply
shurufa-imecli update-source shurufa233-github-cn
shurufa-imecli update-source shurufa233-github-cn --mirror "https://gh-proxy.com/{url}"
```

## Rime Import

Rime dictionaries usually use `.dict.yaml` files with a YAML header ending in
`...`, followed by tab-separated rows:

```text
词条<TAB>pin yin<TAB>weight
```

When a Rime dictionary declares `columns`, the importer follows that column
layout instead of assuming the default order. Inline and block styles are both
accepted:

```yaml
columns: [text, code, weight, stem, comment]
```

```yaml
columns:
  - code
  - text
  - weight
  - comment
```

The optional `stem` column is ignored safely; `text`, `code`, and `weight` are
used to produce shurufa233 entries. Optional `comment`, `comments`, or
`annotation` columns are preserved as candidate comments so native and React
candidate strips can show source-provided hints for symbols, emoji, phrases, and
specialized dictionaries.

Some large Rime dictionaries, including rime-ice style Tencent vector word
lists, declare `columns: [text, weight]` and rely on previously imported
character dictionaries for automatic annotation. When `import_tables` brings in
annotated character rows first, `shurufa-dictimport` now infers a weight-only
word's reading from those imported character readings. Rows with characters
that cannot be inferred are skipped instead of producing broken candidates.

Rime user phrases such as `custom_phrase.txt` are also supported. These files
often have no YAML header and use the same table shape:

```text
词条<TAB>编码<TAB>权重
```

For easier migration from existing personal Rime folders, `shurufa-dictimport`
also accepts whitespace-separated custom phrase rows when there is no YAML
header, so examples such as `What the fuck! wtf 3` and
`http://rime.im/ rime 1` import as fixed high-priority personal phrases.
Because Rime usually gives `custom_phrase` a high `initial_quality`, headerless
custom phrase rows are mapped into a high shurufa233 weight band while keeping
their row weight for ordering within the personal phrase set.

Convert one or more Rime dictionaries into the shurufa233 JSON format:

```powershell
go run ./cmd/dictimport `
  -language zh-CN `
  -version rime-luna-2026-07-05 `
  -source rime-luna-pinyin `
  -out data/dictionaries/zh-CN.rime-luna.json `
  path\to\luna_pinyin.dict.yaml
```

Convert personal Rime custom phrases into a shurufa233 dictionary:

```powershell
go run ./cmd/dictimport `
  -language zh-CN `
  -version custom-phrase-2026-07-05 `
  -source rime-custom-phrase `
  -out data/dictionaries/zh-CN.custom-phrase.json `
  path\to\custom_phrase.txt
```

For runtime editing without rebuilding a dictionary release, fixed phrases can
also be managed through the daemon and CLI. They are persisted in
`user-phrases.json`, loaded as `kind=phrase`, `source=user-phrase`, and kept
separate from learned user scores:

Rime synchronized user dictionaries such as `luna_pinyin.userdb.txt` and
`rime_ice.userdb.txt` are treated as learned word scores rather than fixed
phrases. Import them at runtime with `shurufa-imecli wordbook import
path\to\luna_pinyin.userdb.txt` or `PUT /wordbook` using
`format=rime-userdb`. A row such as `cha jian 插件 c=4 d=0.5 t=8` becomes the
stable shurufa233 score key `chajian|插件`, so the final user data format stays
portable even though migration understands Rime's sync text.

```powershell
shurufa-imecli phrases add msd "马上到！" 60000
shurufa-imecli phrases import .\user-phrases.json --replace
shurufa-imecli phrases export
```

Convert Rime symbol tables such as `symbols.yaml` or `symbols.custom.yaml`:

```powershell
go run ./cmd/dictimport `
  -language zh-CN `
  -version rime-symbols-2026-07-05 `
  -source rime-symbols `
  -out data\dictionaries\zh-CN.rime-symbols.json `
  path\to\symbols.yaml
```

The importer recognizes common `punctuator/symbols` and
`punctuator/symbols/+` flow-list rows such as:

```yaml
patch:
  punctuator/symbols/+:
    '/fs': [℃, ℉, °]
    '/xh': ['※', '★', '☆']
```

Block-list rows are accepted too:

```yaml
patch:
  punctuator/symbols/+:
    '/dw':
      - ℃
      - ℉
      - °
```

Rime's leading slash is dropped for the stored shurufa233 reading, so `/fs`
becomes the `fs` candidate code. The runtime still accepts `/fs` as a
slash-prefixed symbol input, preserves the slash in the preedit buffer, and
filters candidates to symbol, emoji, kaomoji, and agent rows. Imported rows are
tagged as `symbol`, `emoji`, or `kaomoji` where possible so native and React
candidate strips can keep showing the same readable badges.

Rime OpenCC emoji/symbol tables are also accepted. Projects such as
`rime/rime-emoji` and Rime Ice ship files like `opencc/emoji_word.txt` or
`opencc/emoji.txt` with rows shaped as:

```text
微笑<TAB>微笑 😊 [微笑]
ID<TAB>ID 🆔️ 🪪
```

ASCII keys such as `ID` are imported directly as `id` candidates. For Chinese
keys such as `微笑`, pass a Rime dictionary first so `shurufa-dictimport` can
reuse its text-to-pinyin readings and then convert the OpenCC emoji rows into
real pinyin candidates:

```powershell
go run ./cmd/dictimport `
  -language zh-CN `
  -version rime-ice-opencc-emoji-2026-07-05 `
  -source rime-ice-opencc `
  -out data\dictionaries\zh-CN.rime-opencc-emoji.json.gz `
  path\to\rime_ice.dict.yaml `
  path\to\opencc\emoji.txt
```

This keeps emoji and symbol expansion sourced from maintained Rime GitHub
projects instead of hand-maintaining a separate shurufa233-only table. The same
imported rows also appear in the shared catalog API (`GET /catalog`,
`shurufa-imecli symbols`, and `catalog-json` in the C ABI), which is the
foundation for a WeChat-style emoji/symbol panel in both React/Wails and future
native candidate windows.

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
  -source-preset rime-ice-source `
  -source-url https://github.com/iDvel/rime-ice `
  -source-commit <pinned-upstream-commit> `
  -license GPL-3.0 `
  -convert-command "go run ./cmd/dictimport -language zh-CN -version rime-ice-2026-07-05 -source rime-ice -out data/dictionaries/zh-CN.rime-ice.json.gz path\to\rime_ice.dict.yaml" `
  -base-url https://github.com/neko233-com/shurufa233/releases/latest/download `
  -out data/dictionaries/dictionary-manifest.json `
  data/dictionaries/zh-CN.rime-ice.json.gz
```

`shurufa-dictmanifest` reads each dictionary artifact, validates that it is a
real shurufa233 dictionary, detects gzip automatically, computes both artifact
and decompressed-content SHA-256 hashes, and emits the manifest shape consumed
by `GET /updates/check` and `POST /updates/apply`.
The optional source fields are copied to the manifest and each dictionary row,
so settings UI, daemon logs, and future native panels can show where a hot-update
dictionary came from without baking upstream details into C++.
