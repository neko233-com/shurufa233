package main

import (
	"os"
	"path/filepath"
	"strings"

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
	gzFiles, err := filepath.Glob(filepath.Join(dir, "*.json.gz"))
	if err == nil {
		files = append(files, gzFiles...)
	}
	entries := make([][]engine.Entry, 0, len(files))
	for _, file := range files {
		name := filepath.Base(file)
		if isDictionaryMetadataFile(name) {
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
	dict, err := engine.DecodeDictionary(file)
	if err != nil {
		return nil, err
	}
	return dict.Entries, nil
}

func isDictionaryMetadataFile(name string) bool {
	name = strings.TrimSuffix(name, ".gz")
	return name == "manifest.json" || name == "dictionary-manifest.json"
}
