package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
