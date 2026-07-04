package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

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

func TestParseCandidateActionArgsViewDefaults(t *testing.T) {
	input, req, err := parseCandidateActionArgs([]string{"nihao"})
	if err != nil {
		t.Fatal(err)
	}
	if input != "nihao" {
		t.Fatalf("input = %q, want nihao", input)
	}
	if req.Action != "view" {
		t.Fatalf("action = %q, want view", req.Action)
	}
}

func TestParseCandidateActionArgsSelectDisplayIndex(t *testing.T) {
	input, req, err := parseCandidateActionArgs([]string{"ceshi", "select", "--start", "7", "--display-index", "2", "--limit=5"})
	if err != nil {
		t.Fatal(err)
	}
	if input != "ceshi" {
		t.Fatalf("input = %q, want ceshi", input)
	}
	if req.Action != "select" || req.Start != 7 || req.DisplayIndex != 2 || req.Limit != 5 {
		t.Fatalf("request = %#v", req)
	}
}

func TestParseCandidateActionArgsRejectsBadOption(t *testing.T) {
	_, _, err := parseCandidateActionArgs([]string{"nihao", "--wat", "1"})
	if err == nil {
		t.Fatal("expected unknown option error")
	}
}

func TestCandidateActionCallsPreviewThenActionEndpoint(t *testing.T) {
	var previewCalled bool
	var actionCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/engine/preview":
			previewCalled = true
			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode preview: %v", err)
			}
			if req["input"] != "nihao" {
				t.Fatalf("preview input = %q", req["input"])
			}
			_ = json.NewEncoder(w).Encode(engineState{Buffer: "nihao"})
		case "/ime/candidate-action":
			actionCalled = true
			var req candidateActionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode action: %v", err)
			}
			if req.Action != "next-page" || req.Limit != 2 {
				t.Fatalf("candidate action request = %#v", req)
			}
			_ = json.NewEncoder(w).Encode(candidateActionResponse{
				OK:     true,
				Action: req.Action,
				Start:  2,
				Limit:  2,
				Total:  4,
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	previousBase := apiBase
	apiBase = server.URL
	defer func() { apiBase = previousBase }()

	if err := candidateAction(server.Client(), []string{"nihao", "next-page", "--limit", "2"}); err != nil {
		t.Fatal(err)
	}
	if !previewCalled || !actionCalled {
		t.Fatalf("previewCalled=%v actionCalled=%v", previewCalled, actionCalled)
	}
}

func TestReadWordbookFileAcceptsWrappedScores(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wordbook.json")
	if err := os.WriteFile(path, []byte(`{"userScores":{"nihao|你好":25}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readWordbookFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["nihao|你好"] != 25 {
		t.Fatalf("scores = %#v", got)
	}
}

func TestReadWordbookFileAcceptsRawScores(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wordbook.json")
	if err := os.WriteFile(path, []byte(`{"ceshi|测试":50}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readWordbookFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["ceshi|测试"] != 50 {
		t.Fatalf("scores = %#v", got)
	}
}
