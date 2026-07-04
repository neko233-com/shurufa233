package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/shurufa233/core/engine"
)

func TestIsAllowedLocalOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "localhost vite", origin: "http://localhost:5173", want: true},
		{name: "loopback vite", origin: "http://127.0.0.1:5173", want: true},
		{name: "ipv6 loopback vite", origin: "http://[::1]:5173", want: true},
		{name: "wails app", origin: "wails://wails", want: true},
		{name: "empty", origin: "", want: false},
		{name: "wrong port", origin: "http://127.0.0.1:5174", want: false},
		{name: "remote host", origin: "https://example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAllowedLocalOrigin(tt.origin); got != tt.want {
				t.Fatalf("isAllowedLocalOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestWithCORSPreflightAllowsLoopbackVite(t *testing.T) {
	state := &AppState{}
	req := httptest.NewRequest(http.MethodOptions, "/engine/preview", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	rec := httptest.NewRecorder()

	state.withCORS(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("preflight should not call wrapped handler")
	})(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "DELETE") {
		t.Fatalf("Access-Control-Allow-Methods = %q, want DELETE", got)
	}
}

func TestPutConfigNormalizesCandidatePool(t *testing.T) {
	state := &AppState{
		config:   engine.DefaultConfig(),
		engine:   engine.New(engine.DefaultConfig()),
		sessions: map[string]*engine.Engine{},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
	}
	state.sessions["default"] = state.engine

	next := engine.DefaultConfig()
	next.MaxCandidates = 7
	next.CandidatePageSize = 99
	body, err := json.Marshal(next)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	state.putConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if state.config.MaxCandidates != engine.DefaultConfig().MaxCandidates {
		t.Fatalf("maxCandidates = %d, want %d", state.config.MaxCandidates, engine.DefaultConfig().MaxCandidates)
	}
	if state.config.CandidatePageSize != 9 {
		t.Fatalf("candidatePageSize = %d, want 9", state.config.CandidatePageSize)
	}
}

func TestImeSkinIncludesCandidatePageSize(t *testing.T) {
	config := engine.DefaultConfig()
	config.CandidatePageSize = 5
	config.CandidateLayout = "vertical"
	config.ShowCandidateComments = false
	state := &AppState{config: config}
	req := httptest.NewRequest(http.MethodGet, "/ime/skin", nil)
	rec := httptest.NewRecorder()

	state.imeSkin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	parts := strings.Split(rec.Body.String(), "|")
	if len(parts) != 12 {
		t.Fatalf("skin payload parts = %#v", parts)
	}
	if parts[9] != "5" {
		t.Fatalf("candidate page size payload = %q, want 5", parts[9])
	}
	if parts[10] != "vertical" {
		t.Fatalf("candidate layout payload = %q, want vertical", parts[10])
	}
	if parts[11] != "false" {
		t.Fatalf("candidate comment visibility payload = %q, want false", parts[11])
	}
}

func TestNormalizeConfigKeepsCandidateLayout(t *testing.T) {
	next := engine.DefaultConfig()
	next.CandidateLayout = "rime"

	got := normalizeConfig(next)

	if got.CandidateLayout != "vertical" {
		t.Fatalf("candidateLayout = %q, want vertical", got.CandidateLayout)
	}
}

func TestNormalizeConfigKeepsCandidateSkinReadable(t *testing.T) {
	next := engine.DefaultConfig()
	next.Skin.Surface = "#ffffff"
	next.Skin.Text = "#ffffff"
	next.Skin.MutedText = "#ffffff"
	next.Skin.Accent = "#2563eb"
	next.Skin.HighlightText = "#2563eb"

	got := normalizeConfig(next)

	if got.Skin.Text == "#ffffff" {
		t.Fatalf("text color was not corrected: %#v", got.Skin)
	}
	if contrastRatio(got.Skin.Text, got.Skin.Surface) < 4.5 {
		t.Fatalf("text contrast = %.2f, want >= 4.5", contrastRatio(got.Skin.Text, got.Skin.Surface))
	}
	if contrastRatio(got.Skin.MutedText, got.Skin.Surface) < 3.0 {
		t.Fatalf("muted contrast = %.2f, want >= 3.0", contrastRatio(got.Skin.MutedText, got.Skin.Surface))
	}
	if contrastRatio(got.Skin.HighlightText, got.Skin.Accent) < 4.5 {
		t.Fatalf("highlight contrast = %.2f, want >= 4.5", contrastRatio(got.Skin.HighlightText, got.Skin.Accent))
	}
}

func TestNormalizeConfigRejectsInvalidSkinColors(t *testing.T) {
	next := engine.DefaultConfig()
	next.Skin.Surface = "white"
	next.Skin.Accent = "#12"
	next.Skin.Text = "#gggggg"
	next.Skin.MutedText = "transparent"
	next.Skin.Border = "none"
	next.Skin.HighlightText = "currentColor"

	got := normalizeConfig(next)
	defaults := engine.DefaultConfig()

	if got.Skin.Surface != defaults.Skin.Surface {
		t.Fatalf("surface = %q, want %q", got.Skin.Surface, defaults.Skin.Surface)
	}
	if got.Skin.Accent != defaults.Skin.Accent {
		t.Fatalf("accent = %q, want %q", got.Skin.Accent, defaults.Skin.Accent)
	}
	if got.Skin.Border != defaults.Skin.Border {
		t.Fatalf("border = %q, want %q", got.Skin.Border, defaults.Skin.Border)
	}
	if !isHexColor(got.Skin.Text) || !isHexColor(got.Skin.MutedText) || !isHexColor(got.Skin.HighlightText) {
		t.Fatalf("normalized skin contains invalid text colors: %#v", got.Skin)
	}
}

func TestNormalizeConfigKeepsHalfWidthPunctuation(t *testing.T) {
	next := engine.DefaultConfig()
	next.Punctuation = " HALF "

	got := normalizeConfig(next)

	if got.Punctuation != "half" {
		t.Fatalf("punctuation = %q, want half", got.Punctuation)
	}
}

func TestNormalizeConfigDefaultsInvalidPunctuation(t *testing.T) {
	next := engine.DefaultConfig()
	next.Punctuation = "broken"

	got := normalizeConfig(next)

	if got.Punctuation != engine.DefaultConfig().Punctuation {
		t.Fatalf("punctuation = %q, want %q", got.Punctuation, engine.DefaultConfig().Punctuation)
	}
}

func TestNormalizeConfigKeepsMicrosoftDoublePinyinScheme(t *testing.T) {
	next := engine.DefaultConfig()
	next.DoublePinyin = true
	next.DoublePinyinScheme = " MS "

	got := normalizeConfig(next)

	if !got.DoublePinyin || got.DoublePinyinScheme != "microsoft" {
		t.Fatalf("double pinyin config = enabled:%v scheme:%q, want microsoft", got.DoublePinyin, got.DoublePinyinScheme)
	}
}

func TestNormalizeConfigDefaultsInvalidDoublePinyinScheme(t *testing.T) {
	next := engine.DefaultConfig()
	next.DoublePinyinScheme = "broken"

	got := normalizeConfig(next)

	if got.DoublePinyinScheme != engine.DefaultConfig().DoublePinyinScheme {
		t.Fatalf("doublePinyinScheme = %q, want %q", got.DoublePinyinScheme, engine.DefaultConfig().DoublePinyinScheme)
	}
}

func TestImeCandidatesReturnsMetadataAndPagedRows(t *testing.T) {
	config := engine.DefaultConfig()
	config.MaxCandidates = 42
	session := engine.New(config)
	session.AddEntries([]engine.Entry{
		{Reading: "page", Text: "候0", Weight: 100},
		{Reading: "page", Text: "候1", Weight: 99},
		{Reading: "page", Text: "候2", Weight: 98},
		{Reading: "page", Text: "候3", Weight: 97},
		{Reading: "page", Text: "候4", Weight: 96},
		{Reading: "page", Text: "候5", Weight: 95},
		{Reading: "page", Text: "候6", Weight: 94},
		{Reading: "page", Text: "候7", Weight: 93},
		{Reading: "page", Text: "候8", Weight: 92},
	})
	state := &AppState{
		config:   config,
		engine:   session,
		sessions: map[string]*engine.Engine{"default": session},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
	}

	session.Preview("zan")
	req := httptest.NewRequest(http.MethodGet, "/ime/candidates?start=0&limit=7", nil)
	rec := httptest.NewRecorder()
	state.imeCandidates(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !strings.Contains(got, "1\t👍\tzan\t6400\temoji\tbuiltin-symbols\t赞") {
		t.Fatalf("expected emoji metadata row, got %q", got)
	}

	session.Preview("page")
	req = httptest.NewRequest(http.MethodGet, "/ime/candidates?start=7&limit=2", nil)
	rec = httptest.NewRecorder()
	state.imeCandidates(rec, req)
	rows := strings.Split(strings.TrimRight(rec.Body.String(), "\n"), "\n")
	if len(rows) != 2 {
		t.Fatalf("rows = %#v, want 2 rows", rows)
	}
	if !strings.HasPrefix(rows[0], "1\t候7\tpage\t93\t\t") {
		t.Fatalf("first paged row = %q", rows[0])
	}
	if !strings.HasPrefix(rows[1], "2\t候8\tpage\t92\t\t") {
		t.Fatalf("second paged row = %q", rows[1])
	}
}

func TestImeSelectCharCommitsCandidateCharacter(t *testing.T) {
	config := engine.DefaultConfig()
	session := engine.New(config)
	session.AddEntries([]engine.Entry{{Reading: "houxuan", Text: "候选", Weight: 20000}})
	state := &AppState{
		config:   config,
		engine:   session,
		sessions: map[string]*engine.Engine{"default": session},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
	}

	session.Preview("houxuan")
	req := httptest.NewRequest(http.MethodPost, "/ime/select-char?index=0&side=last", nil)
	rec := httptest.NewRecorder()
	state.imeSelectChar(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "选" {
		t.Fatalf("committed char = %q, want 选", got)
	}
	if state := session.State(); state.Buffer != "" || len(state.Candidates) != 0 {
		t.Fatalf("selection should clear composition, got %#v", state)
	}
}

func TestImeCandidateActionPagesAndCommitsByDisplayIndex(t *testing.T) {
	config := engine.DefaultConfig()
	config.CandidatePageSize = 2
	session := engine.New(config)
	session.AddEntries([]engine.Entry{
		{Reading: "ceshi", Text: "测试一", Weight: 10010},
		{Reading: "ceshi", Text: "测试二", Weight: 10009},
		{Reading: "ceshi", Text: "测试三", Weight: 10008},
		{Reading: "ceshi", Text: "测试四", Weight: 10007},
	})
	state := &AppState{
		config:   config,
		engine:   session,
		sessions: map[string]*engine.Engine{"default": session},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
	}
	session.Preview("ceshi")

	req := httptest.NewRequest(http.MethodPost, "/ime/candidate-action", strings.NewReader(`{"action":"next-page","start":0}`))
	rec := httptest.NewRecorder()
	state.imeCandidateAction(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var page candidateActionResult
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if !page.OK || page.Start != 2 || page.Limit != 2 || len(page.Candidates.Items) != 2 {
		t.Fatalf("candidate action page = %#v", page)
	}
	if page.Candidates.Items[0].Text != "测试三" {
		t.Fatalf("first next-page candidate = %#v", page.Candidates.Items[0])
	}

	req = httptest.NewRequest(http.MethodPost, "/ime/candidate-action", strings.NewReader(`{"action":"select","start":2,"displayIndex":2}`))
	rec = httptest.NewRecorder()
	state.imeCandidateAction(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var commit candidateActionResult
	if err := json.Unmarshal(rec.Body.Bytes(), &commit); err != nil {
		t.Fatal(err)
	}
	if commit.Committed != "测试四" || commit.State.Buffer != "" {
		t.Fatalf("candidate action commit = %#v", commit)
	}
}

func TestImeCandidateActionCommitsFirstCharacter(t *testing.T) {
	config := engine.DefaultConfig()
	session := engine.New(config)
	session.AddEntries([]engine.Entry{{Reading: "houxuan", Text: "候选", Weight: 20000}})
	state := &AppState{
		config:   config,
		engine:   session,
		sessions: map[string]*engine.Engine{"default": session},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
	}
	session.Preview("houxuan")

	req := httptest.NewRequest(http.MethodPost, "/ime/candidate-action?action=first-char&index=0", nil)
	rec := httptest.NewRecorder()
	state.imeCandidateAction(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var result candidateActionResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Committed != "候" {
		t.Fatalf("candidate action first-char = %#v", result)
	}
}

func TestImeModeCanToggleSessionMode(t *testing.T) {
	config := engine.DefaultConfig()
	session := engine.New(config)
	state := &AppState{
		config:   config,
		engine:   session,
		sessions: map[string]*engine.Engine{"default": session},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
	}

	session.Preview("nihao")
	req := httptest.NewRequest(http.MethodPost, "/ime/mode", strings.NewReader(`{"toggle":true}`))
	rec := httptest.NewRecorder()
	state.imeMode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got engine.State
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Mode != "en" {
		t.Fatalf("mode = %q, want en", got.Mode)
	}
	if got.Buffer != "" || len(got.Candidates) != 0 {
		t.Fatalf("toggle should clear composition, got %#v", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/ime/mode", nil)
	rec = httptest.NewRecorder()
	state.imeMode(rec, req)
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Mode != "en" {
		t.Fatalf("GET mode = %q, want en", got.Mode)
	}
	if state.config.Mode != "zh" {
		t.Fatalf("session mode should not rewrite saved default config, got %q", state.config.Mode)
	}
}

func TestImeModeAcceptsQueryParametersForNativeFallback(t *testing.T) {
	config := engine.DefaultConfig()
	session := engine.New(config)
	state := &AppState{
		config:   config,
		engine:   session,
		sessions: map[string]*engine.Engine{"default": session},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
	}

	req := httptest.NewRequest(http.MethodPost, "/ime/mode?toggle=1", nil)
	rec := httptest.NewRecorder()
	state.imeMode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("toggle status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got engine.State
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Mode != "en" {
		t.Fatalf("mode after query toggle = %q, want en", got.Mode)
	}

	req = httptest.NewRequest(http.MethodPost, "/ime/mode?mode=zh", nil)
	rec = httptest.NewRecorder()
	state.imeMode(rec, req)
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Mode != "zh" {
		t.Fatalf("mode after query set = %q, want zh", got.Mode)
	}
}

func TestWordbookPutAndDeleteManageUserScores(t *testing.T) {
	config := engine.DefaultConfig()
	session := engine.New(config)
	session.AddEntries([]engine.Entry{
		{Reading: "ceshi", Text: "测试", Weight: 100},
		{Reading: "ceshi", Text: "侧室", Weight: 90},
	})
	state := &AppState{
		config:   config,
		engine:   session,
		sessions: map[string]*engine.Engine{"default": session},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
	}

	body := strings.NewReader(`{"userScores":{"ceshi|侧室":1000}}`)
	req := httptest.NewRequest(http.MethodPut, "/wordbook", body)
	rec := httptest.NewRecorder()
	state.wordbook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := session.Preview("ceshi"); len(got.Candidates) == 0 || got.Candidates[0].Text != "侧室" {
		t.Fatalf("expected imported user word to rerank candidates, got %#v", got.Candidates)
	}

	req = httptest.NewRequest(http.MethodDelete, "/wordbook?key=ceshi%7C%E4%BE%A7%E5%AE%A4", nil)
	rec = httptest.NewRecorder()
	state.wordbook(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := session.Preview("ceshi"); len(got.Candidates) == 0 || got.Candidates[0].Text != "测试" {
		t.Fatalf("expected deleted user word to restore static ranking, got %#v", got.Candidates)
	}

	var stored userScoreStore
	data, err := os.ReadFile(state.userScoresPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if len(stored.Scores) != 0 {
		t.Fatalf("stored scores = %#v, want empty", stored.Scores)
	}
}

func TestPhrasesPutAndDeleteManageUserPhrases(t *testing.T) {
	config := engine.DefaultConfig()
	session := engine.New(config)
	session.AddEntries([]engine.Entry{{Reading: "msd", Text: "默认短语", Weight: 20000}})
	state := &AppState{
		config:   config,
		engine:   session,
		sessions: map[string]*engine.Engine{"default": session},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
	}

	body := strings.NewReader(`{"entries":[{"reading":"msd","text":"马上到！"}],"merge":true}`)
	req := httptest.NewRequest(http.MethodPut, "/phrases", body)
	rec := httptest.NewRecorder()
	state.phrases(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := session.Preview("msd"); len(got.Candidates) == 0 || got.Candidates[0].Text != "马上到！" {
		t.Fatalf("expected user phrase to rank first, got %#v", got.Candidates)
	}

	var stored userPhraseStore
	data, err := os.ReadFile(state.userPhrasesPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if len(stored.Entries) != 1 || stored.Entries[0].Source != engine.UserPhraseSource {
		t.Fatalf("stored phrases = %#v", stored.Entries)
	}

	req = httptest.NewRequest(http.MethodDelete, "/phrases?key=msd%7C%E9%A9%AC%E4%B8%8A%E5%88%B0%EF%BC%81", nil)
	rec = httptest.NewRecorder()
	state.phrases(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := session.Preview("msd"); len(got.Candidates) == 0 || got.Candidates[0].Text != "默认短语" {
		t.Fatalf("expected deleted phrase to restore default candidate, got %#v", got.Candidates)
	}
}

func TestAgentComposeReturnsStructuredCandidates(t *testing.T) {
	got := composeAgentResponse("/rewrite", "这段话有点啰嗦")
	if got.Context != "这段话有点啰嗦" {
		t.Fatalf("context = %q", got.Context)
	}
	if len(got.Items) != 2 {
		t.Fatalf("items = %#v, want 2 structured candidates", got.Items)
	}
	if len(got.Candidates) != len(got.Items) {
		t.Fatalf("legacy candidates should mirror items: %#v vs %#v", got.Candidates, got.Items)
	}
	if got.Items[0].Intent != "rewrite" || got.Items[0].Action != "agent.rewrite.polish" {
		t.Fatalf("unexpected first item metadata: %#v", got.Items[0])
	}
	if got.Items[0].Source != "builtin-agent" || !strings.Contains(got.Items[0].Text, "这段话有点啰嗦") {
		t.Fatalf("unexpected first item payload: %#v", got.Items[0])
	}
}

func TestAgentComposeDefaultUsesContextSignal(t *testing.T) {
	got := composeAgentResponse("总结一下", "上文：性能测试失败")
	if len(got.Items) != 1 {
		t.Fatalf("items = %#v, want 1", got.Items)
	}
	if !strings.Contains(got.Items[0].Text, "结合当前上下文") {
		t.Fatalf("default candidate should mention context use: %#v", got.Items[0])
	}
	if got.Items[0].Context != "上文：性能测试失败" {
		t.Fatalf("item context = %q", got.Items[0].Context)
	}
}

func TestNewSessionLoadsLocalDictionaries(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "shurufa233", "config.json")
	dictionaryDir := filepath.Join(filepath.Dir(configPath), "dictionaries")
	if err := os.MkdirAll(dictionaryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dictionary := `{
		"language": "zh-CN",
		"version": "test",
		"entries": [
			{ "reading": "shi", "text": "是", "weight": 16000 },
			{ "reading": "shi", "text": "时", "weight": 11800 },
			{ "reading": "shi", "text": "事", "weight": 10800 },
			{ "reading": "shi", "text": "市", "weight": 9800 },
			{ "reading": "shi", "text": "使", "weight": 9400 },
			{ "reading": "shi", "text": "试", "weight": 9300 },
			{ "reading": "shi", "text": "十", "weight": 9000 },
			{ "reading": "shi", "text": "识", "weight": 8600 }
		]
	}`
	if err := os.WriteFile(filepath.Join(dictionaryDir, "zh-CN.test.json"), []byte(dictionary), 0o644); err != nil {
		t.Fatal(err)
	}

	state := &AppState{
		sessions: map[string]*engine.Engine{},
		path:     configPath,
	}
	if err := state.load(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ime/key?session=tsf-test", nil)
	session := state.sessionForRequest(req)
	got := session.Preview("shi")
	if len(got.Candidates) < 8 {
		t.Fatalf("expected local dictionary candidates in new session, got %#v", got.Candidates)
	}
}

func TestDictionaryURLsPreferConfiguredMirrors(t *testing.T) {
	config := engine.DefaultConfig()
	config.Update.MirrorBaseURLs = []string{
		" https://cdn.example.com/shurufa233/ ",
		"https://mirror.example.cn/releases",
	}

	got := dictionaryURLs(config, "https://github.com/neko233-com/shurufa233/releases/latest/download/zh-CN.2026.07.04.json")
	want := []string{
		"https://mirror.example.cn/releases/zh-CN.2026.07.04.json",
		"https://cdn.example.com/shurufa233/zh-CN.2026.07.04.json",
		"https://github.com/neko233-com/shurufa233/releases/latest/download/zh-CN.2026.07.04.json",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("dictionaryURLs = %#v, want %#v", got, want)
	}
}

func TestDictionaryAutoUpdateAppliesConfiguredRelease(t *testing.T) {
	dictionary := `{
		"language": "zh-CN",
		"version": "remote-test",
		"entries": [
			{ "reading": "ceshi", "text": "测试热更", "weight": 20000 }
		]
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dictionary-manifest.json":
			_, _ = fmt.Fprintf(w, `{
				"version": "remote-test",
				"channel": "stable",
				"generatedAt": "2026-07-04T00:00:00Z",
				"dictionaries": [
					{ "language": "zh-CN", "version": "remote-test", "url": "%s/zh-CN.remote-test.json" }
				]
			}`, serverURL(r))
		case "/zh-CN.remote-test.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(dictionary))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := engine.DefaultConfig()
	config.Update.ManifestURLs = []string{server.URL + "/dictionary-manifest.json"}
	config.Update.AutoCheck = true
	config.Update.AutoApply = true
	config.Update.InstalledVersion = "builtin"

	state := &AppState{
		config:   config,
		engine:   engine.New(config),
		sessions: map[string]*engine.Engine{},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
		client:   server.Client(),
	}
	state.sessions["default"] = state.engine

	state.runDictionaryAutoUpdateOnce()

	got := state.engine.Preview("ceshi")
	if len(got.Candidates) == 0 || got.Candidates[0].Text != "测试热更" {
		t.Fatalf("auto-updated candidates = %#v", got.Candidates)
	}
	if state.config.Update.InstalledVersion != "remote-test" {
		t.Fatalf("installed version = %q, want remote-test", state.config.Update.InstalledVersion)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(state.path), "dictionaries", "zh-CN.remote-test.json")); err != nil {
		t.Fatalf("expected downloaded dictionary file: %v", err)
	}
}

func TestDictionaryAutoUpdateAppliesGzipRelease(t *testing.T) {
	dictionary := []byte(`{
		"language": "zh-CN",
		"version": "remote-gzip",
		"entries": [
			{ "reading": "yasuo", "text": "压缩热更", "weight": 20000 }
		]
	}`)
	compressed := gzipBytes(t, dictionary)
	rawSHA := sha256Hex(compressed)
	contentSHA := sha256Hex(dictionary)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dictionary-manifest.json":
			_, _ = fmt.Fprintf(w, `{
				"version": "remote-gzip",
				"channel": "stable",
				"generatedAt": "2026-07-04T00:00:00Z",
				"dictionaries": [
					{
						"language": "zh-CN",
						"version": "remote-gzip",
						"url": "%s/zh-CN.remote-gzip.json.gz",
						"compression": "gzip",
						"sha256": "%s",
						"contentSha256": "%s"
					}
				]
			}`, serverURL(r), rawSHA, contentSHA)
		case "/zh-CN.remote-gzip.json.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(compressed)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := engine.DefaultConfig()
	config.Update.ManifestURLs = []string{server.URL + "/dictionary-manifest.json"}
	config.Update.AutoCheck = true
	config.Update.AutoApply = true
	config.Update.InstalledVersion = "builtin"

	state := &AppState{
		config:   config,
		engine:   engine.New(config),
		sessions: map[string]*engine.Engine{},
		path:     filepath.Join(t.TempDir(), "shurufa233", "config.json"),
		client:   server.Client(),
	}
	state.sessions["default"] = state.engine

	state.runDictionaryAutoUpdateOnce()

	got := state.engine.Preview("yasuo")
	if len(got.Candidates) == 0 || got.Candidates[0].Text != "压缩热更" {
		t.Fatalf("auto-updated gzip candidates = %#v", got.Candidates)
	}
	filePath := filepath.Join(filepath.Dir(state.path), "dictionaries", "zh-CN.remote-gzip.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("expected decompressed dictionary file: %v", err)
	}
	if !bytes.Contains(data, []byte("压缩热更")) {
		t.Fatalf("expected decompressed JSON at %s, got %q", filePath, data)
	}
}

func serverURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
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

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}
