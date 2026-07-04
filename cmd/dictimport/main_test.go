package main

import (
	"strings"
	"testing"
)

func TestParseRimeDictionary(t *testing.T) {
	input := `# Rime dictionary
# encoding: utf-8
---
name: sample
version: "test"
...
你好	ni hao	1200
测试	ce shi
`
	entries, err := parseRimeDictionary(strings.NewReader(input), "rime-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want 2", entries)
	}
	if entries[0].Text != "你好" || entries[0].Reading != "nihao" || entries[0].Weight != 1200 {
		t.Fatalf("first entry = %#v", entries[0])
	}
	if entries[1].Text != "测试" || entries[1].Reading != "ceshi" || entries[1].Weight != 1000 {
		t.Fatalf("second entry = %#v", entries[1])
	}
}

func TestMergeEntriesKeepsHighestWeight(t *testing.T) {
	input := `---
...
你好	ni hao	1200
你好	nihao	1500
`
	entries, err := parseRimeDictionary(strings.NewReader(input), "rime-test")
	if err != nil {
		t.Fatal(err)
	}
	merged := mergeEntries(entries)
	if len(merged) != 1 {
		t.Fatalf("merged = %#v, want 1", merged)
	}
	if merged[0].Weight != 1500 {
		t.Fatalf("weight = %d, want 1500", merged[0].Weight)
	}
}
