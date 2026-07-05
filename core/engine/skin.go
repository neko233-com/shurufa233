package engine

import "strings"

var builtinSkinPresets = []SkinPreset{
	{
		ID:                    "wechat-clean",
		Name:                  "WeChat Clean",
		Description:           "Microsoft YaHei UI, horizontal candidates, light border, and a calm green accent for a WeChat-like strip.",
		Tags:                  []string{"wechat", "horizontal", "light"},
		Skin:                  skinWithTheme("#16a34a", "#ffffff", "#111827", "#64748b", "#d7dee8", "#ffffff", "wechat-clean"),
		CandidatePageSize:     7,
		CandidateLayout:       "horizontal",
		ShowCandidateComments: false,
	},
	{
		ID:                    "wechat-dark",
		Name:                  "WeChat Dark",
		Description:           "Dark WeChat-like candidate strip for game and night use while keeping native TSF rendering readable.",
		Tags:                  []string{"wechat", "horizontal", "dark", "game"},
		Skin:                  skinWithTheme("#22c55e", "#111827", "#f8fafc", "#94a3b8", "#334155", "#ffffff", "wechat-dark"),
		CandidatePageSize:     7,
		CandidateLayout:       "horizontal",
		ShowCandidateComments: false,
	},
	{
		ID:                    "microsoft-light",
		Name:                  "Microsoft Light",
		Description:           "Neutral Windows 11-like candidate strip that coexists visually with Microsoft Pinyin.",
		Tags:                  []string{"microsoft", "windows11", "horizontal", "light"},
		Skin:                  skinWithTheme("#2563eb", "#ffffff", "#111827", "#64748b", "#d1d5db", "#ffffff", "microsoft-light"),
		CandidatePageSize:     7,
		CandidateLayout:       "horizontal",
		ShowCandidateComments: false,
	},
	{
		ID:                    "rime-vertical",
		Name:                  "Rime Vertical",
		Description:           "Vertical candidate list with comments kept visible for Rime/Weasel/Squirrel migration checks.",
		Tags:                  []string{"rime", "vertical", "comments"},
		Skin:                  skinWithTheme("#38bdf8", "#111827", "#f8fafc", "#94a3b8", "#334155", "#ffffff", "rime-vertical"),
		CandidatePageSize:     5,
		CandidateLayout:       "vertical",
		ShowCandidateComments: true,
	},
}

func skinWithTheme(accent string, surface string, text string, muted string, border string, highlight string, theme string) Skin {
	return Skin{
		FontFamily:    "Microsoft YaHei UI",
		FontSize:      15,
		Accent:        accent,
		Surface:       surface,
		Text:          text,
		MutedText:     muted,
		Border:        border,
		HighlightText: highlight,
		Theme:         theme,
	}
}

func BuiltinSkinPresets() []SkinPreset {
	out := make([]SkinPreset, len(builtinSkinPresets))
	for i, preset := range builtinSkinPresets {
		out[i] = cloneSkinPreset(preset)
	}
	return out
}

func SkinPresetByID(id string) (SkinPreset, bool) {
	normalized := normalizeSkinPresetID(id)
	for _, preset := range builtinSkinPresets {
		if preset.ID == normalized {
			return cloneSkinPreset(preset), true
		}
	}
	return SkinPreset{}, false
}

func ApplySkinPresetConfig(config Config, id string) (Config, bool) {
	preset, ok := SkinPresetByID(id)
	if !ok {
		return config, false
	}
	next := config
	next.Skin = preset.Skin
	if strings.TrimSpace(config.Skin.FontFamily) != "" {
		next.Skin.FontFamily = config.Skin.FontFamily
	}
	next.CandidatePageSize = normalizeCandidatePageSize(preset.CandidatePageSize)
	next.CandidateLayout = normalizeCandidateLayout(preset.CandidateLayout)
	next.ShowCandidateComments = preset.ShowCandidateComments
	return next, true
}

func normalizeSkinPresetID(id string) string {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "", "wechat", "weixin", "wechat-light", "wx":
		return "wechat-clean"
	case "dark", "wechat-night", "night", "game":
		return "wechat-dark"
	case "microsoft", "windows", "windows11", "win11", "ms":
		return "microsoft-light"
	case "rime", "weasel", "squirrel", "vertical":
		return "rime-vertical"
	default:
		return strings.ToLower(strings.TrimSpace(id))
	}
}

func cloneSkinPreset(preset SkinPreset) SkinPreset {
	preset.Tags = append([]string(nil), preset.Tags...)
	return preset
}
