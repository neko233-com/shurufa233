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

func TestParseCandidateActionArgsAssociateContext(t *testing.T) {
	input, req, err := parseCandidateActionArgs([]string{"associate", "--context", "你好", "--limit", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if input != "" || req.Action != "associate" || req.Context != "你好" || req.Limit != 2 {
		t.Fatalf("associate request = input:%q req:%#v", input, req)
	}
}

func TestParseCandidateActionArgsRejectsBadOption(t *testing.T) {
	_, _, err := parseCandidateActionArgs([]string{"nihao", "--wat", "1"})
	if err == nil {
		t.Fatal("expected unknown option error")
	}
}

func TestAssociateCallsEndpoint(t *testing.T) {
	var associateCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/engine/associate" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		associateCalled = true
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode associate: %v", err)
		}
		if req["context"] != "你好" {
			t.Fatalf("associate context = %#v", req["context"])
		}
		_ = json.NewEncoder(w).Encode(engineState{
			Candidates: []candidate{{Text: "世界", Reading: "shijie", Kind: "association", Comment: "联想"}},
		})
	}))
	defer server.Close()
	previousBase := apiBase
	apiBase = server.URL
	defer func() { apiBase = previousBase }()

	if err := associate(server.Client(), []string{"你好"}); err != nil {
		t.Fatal(err)
	}
	if !associateCalled {
		t.Fatal("associate endpoint was not called")
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

func TestCandidateActionAssociateSkipsPreview(t *testing.T) {
	var actionCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/engine/preview" {
			t.Fatal("associate action should not call preview")
		}
		if r.URL.Path != "/ime/candidate-action" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		actionCalled = true
		var req candidateActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode action: %v", err)
		}
		if req.Action != "associate" || req.Context != "微信" {
			t.Fatalf("candidate association request = %#v", req)
		}
		_ = json.NewEncoder(w).Encode(candidateActionResponse{
			OK:     true,
			Action: req.Action,
			Total:  1,
		})
	}))
	defer server.Close()
	previousBase := apiBase
	apiBase = server.URL
	defer func() { apiBase = previousBase }()

	if err := candidateAction(server.Client(), []string{"associate", "--context", "微信"}); err != nil {
		t.Fatal(err)
	}
	if !actionCalled {
		t.Fatal("candidate action endpoint was not called")
	}
}

func TestCandidateActionAcceptsForgetAction(t *testing.T) {
	input, req, err := parseCandidateActionArgs([]string{"ceshi", "forget", "--index", "0"})
	if err != nil {
		t.Fatal(err)
	}
	if input != "ceshi" || req.Action != "forget" || req.Index != 0 {
		t.Fatalf("forget request = input:%q req:%#v", input, req)
	}
}

func TestPreviewSendsSlashPrefixInput(t *testing.T) {
	var previewCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/engine/preview" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		previewCalled = true
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode preview: %v", err)
		}
		if req["input"] != "/fs" {
			t.Fatalf("preview input = %q", req["input"])
		}
		_ = json.NewEncoder(w).Encode(engineState{
			Buffer: "/fs",
			Candidates: []candidate{{
				Text:    "℃",
				Reading: "fs",
				Kind:    "symbol",
				Source:  "builtin-symbols",
				Weight:  6200,
			}},
		})
	}))
	defer server.Close()
	previousBase := apiBase
	apiBase = server.URL
	defer func() { apiBase = previousBase }()

	if err := preview(server.Client(), "/fs"); err != nil {
		t.Fatal(err)
	}
	if !previewCalled {
		t.Fatal("preview endpoint was not called")
	}
}

func TestPhrasesAddCallsEndpoint(t *testing.T) {
	var phraseCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/phrases" || r.Method != http.MethodPut {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		phraseCalled = true
		var req struct {
			Entries []phraseEntry `json:"entries"`
			Merge   bool          `json:"merge"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode phrase request: %v", err)
		}
		if !req.Merge || len(req.Entries) != 1 || req.Entries[0].Reading != "msd" || req.Entries[0].Text != "马上到！" {
			t.Fatalf("phrase request = %#v", req)
		}
		_ = json.NewEncoder(w).Encode(phraseResponse{Count: 1})
	}))
	defer server.Close()
	previousBase := apiBase
	apiBase = server.URL
	defer func() { apiBase = previousBase }()

	if err := phrases(server.Client(), []string{"add", "msd", "马上到！", "60000"}); err != nil {
		t.Fatal(err)
	}
	if !phraseCalled {
		t.Fatal("phrase endpoint was not called")
	}
}

func TestRejectsAddCallsEndpoint(t *testing.T) {
	var rejectCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rejects" || r.Method != http.MethodPut {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		rejectCalled = true
		var req struct {
			Entries []phraseEntry `json:"entries"`
			Merge   bool          `json:"merge"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode reject request: %v", err)
		}
		if !req.Merge || len(req.Entries) != 1 || req.Entries[0].Reading != "ceshi" || req.Entries[0].Text != "错词" {
			t.Fatalf("reject request = %#v", req)
		}
		_ = json.NewEncoder(w).Encode(rejectResponse{Count: 1})
	}))
	defer server.Close()
	previousBase := apiBase
	apiBase = server.URL
	defer func() { apiBase = previousBase }()

	if err := rejects(server.Client(), []string{"add", "ceshi", "错词"}); err != nil {
		t.Fatal(err)
	}
	if !rejectCalled {
		t.Fatal("reject endpoint was not called")
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

func TestReadPhraseFileAcceptsWrappedEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "phrases.json")
	if err := os.WriteFile(path, []byte(`{"entries":[{"reading":"msd","text":"马上到！","weight":60000}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readPhraseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Reading != "msd" || got[0].Text != "马上到！" || got[0].Weight != 60000 {
		t.Fatalf("phrases = %#v", got)
	}
}

func TestReadPhraseFileAcceptsRawEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "phrases.json")
	if err := os.WriteFile(path, []byte(`[{"reading":"yx","text":"邮箱"}]`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readPhraseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Reading != "yx" || got[0].Text != "邮箱" {
		t.Fatalf("phrases = %#v", got)
	}
}

func TestParseCatalogArgs(t *testing.T) {
	kind, query, limit, err := parseCatalogArgs([]string{"emoji", "zan", "--limit", "12"})
	if err != nil {
		t.Fatal(err)
	}
	if kind != "emoji" || query != "zan" || limit != 12 {
		t.Fatalf("catalog args = kind:%q query:%q limit:%d", kind, query, limit)
	}
}

func TestCatalogCallsEndpoint(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/catalog" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		called = true
		if r.URL.Query().Get("kind") != "symbol" || r.URL.Query().Get("q") != "/fs" || r.URL.Query().Get("limit") != "5" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(catalogResponse{
			Kind:  "symbol",
			Query: "/fs",
			Count: 1,
			Entries: []phraseEntry{{
				Reading: "fs",
				Text:    "℃",
				Kind:    "symbol",
				Source:  "builtin-symbols",
				Weight:  6200,
			}},
		})
	}))
	defer server.Close()
	previousBase := apiBase
	apiBase = server.URL
	defer func() { apiBase = previousBase }()

	if err := catalog(server.Client(), []string{"symbol", "/fs", "--limit=5"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("catalog endpoint was not called")
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
