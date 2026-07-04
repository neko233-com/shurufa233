package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestNewEngineLoadsLocalGzipDictionaries(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHURUFA233_DICTIONARY_DIR", dir)
	t.Setenv("SHURUFA233_USER_SCORES", filepath.Join(t.TempDir(), "user-scores.json"))

	dictionary := `{
		"language": "zh-CN",
		"version": "gzip",
		"entries": [{ "reading": "yasuo", "text": "压缩", "weight": 30000 }]
	}`
	if err := os.WriteFile(filepath.Join(dir, "zh-CN.gzip.json.gz"), gzipBytes(t, []byte(dictionary)), 0o644); err != nil {
		t.Fatal(err)
	}

	session := engine.New(engine.DefaultConfig())
	for _, entries := range loadLocalDictionaryEntries() {
		session.AddEntries(entries)
	}
	state := session.Preview("yasuo")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "压缩" {
		t.Fatalf("expected local gzip dictionary candidate, got %#v", state.Candidates)
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

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestLoadConfigKeepsHalfWidthPunctuation(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SHURUFA233_CONFIG", configPath)
	config := engine.DefaultConfig()
	config.Punctuation = " HALF "
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got := loadConfig()
	if got.Punctuation != "half" {
		t.Fatalf("punctuation = %q, want half", got.Punctuation)
	}
}

func TestLoadConfigKeepsDoublePinyinScheme(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SHURUFA233_CONFIG", configPath)
	config := engine.DefaultConfig()
	config.DoublePinyin = true
	config.DoublePinyinScheme = "microsoft"
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got := loadConfig()
	if !got.DoublePinyin || got.DoublePinyinScheme != "microsoft" {
		t.Fatalf("double pinyin config = enabled:%v scheme:%q, want microsoft", got.DoublePinyin, got.DoublePinyinScheme)
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

func TestBuildCandidatePayloadV2IncludesMetadata(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	state := session.Preview("zan")
	if len(state.Candidates) == 0 {
		t.Fatal("expected zan candidates")
	}

	payload := buildCandidatePayloadV2(session, 0, 3)
	if !payload.OK {
		t.Fatal("payload should be ok")
	}
	if payload.Total == 0 || len(payload.Items) == 0 {
		t.Fatalf("expected candidate items, got %#v", payload)
	}
	if payload.Items[0].Text == "" || payload.Items[0].Reading == "" {
		t.Fatalf("expected text and reading metadata, got %#v", payload.Items[0])
	}
	if payload.Items[0].Score != payload.Items[0].Weight+payload.Items[0].UserScore {
		t.Fatalf("score does not include user score: %#v", payload.Items[0])
	}
}

func TestCapabilitiesIncludeCandidateCharCommit(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "candidate-char-commit" {
			return
		}
	}
	t.Fatalf("capabilities missing candidate-char-commit: %#v", abiFeatureList)
}

func TestDecodeUserScoresPayloadAcceptsWrappedAndRawScores(t *testing.T) {
	wrapped, err := decodeUserScoresPayload(`{"userScores":{"nihao|你好":25}}`)
	if err != nil {
		t.Fatal(err)
	}
	if wrapped["nihao|你好"] != 25 {
		t.Fatalf("wrapped score not decoded: %#v", wrapped)
	}

	raw, err := decodeUserScoresPayload(`{"xiexie|谢谢":10}`)
	if err != nil {
		t.Fatal(err)
	}
	if raw["xiexie|谢谢"] != 10 {
		t.Fatalf("raw score not decoded: %#v", raw)
	}
}

func TestComposeAgentABIUsesContextForRewrite(t *testing.T) {
	got := composeAgentABI("/rewrite", "这段话有点啰嗦")
	if !got.OK || len(got.Items) < 2 {
		t.Fatalf("expected rewrite items, got %#v", got)
	}
	if got.Items[0].Intent != "rewrite" || got.Items[0].Source != "builtin-agent" {
		t.Fatalf("unexpected agent metadata: %#v", got.Items[0])
	}
	if !strings.Contains(got.Items[0].Text, "这段话有点啰嗦") {
		t.Fatalf("expected context in rewrite prompt, got %#v", got.Items[0])
	}
}
