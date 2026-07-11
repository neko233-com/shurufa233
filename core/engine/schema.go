package engine

import "strings"

var builtinSchemaPresets = []SchemaPreset{
	{
		ID:                    "wechat-pinyin",
		Name:                  "微信/微软全拼",
		Kind:                  "full-pinyin",
		Description:           "默认中文拼音方案，横排候选、中文标点、兼容微软输入法的日常交互。",
		Tags:                  []string{"wechat", "microsoft", "default"},
		Language:              "zh-CN",
		DoublePinyin:          false,
		FuzzyInitials:         []string{"zh=z", "ch=c", "sh=s"},
		Punctuation:           "full",
		KeyProfile:            "wechat",
		ShiftToggleMode:       true,
		SemicolonQuickSelect:  true,
		QuoteQuickSelect:      true,
		BracketPageKeys:       true,
		MinusEqualPageKeys:    true,
		CommaPeriodPageKeys:   false,
		CandidateLayout:       "horizontal",
		ShowCandidateComments: false,
	},
	{
		ID:                     "rime-luna-pinyin",
		Name:                   "Rime 朙月拼音",
		Kind:                   "full-pinyin",
		RimeID:                 "luna_pinyin",
		Description:            "面向小狼毫/鼠须管词库迁移的 Rime 朙月拼音兼容方案，保留候选注释与竖排习惯。",
		Tags:                   []string{"rime", "luna-pinyin"},
		Language:               "zh-CN",
		DoublePinyin:           false,
		FuzzyInitials:          []string{"zh=z", "ch=c", "sh=s"},
		Punctuation:            "full",
		KeyProfile:             "rime",
		ShiftToggleMode:        true,
		SemicolonQuickSelect:   false,
		QuoteQuickSelect:       false,
		BracketPageKeys:        true,
		MinusEqualPageKeys:     true,
		CommaPeriodPageKeys:    true,
		CandidateLayout:        "vertical",
		ShowCandidateComments:  true,
		DictionarySourcePreset: "rime-luna-pinyin-source",
	},
	{
		ID:                     "rime-ice-pinyin",
		Name:                   "雾凇拼音",
		Kind:                   "full-pinyin",
		RimeID:                 "rime_ice",
		Description:            "面向 Rime Ice 大词库迁移的全拼方案，适合用 GitHub 热更词库替代从零维护。",
		Tags:                   []string{"rime", "rime-ice", "dictionary"},
		Language:               "zh-CN",
		DoublePinyin:           false,
		FuzzyInitials:          []string{"zh=z", "ch=c", "sh=s"},
		Punctuation:            "full",
		KeyProfile:             "rime",
		ShiftToggleMode:        true,
		SemicolonQuickSelect:   false,
		QuoteQuickSelect:       false,
		BracketPageKeys:        true,
		MinusEqualPageKeys:     true,
		CommaPeriodPageKeys:    true,
		CandidateLayout:        "horizontal",
		ShowCandidateComments:  true,
		DictionarySourcePreset: "rime-ice-source",
	},
	{
		ID:                    "double-pinyin-xiaohe",
		Name:                  "小鹤双拼",
		Kind:                  "double-pinyin",
		RimeID:                "double_pinyin_flypy",
		Description:           "小鹤双拼键位，保留全拼 fallback，适合从 Rime 小鹤方案迁移。",
		Tags:                  []string{"rime", "double-pinyin", "flypy"},
		Language:              "zh-CN",
		DoublePinyin:          true,
		DoublePinyinScheme:    "xiaohe",
		FuzzyInitials:         []string{"zh=z", "ch=c", "sh=s"},
		Punctuation:           "full",
		KeyProfile:            "rime",
		ShiftToggleMode:       true,
		SemicolonQuickSelect:  false,
		QuoteQuickSelect:      false,
		BracketPageKeys:       true,
		MinusEqualPageKeys:    true,
		CommaPeriodPageKeys:   true,
		CandidateLayout:       "horizontal",
		ShowCandidateComments: true,
	},
	{
		ID:                    "double-pinyin-microsoft",
		Name:                  "微软双拼",
		Kind:                  "double-pinyin",
		Description:           "微软双拼键位，TSF 胶水层会把分号作为 ing 韵母转交给 Go 内核。",
		Tags:                  []string{"microsoft", "double-pinyin"},
		Language:              "zh-CN",
		DoublePinyin:          true,
		DoublePinyinScheme:    "microsoft",
		FuzzyInitials:         []string{"zh=z", "ch=c", "sh=s"},
		Punctuation:           "full",
		KeyProfile:            "microsoft",
		ShiftToggleMode:       true,
		SemicolonQuickSelect:  true,
		QuoteQuickSelect:      true,
		BracketPageKeys:       true,
		MinusEqualPageKeys:    true,
		CommaPeriodPageKeys:   false,
		CandidateLayout:       "horizontal",
		ShowCandidateComments: false,
	},
	{
		ID:                    "double-pinyin-ziranma",
		Name:                  "自然码双拼",
		Kind:                  "double-pinyin",
		RimeID:                "double_pinyin_ziranma",
		Description:           "自然码双拼键位，兼容主流输入法的经典双拼方案，无零声母键。",
		Tags:                  []string{"ziranma", "natural-code", "double-pinyin"},
		Language:              "zh-CN",
		DoublePinyin:          true,
		DoublePinyinScheme:    "ziranma",
		FuzzyInitials:         []string{"zh=z", "ch=c", "sh=s"},
		Punctuation:           "full",
		KeyProfile:            "wechat",
		ShiftToggleMode:       true,
		SemicolonQuickSelect:  true,
		QuoteQuickSelect:      true,
		BracketPageKeys:       true,
		MinusEqualPageKeys:    true,
		CommaPeriodPageKeys:   false,
		CandidateLayout:       "horizontal",
		ShowCandidateComments: false,
	},
	{
		ID:                    "double-pinyin-sogou",
		Name:                  "搜狗双拼",
		Kind:                  "double-pinyin",
		Description:           "搜狗/微软兼容双拼入口，当前共用微软双拼底层映射，后续可独立替换键位表。",
		Tags:                  []string{"sogou", "double-pinyin"},
		Language:              "zh-CN",
		DoublePinyin:          true,
		DoublePinyinScheme:    "microsoft",
		FuzzyInitials:         []string{"zh=z", "ch=c", "sh=s"},
		Punctuation:           "full",
		KeyProfile:            "microsoft",
		ShiftToggleMode:       true,
		SemicolonQuickSelect:  true,
		QuoteQuickSelect:      true,
		BracketPageKeys:       true,
		MinusEqualPageKeys:    true,
		CommaPeriodPageKeys:   false,
		CandidateLayout:       "horizontal",
		ShowCandidateComments: false,
	},
}

func BuiltinSchemaPresets() []SchemaPreset {
	out := make([]SchemaPreset, len(builtinSchemaPresets))
	copy(out, builtinSchemaPresets)
	for i := range out {
		out[i].Tags = append([]string(nil), out[i].Tags...)
		out[i].FuzzyInitials = append([]string(nil), out[i].FuzzyInitials...)
	}
	return out
}

func SchemaPresetByID(id string) (SchemaPreset, bool) {
	normalized := NormalizeSchemaID(id)
	for _, preset := range builtinSchemaPresets {
		if preset.ID == normalized {
			return cloneSchemaPreset(preset), true
		}
	}
	return SchemaPreset{}, false
}

func NormalizeSchemaID(id string) string {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "", "default", "pinyin", "full-pinyin", "wechat", "microsoft-pinyin", "ms-pinyin":
		return "wechat-pinyin"
	case "luna", "luna-pinyin", "luna_pinyin", "rime-luna", "rime-luna-pinyin":
		return "rime-luna-pinyin"
	case "rime-ice", "rime_ice", "ice", "rime-ice-pinyin":
		return "rime-ice-pinyin"
	case "xiaohe", "flypy", "double-pinyin-flypy", "double_pinyin_flypy", "double-pinyin-xiaohe":
		return "double-pinyin-xiaohe"
	case "ziranma", "zrm", "natural", "natural-code", "double-pinyin-ziranma", "double_pinyin_ziranma":
		return "double-pinyin-ziranma"
	case "microsoft", "ms", "double-pinyin-ms", "double-pinyin-microsoft":
		return "double-pinyin-microsoft"
	case "sogou", "double-pinyin-sogou":
		return "double-pinyin-sogou"
	default:
		return strings.ToLower(strings.TrimSpace(id))
	}
}

func NormalizeSchemaConfig(config Config) Config {
	if preset, ok := SchemaPresetByID(config.Schema); ok && schemaMatchesConfig(preset, config) {
		config.Schema = NormalizeSchemaID(config.Schema)
		return config
	}
	config.Schema = DeriveSchemaID(config)
	return config
}

func DeriveSchemaID(config Config) string {
	if config.DoublePinyin {
		switch normalizeDoublePinyinScheme(config.DoublePinyinScheme) {
		case "microsoft":
			return "double-pinyin-microsoft"
		case "ziranma":
			return "double-pinyin-ziranma"
		default:
			return "double-pinyin-xiaohe"
		}
	}
	if strings.EqualFold(strings.TrimSpace(config.CandidateLayout), "vertical") {
		return "rime-luna-pinyin"
	}
	if strings.EqualFold(strings.TrimSpace(config.Update.SourcePreset), "rime-ice-source") {
		return "rime-ice-pinyin"
	}
	return "wechat-pinyin"
}

func ApplySchemaPresetConfig(config Config, id string) (Config, bool) {
	preset, ok := SchemaPresetByID(id)
	if !ok {
		return NormalizeSchemaConfig(config), false
	}
	config.Schema = preset.ID
	if preset.Language != "" {
		config.Language = preset.Language
	}
	config.DoublePinyin = preset.DoublePinyin
	config.DoublePinyinScheme = normalizeDoublePinyinScheme(preset.DoublePinyinScheme)
	if len(preset.FuzzyInitials) > 0 {
		config.FuzzyInitials = append([]string(nil), preset.FuzzyInitials...)
	}
	if preset.Punctuation != "" {
		config.Punctuation = preset.Punctuation
	}
	if preset.KeyProfile != "" {
		config.KeyProfile = preset.KeyProfile
		config.ShiftToggleMode = preset.ShiftToggleMode
		config.SemicolonQuickSelect = preset.SemicolonQuickSelect
		config.QuoteQuickSelect = preset.QuoteQuickSelect
		config.BracketPageKeys = preset.BracketPageKeys
		config.MinusEqualPageKeys = preset.MinusEqualPageKeys
		config.CommaPeriodPageKeys = preset.CommaPeriodPageKeys
	}
	if preset.CandidateLayout != "" {
		config.CandidateLayout = preset.CandidateLayout
	}
	config.ShowCandidateComments = preset.ShowCandidateComments
	// Product profiles are an experience preset, not only a spelling table.
	// Switching back from a Rime-oriented profile must immediately restore the
	// Microsoft/WeChat strip contract instead of retaining a tall legacy page.
	if NormalizeKeyProfile(preset.KeyProfile) != "rime" {
		config.CandidatePageSize = 7
		config.CandidateWindowMode = "win11"
		config.ShowCandidateComments = false
	}
	if preset.DictionarySourcePreset != "" {
		config.Update.SourcePreset = preset.DictionarySourcePreset
	}
	return NormalizeSchemaConfig(config), true
}

func cloneSchemaPreset(preset SchemaPreset) SchemaPreset {
	preset.Tags = append([]string(nil), preset.Tags...)
	preset.FuzzyInitials = append([]string(nil), preset.FuzzyInitials...)
	return preset
}

func NormalizeKeyBehavior(config Config) Config {
	profile := config.KeyProfile
	if strings.TrimSpace(profile) == "" {
		if preset, ok := SchemaPresetByID(config.Schema); ok && preset.KeyProfile != "" {
			profile = preset.KeyProfile
		}
	}
	switch NormalizeKeyProfile(profile) {
	case "custom":
		config.KeyProfile = "custom"
		return config
	case "rime":
		config.KeyProfile = "rime"
		config.ShiftToggleMode = true
		config.SemicolonQuickSelect = false
		config.QuoteQuickSelect = false
		config.BracketPageKeys = true
		config.MinusEqualPageKeys = true
		config.CommaPeriodPageKeys = true
		return config
	case "microsoft":
		config.KeyProfile = "microsoft"
	default:
		config.KeyProfile = "wechat"
	}
	config.ShiftToggleMode = true
	config.SemicolonQuickSelect = true
	config.QuoteQuickSelect = true
	config.BracketPageKeys = true
	config.MinusEqualPageKeys = true
	config.CommaPeriodPageKeys = false
	return config
}

func NormalizeKeyProfile(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "", "default", "wechat", "weixin", "wx":
		return "wechat"
	case "microsoft", "ms", "ms-pinyin", "windows":
		return "microsoft"
	case "rime", "weasel", "squirrel", "luna", "luna-pinyin":
		return "rime"
	case "custom", "advanced":
		return "custom"
	default:
		return "wechat"
	}
}

func schemaMatchesConfig(preset SchemaPreset, config Config) bool {
	if preset.DoublePinyin != config.DoublePinyin {
		return false
	}
	if preset.DoublePinyin && normalizeDoublePinyinScheme(preset.DoublePinyinScheme) != normalizeDoublePinyinScheme(config.DoublePinyinScheme) {
		return false
	}
	if preset.CandidateLayout != "" && normalizeCandidateLayout(preset.CandidateLayout) != normalizeCandidateLayout(config.CandidateLayout) {
		return false
	}
	if preset.KeyProfile != "" && strings.TrimSpace(config.KeyProfile) != "" &&
		NormalizeKeyProfile(preset.KeyProfile) != NormalizeKeyProfile(config.KeyProfile) {
		return false
	}
	if preset.DictionarySourcePreset != "" && strings.TrimSpace(config.Update.SourcePreset) != preset.DictionarySourcePreset {
		return false
	}
	return true
}
