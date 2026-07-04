package main

import (
	"encoding/json"
	"os"
	"path/filepath"

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
	if config.MaxCandidates <= 0 {
		config.MaxCandidates = engine.DefaultConfig().MaxCandidates
	}
	if config.Language == "" {
		config.Language = engine.DefaultConfig().Language
	}
	if config.Mode == "" {
		config.Mode = engine.DefaultConfig().Mode
	}
	return config
}
