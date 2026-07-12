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

func TestParseRimeDictionaryHonorsInlineColumns(t *testing.T) {
	input := `---
name: reordered
columns: [text, weight, code]
...
列序	3200	lie xu
`
	entries, err := parseRimeDictionary(strings.NewReader(input), "rime-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want 1", entries)
	}
	if entries[0].Text != "列序" || entries[0].Reading != "liexu" || entries[0].Weight != 3200 {
		t.Fatalf("entry = %#v", entries[0])
	}
}

func TestParseRimeDictionaryHonorsCommentColumn(t *testing.T) {
	input := `---
name: comments
columns: [text, code, weight, comment]
...
候选	hou xuan	3200	候选注释
`
	entries, err := parseRimeDictionary(strings.NewReader(input), "rime-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want 1", entries)
	}
	if entries[0].Comment != "候选注释" {
		t.Fatalf("entry comment = %#v", entries[0])
	}
}

func TestParseRimeDictionaryHonorsBlockColumns(t *testing.T) {
	input := `---
name: block_columns
columns:
  - code
  - text
  - weight
...
dao ru	导入	2800
`
	entries, err := parseRimeDictionary(strings.NewReader(input), "rime-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want 1", entries)
	}
	if entries[0].Text != "导入" || entries[0].Reading != "daoru" || entries[0].Weight != 2800 {
		t.Fatalf("entry = %#v", entries[0])
	}
}

func TestParseRimeDictionaryIgnoresStemColumn(t *testing.T) {
	input := `---
name: stem
columns: [text, code, weight, stem]
...
测试	ce shi	900	c-sh
`
	entries, err := parseRimeDictionary(strings.NewReader(input), "rime-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Text != "测试" || entries[0].Reading != "ceshi" || entries[0].Weight != 900 {
		t.Fatalf("entries = %#v", entries)
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

func TestParseRimeSymbolsYAML(t *testing.T) {
	input := `# Rime symbols.yaml
patch:
  punctuator/symbols/+:
    '/fs': [℃, ℉, °]
    '/xh': ['※', "★", ☆] # stars
    vdw: [℃℃, ℉℉]
    Vlm: [ⅰ, Ⅰ]
    '/yx': [🙂, 😂]
    '/yw': ['(╯°□°）╯︵ ┻━┻', "=_="]
`
	entries, imports, err := parseRimeDocument(strings.NewReader(input), "rime-symbols")
	if err != nil {
		t.Fatal(err)
	}
	if len(imports) != 0 {
		t.Fatalf("imports = %#v, want none", imports)
	}
	got := map[string][]engine.Entry{}
	for _, entry := range entries {
		got[entry.Reading] = append(got[entry.Reading], entry)
	}
	if len(got["fs"]) != 3 || got["fs"][0].Text != "℃" || got["fs"][0].Kind != "symbol" {
		t.Fatalf("fs symbols = %#v", got["fs"])
	}
	if len(got["xh"]) != 3 || got["xh"][1].Text != "★" {
		t.Fatalf("xh symbols = %#v", got["xh"])
	}
	if len(got["dw"]) != 2 || got["dw"][0].Text != "℃℃" {
		t.Fatalf("dw v-mode symbols = %#v", got["dw"])
	}
	if len(got["lm"]) != 2 || got["lm"][0].Text != "ⅰ" {
		t.Fatalf("lm caps-v-mode symbols = %#v", got["lm"])
	}
	if len(got["yx"]) != 2 || got["yx"][0].Kind != "emoji" {
		t.Fatalf("yx emoji = %#v", got["yx"])
	}
	if len(got["yw"]) != 2 || got["yw"][0].Kind != "kaomoji" {
		t.Fatalf("yw kaomoji = %#v", got["yw"])
	}
}

func TestParseRimeSymbolsBlockListYAML(t *testing.T) {
	input := `patch:
  punctuator/symbols/+:
    '/dw':
      - ℃
      - "℉"
      - '°' # degree
    '/bq':
      - 🙂
      - '(T_T)'
    vxh:
      - ※
      - ★
`
	entries, imports, err := parseRimeDocument(strings.NewReader(input), "rime-symbols")
	if err != nil {
		t.Fatal(err)
	}
	if len(imports) != 0 {
		t.Fatalf("imports = %#v, want none", imports)
	}
	got := map[string][]engine.Entry{}
	for _, entry := range entries {
		got[entry.Reading] = append(got[entry.Reading], entry)
	}
	if len(got["dw"]) != 3 || got["dw"][0].Text != "℃" || got["dw"][1].Text != "℉" || got["dw"][2].Text != "°" {
		t.Fatalf("dw block symbols = %#v", got["dw"])
	}
	if got["dw"][0].Weight != rimeSymbolWeightBase || got["dw"][2].Weight != rimeSymbolWeightBase-2 {
		t.Fatalf("dw block weights = %#v", got["dw"])
	}
	if len(got["bq"]) != 2 || got["bq"][0].Kind != "emoji" || got["bq"][1].Kind != "kaomoji" {
		t.Fatalf("bq block kinds = %#v", got["bq"])
	}
	if len(got["xh"]) != 2 || got["xh"][0].Text != "※" || got["xh"][1].Text != "★" {
		t.Fatalf("xh v-mode block symbols = %#v", got["xh"])
	}
}

func TestParseRimeNestedSymbolsVModeYAML(t *testing.T) {
	input := `patch:
  punctuator:
    symbols:
      vdw: [℃, ℉, °]
`
	entries, imports, err := parseRimeDocument(strings.NewReader(input), "rime-symbols")
	if err != nil {
		t.Fatal(err)
	}
	if len(imports) != 0 {
		t.Fatalf("imports = %#v, want none", imports)
	}
	got := map[string][]engine.Entry{}
	for _, entry := range entries {
		got[entry.Reading] = append(got[entry.Reading], entry)
	}
	if len(got["dw"]) != 3 || got["dw"][0].Text != "℃" {
		t.Fatalf("nested v-mode symbols = %#v", got["dw"])
	}
}

func TestRimeSymbolsImportIntoEngine(t *testing.T) {
	input := `patch:
  punctuator/symbols/+:
    '/fs': [℃, ℉, °]
    vxh: [※, ★, ☆]
`
	entries, err := parseRimeDictionary(strings.NewReader(input), "rime-symbols")
	if err != nil {
		t.Fatal(err)
	}
	e := engine.New(engine.DefaultConfig())
	e.AddEntries(entries)
	state := e.Preview("fs")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "℃" || state.Candidates[0].Kind != "symbol" {
		t.Fatalf("expected imported Rime symbol candidate, got %#v", state.Candidates)
	}
	state = e.Preview("vxh")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "※" || state.Candidates[0].Kind != "symbol" {
		t.Fatalf("expected imported Rime v-mode symbol candidate, got %#v", state.Candidates)
	}
}

func TestParseOpenCCEmojiTextWithASCIIKey(t *testing.T) {
	input := "ID\tID 🆔️ 🪪\nWIFI\tWIFI 🛜\n"
	entries, imports, err := parseRimeDocument(strings.NewReader(input), "rime-opencc")
	if err != nil {
		t.Fatal(err)
	}
	if len(imports) != 0 {
		t.Fatalf("imports = %#v, want none", imports)
	}
	got := map[string][]engine.Entry{}
	for _, entry := range entries {
		got[entry.Reading] = append(got[entry.Reading], entry)
	}
	if len(got["id"]) != 2 || got["id"][0].Text != "🆔️" || got["id"][1].Text != "🪪" {
		t.Fatalf("id opencc entries = %#v", got["id"])
	}
	if got["id"][0].Kind != "emoji" || got["id"][0].Weight != openCCWeightBase-1 {
		t.Fatalf("id opencc metadata = %#v", got["id"][0])
	}
	if len(got["wifi"]) != 1 || got["wifi"][0].Text != "🛜" {
		t.Fatalf("wifi opencc entries = %#v", got["wifi"])
	}
}

func TestParseOpenCCEmojiTextUsesReadingHints(t *testing.T) {
	input := "微笑\t微笑 😊 [微笑]\n"
	entries, imports, err := parseRimeDocumentWithHints(strings.NewReader(input), "rime-opencc", rimeParseHints{
		ReadingsByText: map[string][]string{
			"微笑": {"weixiao"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(imports) != 0 {
		t.Fatalf("imports = %#v, want none", imports)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want 2", entries)
	}
	if entries[0].Reading != "weixiao" || entries[0].Text != "😊" || entries[0].Kind != "emoji" {
		t.Fatalf("first hinted opencc entry = %#v", entries[0])
	}
	if entries[1].Reading != "weixiao" || entries[1].Text != "[微笑]" || entries[1].Kind != "kaomoji" {
		t.Fatalf("second hinted opencc entry = %#v", entries[1])
	}
}

func TestCollectorOpenCCRowsUsePreviouslyImportedRimeDictionary(t *testing.T) {
	root := t.TempDir()
	dictPath := filepath.Join(root, "base.dict.yaml")
	openCCPath := filepath.Join(root, "emoji.txt")
	writeFile(t, dictPath, `---
name: base
...
微笑	wei xiao	900
`)
	writeFile(t, openCCPath, "微笑\t微笑 😊\n")

	collector := newRimeCollector("rime-opencc", true, "error", nil)
	if _, err := collector.collect(dictPath); err != nil {
		t.Fatal(err)
	}
	entries, err := collector.collect(openCCPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Reading != "weixiao" || entries[0].Text != "😊" {
		t.Fatalf("opencc entries = %#v", entries)
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

func TestCollectorInfersWeightOnlyDictionaryReadingsFromImportedCharacters(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "rime_ice.dict.yaml"), `---
name: rime_ice
import_tables:
  - cn_dicts/8105
  - cn_dicts/tencent
...
`)
	writeFile(t, filepath.Join(root, "cn_dicts", "8105.dict.yaml"), `---
name: 8105
...
你	ni	100
好	hao	100
长	chang	90
长	zhang	10
大	da	100
〇	ling	100
八	ba	100
`)
	writeFile(t, filepath.Join(root, "cn_dicts", "tencent.dict.yaml"), `---
name: tencent
columns:
  - text
  - weight
...
你好	800
长大	700
〇八	600
缺字	500
`)

	entries, err := newRimeCollector("rime-test", true, "error", nil).collect(filepath.Join(root, "rime_ice.dict.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]engine.Entry{}
	for _, entry := range entries {
		got[entry.Text] = entry
	}
	for text, reading := range map[string]string{
		"你好": "nihao",
		"长大": "changda",
		"〇八": "lingba",
	} {
		entry, ok := got[text]
		if !ok || entry.Reading != reading {
			t.Fatalf("entry for %s = %#v, want reading %s; all entries=%#v", text, entry, reading, entries)
		}
	}
	if _, ok := got["缺字"]; ok {
		t.Fatalf("uninferrable weight-only entry was imported: %#v", got["缺字"])
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

func TestCompactEntriesKeepsHigherFrequencyDailyPhraseGroups(t *testing.T) {
	entries := []engine.Entry{
		{Reading: "gaopin", Text: "高频", Weight: 2000},
		{Reading: "nihaoma", Text: "你好吗", Weight: 25000},
	}

	got := compactEntries(entries, 1)
	if len(got) != 1 || got[0].Reading != "nihaoma" {
		t.Fatalf("compact entries = %#v, want daily phrase group", got)
	}
}

func TestCompactEntriesSpendsBudgetOnReadingCoverage(t *testing.T) {
	entries := []engine.Entry{
		{Reading: "gaopin", Text: "高频", Weight: 900000},
		{Reading: "gaopin", Text: "高品", Weight: 800000},
		{Reading: "gaopin", Text: "高聘", Weight: 700000},
		{Reading: "wanan", Text: "晚安", Weight: 1000},
		{Reading: "shaodeng", Text: "稍等", Weight: 900},
		{Reading: "heshui", Text: "喝水", Weight: 800},
	}

	got := compactEntries(entries, 4)
	readings := map[string]bool{}
	for _, entry := range got {
		readings[entry.Reading] = true
	}
	if len(readings) < 3 || !readings["wanan"] || !readings["shaodeng"] {
		t.Fatalf("compact coverage = %#v, want at least three reading groups including daily phrases", got)
	}
}

func TestCompactEntriesDropsOrdinaryLatinEntriesFromChineseRuntimeLexicon(t *testing.T) {
	entries := []engine.Entry{
		{Reading: "hello", Text: "hello", Weight: 999999},
		{Reading: "wanan", Text: "晚安", Weight: 100},
		{Reading: "bq", Text: "😊", Kind: "emoji", Weight: 1},
	}
	got := compactEntries(entries, 2)
	if len(got) != 2 || got[0].Text == "hello" || got[1].Text == "hello" {
		t.Fatalf("compactEntries should reserve the Chinese runtime budget, got %#v", got)
	}
}

func TestCompactEntriesProtectsPreferredDailyReadings(t *testing.T) {
	entries := []engine.Entry{
		{Reading: "wanghongci", Text: "网红词", Weight: 900000},
		{Reading: "wanan", Text: "晚安", Weight: 10},
	}
	got := compactEntriesWithPreferredReadings(entries, 1, map[string]bool{"wanan": true})
	if len(got) != 1 || got[0].Text != "晚安" {
		t.Fatalf("preferred daily reading was compacted out: %#v", got)
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
