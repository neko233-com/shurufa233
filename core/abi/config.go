package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/neko233-com/shurufa233/core/engine"
)

func configFile() (string, error) {
	if override := os.Getenv("SHURUFA233_CONFIG"); override != "" {
		return override, nil
	}
	base := os.Getenv("APPDATA")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "shurufa233", "config.json"), nil
}

func loadConfig() engine.Config {
	config := engine.DefaultConfig()
	path, err := configFile()
	if err != nil {
		return config
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return config
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return engine.DefaultConfig()
	}
	return normalizeConfig(config)
}

func persistConfig(config engine.Config) error {
	path, err := configFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	config = normalizeConfig(config)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		if retryErr := os.Rename(tmp, path); retryErr != nil {
			_ = os.Remove(tmp)
			return retryErr
		}
	}
	return nil
}

func normalizeConfig(config engine.Config) engine.Config {
	if config.MaxCandidates <= 0 {
		config.MaxCandidates = engine.DefaultConfig().MaxCandidates
	}
	if config.CandidatePageSize <= 0 {
		config.CandidatePageSize = engine.DefaultConfig().CandidatePageSize
	}
	if config.CandidatePageSize < 3 {
		config.CandidatePageSize = 3
	}
	if config.CandidatePageSize > 9 {
		config.CandidatePageSize = 9
	}
	config.CandidateLayout = normalizeCandidateLayout(config.CandidateLayout)
	if config.Language == "" {
		config.Language = engine.DefaultConfig().Language
	}
	if config.Mode == "" {
		config.Mode = engine.DefaultConfig().Mode
	}
	config.Script = normalizeScript(config.Script, engine.DefaultConfig().Script)
	config.DoublePinyinScheme = normalizeDoublePinyinScheme(config.DoublePinyinScheme)
	switch strings.ToLower(strings.TrimSpace(config.Punctuation)) {
	case "half":
		config.Punctuation = "half"
	default:
		config.Punctuation = engine.DefaultConfig().Punctuation
	}
	config.Skin = engine.NormalizeSkin(config.Skin)
	if config.Update.SourcePreset == "" {
		config.Update.SourcePreset = engine.DefaultConfig().Update.SourcePreset
	}
	if config.Update.Channel == "" {
		config.Update.Channel = engine.DefaultConfig().Update.Channel
	}
	if len(config.Update.ManifestURLs) == 0 {
		config.Update.ManifestURLs = engine.DefaultConfig().Update.ManifestURLs
	}
	if config.Update.MirrorBaseURLs == nil {
		config.Update.MirrorBaseURLs = engine.DefaultConfig().Update.MirrorBaseURLs
	}
	if config.Update.InstalledVersion == "" {
		config.Update.InstalledVersion = engine.DefaultConfig().Update.InstalledVersion
	}
	config = engine.NormalizeSchemaConfig(config)
	config = engine.NormalizeKeyBehavior(config)
	config.RecognizerPatterns = engine.NormalizeRecognizerPatterns(config.RecognizerPatterns)
	config.AppRules = engine.NormalizeAppRules(config.AppRules)
	config.Agent = engine.NormalizeAgent(config.Agent)
	config.Sync = engine.NormalizeSync(config.Sync)
	return engine.NormalizeConfig(config)
}

func normalizeScript(script string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(script)) {
	case "", "simplified", "simp", "s", "zh-cn", "cn":
		if strings.TrimSpace(script) == "" && fallback != "" {
			return fallback
		}
		return "simplified"
	case "traditional", "trad", "t", "zh-tw", "zh-hk", "tw", "hk":
		return "traditional"
	default:
		if fallback == "" {
			return "simplified"
		}
		return fallback
	}
}

func normalizeCandidateLayout(layout string) string {
	switch strings.ToLower(strings.TrimSpace(layout)) {
	case "", "horizontal", "linear", "inline", "wechat", "microsoft":
		return "horizontal"
	case "vertical", "stacked", "rime":
		return "vertical"
	case "auto":
		return "auto"
	default:
		return "horizontal"
	}
}

func normalizeDoublePinyinScheme(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "", "xiaohe", "flypy":
		return "xiaohe"
	case "microsoft", "ms", "sogou":
		return "microsoft"
	default:
		return "xiaohe"
	}
}
