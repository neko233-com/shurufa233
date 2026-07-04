package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/neko233-com/shurufa233/core/engine"
)

const listenAddr = "127.0.0.1:23333"

type AppState struct {
	mu       sync.RWMutex
	config   engine.Config
	engine   *engine.Engine
	sessions map[string]*engine.Engine
	path     string
	client   *http.Client
}

type previewRequest struct {
	Input string `json:"input"`
}

type agentRequest struct {
	Input   string `json:"input"`
	Context string `json:"context,omitempty"`
}

type agentResponse struct {
	Input      string   `json:"input"`
	Candidates []string `json:"candidates"`
	Actions    []string `json:"actions"`
}

type dictionaryManifest struct {
	Version      string                 `json:"version"`
	Channel      string                 `json:"channel"`
	GeneratedAt  string                 `json:"generatedAt"`
	Dictionaries []dictionaryDescriptor `json:"dictionaries"`
}

type dictionaryDescriptor struct {
	Language string `json:"language"`
	Version  string `json:"version"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256,omitempty"`
}

type updateCheck struct {
	CurrentVersion  string             `json:"currentVersion"`
	LatestVersion   string             `json:"latestVersion"`
	UpdateAvailable bool               `json:"updateAvailable"`
	ManifestURL     string             `json:"manifestUrl,omitempty"`
	Manifest        dictionaryManifest `json:"manifest,omitempty"`
}

type userScoreStore struct {
	Version   int            `json:"version"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Scores    map[string]int `json:"scores"`
}

func main() {
	configPath, err := configFile()
	if err != nil {
		log.Fatal(err)
	}

	state := &AppState{
		sessions: make(map[string]*engine.Engine),
		path:     configPath,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	if err := state.load(); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", state.withCORS(state.health))
	mux.HandleFunc("GET /config", state.withCORS(state.getConfig))
	mux.HandleFunc("PUT /config", state.withCORS(state.putConfig))
	mux.HandleFunc("POST /engine/preview", state.withCORS(state.preview))
	mux.HandleFunc("GET /wordbook", state.withCORS(state.wordbook))
	mux.HandleFunc("GET /updates/check", state.withCORS(state.checkUpdates))
	mux.HandleFunc("POST /updates/apply", state.withCORS(state.applyUpdates))
	mux.HandleFunc("POST /ime/key", state.withCORS(state.imeKey))
	mux.HandleFunc("POST /ime/backspace", state.withCORS(state.imeBackspace))
	mux.HandleFunc("POST /ime/clear", state.withCORS(state.imeClear))
	mux.HandleFunc("POST /ime/select", state.withCORS(state.imeSelect))
	mux.HandleFunc("GET /ime/count", state.withCORS(state.imeCount))
	mux.HandleFunc("GET /ime/candidates", state.withCORS(state.imeCandidates))
	mux.HandleFunc("GET /ime/skin", state.withCORS(state.imeSkin))
	mux.HandleFunc("POST /agent/compose", state.withCORS(state.agentCompose))

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("shurufa233 daemon listening on http://%s", listenAddr)
	log.Fatal(server.ListenAndServe())
}

func (s *AppState) load() error {
	config := engine.DefaultConfig()
	data, err := os.ReadFile(s.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}
	}
	config = normalizeConfig(config)
	s.config = config
	s.engine = engine.New(config)
	s.engine.ImportUserScores(s.loadUserScores())
	s.sessions["default"] = s.engine
	if err := s.loadLocalDictionariesLocked(); err != nil {
		return err
	}
	return s.saveLocked()
}

func (s *AppState) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"ok":         true,
		"service":    "shurufa233-daemon",
		"configPath": s.path,
		"updatedAt":  time.Now().UTC(),
	})
}

func (s *AppState) getConfig(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, s.config)
}

func (s *AppState) putConfig(w http.ResponseWriter, r *http.Request) {
	var next engine.Config
	if err := json.NewDecoder(r.Body).Decode(&next); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = next
	s.engine.Configure(next)
	for _, session := range s.sessions {
		session.Configure(next)
	}
	if err := s.saveLocked(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, s.config)
}

func (s *AppState) preview(w http.ResponseWriter, r *http.Request) {
	var req previewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, s.engine.Preview(req.Input))
}

func (s *AppState) wordbook(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, map[string]any{
		"userScores": s.engine.UserScores(),
	})
}

func (s *AppState) imeKey(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}
	session := s.sessionForRequest(r)
	session.InputKey([]rune(key)[0])
	w.WriteHeader(http.StatusNoContent)
}

func (s *AppState) imeBackspace(w http.ResponseWriter, r *http.Request) {
	session := s.sessionForRequest(r)
	session.Backspace()
	w.WriteHeader(http.StatusNoContent)
}

func (s *AppState) imeClear(w http.ResponseWriter, r *http.Request) {
	session := s.sessionForRequest(r)
	session.Clear()
	w.WriteHeader(http.StatusNoContent)
}

func (s *AppState) imeSelect(w http.ResponseWriter, r *http.Request) {
	index := 0
	if raw := r.URL.Query().Get("index"); raw != "" {
		_, _ = fmt.Sscanf(raw, "%d", &index)
	}
	session := s.sessionForRequest(r)
	state, err := session.Select(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.saveUserScores(session.UserScores()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(state.Committed))
}

func (s *AppState) imeCount(w http.ResponseWriter, r *http.Request) {
	session := s.sessionForRequest(r)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "%d", len(session.State().Candidates))
}

func (s *AppState) imeCandidates(w http.ResponseWriter, r *http.Request) {
	session := s.sessionForRequest(r)
	state := session.State()
	parts := make([]string, 0, len(state.Candidates))
	for i, candidate := range state.Candidates {
		parts = append(parts, fmt.Sprintf("%d\t%s\t%s\t%d",
			i+1,
			candidate.Text,
			candidate.Reading,
			candidate.Weight+candidate.UserScore,
		))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(strings.Join(parts, "\n")))
}

func (s *AppState) imeSkin(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "%s|%d|%s|%s|%s|%s|%s|%s|%s",
		s.config.Skin.FontFamily,
		s.config.Skin.FontSize,
		s.config.Skin.Accent,
		s.config.Skin.Surface,
		s.config.Skin.Text,
		s.config.Skin.MutedText,
		s.config.Skin.Border,
		s.config.Skin.HighlightText,
		s.config.Skin.Theme,
	)
}

func (s *AppState) sessionForRequest(r *http.Request) *engine.Engine {
	id := "default"
	if r != nil {
		if raw := strings.TrimSpace(r.URL.Query().Get("session")); raw != "" {
			id = raw
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[id]
	if session == nil {
		session = engine.New(s.config)
		session.ImportUserScores(s.loadUserScores())
		s.sessions[id] = session
	}
	return session
}

func (s *AppState) agentCompose(w http.ResponseWriter, r *http.Request) {
	var req agentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	input := strings.TrimSpace(req.Input)
	if input == "" {
		http.Error(w, "missing input", http.StatusBadRequest)
		return
	}
	writeJSON(w, composeAgentResponse(input, req.Context))
}

func (s *AppState) checkUpdates(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	manifest, manifestURL, err := s.fetchManifest(config.Update.ManifestURLs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	current := config.Update.InstalledVersion
	writeJSON(w, updateCheck{
		CurrentVersion:  current,
		LatestVersion:   manifest.Version,
		UpdateAvailable: manifest.Version != "" && manifest.Version != current,
		ManifestURL:     manifestURL,
		Manifest:        manifest,
	})
}

func composeAgentResponse(input string, context string) agentResponse {
	lower := strings.ToLower(input)
	response := agentResponse{
		Input:   input,
		Actions: []string{"commit", "copy", "open-settings"},
	}
	switch {
	case strings.HasPrefix(lower, "/rewrite "):
		text := strings.TrimSpace(input[len("/rewrite "):])
		response.Candidates = []string{
			"请润色这段文字：" + text,
			"把下面内容改得更自然、更简洁：" + text,
		}
	case strings.HasPrefix(lower, "/translate "):
		text := strings.TrimSpace(input[len("/translate "):])
		response.Candidates = []string{
			"请把这段内容翻译成中文：" + text,
			"请把这段内容翻译成英文：" + text,
		}
	case strings.HasPrefix(lower, "/ask "):
		text := strings.TrimSpace(input[len("/ask "):])
		response.Candidates = []string{
			"请回答：" + text,
			"请分步骤分析：" + text,
		}
	default:
		prefix := "作为输入法 agent，请处理："
		if strings.TrimSpace(context) != "" {
			prefix = "结合当前上下文，作为输入法 agent，请处理："
		}
		response.Candidates = []string{prefix + input}
	}
	return response
}

func (s *AppState) applyUpdates(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	manifest, manifestURL, err := s.fetchManifest(s.config.Update.ManifestURLs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if len(manifest.Dictionaries) == 0 {
		http.Error(w, "manifest has no dictionaries", http.StatusBadGateway)
		return
	}

	dir := s.dictionaryDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	applied := make([]string, 0, len(manifest.Dictionaries))
	for _, item := range manifest.Dictionaries {
		if s.config.Language != "" && item.Language != s.config.Language {
			continue
		}
		data, err := s.downloadDictionary(item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if item.SHA256 != "" {
			sum := sha256.Sum256(data)
			if !strings.EqualFold(fmt.Sprintf("%x", sum[:]), item.SHA256) {
				http.Error(w, "dictionary sha256 mismatch", http.StatusBadGateway)
				return
			}
		}
		filePath := filepath.Join(dir, item.Language+"."+item.Version+".json")
		if err := os.WriteFile(filePath, data, 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		loaded, err := s.engine.LoadDictionary(strings.NewReader(string(data)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		applied = append(applied, loaded.Language+"@"+loaded.Version)
	}

	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), manifestData, 0o644)
	s.config.Update.InstalledVersion = manifest.Version
	if err := s.saveLocked(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"ok":          true,
		"manifestUrl": manifestURL,
		"version":     manifest.Version,
		"applied":     applied,
	})
}

func (s *AppState) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *AppState) fetchManifest(urls []string) (dictionaryManifest, string, error) {
	var lastErr error
	for _, url := range urls {
		if strings.TrimSpace(url) == "" {
			continue
		}
		resp, err := s.client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("%s returned HTTP %d", url, resp.StatusCode)
			continue
		}
		var manifest dictionaryManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			lastErr = err
			continue
		}
		return manifest, url, nil
	}
	if lastErr == nil {
		lastErr = errors.New("no update manifest URLs configured")
	}
	return dictionaryManifest{}, "", lastErr
}

func (s *AppState) downloadDictionary(item dictionaryDescriptor) ([]byte, error) {
	urls := s.dictionaryURLs(item.URL)
	var lastErr error
	for _, url := range urls {
		resp, err := s.client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("%s returned HTTP %d", url, resp.StatusCode)
			continue
		}
		return data, nil
	}
	return nil, lastErr
}

func (s *AppState) dictionaryURLs(rawURL string) []string {
	urls := []string{rawURL}
	name := filepath.Base(strings.ReplaceAll(rawURL, "\\", "/"))
	for _, base := range s.config.Update.MirrorBaseURLs {
		base = strings.TrimRight(strings.TrimSpace(base), "/")
		if base == "" {
			continue
		}
		urls = append([]string{base + "/" + name}, urls...)
	}
	return urls
}

func (s *AppState) loadLocalDictionariesLocked() error {
	dir := s.dictionaryDir()
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	for _, file := range files {
		if filepath.Base(file) == "manifest.json" {
			continue
		}
		f, err := os.Open(file)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		_, loadErr := s.engine.LoadDictionary(f)
		closeErr := f.Close()
		if loadErr != nil {
			return loadErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func (s *AppState) dictionaryDir() string {
	return filepath.Join(filepath.Dir(s.path), "dictionaries")
}

func (s *AppState) userScoresPath() string {
	return filepath.Join(filepath.Dir(s.path), "user-scores.json")
}

func (s *AppState) loadUserScores() map[string]int {
	data, err := os.ReadFile(s.userScoresPath())
	if err != nil {
		return nil
	}
	var store userScoreStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil
	}
	return store.Scores
}

func (s *AppState) saveUserScores(scores map[string]int) error {
	if len(scores) == 0 {
		return nil
	}
	path := s.userScoresPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	merged := make(map[string]int, len(scores))
	for key, value := range scores {
		merged[key] = value
	}
	if data, err := os.ReadFile(path); err == nil {
		var existing userScoreStore
		if json.Unmarshal(data, &existing) == nil {
			for key, value := range existing.Scores {
				if value > merged[key] {
					merged[key] = value
				}
			}
		}
	}
	store := userScoreStore{
		Version:   1,
		UpdatedAt: time.Now().UTC(),
		Scores:    merged,
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		if retryErr := os.Rename(tmp, path); retryErr != nil {
			_ = os.Remove(tmp)
			return retryErr
		}
	}
	return nil
}

func (s *AppState) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "content-type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func configFile() (string, error) {
	if override := os.Getenv("SHURUFA233_CONFIG"); override != "" {
		return override, nil
	}
	base := os.Getenv("APPDATA")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "shurufa233", "config.json"), nil
}

func normalizeConfig(config engine.Config) engine.Config {
	defaults := engine.DefaultConfig()
	if config.MaxCandidates <= 0 {
		config.MaxCandidates = defaults.MaxCandidates
	}
	if config.Language == "" {
		config.Language = defaults.Language
	}
	if config.Mode == "" {
		config.Mode = defaults.Mode
	}
	if config.Skin.FontFamily == "" {
		config.Skin = defaults.Skin
	}
	if config.Skin.Surface == "" {
		config.Skin.Surface = defaults.Skin.Surface
	}
	if config.Skin.Text == "" {
		config.Skin.Text = defaults.Skin.Text
	}
	if config.Skin.MutedText == "" {
		config.Skin.MutedText = defaults.Skin.MutedText
	}
	if config.Skin.Border == "" {
		config.Skin.Border = defaults.Skin.Border
	}
	if config.Skin.HighlightText == "" {
		config.Skin.HighlightText = defaults.Skin.HighlightText
	}
	if config.Update.Channel == "" {
		config.Update.Channel = defaults.Update.Channel
	}
	if len(config.Update.ManifestURLs) == 0 {
		config.Update.ManifestURLs = defaults.Update.ManifestURLs
	}
	if config.Update.MirrorBaseURLs == nil {
		config.Update.MirrorBaseURLs = defaults.Update.MirrorBaseURLs
	}
	if config.Update.InstalledVersion == "" {
		config.Update.InstalledVersion = defaults.Update.InstalledVersion
	}
	return config
}
