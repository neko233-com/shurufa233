package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neko233-com/shurufa233/core/engine"
)

func TestNewEngineLoadsLocalDictionaries(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHURUFA233_DICTIONARY_DIR", dir)
	t.Setenv("SHURUFA233_USER_SCORES", filepath.Join(t.TempDir(), "user-scores.json"))

	dictionary := `{
		"language": "zh-CN",
		"version": "test",
		"entries": [{ "reading": "rexiu", "text": "热修", "weight": 30000 }]
	}`
	if err := os.WriteFile(filepath.Join(dir, "zh-CN.test.json"), []byte(dictionary), 0o644); err != nil {
		t.Fatal(err)
	}

	session := engine.New(engine.DefaultConfig())
	for _, entries := range loadLocalDictionaryEntries() {
		session.AddEntries(entries)
	}
	state := session.Preview("rexiu")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "热修" {
		t.Fatalf("expected local dictionary candidate 热修, got %#v", state.Candidates)
	}
}
