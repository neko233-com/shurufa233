package engine

type DictionarySourcePreset struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Kind           string                `json:"kind"`
	Description    string                `json:"description"`
	Homepage       string                `json:"homepage"`
	License        string                `json:"license"`
	Installable    bool                  `json:"installable"`
	ManifestURLs   []string              `json:"manifestUrls,omitempty"`
	MirrorBaseURLs []string              `json:"mirrorBaseUrls,omitempty"`
	RawSources     []DictionaryRawSource `json:"rawSources,omitempty"`
	ConvertCommand string                `json:"convertCommand,omitempty"`
	SyncCommand    string                `json:"syncCommand,omitempty"`
}

type DictionaryRawSource struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Role  string `json:"role"`
}

var builtinDictionarySources = []DictionarySourcePreset{
	{
		ID:          "shurufa233-github",
		Name:        "shurufa233 Release",
		Kind:        "manifest",
		Description: "默认可安装热更源。由 shurufa233 发布转换后的 Rime/OpenCC 词库包，daemon 可直接检查和应用。",
		Homepage:    "https://github.com/neko233-com/shurufa233",
		License:     "mixed-source-manifest",
		Installable: true,
		ManifestURLs: []string{
			"https://github.com/neko233-com/shurufa233/releases/latest/download/dictionary-manifest.json",
		},
	},
	{
		ID:          "shurufa233-github-cn",
		Name:        "shurufa233 Release (GitHub mirror-ready)",
		Kind:        "manifest",
		Description: "中国地区可选热更源。保留 GitHub Release 作为权威地址，并优先尝试 mirrorBaseUrls 中的 GitHub 代理模板；用户可在设置里替换为自建或企业镜像。",
		Homepage:    "https://github.com/neko233-com/shurufa233",
		License:     "mixed-source-manifest",
		Installable: true,
		ManifestURLs: []string{
			"https://github.com/neko233-com/shurufa233/releases/latest/download/dictionary-manifest.json",
		},
		MirrorBaseURLs: []string{
			"https://gh-proxy.com/{url}",
		},
	},
	{
		ID:          "rime-luna-source",
		Name:        "Rime 朙月拼音",
		Kind:        "rime-source",
		Description: "Rime 官方 Luna Pinyin 基础词库，适合作为拼音基础词频来源，不直接安装，需先转换成 shurufa233 JSON manifest。",
		Homepage:    "https://github.com/rime/rime-luna-pinyin",
		License:     "LGPL",
		RawSources: []DictionaryRawSource{
			{Label: "luna_pinyin.dict.yaml", URL: "https://raw.githubusercontent.com/rime/rime-luna-pinyin/master/luna_pinyin.dict.yaml", Role: "dictionary"},
		},
		ConvertCommand: `shurufa-dictimport -language zh-CN -version luna-pinyin -source rime-luna-pinyin -out data\dictionaries\zh-CN.luna-pinyin.json path\to\luna_pinyin.dict.yaml`,
		SyncCommand:    `shurufa-dictsync -preset rime-luna-source -version luna-pinyin-YYYY.MM.DD`,
	},
	{
		ID:          "rime-ice-source",
		Name:        "雾凇拼音 Rime Ice",
		Kind:        "rime-source",
		Description: "长期维护的简体中文 Rime 配置和精校词库，适合作为生产主词库来源。建议本地 clone 后让 import_tables 递归解析。",
		Homepage:    "https://github.com/iDvel/rime-ice",
		License:     "GPL-3.0",
		RawSources: []DictionaryRawSource{
			{Label: "rime_ice.dict.yaml", URL: "https://raw.githubusercontent.com/iDvel/rime-ice/main/rime_ice.dict.yaml", Role: "entry-dictionary"},
			{Label: "symbols_v.yaml", URL: "https://raw.githubusercontent.com/iDvel/rime-ice/main/symbols_v.yaml", Role: "symbols"},
			{Label: "symbols_caps_v.yaml", URL: "https://raw.githubusercontent.com/iDvel/rime-ice/main/symbols_caps_v.yaml", Role: "symbols"},
			{Label: "opencc/emoji.txt", URL: "https://raw.githubusercontent.com/iDvel/rime-ice/main/opencc/emoji.txt", Role: "opencc-emoji"},
		},
		ConvertCommand: `shurufa-dictimport -language zh-CN -version rime-ice -source rime-ice -missing-imports=warn -out data\dictionaries\zh-CN.rime-ice.json.gz path\to\rime-ice\rime_ice.dict.yaml`,
		SyncCommand:    `shurufa-dictsync -preset rime-ice-source -version rime-ice-YYYY.MM.DD`,
	},
	{
		ID:          "rime-emoji-source",
		Name:        "Rime Emoji / OpenCC",
		Kind:        "opencc-source",
		Description: "Rime emoji OpenCC 数据，可转换为 emoji、kaomoji、symbol 候选并进入共享 catalog。",
		Homepage:    "https://github.com/rime/rime-emoji",
		License:     "LGPL",
		RawSources: []DictionaryRawSource{
			{Label: "opencc/emoji_word.txt", URL: "https://raw.githubusercontent.com/rime/rime-emoji/master/opencc/emoji_word.txt", Role: "opencc-emoji"},
			{Label: "opencc/emoji_category.txt", URL: "https://raw.githubusercontent.com/rime/rime-emoji/master/opencc/emoji_category.txt", Role: "opencc-emoji"},
		},
		ConvertCommand: `shurufa-dictimport -language zh-CN -version rime-emoji -source rime-emoji -out data\dictionaries\zh-CN.rime-emoji.json path\to\rime-emoji\opencc\emoji_word.txt`,
		SyncCommand:    `shurufa-dictsync -preset rime-emoji-source -version rime-emoji-YYYY.MM.DD`,
	},
}

func BuiltinDictionarySources() []DictionarySourcePreset {
	out := make([]DictionarySourcePreset, len(builtinDictionarySources))
	copy(out, builtinDictionarySources)
	return out
}

func DictionarySourceByID(id string) (DictionarySourcePreset, bool) {
	for _, source := range builtinDictionarySources {
		if source.ID == id {
			return source, true
		}
	}
	return DictionarySourcePreset{}, false
}

func UpdateConfigFromDictionarySource(config Config, source DictionarySourcePreset) Config {
	config.Update.SourcePreset = source.ID
	if len(source.ManifestURLs) > 0 {
		config.Update.ManifestURLs = append([]string(nil), source.ManifestURLs...)
	}
	if source.MirrorBaseURLs != nil {
		config.Update.MirrorBaseURLs = append([]string(nil), source.MirrorBaseURLs...)
	}
	return config
}
