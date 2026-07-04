package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAgentArgsWithContext(t *testing.T) {
	input, context := parseAgentArgs([]string{"--context", "选中文本", "/rewrite"})
	if input != "/rewrite" {
		t.Fatalf("input = %q, want /rewrite", input)
	}
	if context != "选中文本" {
		t.Fatalf("context = %q, want 选中文本", context)
	}
}

func TestParseAgentArgsKeepsPromptWords(t *testing.T) {
	input, context := parseAgentArgs([]string{"/ask", "怎么优化", "-c", "当前代码"})
	if input != "/ask 怎么优化" {
		t.Fatalf("input = %q", input)
	}
	if context != "当前代码" {
		t.Fatalf("context = %q", context)
	}
}

func TestReadWordbookFileAcceptsWrappedScores(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wordbook.json")
	if err := os.WriteFile(path, []byte(`{"userScores":{"nihao|你好":25}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readWordbookFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["nihao|你好"] != 25 {
		t.Fatalf("scores = %#v", got)
	}
}

func TestReadWordbookFileAcceptsRawScores(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wordbook.json")
	if err := os.WriteFile(path, []byte(`{"ceshi|测试":50}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readWordbookFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["ceshi|测试"] != 50 {
		t.Fatalf("scores = %#v", got)
	}
}
