package engine

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestSelectCandidateFirstAndLastChar(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{{Reading: "houxuan", Text: "候选", Weight: 20000}})

	state := e.Preview("houxuan")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "候选" {
		t.Fatalf("expected 候选 candidate, got %#v", state.Candidates)
	}
	committed, err := e.SelectChar(0, "first")
	if err != nil {
		t.Fatal(err)
	}
	if committed.Committed != "候" || committed.Buffer != "" {
		t.Fatalf("first char commit = %#v", committed)
	}

	e.Preview("houxuan")
	committed, err = e.SelectChar(0, "last")
	if err != nil {
		t.Fatal(err)
	}
	if committed.Committed != "选" || committed.Buffer != "" {
		t.Fatalf("last char commit = %#v", committed)
	}
}

func TestSelectCandidateCharRejectsInvalidSide(t *testing.T) {
	e := New(DefaultConfig())
	e.Preview("nihao")
	if _, err := e.SelectChar(0, "middle"); err == nil {
		t.Fatal("expected invalid side error")
	}
}

func TestCandidatePageSizeConfigIsClamped(t *testing.T) {
	config := DefaultConfig()
	config.CandidatePageSize = 99
	config.CandidateLayout = "rime"
	e := New(config)
	if e.config.CandidatePageSize != 9 {
		t.Fatalf("candidate page size = %d, want 9", e.config.CandidatePageSize)
	}
	if e.config.CandidateLayout != "vertical" {
		t.Fatalf("candidate layout = %q, want vertical", e.config.CandidateLayout)
	}

	config.CandidatePageSize = 1
	config.CandidateLayout = "sideways"
	e.Configure(config)
	if e.config.CandidatePageSize != 3 {
		t.Fatalf("candidate page size = %d, want 3", e.config.CandidatePageSize)
	}
	if e.config.CandidateLayout != "horizontal" {
		t.Fatalf("candidate layout = %q, want horizontal", e.config.CandidateLayout)
	}
}

func TestBuiltinSchemaPresetsIncludeRimeAndDoublePinyin(t *testing.T) {
	presets := BuiltinSchemaPresets()
	seen := map[string]bool{}
	for _, preset := range presets {
		seen[preset.ID] = true
	}
	for _, id := range []string{"wechat-pinyin", "rime-luna-pinyin", "rime-ice-pinyin", "double-pinyin-xiaohe", "double-pinyin-microsoft"} {
		if !seen[id] {
			t.Fatalf("missing schema preset %q in %#v", id, presets)
		}
	}
}

func TestReverseLookupFindsReadingsAndSources(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{{Reading: "shurufa", Text: "输入法", Kind: "phrase", Source: "rime-test", Comment: "测试", Weight: 22000}})

	got := e.ReverseLookup(ReverseLookupRequest{Query: "输入法", Limit: 5})

	if got.Query != "输入法" || len(got.Entries) == 0 {
		t.Fatalf("reverse lookup = %#v", got)
	}
	if got.Entries[0].Reading != "shurufa" || got.Entries[0].Source != "rime-test" {
		t.Fatalf("reverse lookup entry = %#v", got.Entries[0])
	}
}

func TestApplySchemaPresetConfigEnablesMicrosoftDoublePinyin(t *testing.T) {
	config, ok := ApplySchemaPresetConfig(DefaultConfig(), "double-pinyin-microsoft")
	if !ok {
		t.Fatal("expected microsoft double pinyin schema")
	}
	if config.Schema != "double-pinyin-microsoft" || !config.DoublePinyin || config.DoublePinyinScheme != "microsoft" {
		t.Fatalf("schema config = %#v", config)
	}
}

func TestNormalizeSchemaConfigDerivesManualDoublePinyin(t *testing.T) {
	config := DefaultConfig()
	config.Schema = ""
	config.DoublePinyin = true
	config.DoublePinyinScheme = "xiaohe"

	got := NormalizeSchemaConfig(config)

	if got.Schema != "double-pinyin-xiaohe" {
		t.Fatalf("schema = %q, want double-pinyin-xiaohe", got.Schema)
	}
}

func TestGreedySegmentation(t *testing.T) {
	e := New(DefaultConfig())
	state := e.Preview("womende")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "我们的" {
		t.Fatalf("expected segmented candidate 我们的, got %#v", state.Candidates)
	}
}

func TestSegmentedCandidateUsesBestScoredPath(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{
		{Reading: "nide", Text: "泥德", Weight: 100},
	})

	state := e.Preview("nide")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "你的" {
		t.Fatalf("expected best segmented candidate 你的 to outrank low-quality exact word, got %#v", state.Candidates)
	}
	if state.Candidates[0].Kind != "phrase" || state.Candidates[0].Source != "segmenter" {
		t.Fatalf("expected segmented phrase metadata, got %#v", state.Candidates[0])
	}
}

func TestSegmentedCandidateDoesNotOverrideStrongExactPhrase(t *testing.T) {
	e := New(DefaultConfig())
	state := e.Preview("nihao")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "你好" {
		t.Fatalf("expected exact phrase 你好 to remain first, got %#v", state.Candidates)
	}
	if state.Candidates[0].Source == "segmenter" {
		t.Fatalf("exact phrase should not be replaced by segmented metadata: %#v", state.Candidates[0])
	}
}

func TestSegmentedCandidateUsesUserScoresInPath(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{
		{Reading: "aa", Text: "甲", Weight: 500},
		{Reading: "aa", Text: "乙", Weight: 100},
		{Reading: "bb", Text: "丙", Weight: 500},
	})

	state := e.Preview("aabb")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "甲丙" {
		t.Fatalf("expected static segment path 甲丙, got %#v", state.Candidates)
	}

	e.ImportUserScores(map[string]int{"aa|乙": 1000})
	state = e.Preview("aabb")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "乙丙" {
		t.Fatalf("expected learned segment path 乙丙, got %#v", state.Candidates)
	}
}

func TestApostropheSeparatorPreservesBufferAndForcesSegmentation(t *testing.T) {
	e := New(DefaultConfig())

	plain := e.Preview("xian")
	if len(plain.Candidates) == 0 || plain.Candidates[0].Text != "先" {
		t.Fatalf("expected plain xian to keep exact candidate first, got %#v", plain.Candidates)
	}

	state := e.Preview("xi'an")
	if state.Buffer != "xi'an" {
		t.Fatalf("expected apostrophe buffer to be preserved, got %q", state.Buffer)
	}
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "西安" {
		t.Fatalf("expected xi'an to force 西安 segmentation, got %#v", state.Candidates)
	}
	if state.Candidates[0].Kind != "phrase" || state.Candidates[0].Source != "separator" {
		t.Fatalf("expected separator phrase metadata, got %#v", state.Candidates[0])
	}
}

func TestInputKeyAcceptsApostropheSeparator(t *testing.T) {
	e := New(DefaultConfig())
	for _, r := range "xi'an" {
		e.InputKey(r)
	}
	state := e.State()
	if state.Buffer != "xi'an" {
		t.Fatalf("expected typed apostrophe buffer, got %q", state.Buffer)
	}
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "西安" {
		t.Fatalf("expected typed xi'an to produce 西安, got %#v", state.Candidates)
	}
}

func TestSlashSymbolPrefixPreservesBufferAndFiltersCandidates(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{{Reading: "fs", Text: "普通词", Weight: 90000}})

	plain := e.Preview("fs")
	if len(plain.Candidates) == 0 || plain.Candidates[0].Text != "普通词" {
		t.Fatalf("expected plain fs to allow ordinary candidates, got %#v", plain.Candidates)
	}

	state := e.Preview("/fs")
	if state.Buffer != "/fs" {
		t.Fatalf("expected slash prefix buffer, got %q", state.Buffer)
	}
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "℃" {
		t.Fatalf("expected /fs to prefer symbol candidates, got %#v", state.Candidates)
	}
	for _, candidate := range state.Candidates {
		if candidate.Text == "普通词" {
			t.Fatalf("slash prefix should filter ordinary candidates, got %#v", state.Candidates)
		}
	}
}

func TestInputKeyAcceptsSlashSymbolPrefix(t *testing.T) {
	e := New(DefaultConfig())
	for _, r := range "/xh" {
		e.InputKey(r)
	}
	state := e.State()
	if state.Buffer != "/xh" {
		t.Fatalf("expected typed slash prefix buffer, got %q", state.Buffer)
	}
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "※" {
		t.Fatalf("expected typed /xh to produce symbol candidates, got %#v", state.Candidates)
	}
}

func TestUserPhrasesCanBeAddedListedAndDeleted(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{{Reading: "msd", Text: "默认短语", Weight: 20000}})

	added := e.AddUserPhrases([]Entry{{Reading: "msd", Text: "马上到！"}})
	if len(added) != 1 || added[0].Kind != UserPhraseKind || added[0].Source != UserPhraseSource {
		t.Fatalf("added user phrase metadata = %#v", added)
	}
	state := e.Preview("msd")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "马上到！" {
		t.Fatalf("expected user phrase to rank first, got %#v", state.Candidates)
	}
	if state.Candidates[0].Kind != UserPhraseKind || state.Candidates[0].Source != UserPhraseSource {
		t.Fatalf("expected user phrase candidate metadata, got %#v", state.Candidates[0])
	}

	phrases := e.UserPhrases()
	if len(phrases) != 1 || phrases[0].Text != "马上到！" {
		t.Fatalf("user phrases = %#v", phrases)
	}
	e.DeleteUserPhrase("msd", "马上到！")
	state = e.Preview("msd")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "默认短语" {
		t.Fatalf("expected deleting user phrase to restore default phrase, got %#v", state.Candidates)
	}
}

func TestRejectCandidateHidesAndRestoresCandidate(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{
		{Reading: "ceshi", Text: "错词", Weight: 20000},
		{Reading: "ceshi", Text: "测试", Weight: 10000},
	})

	state := e.Preview("ceshi")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "错词" {
		t.Fatalf("expected wrong candidate first, got %#v", state.Candidates)
	}
	state, rejected, err := e.RejectCandidate(0)
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Reading != "ceshi" || rejected.Text != "错词" {
		t.Fatalf("rejected = %#v", rejected)
	}
	if len(state.Candidates) == 0 || state.Candidates[0].Text == "错词" {
		t.Fatalf("expected rejected candidate to disappear, got %#v", state.Candidates)
	}
	rejects := e.UserRejects()
	if len(rejects) != 1 || rejects[0].Text != "错词" {
		t.Fatalf("user rejects = %#v", rejects)
	}

	e.DeleteUserReject("ceshi", "错词")
	state = e.Preview("ceshi")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "错词" {
		t.Fatalf("expected restored candidate, got %#v", state.Candidates)
	}
}

func TestPinCandidatePromotesAndCanBeRemoved(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{
		{Reading: "ceshi", Text: "测试", Weight: 30000},
		{Reading: "ceshi", Text: "侧室", Weight: 1000},
	})

	state := e.Preview("ceshi")
	if len(state.Candidates) < 2 || state.Candidates[0].Text != "测试" {
		t.Fatalf("expected default high-weight candidate first, got %#v", state.Candidates)
	}
	state, pinned, err := e.PinCandidate(1)
	if err != nil {
		t.Fatal(err)
	}
	if pinned.Reading != "ceshi" || pinned.Text != "侧室" {
		t.Fatalf("pinned = %#v", pinned)
	}
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "侧室" || !state.Candidates[0].Pinned {
		t.Fatalf("expected pinned candidate to rank first, got %#v", state.Candidates)
	}
	pins := e.UserPins()
	if len(pins) != 1 || pins[0].Text != "侧室" {
		t.Fatalf("user pins = %#v", pins)
	}

	e.DeleteUserPin("ceshi", "侧室")
	state = e.Preview("ceshi")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "测试" || state.Candidates[0].Pinned {
		t.Fatalf("expected unpinned ranking to restore, got %#v", state.Candidates)
	}
}

func TestPinCandidateRemovesMatchingReject(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{{Reading: "ceshi", Text: "错词", Weight: 20000}})
	e.AddUserRejects([]Entry{{Reading: "ceshi", Text: "错词"}})
	if state := e.Preview("ceshi"); len(state.Candidates) != 0 {
		t.Fatalf("expected rejected candidate hidden, got %#v", state.Candidates)
	}

	e.AddUserPins([]Entry{{Reading: "ceshi", Text: "错词"}})
	state := e.Preview("ceshi")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "错词" || !state.Candidates[0].Pinned {
		t.Fatalf("expected pin to restore and promote candidate, got %#v", state.Candidates)
	}
	if rejects := e.UserRejects(); len(rejects) != 0 {
		t.Fatalf("expected matching reject removed, got %#v", rejects)
	}
}

func TestTraditionalScriptConvertsCandidateText(t *testing.T) {
	config := DefaultConfig()
	config.Script = "traditional"
	e := New(config)
	e.AddEntries([]Entry{{Reading: "zhongguo", Text: "中国", Weight: 20000}})

	state := e.Preview("zhongguo")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "中國" {
		t.Fatalf("expected traditional candidate text, got %#v", state.Candidates)
	}
	if state.Candidates[0].Reading != "zhongguo" {
		t.Fatalf("candidate reading should stay unchanged, got %#v", state.Candidates[0])
	}
}

func TestTraditionalRejectUsesConvertedCandidateText(t *testing.T) {
	config := DefaultConfig()
	config.Script = "traditional"
	e := New(config)
	e.AddEntries([]Entry{
		{Reading: "ceshi", Text: "错词", Weight: 20000},
		{Reading: "ceshi", Text: "测试", Weight: 10000},
	})

	state := e.Preview("ceshi")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "錯詞" {
		t.Fatalf("expected converted wrong candidate first, got %#v", state.Candidates)
	}
	state, rejected, err := e.RejectCandidate(0)
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Reading != "ceshi" || rejected.Text != "錯詞" {
		t.Fatalf("rejected = %#v", rejected)
	}
	if len(state.Candidates) == 0 || state.Candidates[0].Text == "錯詞" {
		t.Fatalf("expected converted rejected candidate to disappear, got %#v", state.Candidates)
	}
}

func TestAssociateReturnsNextWordCandidates(t *testing.T) {
	e := New(DefaultConfig())

	state := e.Associate("你好", 3)
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "世界" {
		t.Fatalf("expected association candidates for 你好, got %#v", state.Candidates)
	}
	if state.Candidates[0].Kind != "association" || state.Candidates[0].Comment != "联想" {
		t.Fatalf("association metadata = %#v", state.Candidates[0])
	}
}

func TestSelectReturnsAssociationCandidates(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{{Reading: "weixin", Text: "微信", Weight: 20000}})

	state := e.Preview("weixin")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "微信" {
		t.Fatalf("expected 微信 candidate, got %#v", state.Candidates)
	}
	state, err := e.Select(0)
	if err != nil {
		t.Fatal(err)
	}
	if state.Committed != "微信" {
		t.Fatalf("committed = %q, want 微信", state.Committed)
	}
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "输入法" {
		t.Fatalf("expected association after select, got %#v", state.Candidates)
	}
}

func TestAssociationsCanBeDisabled(t *testing.T) {
	config := DefaultConfig()
	config.Associations = false
	e := New(config)

	state := e.Associate("你好", 3)
	if len(state.Candidates) != 0 {
		t.Fatalf("associations should be disabled, got %#v", state.Candidates)
	}
}

func TestReplaceUserPhrasesClearsOldPhrases(t *testing.T) {
	e := New(DefaultConfig())
	e.AddUserPhrases([]Entry{{Reading: "aaa", Text: "旧短语"}})
	e.ReplaceUserPhrases([]Entry{{Reading: "bbb", Text: "新短语", Weight: 60000}})

	if got := e.Preview("aaa"); len(got.Candidates) != 0 {
		t.Fatalf("old user phrase should be removed, got %#v", got.Candidates)
	}
	got := e.Preview("bbb")
	if len(got.Candidates) == 0 || got.Candidates[0].Text != "新短语" {
		t.Fatalf("expected replacement user phrase, got %#v", got.Candidates)
	}
}

func TestAbbreviationCandidates(t *testing.T) {
	e := New(DefaultConfig())

	tests := []struct {
		input string
		want  string
	}{
		{input: "nh", want: "你好"},
		{input: "wx", want: "微信"},
		{input: "srf", want: "输入法"},
		{input: "zw", want: "中文"},
	}

	for _, tt := range tests {
		state := e.Preview(tt.input)
		if len(state.Candidates) == 0 || state.Candidates[0].Text != tt.want {
			t.Fatalf("expected abbreviation %s -> %s, got %#v", tt.input, tt.want, state.Candidates)
		}
	}
}

func TestAbbreviationCandidatesFromImportedDictionary(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{
		{Reading: "zhongguoren", Text: "中国人", Weight: 18000},
		{Reading: "weixinshurufa", Text: "微信输入法", Weight: 19000},
	})

	tests := []struct {
		input string
		want  string
	}{
		{input: "zgr", want: "中国人"},
		{input: "wxsrf", want: "微信输入法"},
	}

	for _, tt := range tests {
		state := e.Preview(tt.input)
		if len(state.Candidates) == 0 || state.Candidates[0].Text != tt.want {
			t.Fatalf("expected imported abbreviation %s -> %s, got %#v", tt.input, tt.want, state.Candidates)
		}
	}
}

func TestAbbreviationKeepsFullPinyinExactCandidateFirst(t *testing.T) {
	e := New(DefaultConfig())
	state := e.Preview("nihao")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "你好" {
		t.Fatalf("expected full pinyin exact candidate first, got %#v", state.Candidates)
	}
}

func TestAbbreviationUsesUserScores(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{
		{Reading: "zhongguoren", Text: "中国人", Weight: 18000},
		{Reading: "zhiguangren", Text: "追光人", Weight: 17000},
	})

	state := e.Preview("zgr")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "中国人" {
		t.Fatalf("expected static abbreviation ranking, got %#v", state.Candidates)
	}

	e.ImportUserScores(map[string]int{"zhiguangren|追光人": 2000})
	state = e.Preview("zgr")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "追光人" {
		t.Fatalf("expected learned abbreviation ranking, got %#v", state.Candidates)
	}
}

func TestEnglishMode(t *testing.T) {
	config := DefaultConfig()
	config.Mode = "en"
	e := New(config)
	state := e.Preview("hello")
	if state.Mode != "en" {
		t.Fatalf("mode = %q, want en", state.Mode)
	}
	if len(state.Candidates) != 1 || state.Candidates[0].Text != "hello" {
		t.Fatalf("expected passthrough English candidate, got %#v", state.Candidates)
	}
}

func TestToggleModeClearsBufferAndNormalizesMode(t *testing.T) {
	config := DefaultConfig()
	config.Mode = "broken"
	e := New(config)
	if got := e.State().Mode; got != "zh" {
		t.Fatalf("initial normalized mode = %q, want zh", got)
	}
	e.Preview("nihao")
	state := e.ToggleMode()
	if state.Mode != "en" {
		t.Fatalf("mode after toggle = %q, want en", state.Mode)
	}
	if state.Buffer != "" || len(state.Candidates) != 0 {
		t.Fatalf("toggle should clear composition, got %#v", state)
	}
	state = e.SetMode("zh")
	if state.Mode != "zh" {
		t.Fatalf("mode after set = %q, want zh", state.Mode)
	}
}

func TestDefaultConfigShowsCandidateComments(t *testing.T) {
	if !DefaultConfig().ShowCandidateComments {
		t.Fatal("showCandidateComments should default to true")
	}
}

func TestKeyBehaviorProfilesNormalize(t *testing.T) {
	wechat := NormalizeKeyBehavior(Config{KeyProfile: "wechat"})
	if !wechat.ShiftToggleMode || !wechat.SemicolonQuickSelect || !wechat.QuoteQuickSelect ||
		!wechat.BracketPageKeys || !wechat.MinusEqualPageKeys || wechat.CommaPeriodPageKeys {
		t.Fatalf("wechat key behavior = %#v", wechat)
	}

	rime := NormalizeKeyBehavior(Config{KeyProfile: "rime"})
	if !rime.ShiftToggleMode || rime.SemicolonQuickSelect || rime.QuoteQuickSelect ||
		!rime.BracketPageKeys || !rime.MinusEqualPageKeys || !rime.CommaPeriodPageKeys {
		t.Fatalf("rime key behavior = %#v", rime)
	}

	custom := NormalizeKeyBehavior(Config{
		KeyProfile:          "custom",
		BracketPageKeys:     true,
		CommaPeriodPageKeys: true,
	})
	if custom.ShiftToggleMode || custom.SemicolonQuickSelect || custom.QuoteQuickSelect ||
		!custom.BracketPageKeys || custom.MinusEqualPageKeys || !custom.CommaPeriodPageKeys {
		t.Fatalf("custom key behavior = %#v", custom)
	}
}

func TestKeyBehaviorCanBeDerivedFromSchema(t *testing.T) {
	config := NormalizeKeyBehavior(Config{Schema: "rime-luna-pinyin"})
	if config.KeyProfile != "rime" || !config.CommaPeriodPageKeys || config.SemicolonQuickSelect {
		t.Fatalf("schema-derived key behavior = %#v", config)
	}
}

func TestFuzzyInitialsExpandCandidates(t *testing.T) {
	e := New(DefaultConfig())

	tests := []struct {
		input string
		want  string
	}{
		{input: "zongwen", want: "中文"},
		{input: "si", want: "是"},
		{input: "surufa", want: "输入法"},
	}

	for _, tt := range tests {
		state := e.Preview(tt.input)
		found := false
		for _, candidate := range state.Candidates {
			if candidate.Text == tt.want {
				found = true
				if candidate.Weight <= 0 {
					t.Fatalf("fuzzy candidate has invalid weight: %#v", candidate)
				}
				break
			}
		}
		if !found {
			t.Fatalf("expected fuzzy candidate %q for %q, got %#v", tt.want, tt.input, state.Candidates)
		}
	}
}

func TestFuzzyInitialsCanBeDisabled(t *testing.T) {
	config := DefaultConfig()
	config.FuzzyInitials = nil
	e := New(config)

	state := e.Preview("zongwen")
	for _, candidate := range state.Candidates {
		if candidate.Text == "中文" {
			t.Fatalf("fuzzy candidate should be disabled, got %#v", state.Candidates)
		}
	}
}

func TestDoublePinyinXiaoheCandidates(t *testing.T) {
	config := DefaultConfig()
	config.DoublePinyin = true
	e := New(config)

	tests := []struct {
		input string
		want  string
	}{
		{input: "nihk", want: "你好"},
		{input: "vswf", want: "中文"},
		{input: "uuru", want: "输入法"},
	}

	for _, tt := range tests {
		state := e.Preview(tt.input)
		if len(state.Candidates) == 0 || state.Candidates[0].Text != tt.want {
			t.Fatalf("expected double pinyin %q -> %q, got %#v", tt.input, tt.want, state.Candidates)
		}
	}
}

func TestDoublePinyinExplicitXiaoheSchemeCandidates(t *testing.T) {
	config := DefaultConfig()
	config.DoublePinyin = true
	config.DoublePinyinScheme = "xiaohe"
	e := New(config)

	state := e.Preview("nihk")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "你好" {
		t.Fatalf("expected explicit xiaohe double pinyin candidate, got %#v", state.Candidates)
	}
}

func TestDoublePinyinMicrosoftCandidates(t *testing.T) {
	config := DefaultConfig()
	config.DoublePinyin = true
	config.DoublePinyinScheme = "microsoft"
	e := New(config)

	tests := []struct {
		input string
		want  string
	}{
		{input: "nill", want: "你来"},
		{input: "vswf", want: "中文"},
		{input: "uuru", want: "输入法"},
	}

	e.AddEntries([]Entry{{Reading: "nilai", Text: "你来", Weight: 18000}})
	for _, tt := range tests {
		state := e.Preview(tt.input)
		if len(state.Candidates) == 0 || state.Candidates[0].Text != tt.want {
			t.Fatalf("expected microsoft double pinyin %q -> %q, got %#v", tt.input, tt.want, state.Candidates)
		}
	}
}

func TestDoublePinyinMicrosoftZeroInitialAndSemicolonFinal(t *testing.T) {
	config := DefaultConfig()
	config.DoublePinyin = true
	config.DoublePinyinScheme = "microsoft"
	e := New(config)
	e.AddEntries([]Entry{
		{Reading: "ai", Text: "哎", Weight: 12000},
		{Reading: "ming", Text: "明", Weight: 12000},
	})

	state := e.Preview("ol")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "哎" {
		t.Fatalf("expected microsoft zero-initial ol -> ai candidate, got %#v", state.Candidates)
	}
	state = e.Preview("m;")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "明" {
		t.Fatalf("expected microsoft semicolon final m; -> ming candidate, got %#v", state.Candidates)
	}
}

func TestDoublePinyinCanBeDisabled(t *testing.T) {
	config := DefaultConfig()
	config.DoublePinyin = false
	e := New(config)

	state := e.Preview("nihk")
	for _, candidate := range state.Candidates {
		if candidate.Text == "你好" {
			t.Fatalf("double pinyin should be disabled, got %#v", state.Candidates)
		}
	}
}

func TestDoublePinyinKeepsFullPinyinFallback(t *testing.T) {
	config := DefaultConfig()
	config.DoublePinyin = true
	e := New(config)

	state := e.Preview("nihao")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "你好" {
		t.Fatalf("expected full pinyin fallback while double pinyin is enabled, got %#v", state.Candidates)
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

func TestLoadDictionaryAcceptsGzipJSON(t *testing.T) {
	e := New(DefaultConfig())
	data := gzipBytes(t, []byte(`{
		"language": "zh-CN",
		"version": "gzip",
		"entries": [{ "reading": "yasuobao", "text": "压缩包", "weight": 9000 }]
	}`))
	_, err := e.LoadDictionary(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	state := e.Preview("yasuobao")
	if len(state.Candidates) == 0 || state.Candidates[0].Text != "压缩包" {
		t.Fatalf("expected gzip dictionary candidate, got %#v", state.Candidates)
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
				if candidate.Comment == "" {
					t.Fatalf("expected builtin candidate comment for %q, got %#v", tt.text, candidate)
				}
			}
		}
		if !found {
			t.Fatalf("expected builtin %s candidate %q for %s, got %#v", tt.kind, tt.text, tt.reading, state.Candidates)
		}
	}
}

func TestCatalogEntriesFiltersSpecialResources(t *testing.T) {
	e := New(DefaultConfig())
	e.AddEntries([]Entry{
		{Reading: "zan", Text: "普通赞", Weight: 90000},
		{Reading: "zan", Text: "👏🏻", Kind: "emoji", Source: "rime-emoji", Comment: "鼓掌", Weight: 7000},
		{Reading: "fs", Text: "℃℃", Kind: "symbol", Source: "rime-symbols", Comment: "温度", Weight: 7100},
	})

	emojis := e.CatalogEntries(CatalogRequest{Kind: "emoji", Query: "zan", Limit: 10})
	if emojis.Kind != "emoji" || emojis.Count == 0 {
		t.Fatalf("emoji catalog = %#v", emojis)
	}
	for _, entry := range emojis.Entries {
		if entry.Kind != "emoji" {
			t.Fatalf("emoji catalog leaked non-emoji entry: %#v", emojis.Entries)
		}
		if entry.Text == "普通赞" {
			t.Fatalf("catalog should not include ordinary words: %#v", emojis.Entries)
		}
	}

	symbols := e.CatalogEntries(CatalogRequest{Kind: "symbol", Query: "/fs", Limit: 10})
	if symbols.Kind != "symbol" || symbols.Count == 0 || symbols.Entries[0].Reading != "fs" {
		t.Fatalf("symbol catalog slash query = %#v", symbols)
	}
}

func TestBuiltinAgentCandidateMetadata(t *testing.T) {
	e := New(DefaultConfig())

	tests := []struct {
		reading string
		text    string
	}{
		{reading: "rewrite", text: "/rewrite "},
		{reading: "runse", text: "/rewrite "},
		{reading: "translate", text: "/translate "},
		{reading: "fanyi", text: "/translate "},
		{reading: "ask", text: "/ask "},
		{reading: "agent", text: "/ask "},
	}

	for _, tt := range tests {
		state := e.Preview(tt.reading)
		if len(state.Candidates) == 0 || state.Candidates[0].Text != tt.text {
			t.Fatalf("expected top agent candidate %q for %s, got %#v", tt.text, tt.reading, state.Candidates)
		}
		if state.Candidates[0].Kind != "agent" || state.Candidates[0].Source != "builtin-agent" {
			t.Fatalf("expected agent metadata for %s, got %#v", tt.reading, state.Candidates[0])
		}
	}
}

func TestBuiltinAgentCandidateDoesNotOverrideCommonWord(t *testing.T) {
	e := New(DefaultConfig())
	state := e.Preview("ai")
	if len(state.Candidates) < 2 {
		t.Fatalf("expected word and agent candidates for ai, got %#v", state.Candidates)
	}
	if state.Candidates[0].Text != "爱" {
		t.Fatalf("agent candidate should not outrank common word 爱, got %#v", state.Candidates)
	}
	foundAgent := false
	for _, candidate := range state.Candidates {
		if candidate.Kind == "agent" && candidate.Text == "/ask " {
			foundAgent = true
		}
	}
	if !foundAgent {
		t.Fatalf("expected ai to expose /ask agent candidate, got %#v", state.Candidates)
	}
}

func TestDynamicDateTimeCandidates(t *testing.T) {
	oldNow := nowFunc
	nowFunc = func() time.Time {
		return time.Date(2026, time.July, 5, 14, 3, 9, 0, time.UTC)
	}
	defer func() { nowFunc = oldNow }()

	e := New(DefaultConfig())
	tests := []struct {
		input string
		want  string
	}{
		{input: "rq", want: "2026-07-05"},
		{input: "date", want: "2026年7月5日"},
		{input: "sj", want: "14:03"},
		{input: "dt", want: "2026-07-05 14:03"},
		{input: "ts", want: "1783260189"},
	}

	for _, tt := range tests {
		state := e.Preview(tt.input)
		found := false
		for _, candidate := range state.Candidates {
			if candidate.Text == tt.want {
				found = true
				if candidate.Kind != "dynamic" || candidate.Source != "builtin-datetime" {
					t.Fatalf("expected dynamic metadata for %q, got %#v", tt.input, candidate)
				}
			}
		}
		if !found {
			t.Fatalf("expected dynamic candidate %q for %q, got %#v", tt.want, tt.input, state.Candidates)
		}
	}
}

func TestDynamicWeekCandidates(t *testing.T) {
	oldNow := nowFunc
	nowFunc = func() time.Time {
		return time.Date(2026, time.July, 5, 14, 3, 9, 0, time.UTC)
	}
	defer func() { nowFunc = oldNow }()

	e := New(DefaultConfig())
	state := e.Preview("xq")
	if len(state.Candidates) < 2 || state.Candidates[0].Text != "星期日" || state.Candidates[1].Text != "周日" {
		t.Fatalf("expected Sunday week candidates, got %#v", state.Candidates)
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
