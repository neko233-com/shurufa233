package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/shurufa233/core/engine"
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

func TestParseRimeDictionaryNormalizesToneMarksAndUmlaut(t *testing.T) {
	input := `---
name: tone
...
女	nǚ	1200
绿	lǜ	1100
略	lüe	1000
旅	lu:	900
测试	cè shì	800
`
	entries, err := parseRimeDictionary(strings.NewReader(input), "rime-test")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, entry := range entries {
		got[entry.Text] = entry.Reading
	}
	want := map[string]string{
		"女":  "nv",
		"绿":  "lv",
		"略":  "lve",
		"旅":  "lv",
		"测试": "ceshi",
	}
	for text, reading := range want {
		if got[text] != reading {
			t.Fatalf("reading for %s = %q, want %q; entries=%#v", text, got[text], reading, entries)
		}
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

func TestParseRimeCustomPhraseText(t *testing.T) {
	input := `# Rime custom_phrase.txt
#@/db_name custom_phrase.txt
# 以 Tab 分割：词汇<TAB>编码<TAB>权重
马上到！	msd	99
邮箱	youxiang
`
	entries, imports, err := parseRimeDocument(strings.NewReader(input), "custom-phrase")
	if err != nil {
		t.Fatal(err)
	}
	if len(imports) != 0 {
		t.Fatalf("imports = %#v, want none", imports)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want 2", entries)
	}
	if entries[0].Text != "马上到！" || entries[0].Reading != "msd" || entries[0].Weight != customPhraseWeightBase+99 {
		t.Fatalf("first custom phrase = %#v", entries[0])
	}
	if entries[1].Text != "邮箱" || entries[1].Reading != "youxiang" || entries[1].Weight != customPhraseWeightBase+1000 {
		t.Fatalf("second custom phrase = %#v", entries[1])
	}
}

func TestParseRimeCustomPhraseWhitespaceFallback(t *testing.T) {
	input := `What the fuck! wtf 3
http://rime.im/ rime 1
Rime rime
`
	entries, err := parseRimeDictionary(strings.NewReader(input), "custom-phrase")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %#v, want 3", entries)
	}
	if entries[0].Text != "What the fuck!" || entries[0].Reading != "wtf" || entries[0].Weight != customPhraseWeightBase+3 {
		t.Fatalf("whitespace custom phrase = %#v", entries[0])
	}
	if entries[1].Text != "http://rime.im/" || entries[1].Reading != "rime" || entries[1].Weight != customPhraseWeightBase+1 {
		t.Fatalf("url custom phrase = %#v", entries[1])
	}
	if entries[2].Text != "Rime" || entries[2].Reading != "rime" || entries[2].Weight != customPhraseWeightBase+1000 {
		t.Fatalf("weightless custom phrase = %#v", entries[2])
	}
}

func TestCustomPhraseImportRanksAboveOrdinaryDictionary(t *testing.T) {
	input := `马上到！	msd	1`
	entries, err := parseRimeDictionary(strings.NewReader(input), "custom-phrase")
	if err != nil {
		t.Fatal(err)
	}
	e := engine.New(engine.DefaultConfig())
	e.AddEntries([]engine.Entry{{Reading: "msd", Text: "普通短语", Weight: 20000}})
	e.AddEntries(entries)
	state := e.Preview("msd")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "马上到！" {
		t.Fatalf("expected custom phrase to rank first, got %#v", state.Candidates)
	}
}

func TestParseRimeYAMLHeaderIsNotImportedAsCustomPhrase(t *testing.T) {
	input := `---
name: rime_ice
version: "test"
...
入口	ru kou	900
`
	entries, err := parseRimeDictionary(strings.NewReader(input), "rime-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Text != "入口" {
		t.Fatalf("entries = %#v, want only body row", entries)
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
