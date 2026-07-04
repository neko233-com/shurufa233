package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/neko233-com/shurufa233/core/engine"
)

func dictionaryDir() (string, error) {
	if override := os.Getenv("SHURUFA233_DICTIONARY_DIR"); override != "" {
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
	return filepath.Join(base, "shurufa233", "dictionaries"), nil
}

func loadLocalDictionaryEntries() [][]engine.Entry {
	dir, err := dictionaryDir()
	if err != nil {
		return nil
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil
	}
	entries := make([][]engine.Entry, 0, len(files))
	for _, file := range files {
		name := filepath.Base(file)
		if name == "manifest.json" || name == "dictionary-manifest.json" {
			continue
		}
		loaded, err := loadDictionaryFile(file)
		if err != nil || len(loaded) == 0 {
			continue
		}
		entries = append(entries, loaded)
	}
	return entries
}

func loadDictionaryFile(path string) ([]engine.Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	var dict engine.DictionaryFile
	if err := json.Unmarshal(data, &dict); err != nil {
		return nil, err
	}
	return dict.Entries, nil
}
