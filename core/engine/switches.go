package engine

import "strings"

func SwitchOptions(config Config) []SwitchOption {
	config = NormalizeSwitchConfig(config)
	return []SwitchOption{
		{
			ID:          "ascii_mode",
			Name:        "中英文",
			RimeName:    "ascii_mode",
			Description: "Rime/小狼毫式中英文模式开关，打开时进入英文直通模式。",
			Value:       normalizeMode(config.Mode) == "en",
			On:          "英文",
			Off:         "中文",
			ConfigField: "mode",
		},
		{
			ID:          "ascii_punct",
			Name:        "中英标点",
			RimeName:    "ascii_punct",
			Description: "打开时使用半角英文标点，关闭时使用中文标点。",
			Value:       normalizePunctuation(config.Punctuation) == "half",
			On:          "半角",
			Off:         "中文标点",
			ConfigField: "punctuation",
		},
		{
			ID:          "simplification",
			Name:        "简繁输出",
			RimeName:    "simplification",
			Description: "打开时输出简体，关闭时输出繁体。",
			Value:       normalizeScript(config.Script) == "simplified",
			On:          "简体",
			Off:         "繁体",
			ConfigField: "script",
		},
		{
			ID:          "candidate_comments",
			Name:        "候选注释",
			RimeName:    "comment",
			Description: "控制 Rime/OpenCC 来源注释是否显示在候选窗。",
			Value:       config.ShowCandidateComments,
			On:          "显示",
			Off:         "隐藏",
			ConfigField: "showCandidateComments",
		},
		{
			ID:          "associations",
			Name:        "联想候选",
			RimeName:    "prediction",
			Description: "控制微信式上屏后联想候选。",
			Value:       config.Associations,
			On:          "开启",
			Off:         "关闭",
			ConfigField: "associations",
		},
		{
			ID:          "vertical_candidates",
			Name:        "候选排列",
			RimeName:    "candidate_layout",
			Description: "打开时使用 Rime 式竖排候选，关闭时使用微信/微软式横排候选。",
			Value:       normalizeCandidateLayout(config.CandidateLayout) == "vertical",
			On:          "竖排",
			Off:         "横排",
			ConfigField: "candidateLayout",
		},
	}
}

func ApplySwitch(config Config, id string, value bool, toggle bool) (Config, SwitchOption, bool) {
	id = NormalizeSwitchID(id)
	if id == "" {
		return config, SwitchOption{}, false
	}
	config = NormalizeSwitchConfig(config)
	current, ok := switchValue(config, id)
	if !ok {
		return config, SwitchOption{}, false
	}
	if toggle {
		value = !current
	}
	switch id {
	case "ascii_mode":
		if value {
			config.Mode = "en"
		} else {
			config.Mode = "zh"
		}
	case "ascii_punct":
		if value {
			config.Punctuation = "half"
		} else {
			config.Punctuation = "full"
		}
	case "simplification":
		if value {
			config.Script = "simplified"
		} else {
			config.Script = "traditional"
		}
	case "candidate_comments":
		config.ShowCandidateComments = value
	case "associations":
		config.Associations = value
	case "vertical_candidates":
		if value {
			config.CandidateLayout = "vertical"
		} else {
			config.CandidateLayout = "horizontal"
		}
	}
	config = NormalizeSwitchConfig(config)
	return config, findSwitch(config, id), true
}

func NormalizeSwitchID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	id = strings.ReplaceAll(id, "-", "_")
	switch id {
	case "ascii", "ascii_mode", "mode", "english", "en":
		return "ascii_mode"
	case "ascii_punct", "punctuation", "punct", "ascii_punctuation":
		return "ascii_punct"
	case "simplification", "simplified", "traditional", "script":
		return "simplification"
	case "comment", "comments", "candidate_comment", "candidate_comments", "show_candidate_comments":
		return "candidate_comments"
	case "association", "associations", "prediction", "predict":
		return "associations"
	case "candidate_layout", "layout", "vertical", "vertical_candidates":
		return "vertical_candidates"
	default:
		return ""
	}
}

func NormalizeSwitchConfig(config Config) Config {
	if config.MaxCandidates <= 0 {
		defaults := DefaultConfig()
		defaults.Update = config.Update
		if config.Skin != (Skin{}) {
			defaults.Skin = config.Skin
		}
		config = defaults
	}
	config.Mode = normalizeMode(config.Mode)
	config.Punctuation = normalizePunctuation(config.Punctuation)
	config.Script = normalizeScript(config.Script)
	config.CandidateLayout = normalizeCandidateLayout(config.CandidateLayout)
	config.CandidatePageSize = normalizeCandidatePageSize(config.CandidatePageSize)
	config.DoublePinyinScheme = normalizeDoublePinyinScheme(config.DoublePinyinScheme)
	config = NormalizeSchemaConfig(config)
	config = NormalizeKeyBehavior(config)
	return config
}

func switchValue(config Config, id string) (bool, bool) {
	for _, option := range SwitchOptions(config) {
		if option.ID == id {
			return option.Value, true
		}
	}
	return false, false
}

func findSwitch(config Config, id string) SwitchOption {
	for _, option := range SwitchOptions(config) {
		if option.ID == id {
			return option
		}
	}
	return SwitchOption{}
}

func normalizePunctuation(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "half", "ascii", "en":
		return "half"
	default:
		return "full"
	}
}
