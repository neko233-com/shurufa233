package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
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

func TestLoadConfigKeepsKeyBehaviorProfile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SHURUFA233_CONFIG", configPath)
	config := engine.DefaultConfig()
	config.KeyProfile = "rime"
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got := loadConfig()
	if got.KeyProfile != "rime" || !got.CommaPeriodPageKeys || got.SemicolonQuickSelect {
		t.Fatalf("key behavior config = %#v", got)
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
	if payload.Items[0].Comment == "" {
		t.Fatalf("expected candidate comment metadata, got %#v", payload.Items[0])
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

func TestCapabilitiesIncludeCandidateComments(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "candidate-comments" {
			return
		}
	}
	t.Fatalf("capabilities missing candidate-comments: %#v", abiFeatureList)
}

func TestCapabilitiesIncludePinyinSeparators(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "pinyin-separators" {
			return
		}
	}
	t.Fatalf("capabilities missing pinyin-separators: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeRimeSymbolPrefix(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "rime-symbol-prefix" {
			return
		}
	}
	t.Fatalf("capabilities missing rime-symbol-prefix: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeUserPhrasesJSON(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "user-phrases-json" {
			return
		}
	}
	t.Fatalf("capabilities missing user-phrases-json: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeRimeCustomPhraseText(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "rime-custom-phrase-text" {
			return
		}
	}
	t.Fatalf("capabilities missing rime-custom-phrase-text: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeUserRejectsJSON(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "user-rejects-json" {
			return
		}
	}
	t.Fatalf("capabilities missing user-rejects-json: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeUserPinsJSON(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "user-pins-json" {
			return
		}
	}
	t.Fatalf("capabilities missing user-pins-json: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeCandidateActionJSON(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "candidate-action-json" {
			return
		}
	}
	t.Fatalf("capabilities missing candidate-action-json: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeExtensionCommandJSON(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "extension-command-json" {
			return
		}
	}
	t.Fatalf("capabilities missing extension-command-json: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeCatalogJSON(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "catalog-json" {
			return
		}
	}
	t.Fatalf("capabilities missing catalog-json: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeReverseLookupJSON(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "reverse-lookup-json" {
			return
		}
	}
	t.Fatalf("capabilities missing reverse-lookup-json: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeAssociationCandidates(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "association-candidates" {
			return
		}
	}
	t.Fatalf("capabilities missing association-candidates: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeDictionarySourcePresets(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "dictionary-source-presets" {
			return
		}
	}
	t.Fatalf("capabilities missing dictionary-source-presets: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeKeyBehaviorConfig(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "key-behavior-config" {
			return
		}
	}
	t.Fatalf("capabilities missing key-behavior-config: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeRimeRecognizerPatterns(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "rime-recognizer-patterns-json" {
			return
		}
	}
	t.Fatalf("capabilities missing rime-recognizer-patterns-json: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeRecognizerDecision(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "recognizer-decision-json" {
			return
		}
	}
	t.Fatalf("capabilities missing recognizer-decision-json: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeKeyEventJSON(t *testing.T) {
	for _, feature := range abiFeatureList {
		if feature == "key-event-json" {
			return
		}
	}
	t.Fatalf("capabilities missing key-event-json: %#v", abiFeatureList)
}

func TestCapabilitiesIncludeApplyAppRulesAndUserDataDelete(t *testing.T) {
	want := map[string]bool{
		"agent-config-json":       false,
		"apply-agent-config-json": false,
		"skin-presets-json":       false,
		"apply-skin-preset-json":  false,
		"profile-sync-json":       false,
		"apply-sync-config-json":  false,
		"apply-app-rules-json":    false,
		"user-data-delete-json":   false,
	}
	for _, feature := range abiFeatureList {
		if _, ok := want[feature]; ok {
			want[feature] = true
		}
	}
	for feature, found := range want {
		if !found {
			t.Fatalf("capabilities missing %s: %#v", feature, abiFeatureList)
		}
	}
}

func TestPreviewPreservesPinyinSeparatorCandidate(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	state := session.Preview("xi'an")
	if state.Buffer != "xi'an" {
		t.Fatalf("buffer = %q, want xi'an", state.Buffer)
	}
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "西安" {
		t.Fatalf("expected separator candidate 西安, got %#v", state.Candidates)
	}
}

func TestPreviewPreservesRimeSymbolPrefixCandidate(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	state := session.Preview("/fs")
	if state.Buffer != "/fs" {
		t.Fatalf("buffer = %q, want /fs", state.Buffer)
	}
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "℃" {
		t.Fatalf("expected symbol candidate ℃, got %#v", state.Candidates)
	}
}

func TestExecuteExtensionCommandImportsUserPhrases(t *testing.T) {
	t.Setenv("SHURUFA233_USER_PHRASES", filepath.Join(t.TempDir(), "user-phrases.json"))
	session := engine.New(engine.DefaultConfig())
	session.AddEntries([]engine.Entry{{Reading: "msd", Text: "默认短语", Weight: 20000}})

	got, handled := executeSessionExtensionCommand(session, "import-user-phrases", `{"entries":[{"reading":"msd","text":"马上到！"}]}`)
	if !handled {
		t.Fatal("import-user-phrases command was not handled")
	}
	result, ok := got.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("import-user-phrases = %#v", got)
	}
	state := session.Preview("msd")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "马上到！" {
		t.Fatalf("expected user phrase to rank first, got %#v", state.Candidates)
	}

	list, handled := executeSessionExtensionCommand(session, "user-phrases", `{}`)
	if !handled {
		t.Fatal("user-phrases command was not handled")
	}
	listResult, ok := list.(map[string]any)
	if !ok || listResult["count"] != 1 {
		t.Fatalf("user-phrases = %#v", list)
	}
}

func TestExecuteExtensionCommandImportsRimeUserDB(t *testing.T) {
	t.Setenv("SHURUFA233_USER_SCORES", filepath.Join(t.TempDir(), "user-scores.json"))
	session := engine.New(engine.DefaultConfig())
	session.AddEntries([]engine.Entry{
		{Reading: "chajian", Text: "插件", Weight: 100},
		{Reading: "chajian", Text: "差件", Weight: 90},
	})

	got, handled := executeSessionExtensionCommand(session, "import-user-scores", `{"format":"rime-userdb","data":"# Rime user dictionary\ncha jian 插件 c=20 d=1 t=3\n"}`)
	if !handled {
		t.Fatal("import-user-scores command was not handled")
	}
	result, ok := got.(map[string]any)
	if !ok || result["ok"] != true || result["imported"] != 1 {
		t.Fatalf("import Rime userdb = %#v", got)
	}
	if state := session.Preview("chajian"); len(state.Candidates) == 0 || state.Candidates[0].Text != "插件" {
		t.Fatalf("expected Rime userdb score to rerank candidates, got %#v", state.Candidates)
	}
}

func TestExecuteExtensionCommandImportsRimeCustomPhrases(t *testing.T) {
	t.Setenv("SHURUFA233_USER_PHRASES", filepath.Join(t.TempDir(), "user-phrases.json"))
	session := engine.New(engine.DefaultConfig())

	got, handled := executeSessionExtensionCommand(session, "import-rime-custom-phrases", `{"data":"马上到！\tmsd\t1\n开会\tkh\t20\n","merge":true}`)
	if !handled {
		t.Fatal("import-rime-custom-phrases command was not handled")
	}
	result, ok := got.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("import-rime-custom-phrases = %#v", got)
	}
	state := session.Preview("msd")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "马上到！" {
		t.Fatalf("expected Rime custom phrase to rank first, got %#v", state.Candidates)
	}

	exported, handled := executeSessionExtensionCommand(session, "user-phrases-rime-text", `{}`)
	if !handled {
		t.Fatal("user-phrases-rime-text command was not handled")
	}
	exportResult, ok := exported.(map[string]any)
	data, dataOK := exportResult["data"].(string)
	if !ok || !dataOK || !strings.Contains(data, "马上到！\tmsd\t1") {
		t.Fatalf("user-phrases-rime-text = %#v", exported)
	}
}

func TestExecuteExtensionCommandRejectsCandidate(t *testing.T) {
	t.Setenv("SHURUFA233_USER_REJECTS", filepath.Join(t.TempDir(), "user-rejects.json"))
	t.Setenv("SHURUFA233_USER_SCORES", filepath.Join(t.TempDir(), "user-scores.json"))
	session := engine.New(engine.DefaultConfig())
	session.AddEntries([]engine.Entry{
		{Reading: "ceshi", Text: "错词", Weight: 20000},
		{Reading: "ceshi", Text: "测试", Weight: 10000},
	})
	session.Preview("ceshi")

	got, handled := executeSessionExtensionCommand(session, "candidate-action", `{"action":"forget","index":0}`)
	if !handled {
		t.Fatal("candidate-action forget was not handled")
	}
	result, ok := got.(candidateActionResult)
	if !ok || !result.OK || result.Rejected == nil || result.Rejected.Text != "错词" {
		t.Fatalf("candidate-action forget = %#v", got)
	}
	if state := session.Preview("ceshi"); len(state.Candidates) == 0 || state.Candidates[0].Text == "错词" {
		t.Fatalf("expected rejected candidate to be hidden, got %#v", state.Candidates)
	}

	list, handled := executeSessionExtensionCommand(session, "user-rejects", `{}`)
	if !handled {
		t.Fatal("user-rejects command was not handled")
	}
	listResult, ok := list.(map[string]any)
	if !ok || listResult["count"] != 1 {
		t.Fatalf("user-rejects = %#v", list)
	}
}

func TestExecuteExtensionCommandPinsCandidate(t *testing.T) {
	t.Setenv("SHURUFA233_USER_PINS", filepath.Join(t.TempDir(), "user-pins.json"))
	session := engine.New(engine.DefaultConfig())
	session.AddEntries([]engine.Entry{
		{Reading: "ceshi", Text: "测试", Weight: 30000},
		{Reading: "ceshi", Text: "侧室", Weight: 1000},
	})
	session.Preview("ceshi")

	got, handled := executeSessionExtensionCommand(session, "candidate-action", `{"action":"pin","index":1}`)
	if !handled {
		t.Fatal("candidate-action pin was not handled")
	}
	result, ok := got.(candidateActionResult)
	if !ok || !result.OK || result.Pinned == nil || result.Pinned.Text != "侧室" {
		t.Fatalf("candidate-action pin = %#v", got)
	}
	state := session.Preview("ceshi")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "侧室" || !state.Candidates[0].Pinned {
		t.Fatalf("expected pinned candidate first, got %#v", state.Candidates)
	}

	list, handled := executeSessionExtensionCommand(session, "user-pins", `{}`)
	if !handled {
		t.Fatal("user-pins command was not handled")
	}
	listResult, ok := list.(map[string]any)
	if !ok || listResult["count"] != 1 {
		t.Fatalf("user-pins = %#v", list)
	}
}

func TestExecuteExtensionCommandCandidateActionPagingAndSelect(t *testing.T) {
	t.Setenv("SHURUFA233_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("SHURUFA233_DICTIONARY_DIR", t.TempDir())
	t.Setenv("SHURUFA233_USER_SCORES", filepath.Join(t.TempDir(), "user-scores.json"))

	session := engine.New(engine.DefaultConfig())
	session.AddEntries([]engine.Entry{
		{Reading: "ceshi", Text: "测试一", Weight: 10010},
		{Reading: "ceshi", Text: "测试二", Weight: 10009},
		{Reading: "ceshi", Text: "测试三", Weight: 10008},
		{Reading: "ceshi", Text: "测试四", Weight: 10007},
		{Reading: "ceshi", Text: "测试五", Weight: 10006},
	})
	session.Preview("ceshi")

	view, handled := executeSessionExtensionCommand(session, "candidate-action", `{"action":"view","start":0,"limit":2}`)
	if !handled {
		t.Fatal("candidate-action command was not handled")
	}
	firstPage, ok := view.(candidateActionResult)
	if !ok || !firstPage.OK || firstPage.Start != 0 || firstPage.Limit != 2 || len(firstPage.Candidates.Items) != 2 {
		t.Fatalf("candidate-action view = %#v", view)
	}

	next, handled := executeSessionExtensionCommand(session, "candidate-action", `{"action":"next-page","start":0,"limit":2}`)
	if !handled {
		t.Fatal("candidate-action next-page was not handled")
	}
	nextPage, ok := next.(candidateActionResult)
	if !ok || nextPage.Start != 2 || len(nextPage.Candidates.Items) != 2 {
		t.Fatalf("candidate-action next-page = %#v", next)
	}

	selected, handled := executeSessionExtensionCommand(session, "candidate-action", `{"action":"select","start":2,"displayIndex":2,"limit":2}`)
	if !handled {
		t.Fatal("candidate-action select was not handled")
	}
	commit, ok := selected.(candidateActionResult)
	if !ok || !commit.OK || commit.Committed != "测试四" || commit.State.Buffer != "" {
		t.Fatalf("candidate-action select = %#v", selected)
	}
}

func TestExecuteExtensionCommandCatalog(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	session.AddEntries([]engine.Entry{{Reading: "fs", Text: "℃℃", Kind: "symbol", Source: "rime-symbols", Weight: 9000}})

	got, handled := executeSessionExtensionCommand(session, "catalog-json", `{"kind":"symbol","query":"/fs","limit":5}`)
	if !handled {
		t.Fatal("catalog-json command was not handled")
	}
	result, ok := got.(engine.CatalogResponse)
	if !ok || result.Kind != "symbol" || result.Count == 0 || result.Entries[0].Reading != "fs" {
		t.Fatalf("catalog-json command = %#v", got)
	}
}

func TestExecuteExtensionCommandReverseLookup(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	session.AddEntries([]engine.Entry{{Reading: "shurufa", Text: "输入法", Kind: "phrase", Source: "test", Weight: 20000}})

	got, handled := executeSessionExtensionCommand(session, "reverse-lookup-json", `{"query":"输入法","limit":5}`)
	if !handled {
		t.Fatal("reverse-lookup-json command was not handled")
	}
	result, ok := got.(engine.ReverseLookupResponse)
	if !ok || len(result.Entries) == 0 || result.Entries[0].Reading != "shurufa" {
		t.Fatalf("reverse-lookup-json command = %#v", got)
	}
}

func TestExecuteExtensionCommandAssociate(t *testing.T) {
	session := engine.New(engine.DefaultConfig())

	got, handled := executeSessionExtensionCommand(session, "associate", `{"context":"你好","limit":2}`)
	if !handled {
		t.Fatal("associate command was not handled")
	}
	state, ok := got.(engine.State)
	if !ok || len(state.Candidates) == 0 || state.Candidates[0].Text != "世界" {
		t.Fatalf("associate command = %#v", got)
	}
}

func TestExecuteExtensionCommandDictionarySources(t *testing.T) {
	session := engine.New(engine.DefaultConfig())

	got, handled := executeSessionExtensionCommand(session, "dictionary-sources-json", `{}`)
	if !handled {
		t.Fatal("dictionary-sources-json command was not handled")
	}
	result, ok := got.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("dictionary-sources-json = %#v", got)
	}
	sources, ok := result["sources"].([]engine.DictionarySourcePreset)
	if !ok || len(sources) == 0 {
		t.Fatalf("dictionary sources = %#v", result["sources"])
	}
}

func TestExecuteExtensionCommandSchemaPresets(t *testing.T) {
	session := engine.New(engine.DefaultConfig())

	got, handled := executeSessionExtensionCommand(session, "schema-presets-json", `{}`)
	if !handled {
		t.Fatal("schema-presets-json command was not handled")
	}
	result, ok := got.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("schema-presets-json = %#v", got)
	}
	schemas, ok := result["schemas"].([]engine.SchemaPreset)
	if !ok || len(schemas) == 0 {
		t.Fatalf("schema presets = %#v", result["schemas"])
	}
}

func TestExecuteCommandAppliesSkinPreset(t *testing.T) {
	t.Setenv("SHURUFA233_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	session := engine.New(engine.DefaultConfig())

	listed, handled := executeSessionExtensionCommand(session, "skin-presets-json", `{}`)
	if !handled {
		t.Fatal("skin-presets-json command was not handled")
	}
	listResult, ok := listed.(map[string]any)
	if !ok || listResult["ok"] != true {
		t.Fatalf("skin-presets-json = %#v", listed)
	}
	presets, ok := listResult["presets"].([]engine.SkinPreset)
	if !ok || len(presets) == 0 {
		t.Fatalf("skin presets = %#v", listResult["presets"])
	}

	applied, handled := executeSessionExtensionCommand(session, "apply-skin-preset-json", `{"id":"wechat-dark"}`)
	if !handled {
		t.Fatal("apply-skin-preset-json command was not handled")
	}
	appliedResult, ok := applied.(map[string]any)
	if !ok || appliedResult["ok"] != true {
		t.Fatalf("apply-skin-preset-json = %#v", applied)
	}
	config, ok := appliedResult["config"].(engine.Config)
	if !ok || config.Skin.Theme != "wechat-dark" || config.CandidateLayout != "horizontal" || config.ShowCandidateComments {
		t.Fatalf("applied skin config = %#v", appliedResult["config"])
	}
}

func TestExecuteExtensionCommandRimeCustomYAML(t *testing.T) {
	session := engine.New(engine.DefaultConfig())

	got, handled := executeSessionExtensionCommand(session, "rime-custom-json", `{"yaml":"patch:\n  schema_list:\n    - schema: double_pinyin_flypy\n  menu/page_size: 8\n  style/horizontal: false\n  speller/algebra:\n    - derive/^zh/z/\n"}`)
	if !handled {
		t.Fatal("rime-custom-json command was not handled")
	}
	result, ok := got.(engine.RimeCustomResult)
	if !ok || !result.OK {
		t.Fatalf("rime-custom-json = %#v", got)
	}
	if session.Config().Schema != "double-pinyin-xiaohe" || session.Config().CandidatePageSize != 8 || session.Config().CandidateLayout != "vertical" {
		t.Fatalf("rime custom did not update session config: %#v", session.Config())
	}
	if len(session.Config().SpellerAlgebra) != 1 || !containsString(session.Config().FuzzyInitials, "zh=z") {
		t.Fatalf("rime speller algebra did not update session config: %#v", session.Config())
	}
}

func TestExecuteExtensionCommandRimeSwitches(t *testing.T) {
	session := engine.New(engine.DefaultConfig())

	got, handled := executeSessionExtensionCommand(session, "switches-json", `{}`)
	if !handled {
		t.Fatal("switches-json command was not handled")
	}
	result, ok := got.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("switches-json = %#v", got)
	}
	switches, ok := result["switches"].([]engine.SwitchOption)
	if !ok || len(switches) == 0 {
		t.Fatalf("switches = %#v", result["switches"])
	}

	applied, handled := executeSessionExtensionCommand(session, "apply-switch-json", `{"id":"ascii_mode","value":true}`)
	if !handled {
		t.Fatal("apply-switch-json command was not handled")
	}
	appliedResult, ok := applied.(map[string]any)
	if !ok || appliedResult["ok"] != true {
		t.Fatalf("apply-switch-json = %#v", applied)
	}
	if session.Config().Mode != "en" || session.State().Mode != "en" {
		t.Fatalf("ascii_mode switch did not update session config/state: config=%#v state=%#v", session.Config(), session.State())
	}
}

func TestExecuteExtensionCommandRimeRecognizerPatterns(t *testing.T) {
	session := engine.New(engine.DefaultConfig())

	got, handled := executeSessionExtensionCommand(session, "rime-recognizer-patterns-json", `{}`)
	if !handled {
		t.Fatal("rime-recognizer-patterns-json command was not handled")
	}
	result, ok := got.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("rime-recognizer-patterns-json = %#v", got)
	}
	patterns, ok := result["patterns"].(map[string]string)
	if !ok || patterns["url"] == "" || patterns["email"] == "" {
		t.Fatalf("recognizer patterns = %#v", result["patterns"])
	}
}

func TestExecuteExtensionCommandRecognizerDecision(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	got, handled := executeSessionExtensionCommand(session, "recognizer-decision-json", `{"input":"www.example.com"}`)
	if !handled {
		t.Fatal("recognizer-decision-json command was not handled")
	}
	decision, ok := got.(engine.RecognizerDecision)
	if !ok || !decision.OK || !decision.Matched || decision.Name != "url" || !decision.PassThrough {
		t.Fatalf("recognizer decision = %#v", got)
	}
}

func TestExecuteExtensionCommandAppContextRules(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	got, handled := executeSessionExtensionCommand(session, "app-rules-json", `{}`)
	if !handled {
		t.Fatal("app-rules-json command was not handled")
	}
	result, ok := got.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("app-rules-json = %#v", got)
	}
	rules, ok := result["rules"].([]engine.AppRule)
	if !ok || len(rules) == 0 {
		t.Fatalf("app rules = %#v", result["rules"])
	}

	decision, handled := executeSessionExtensionCommand(session, "resolve-app-context-json", `{"appContext":{"processName":"WeGame.exe"}}`)
	if !handled {
		t.Fatal("resolve-app-context-json command was not handled")
	}
	resolved, ok := decision.(engine.AppContextDecision)
	if !ok || !resolved.OK || !resolved.Matched || resolved.Mode != "en" || !resolved.DisableCandidates {
		t.Fatalf("resolve app context = %#v", decision)
	}
}

func TestExecuteExtensionCommandApplyAppRules(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SHURUFA233_CONFIG", configPath)
	session := engine.New(engine.DefaultConfig())

	got, handled := executeSessionExtensionCommand(session, "apply-app-rules-json", `{"rules":[{"id":"editor","name":"Editor","processNames":["Notepad.exe"],"mode":"en","disableCandidates":true,"priority":99}]}`)
	if !handled {
		t.Fatal("apply-app-rules-json command was not handled")
	}
	result, ok := got.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("apply app rules = %#v", got)
	}

	decision := engine.ResolveAppContext(session.Config(), engine.AppContext{ProcessName: "Notepad.exe"})
	if !decision.Matched || decision.Mode != "en" || !decision.DisableCandidates {
		t.Fatalf("applied app rules did not affect context decision: %#v", decision)
	}
	saved, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), `"id": "editor"`) {
		t.Fatalf("saved config does not include app rule: %s", saved)
	}
}

func TestExecuteExtensionCommandAgentConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SHURUFA233_CONFIG", configPath)
	session := engine.New(engine.DefaultConfig())

	got, handled := executeSessionExtensionCommand(session, "apply-agent-config-json", `{"agent":{"enabled":true,"provider":"local","model":"qwen","endpoint":"http://127.0.0.1:8787","timeoutMs":2500,"triggers":["/ask","/rewrite"],"actions":["copy","handoff"]}}`)
	if !handled {
		t.Fatal("apply-agent-config-json command was not handled")
	}
	result, ok := got.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("apply agent config = %#v", got)
	}
	config := loadConfig()
	if config.Agent.Provider != "local" || config.Agent.Model != "qwen" || config.Agent.TimeoutMs != 2500 || len(config.Agent.Triggers) != 2 {
		t.Fatalf("persisted agent config = %#v", config.Agent)
	}

	list, handled := executeSessionExtensionCommand(session, "agent-config-json", `{}`)
	if !handled {
		t.Fatal("agent-config-json command was not handled")
	}
	payload, ok := list.(map[string]any)
	if !ok || payload["ok"] != true {
		t.Fatalf("agent config payload = %#v", list)
	}
}

func TestExecuteExtensionCommandProfileBundle(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	session.ImportUserScores(map[string]int{"nihao|你好": 25})
	session.AddUserPhrases([]engine.Entry{{Reading: "msd", Text: "马上到", Weight: 60000}})

	got, handled := executeSessionExtensionCommand(session, "profile-json", `{}`)
	if !handled {
		t.Fatal("profile-json command was not handled")
	}
	profile, ok := got.(profileBundle)
	if !ok || !profile.OK || profile.Counts["userScores"] != 1 || profile.Counts["phrases"] != 1 {
		t.Fatalf("profile-json = %#v", got)
	}

	imported, handled := executeSessionExtensionCommand(session, "import-profile-json", `{"merge":false,"userScores":{"ceshi|测试":50},"phrases":[{"reading":"yyds","text":"永远的神","weight":70000}]}`)
	if !handled {
		t.Fatal("import-profile-json command was not handled")
	}
	result, ok := imported.(map[string]any)
	if !ok || result["ok"] != true {
		t.Fatalf("import-profile-json = %#v", imported)
	}
	if session.UserScores()["ceshi|测试"] != 50 || len(session.UserPhrases()) != 1 || session.UserPhrases()[0].Reading != "yyds" {
		t.Fatalf("profile import did not update session: scores=%#v phrases=%#v", session.UserScores(), session.UserPhrases())
	}
}

func TestExecuteExtensionCommandProfileSync(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHURUFA233_CONFIG", filepath.Join(dir, "config.json"))
	session := engine.New(engine.DefaultConfig())
	session.ImportUserScores(map[string]int{"nihao|你好": 25})
	session.AddUserPhrases([]engine.Entry{{Reading: "msd", Text: "马上到", Weight: 60000}})
	syncDir := filepath.Join(dir, "sync")

	got, handled := executeSessionExtensionCommand(session, "apply-sync-config-json", `{"sync":{"enabled":true,"provider":"local-directory","directory":"`+filepath.ToSlash(syncDir)+`","conflictPolicy":"merge-newer"}}`)
	if !handled {
		t.Fatal("apply-sync-config-json command was not handled")
	}
	if result, ok := got.(map[string]any); !ok || result["ok"] != true {
		t.Fatalf("apply sync config = %#v", got)
	}

	exported, handled := executeSessionExtensionCommand(session, "sync-export", `{}`)
	if !handled {
		t.Fatal("sync-export command was not handled")
	}
	exportResult, ok := exported.(map[string]any)
	if !ok || exportResult["ok"] != true {
		t.Fatalf("sync export = %#v", exported)
	}
	if _, err := os.Stat(filepath.Join(syncDir, "shurufa233-profile.json")); err != nil {
		t.Fatalf("sync profile was not written: %v", err)
	}

	session.ReplaceUserScores(map[string]int{})
	session.ReplaceUserPhrases(nil)
	imported, handled := executeSessionExtensionCommand(session, "sync-import", `{}`)
	if !handled {
		t.Fatal("sync-import command was not handled")
	}
	importResult, ok := imported.(map[string]any)
	if !ok || importResult["ok"] != true {
		t.Fatalf("sync import = %#v", imported)
	}
	if session.UserScores()["nihao|你好"] != 25 || len(session.UserPhrases()) != 1 {
		t.Fatalf("sync import did not restore session: scores=%#v phrases=%#v", session.UserScores(), session.UserPhrases())
	}
}

func TestExecuteExtensionCommandDeletesUserData(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHURUFA233_USER_PHRASES", filepath.Join(dir, "user-phrases.json"))
	t.Setenv("SHURUFA233_USER_REJECTS", filepath.Join(dir, "user-rejects.json"))
	t.Setenv("SHURUFA233_USER_PINS", filepath.Join(dir, "user-pins.json"))
	session := engine.New(engine.DefaultConfig())
	session.AddEntries([]engine.Entry{
		{Reading: "ceshi", Text: "测试", Weight: 30000},
		{Reading: "ceshi", Text: "错词", Weight: 20000},
	})
	session.AddUserPhrases([]engine.Entry{{Reading: "msd", Text: "马上到！", Weight: 60000}})
	session.AddUserRejects([]engine.Entry{{Reading: "ceshi", Text: "错词", Weight: 1}})
	session.AddUserPins([]engine.Entry{{Reading: "ceshi", Text: "测试", Weight: 1}})

	got, handled := executeSessionExtensionCommand(session, "delete-user-phrase", `{"reading":"msd","text":"马上到！"}`)
	if !handled {
		t.Fatal("delete-user-phrase command was not handled")
	}
	phrases, ok := got.(map[string]any)
	if !ok || phrases["ok"] != true || phrases["total"] != 0 {
		t.Fatalf("delete user phrase = %#v", got)
	}

	got, handled = executeSessionExtensionCommand(session, "delete-user-reject", `{"reading":"ceshi","text":"错词"}`)
	if !handled {
		t.Fatal("delete-user-reject command was not handled")
	}
	rejects, ok := got.(map[string]any)
	if !ok || rejects["ok"] != true || rejects["total"] != 0 {
		t.Fatalf("delete user reject = %#v", got)
	}

	got, handled = executeSessionExtensionCommand(session, "delete-user-pin", `{"reading":"ceshi","text":"测试"}`)
	if !handled {
		t.Fatal("delete-user-pin command was not handled")
	}
	pins, ok := got.(map[string]any)
	if !ok || pins["ok"] != true || pins["total"] != 0 {
		t.Fatalf("delete user pin = %#v", got)
	}
}

func TestExecuteExtensionCommandCandidateActionCommitsCandidateChar(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	session.Preview("zhongwen")

	got, handled := executeSessionExtensionCommand(session, "candidate-action", `{"action":"first-char","index":0}`)
	if !handled {
		t.Fatal("candidate-action first-char was not handled")
	}
	result, ok := got.(candidateActionResult)
	if !ok || !result.OK || result.Committed != "中" {
		t.Fatalf("candidate-action first-char = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventInputsAndCommits(t *testing.T) {
	t.Setenv("SHURUFA233_USER_SCORES", filepath.Join(t.TempDir(), "user-scores.json"))
	session := engine.New(engine.DefaultConfig())
	for _, key := range []string{"n", "i", "h", "a", "o"} {
		got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":"`+key+`","character":"`+key+`"}`)
		if !handled {
			t.Fatal("key-event-json command was not handled")
		}
		result, ok := got.(keyEventResult)
		if !ok || !result.OK || !result.Handled || result.Action != "input" {
			t.Fatalf("key-event input %q = %#v", key, got)
		}
	}
	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":"space"}`)
	if !handled {
		t.Fatal("key-event-json space was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || !result.Handled || result.Action != "commit-candidate" || result.Committed != "你好" || result.State.Buffer != "" {
		t.Fatalf("key-event space = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventRespectsGameContext(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":"n","character":"n","appContext":{"processName":"WeGame.exe"}}`)
	if !handled {
		t.Fatal("key-event-json command was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || result.Handled || result.PassThrough != "n" || result.Decision == nil || !result.Decision.DisableCandidates {
		t.Fatalf("key-event game context = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventRecognizerKeepsURLPunctuationLiteral(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	for _, key := range []string{"w", "w", "w", ".", "e", "x", "a", "m", "p", "l", "e", ".", "c", "o", "m"} {
		got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":"`+key+`","character":"`+key+`"}`)
		if !handled {
			t.Fatal("key-event-json command was not handled")
		}
		result, ok := got.(keyEventResult)
		if !ok || !result.OK || !result.Handled {
			t.Fatalf("key-event URL input %q = %#v", key, got)
		}
		if key == "." && result.Action != "input-recognizer-literal" {
			t.Fatalf("URL dot should remain literal input, got %#v", result)
		}
	}

	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":",","character":","}`)
	if !handled {
		t.Fatal("key-event-json comma was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || !result.Handled || result.Action != "commit-recognizer-punctuation" ||
		result.Committed != "www.example.com," || result.Recognizer == nil || result.Recognizer.Name != "url" || result.State.Buffer != "" {
		t.Fatalf("recognizer URL comma = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventRecognizerCommitsEmailWithSpace(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	for _, key := range []string{"d", "e", "v", "@", "e", "x", "a", "m", "p", "l", "e", ".", "c", "o", "m"} {
		got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":"`+key+`","character":"`+key+`"}`)
		if !handled {
			t.Fatal("key-event-json command was not handled")
		}
		result, ok := got.(keyEventResult)
		if !ok || !result.OK || !result.Handled {
			t.Fatalf("key-event email input %q = %#v", key, got)
		}
	}

	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":"space"}`)
	if !handled {
		t.Fatal("key-event-json space was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || !result.Handled || result.Action != "commit-recognizer-literal" ||
		result.Committed != "dev@example.com " || result.Recognizer == nil || result.Recognizer.Name != "email" || result.State.Buffer != "" {
		t.Fatalf("recognizer email space = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventShiftToggle(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":"shift","action":"tap"}`)
	if !handled {
		t.Fatal("key-event-json command was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || !result.Handled || result.Action != "toggle-mode" || result.State.Mode != "en" {
		t.Fatalf("key-event shift toggle = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventCommitsFullWidthPunctuation(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":",","character":","}`)
	if !handled {
		t.Fatal("key-event-json command was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || !result.Handled || result.Action != "commit-punctuation" || result.Committed != "，" {
		t.Fatalf("key-event punctuation = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventCommitsCandidateBeforePunctuation(t *testing.T) {
	t.Setenv("SHURUFA233_USER_SCORES", filepath.Join(t.TempDir(), "user-scores.json"))
	session := engine.New(engine.DefaultConfig())
	session.Preview("nihao")
	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":",","character":","}`)
	if !handled {
		t.Fatal("key-event-json command was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || !result.Handled || result.Action != "commit-candidate-punctuation" || result.Committed != "你好，" || result.State.Buffer != "" {
		t.Fatalf("key-event candidate punctuation = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventHalfPunctuationPassesThroughWhenIdle(t *testing.T) {
	config := engine.DefaultConfig()
	config.Punctuation = "half"
	session := engine.New(config)
	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":".","character":"."}`)
	if !handled {
		t.Fatal("key-event-json command was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || result.Handled || result.PassThrough != "." || result.Reason != "half-punctuation" {
		t.Fatalf("key-event half punctuation = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventUsesConfiguredRimePunctuationShape(t *testing.T) {
	config := engine.DefaultConfig()
	config.PunctuationFullShape = map[string][]string{
		",": {"，自定义"},
	}
	session := engine.New(config)
	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":",","character":","}`)
	if !handled {
		t.Fatal("key-event-json command was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || !result.Handled || result.Committed != "，自定义" {
		t.Fatalf("key-event configured punctuation = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventQuickSelectsSecondAndThirdCandidate(t *testing.T) {
	t.Setenv("SHURUFA233_USER_SCORES", filepath.Join(t.TempDir(), "user-scores.json"))
	session := engine.New(engine.DefaultConfig())
	session.AddEntries([]engine.Entry{
		{Reading: "quick", Text: "候一", Weight: 100},
		{Reading: "quick", Text: "候二", Weight: 90},
		{Reading: "quick", Text: "候三", Weight: 80},
	})
	session.Preview("quick")

	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":";","character":";"}`)
	if !handled {
		t.Fatal("key-event-json semicolon was not handled")
	}
	second, ok := got.(keyEventResult)
	if !ok || !second.OK || !second.Handled || second.Action != "commit-candidate" || second.Index != 1 || second.Committed != "候二" || second.Reason != "quick-select" {
		t.Fatalf("semicolon quick select = %#v", got)
	}

	session.Preview("quick")
	got, handled = executeSessionExtensionCommand(session, "key-event-json", `{"key":"'","character":"'"}`)
	if !handled {
		t.Fatal("key-event-json quote was not handled")
	}
	third, ok := got.(keyEventResult)
	if !ok || !third.OK || !third.Handled || third.Action != "commit-candidate" || third.Index != 2 || third.Committed != "候三" || third.Reason != "quick-select" {
		t.Fatalf("quote quick select = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventRimeProfilePagesWithCommaPeriod(t *testing.T) {
	config := engine.DefaultConfig()
	config.KeyProfile = "rime"
	session := engine.New(config)
	for i := 0; i < 10; i++ {
		session.AddEntries([]engine.Entry{{Reading: "page", Text: "候" + strconv.Itoa(i), Weight: 100 - i}})
	}
	session.Preview("page")

	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":".","character":".","limit":5}`)
	if !handled {
		t.Fatal("key-event-json period was not handled")
	}
	next, ok := got.(keyEventResult)
	if !ok || !next.OK || !next.Handled || next.Action != "page-candidates" || next.PageDelta != 1 {
		t.Fatalf("rime period page = %#v", got)
	}

	got, handled = executeSessionExtensionCommand(session, "key-event-json", `{"key":",","character":",","limit":5}`)
	if !handled {
		t.Fatal("key-event-json comma was not handled")
	}
	prev, ok := got.(keyEventResult)
	if !ok || !prev.OK || !prev.Handled || prev.Action != "page-candidates" || prev.PageDelta != -1 {
		t.Fatalf("rime comma page = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventNavigationIntent(t *testing.T) {
	session := engine.New(engine.DefaultConfig())
	session.AddEntries([]engine.Entry{
		{Reading: "move", Text: "候一", Weight: 100},
		{Reading: "move", Text: "候二", Weight: 90},
	})
	session.Preview("move")

	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":"right"}`)
	if !handled {
		t.Fatal("key-event-json right was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || !result.Handled || result.Action != "move-selection" || result.MoveDelta != 1 {
		t.Fatalf("right navigation intent = %#v", got)
	}
}

func TestExecuteExtensionCommandKeyEventMicrosoftDoublePinyinKeepsSemicolonInput(t *testing.T) {
	config := engine.DefaultConfig()
	config.DoublePinyin = true
	config.DoublePinyinScheme = "microsoft"
	session := engine.New(config)
	session.AddEntries([]engine.Entry{
		{Reading: "quick", Text: "候一", Weight: 100},
		{Reading: "quick", Text: "候二", Weight: 90},
	})
	session.Preview("quick")

	got, handled := executeSessionExtensionCommand(session, "key-event-json", `{"key":";","character":";"}`)
	if !handled {
		t.Fatal("key-event-json semicolon was not handled")
	}
	result, ok := got.(keyEventResult)
	if !ok || !result.OK || !result.Handled || result.Action == "commit-candidate" || result.Committed == "候二" {
		t.Fatalf("microsoft double pinyin semicolon should not quick-select: %#v", got)
	}
}

func TestExecuteExtensionCommandPreviewAndCandidatePayload(t *testing.T) {
	t.Setenv("SHURUFA233_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("SHURUFA233_DICTIONARY_DIR", t.TempDir())
	t.Setenv("SHURUFA233_USER_SCORES", filepath.Join(t.TempDir(), "user-scores.json"))

	session := engine.New(engine.DefaultConfig())
	preview, handled := executeSessionExtensionCommand(session, "preview", `{"input":"zan"}`)
	if !handled {
		t.Fatal("preview command was not handled")
	}
	state, ok := preview.(engine.State)
	if !ok || len(state.Candidates) == 0 || state.Candidates[0].Text != "👍" {
		t.Fatalf("preview command = %#v", preview)
	}

	payload, handled := executeSessionExtensionCommand(session, "candidate-payload-v2", `{"start":0,"limit":3}`)
	if !handled {
		t.Fatal("candidate-payload-v2 command was not handled")
	}
	candidates, ok := payload.(candidatePayloadV2)
	if !ok || !candidates.OK || len(candidates.Items) == 0 {
		t.Fatalf("candidate payload command = %#v", payload)
	}
	if candidates.Items[0].Comment == "" {
		t.Fatalf("expected candidate comment through extension command, got %#v", candidates.Items[0])
	}
}

func TestExecuteExtensionCommandAgentCompose(t *testing.T) {
	got, handled := executeSessionExtensionCommand(engine.New(engine.DefaultConfig()), "agent-compose", `{"input":"/ask 输入法怎么优化","context":"保留上下文"}`)
	if !handled {
		t.Fatal("agent-compose command was not handled")
	}
	result, ok := got.(agentComposeResult)
	if !ok || !result.OK || len(result.Items) == 0 {
		t.Fatalf("agent-compose command = %#v", got)
	}
	if result.Items[0].Intent != "ask" || !strings.Contains(result.Items[0].Text, "输入法怎么优化") || result.Provider == "" {
		t.Fatalf("unexpected agent-compose command result: %#v", result.Items[0])
	}
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
