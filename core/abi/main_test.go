package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestNewEngineLoadsConfigForFuzzyInitials(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SHURUFA233_CONFIG", configPath)
	t.Setenv("SHURUFA233_DICTIONARY_DIR", t.TempDir())
	t.Setenv("SHURUFA233_USER_SCORES", filepath.Join(t.TempDir(), "user-scores.json"))

	config := engine.DefaultConfig()
	config.FuzzyInitials = nil
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	session := engine.New(loadConfig())
	state := session.Preview("zongwen")
	for _, candidate := range state.Candidates {
		if candidate.Text == "中文" {
			t.Fatalf("expected ABI core to honor disabled fuzzy initials, got %#v", state.Candidates)
		}
	}
}

func TestPersistUserScoresAsync(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user-scores.json")
	t.Setenv("SHURUFA233_USER_SCORES", path)

	persistUserScores(map[string]int{"nihao|你好": 25})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			var store userScoreStore
			if json.Unmarshal(data, &store) == nil && store.Scores["nihao|你好"] == 25 {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("expected async user score persistence at %s", path)
}

func TestPersistUserScoresSyncMergesExistingScores(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user-scores.json")
	t.Setenv("SHURUFA233_USER_SCORES", path)

	persistUserScoresSync(map[string]int{"nihao|你好": 3})
	persistUserScoresSync(map[string]int{"nihao|你好": 1, "xiaolian|笑脸": 2})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read user scores: %v", err)
	}
	var store userScoreStore
	if err := json.Unmarshal(data, &store); err != nil {
		t.Fatalf("parse user scores: %v", err)
	}
	if store.Scores["nihao|你好"] != 3 {
		t.Fatalf("expected higher existing score to win, got %#v", store.Scores)
	}
	if store.Scores["xiaolian|笑脸"] != 2 {
		t.Fatalf("expected new score to be merged, got %#v", store.Scores)
	}
}
