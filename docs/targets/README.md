# Target screenshots

Product UX ground truth for shurufa233. **Rime Ice screenshots are not the primary bar.**

## Landed

| File | Source | Role |
| --- | --- | --- |
| [`ms-pinyin-target.png`](./ms-pinyin-target.png) | Windows 中文输入法 / Microsoft Pinyin (Win11) | **P0 visual + interaction bar** for composition UI |

## Still welcome

| File | Source | Role |
| --- | --- | --- |
| `wechat-strip.png` | 微信输入法 | P0 typing-feel companion (strip rhythm / ranking comfort) |
| `shurufa-target.png` | shurufa233 | Side-by-side self-check against the bars above |

## Spec extracted from `ms-pinyin-target.png`

Two-layer composition chrome (not a single Rime-style annotation soup):

1. **Inline preedit** (in the host editor / composition string)
   - Exact typed spelling with syllable separators: `ju'da'wu'bi`
   - Subtle underline / dotted underline under the composing segment
   - Must stay readable against light or dark editor backgrounds

2. **Floating candidate strip** (caret-anchored, below preedit)
   - Dark semi-opaque rounded bar, soft shadow, no focus steal
   - Horizontal candidates: `1 巨大无比` … `7 剧` (about 7 per page)
   - Selected row: left **vertical accent bar** + rounded pill highlight
   - Unselected: number + text, muted contrast, no heavy comments
   - Trailing chrome: page prev/next + utility icons (settings / more)
   - Compact height; one horizontal row, not vertical Rime list

Agents implementing candidate paint must match this silhouette before inventing new chrome.
