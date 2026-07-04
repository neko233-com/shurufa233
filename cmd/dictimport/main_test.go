package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
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

func TestParseRimeDocumentCollectsImportTables(t *testing.T) {
	input := `---
name: rime_ice
version: "test"
import_tables:
  - cn_dicts/8105
  - "cn_dicts/base" # common words
...
入口	ru kou	900
`
	entries, imports, err := parseRimeDocument(strings.NewReader(input), "rime-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Text != "入口" {
		t.Fatalf("entries = %#v", entries)
	}
	want := []string{filepath.Join("cn_dicts", "8105"), filepath.Join("cn_dicts", "base")}
	if len(imports) != len(want) {
		t.Fatalf("imports = %#v, want %#v", imports, want)
	}
	for index := range want {
		if imports[index] != want[index] {
			t.Fatalf("imports = %#v, want %#v", imports, want)
		}
	}
}

func TestParseRimeDocumentCollectsInlineImportTables(t *testing.T) {
	input := `---
import_tables: [luna_pinyin, cn_dicts/ext]
...
`
	_, imports, err := parseRimeDocument(strings.NewReader(input), "rime-test")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"luna_pinyin", filepath.Join("cn_dicts", "ext")}
	if len(imports) != len(want) {
		t.Fatalf("imports = %#v, want %#v", imports, want)
	}
	for index := range want {
		if imports[index] != want[index] {
			t.Fatalf("imports = %#v, want %#v", imports, want)
		}
	}
}

func TestCollectorResolvesImportTables(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "rime_ice.dict.yaml"), `---
name: rime_ice
import_tables:
  - cn_dicts/8105
  - cn_dicts/base
...
入口	ru kou	900
`)
	writeFile(t, filepath.Join(root, "cn_dicts", "8105.dict.yaml"), `---
name: 8105
...
一	yi	100
`)
	writeFile(t, filepath.Join(root, "cn_dicts", "base.dict.yaml"), `---
name: base
...
你好	ni hao	1200
`)

	entries, err := newRimeCollector("rime-test", true, "error", nil).collect(filepath.Join(root, "rime_ice.dict.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, entry := range entries {
		got[entry.Text] = entry.Reading
	}
	for text, reading := range map[string]string{"一": "yi", "你好": "nihao", "入口": "rukou"} {
		if got[text] != reading {
			t.Fatalf("entries = %#v, missing %s/%s", entries, text, reading)
		}
	}
}

func TestCollectorMissingImportsCanWarn(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "main.dict.yaml"), `---
import_tables:
  - missing
...
入口	ru kou	900
`)

	var warnings strings.Builder
	entries, err := newRimeCollector("rime-test", true, "warn", &warnings).collect(filepath.Join(root, "main.dict.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Text != "入口" {
		t.Fatalf("entries = %#v", entries)
	}
	if !strings.Contains(warnings.String(), "missing") {
		t.Fatalf("warnings = %q", warnings.String())
	}
}

func TestCollectorMissingImportsErrorByDefault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "main.dict.yaml"), `---
import_tables:
  - missing
...
`)

	_, err := newRimeCollector("rime-test", true, "error", nil).collect(filepath.Join(root, "main.dict.yaml"))
	if err == nil {
		t.Fatal("expected missing import error")
	}
}

func TestResolveImportTableRejectsUnsafePaths(t *testing.T) {
	for _, table := range []string{"../outside", "/absolute", `C:drive-relative`, `C:\absolute`} {
		if _, _, err := resolveImportTable(t.TempDir(), table); err == nil {
			t.Fatalf("expected unsafe import error for %q", table)
		}
	}
}

func TestWriteDictionaryOutputCanGzip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dictionary.json.gz")
	if err := writeDictionaryOutput(path, []byte(`{"language":"zh-CN","entries":[]}`), true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(decoded), `"language":"zh-CN"`) {
		t.Fatalf("decoded gzip = %q", decoded)
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

func writeFile(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
