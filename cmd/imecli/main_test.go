package main

import "testing"

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
