package main

import (
	"bytes"
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
	if got := rec.Body.String(); !strings.Contains(got, "1\t👍\tzan\t6400\temoji\tbuiltin-symbols") {
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

func serverURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
