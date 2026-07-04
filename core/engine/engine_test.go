package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidatesAndLearning(t *testing.T) {
	e := New(DefaultConfig())
	for _, r := range "nihao" {
		e.InputKey(r)
	}
	state := e.State()
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "你好" {
		t.Fatalf("expected top candidate 你好, got %#v", state.Candidates)
	}

	committed, err := e.Select(0)
	if err != nil {
		t.Fatal(err)
	}
	if committed.Committed != "你好" {
		t.Fatalf("expected committed text, got %q", committed.Committed)
	}
	if committed.Buffer != "" {
		t.Fatalf("expected buffer to clear, got %q", committed.Buffer)
	}
}

func TestGreedySegmentation(t *testing.T) {
	e := New(DefaultConfig())
	state := e.Preview("womende")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "我们的" {
		t.Fatalf("expected segmented candidate 我们的, got %#v", state.Candidates)
	}
}

func TestEnglishMode(t *testing.T) {
	config := DefaultConfig()
	config.Mode = "en"
	e := New(config)
	state := e.Preview("hello")
	if len(state.Candidates) != 1 || state.Candidates[0].Text != "hello" {
		t.Fatalf("expected passthrough English candidate, got %#v", state.Candidates)
	}
}

func TestLoadDictionary(t *testing.T) {
	e := New(DefaultConfig())
	_, err := e.LoadDictionary(strings.NewReader(`{
		"language": "zh-CN",
		"version": "test",
		"entries": [{ "reading": "maomao", "text": "猫猫", "weight": 20000 }]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	state := e.Preview("maomao")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "猫猫" {
		t.Fatalf("expected hot dictionary candidate, got %#v", state.Candidates)
	}
}

func TestLoadDictionaryAcceptsUTF8BOM(t *testing.T) {
	e := New(DefaultConfig())
	_, err := e.LoadDictionary(strings.NewReader("\ufeff" + `{
		"language": "zh-CN",
		"version": "bom",
		"entries": [{ "reading": "bom", "text": "字节序", "weight": 9000 }]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	state := e.Preview("bom")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "字节序" {
		t.Fatalf("expected BOM dictionary candidate, got %#v", state.Candidates)
	}
}

func TestLoadDictionaryMergesDuplicates(t *testing.T) {
	e := New(DefaultConfig())
	_, err := e.LoadDictionary(strings.NewReader(`{
		"language": "zh-CN",
		"version": "test",
		"entries": [{ "reading": "nihao", "text": "你好", "weight": 20000 }]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	state := e.Preview("nihao")
	count := 0
	for _, candidate := range state.Candidates {
		if candidate.Text == "你好" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one merged 你好 candidate, got %#v", state.Candidates)
	}
	if state.Candidates[0].Weight != 20000 {
		t.Fatalf("expected merged candidate to keep highest weight, got %d", state.Candidates[0].Weight)
	}
}

func TestImportUserScoresAffectsRanking(t *testing.T) {
	e := New(DefaultConfig())
	_, err := e.LoadDictionary(strings.NewReader(`{
		"language": "zh-CN",
		"version": "test",
		"entries": [
			{ "reading": "ceshi", "text": "测试", "weight": 100 },
			{ "reading": "ceshi", "text": "侧室", "weight": 90 }
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	e.ImportUserScores(map[string]int{"ceshi|侧室": 1000})
	state := e.Preview("ceshi")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "侧室" {
		t.Fatalf("expected imported user score to rerank candidates, got %#v", state.Candidates)
	}
}

func TestBuiltinEmojiCandidateMetadata(t *testing.T) {
	e := New(DefaultConfig())

	tests := []struct {
		reading string
		text    string
		kind    string
	}{
		{reading: "kaixin", text: "ヽ(・∀・)ﾉ", kind: "kaomoji"},
		{reading: "zan", text: "👍", kind: "emoji"},
		{reading: "wuyu", text: "=_=", kind: "kaomoji"},
		{reading: "shengluehao", text: "……", kind: "symbol"},
	}

	for _, tt := range tests {
		state := e.Preview(tt.reading)
		found := false
		for _, candidate := range state.Candidates {
			if candidate.Text == tt.text {
				found = true
				if candidate.Kind != tt.kind || candidate.Source != "builtin-symbols" {
					t.Fatalf("expected %s metadata for %q, got %#v", tt.kind, tt.text, candidate)
				}
			}
		}
		if !found {
			t.Fatalf("expected builtin %s candidate %q for %s, got %#v", tt.kind, tt.text, tt.reading, state.Candidates)
		}
	}
}

func TestBundledZhDictionaryHasPagingCandidates(t *testing.T) {
	e := New(DefaultConfig())
	file, err := os.Open(filepath.Join("..", "..", "data", "dictionaries", "zh-CN.2026.07.04.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := e.LoadDictionary(file); err != nil {
		t.Fatal(err)
	}
	state := e.Preview("shi")
	if len(state.Candidates) < 8 {
		t.Fatalf("expected bundled shi candidates to exercise paging, got %#v", state.Candidates)
	}
	if state.Candidates[0].Text != "是" {
		t.Fatalf("expected top bundled shi candidate 是, got %#v", state.Candidates)
	}
}
