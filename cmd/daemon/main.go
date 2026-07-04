package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/neko233-com/shurufa233/core/engine"
)

const listenAddr = "127.0.0.1:23333"
const dictionaryAutoUpdateInterval = 6 * time.Hour
const dictionaryAutoUpdateInitialDelay = 30 * time.Second

type AppState struct {
	mu       sync.RWMutex
	config   engine.Config
	engine   *engine.Engine
	sessions map[string]*engine.Engine
	path     string
	logPath  string
	client   *http.Client
}

type previewRequest struct {
	Input string `json:"input"`
}

type modeRequest struct {
	Mode   string `json:"mode,omitempty"`
	Toggle bool   `json:"toggle,omitempty"`
}

type candidateActionRequest struct {
	Action       string `json:"action,omitempty"`
	Index        int    `json:"index,omitempty"`
	DisplayIndex int    `json:"displayIndex,omitempty"`
	Start        int    `json:"start,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	PageSize     int    `json:"pageSize,omitempty"`
	Delta        int    `json:"delta,omitempty"`
	Side         string `json:"side,omitempty"`
}

type candidateActionResult struct {
	OK         bool                 `json:"ok"`
	Action     string               `json:"action"`
	Start      int                  `json:"start"`
	Limit      int                  `json:"limit"`
	Total      int                  `json:"total"`
	Committed  string               `json:"committed,omitempty"`
	State      engine.State         `json:"state"`
	Candidates candidatePagePayload `json:"candidates"`
	UpdatedAt  time.Time            `json:"updatedAt"`
}

type candidatePagePayload struct {
	OK        bool                `json:"ok"`
	Start     int                 `json:"start"`
	Limit     int                 `json:"limit"`
	Total     int                 `json:"total"`
	Items     []candidatePageItem `json:"items"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

type candidatePageItem struct {
	Index        int    `json:"index"`
	DisplayIndex int    `json:"displayIndex"`
	Text         string `json:"text"`
	Reading      string `json:"reading"`
	Kind         string `json:"kind,omitempty"`
	Source       string `json:"source,omitempty"`
	Comment      string `json:"comment,omitempty"`
	Weight       int    `json:"weight"`
	UserScore    int    `json:"userScore"`
	Score        int    `json:"score"`
}

type agentRequest struct {
	Input   string `json:"input"`
	Context string `json:"context,omitempty"`
}

type agentCandidate struct {
	Text    string `json:"text"`
	Intent  string `json:"intent"`
	Action  string `json:"action"`
	Source  string `json:"source"`
	Context string `json:"context,omitempty"`
}

type agentResponse struct {
	Input      string           `json:"input"`
	Context    string           `json:"context,omitempty"`
	Candidates []string         `json:"candidates"`
	Items      []agentCandidate `json:"items"`
	Actions    []string         `json:"actions"`
}

type dictionaryManifest struct {
	Version      string                 `json:"version"`
	Channel      string                 `json:"channel"`
	GeneratedAt  string                 `json:"generatedAt"`
	Dictionaries []dictionaryDescriptor `json:"dictionaries"`
}

type dictionaryDescriptor struct {
	Language      string `json:"language"`
	Version       string `json:"version"`
	URL           string `json:"url"`
	SHA256        string `json:"sha256,omitempty"`
	Compression   string `json:"compression,omitempty"`
	ContentSHA256 string `json:"contentSha256,omitempty"`
}

type updateCheck struct {
	CurrentVersion  string             `json:"currentVersion"`
	LatestVersion   string             `json:"latestVersion"`
	UpdateAvailable bool               `json:"updateAvailable"`
	ManifestURL     string             `json:"manifestUrl,omitempty"`
	Manifest        dictionaryManifest `json:"manifest,omitempty"`
}

type updateApplyResult struct {
	OK          bool     `json:"ok"`
	ManifestURL string   `json:"manifestUrl"`
	Version     string   `json:"version"`
	Applied     []string `json:"applied"`
}

type userScoreStore struct {
	Version   int            `json:"version"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Scores    map[string]int `json:"scores"`
}

type wordbookRequest struct {
	UserScores map[string]int `json:"userScores,omitempty"`
	Scores     map[string]int `json:"scores,omitempty"`
	Merge      bool           `json:"merge,omitempty"`
}

func main() {
	logFile, logPath, err := setupFileLogging()
	if err != nil {
		log.Printf("warning: daemon file logging is unavailable: %v", err)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	configPath, err := configFile()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("shurufa233 daemon starting; config=%s log=%s", configPath, logPath)

	state := &AppState{
		sessions: make(map[string]*engine.Engine),
		path:     configPath,
		logPath:  logPath,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	if err := state.load(); err != nil {
		log.Fatal(err)
	}
	go state.runDictionaryAutoUpdater()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", state.withCORS(state.health))
	mux.HandleFunc("GET /config", state.withCORS(state.getConfig))
	mux.HandleFunc("PUT /config", state.withCORS(state.putConfig))
	mux.HandleFunc("POST /engine/preview", state.withCORS(state.preview))
	mux.HandleFunc("GET /wordbook", state.withCORS(state.wordbook))
	mux.HandleFunc("PUT /wordbook", state.withCORS(state.wordbook))
	mux.HandleFunc("DELETE /wordbook", state.withCORS(state.wordbook))
	mux.HandleFunc("GET /updates/check", state.withCORS(state.checkUpdates))
	mux.HandleFunc("POST /updates/apply", state.withCORS(state.applyUpdates))
	mux.HandleFunc("POST /ime/key", state.withCORS(state.imeKey))
	mux.HandleFunc("POST /ime/backspace", state.withCORS(state.imeBackspace))
	mux.HandleFunc("POST /ime/clear", state.withCORS(state.imeClear))
	mux.HandleFunc("POST /ime/select", state.withCORS(state.imeSelect))
	mux.HandleFunc("POST /ime/select-char", state.withCORS(state.imeSelectChar))
	mux.HandleFunc("POST /ime/candidate-action", state.withCORS(state.imeCandidateAction))
	mux.HandleFunc("GET /ime/count", state.withCORS(state.imeCount))
	mux.HandleFunc("GET /ime/candidates", state.withCORS(state.imeCandidates))
	mux.HandleFunc("GET /ime/skin", state.withCORS(state.imeSkin))
	mux.HandleFunc("GET /ime/mode", state.withCORS(state.imeMode))
	mux.HandleFunc("POST /ime/mode", state.withCORS(state.imeMode))
	mux.HandleFunc("POST /agent/compose", state.withCORS(state.agentCompose))
	if settingsDir := settingsStaticDir(); settingsDir != "" {
		fileServer := http.StripPrefix("/settings/", http.FileServer(http.Dir(settingsDir)))
		mux.HandleFunc("GET /settings", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/settings/", http.StatusMovedPermanently)
		})
		mux.Handle("GET /settings/", fileServer)
		log.Printf("settings UI serving from %s at http://%s/settings/", settingsDir, listenAddr)
	} else {
		log.Printf("settings UI static files not found; build apps/settings before packaging")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			state.withCORS(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("shurufa233 daemon listening on http://%s", listenAddr)
	log.Fatal(server.ListenAndServe())
}

func settingsStaticDir() string {
	candidates := make([]string, 0, 3)
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "settings"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "settings"),
			filepath.Join(cwd, "apps", "settings", "dist"),
		)
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
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
		"logPath":    s.logPath,
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
	next = normalizeConfig(next)
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

func (s *AppState) wordbook(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		defer s.mu.RUnlock()
		writeJSON(w, wordbookResponse(s.engine.UserScores()))
	case http.MethodPut:
		var req wordbookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		next := req.UserScores
		if next == nil {
			next = req.Scores
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if req.Merge {
			merged := s.engine.UserScores()
			for key, value := range next {
				if value > 0 {
					merged[key] = value
				}
			}
			next = merged
		}
		s.replaceUserScoresLocked(next)
		if err := s.writeUserScores(next); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, wordbookResponse(s.engine.UserScores()))
	case http.MethodDelete:
		key := strings.TrimSpace(r.URL.Query().Get("key"))
		s.mu.Lock()
		defer s.mu.Unlock()
		if key == "" {
			s.replaceUserScoresLocked(nil)
		} else {
			for _, session := range s.sessions {
				if session != nil {
					session.DeleteUserScore(key)
				}
			}
		}
		scores := s.engine.UserScores()
		if err := s.writeUserScores(scores); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, wordbookResponse(scores))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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

func (s *AppState) imeSelectChar(w http.ResponseWriter, r *http.Request) {
	index := 0
	if raw := r.URL.Query().Get("index"); raw != "" {
		_, _ = fmt.Sscanf(raw, "%d", &index)
	}
	side := strings.TrimSpace(r.URL.Query().Get("side"))
	if side == "" {
		side = "first"
	}
	session := s.sessionForRequest(r)
	state, err := session.SelectChar(index, side)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(state.Committed))
}

func (s *AppState) imeCandidateAction(w http.ResponseWriter, r *http.Request) {
	var req candidateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	applyCandidateActionQuery(&req, r)
	session := s.sessionForRequest(r)
	result, err := s.executeCandidateAction(session, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, result)
}

func (s *AppState) imeCount(w http.ResponseWriter, r *http.Request) {
	session := s.sessionForRequest(r)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "%d", len(session.State().Candidates))
}

func (s *AppState) imeCandidates(w http.ResponseWriter, r *http.Request) {
	session := s.sessionForRequest(r)
	state := session.State()
	start := parseBoundedInt(r.URL.Query().Get("start"), 0, len(state.Candidates))
	limit := parseBoundedInt(r.URL.Query().Get("limit"), len(state.Candidates), 9)
	if limit <= 0 || limit > 9 {
		limit = 9
	}
	end := start + limit
	if end > len(state.Candidates) {
		end = len(state.Candidates)
	}
	parts := make([]string, 0, max(0, end-start))
	for i, candidate := range state.Candidates[start:end] {
		parts = append(parts, fmt.Sprintf("%d\t%s\t%s\t%d\t%s\t%s\t%s",
			i+1,
			sanitizePayloadField(candidate.Text),
			sanitizePayloadField(candidate.Reading),
			candidate.Weight+candidate.UserScore,
			sanitizePayloadField(candidate.Kind),
			sanitizePayloadField(candidate.Source),
			sanitizePayloadField(candidate.Comment),
		))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(strings.Join(parts, "\n")))
}

func applyCandidateActionQuery(req *candidateActionRequest, r *http.Request) {
	query := r.URL.Query()
	if raw := strings.TrimSpace(query.Get("action")); raw != "" {
		req.Action = raw
	}
	if raw := strings.TrimSpace(query.Get("side")); raw != "" {
		req.Side = raw
	}
	if raw := strings.TrimSpace(query.Get("index")); raw != "" {
		req.Index = parseBoundedInt(raw, req.Index, -1)
	}
	if raw := strings.TrimSpace(query.Get("displayIndex")); raw != "" {
		req.DisplayIndex = parseBoundedInt(raw, req.DisplayIndex, -1)
	}
	if raw := strings.TrimSpace(query.Get("start")); raw != "" {
		req.Start = parseBoundedInt(raw, req.Start, -1)
	}
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		req.Limit = parseBoundedInt(raw, req.Limit, -1)
	}
	if raw := strings.TrimSpace(query.Get("pageSize")); raw != "" {
		req.PageSize = parseBoundedInt(raw, req.PageSize, -1)
	}
	if raw := strings.TrimSpace(query.Get("delta")); raw != "" {
		req.Delta = parseBoundedInt(raw, req.Delta, -1)
	}
}

func (s *AppState) executeCandidateAction(session *engine.Engine, req candidateActionRequest) (candidateActionResult, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "view"
	}
	limit := req.Limit
	if limit <= 0 {
		limit = req.PageSize
	}
	if limit <= 0 {
		limit = s.config.CandidatePageSize
	}
	if limit <= 0 {
		limit = engine.DefaultConfig().CandidatePageSize
	}
	if limit > 64 {
		limit = 64
	}
	start := req.Start
	if start < 0 {
		start = 0
	}
	total := len(session.State().Candidates)
	if start > total {
		start = total
	}

	switch action {
	case "view", "page", "candidates", "candidate-page":
		return buildCandidateActionResult(session, action, start, limit, ""), nil
	case "next", "next-page", "page-next":
		step := limit
		if req.Delta > 0 {
			step = req.Delta * limit
		}
		start += step
		if start >= total {
			start = maxCandidatePageStart(total, limit)
		}
		return buildCandidateActionResult(session, action, start, limit, ""), nil
	case "prev", "previous", "previous-page", "prev-page", "page-prev":
		step := limit
		if req.Delta > 0 {
			step = req.Delta * limit
		}
		start -= step
		if start < 0 {
			start = 0
		}
		return buildCandidateActionResult(session, action, start, limit, ""), nil
	case "home", "first-page":
		return buildCandidateActionResult(session, action, 0, limit, ""), nil
	case "end", "last-page":
		return buildCandidateActionResult(session, action, maxCandidatePageStart(total, limit), limit, ""), nil
	case "select", "commit", "commit-candidate":
		index := candidateActionIndex(req, start)
		state, err := session.Select(index)
		if err != nil {
			return candidateActionResult{}, err
		}
		if err := s.saveUserScores(session.UserScores()); err != nil {
			return candidateActionResult{}, err
		}
		return buildCandidateActionResultWithState(session, action, 0, limit, state.Committed, state), nil
	case "first-char", "commit-first-char":
		return executeCandidateCharAction(session, req, start, limit, action, "first")
	case "last-char", "commit-last-char":
		return executeCandidateCharAction(session, req, start, limit, action, "last")
	case "select-char", "commit-char", "commit-candidate-char":
		return executeCandidateCharAction(session, req, start, limit, action, req.Side)
	default:
		return candidateActionResult{}, fmt.Errorf("unknown candidate action: %s", action)
	}
}

func executeCandidateCharAction(session *engine.Engine, req candidateActionRequest, start int, limit int, action string, side string) (candidateActionResult, error) {
	index := candidateActionIndex(req, start)
	state, err := session.SelectChar(index, side)
	if err != nil {
		return candidateActionResult{}, err
	}
	return buildCandidateActionResultWithState(session, action, 0, limit, state.Committed, state), nil
}

func candidateActionIndex(req candidateActionRequest, start int) int {
	if req.DisplayIndex > 0 {
		return start + req.DisplayIndex - 1
	}
	return req.Index
}

func buildCandidateActionResult(session *engine.Engine, action string, start int, limit int, committed string) candidateActionResult {
	return buildCandidateActionResultWithState(session, action, start, limit, committed, session.State())
}

func buildCandidateActionResultWithState(session *engine.Engine, action string, start int, limit int, committed string, state engine.State) candidateActionResult {
	candidates := buildCandidatePagePayload(session, start, limit)
	return candidateActionResult{
		OK:         true,
		Action:     action,
		Start:      candidates.Start,
		Limit:      candidates.Limit,
		Total:      candidates.Total,
		Committed:  committed,
		State:      state,
		Candidates: candidates,
		UpdatedAt:  time.Now().UTC(),
	}
}

func buildCandidatePagePayload(session *engine.Engine, start int, limit int) candidatePagePayload {
	state := session.State()
	if start < 0 {
		start = 0
	}
	if start > len(state.Candidates) {
		start = len(state.Candidates)
	}
	if limit <= 0 || limit > 64 {
		limit = 64
	}
	end := start + limit
	if end > len(state.Candidates) {
		end = len(state.Candidates)
	}
	items := make([]candidatePageItem, 0, end-start)
	for i, candidate := range state.Candidates[start:end] {
		index := start + i
		items = append(items, candidatePageItem{
			Index:        index,
			DisplayIndex: i + 1,
			Text:         candidate.Text,
			Reading:      candidate.Reading,
			Kind:         candidate.Kind,
			Source:       candidate.Source,
			Comment:      candidate.Comment,
			Weight:       candidate.Weight,
			UserScore:    candidate.UserScore,
			Score:        candidate.Weight + candidate.UserScore,
		})
	}
	return candidatePagePayload{
		OK:        true,
		Start:     start,
		Limit:     limit,
		Total:     len(state.Candidates),
		Items:     items,
		UpdatedAt: time.Now().UTC(),
	}
}

func maxCandidatePageStart(total int, limit int) int {
	if total <= 0 || limit <= 0 {
		return 0
	}
	return ((total - 1) / limit) * limit
}

func parseBoundedInt(raw string, fallback int, upper int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	if value < 0 {
		return 0
	}
	if upper >= 0 && value > upper {
		return upper
	}
	return value
}

func sanitizePayloadField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func (s *AppState) imeSkin(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "%s|%d|%s|%s|%s|%s|%s|%s|%s|%d|%s|%t",
		s.config.Skin.FontFamily,
		s.config.Skin.FontSize,
		s.config.Skin.Accent,
		s.config.Skin.Surface,
		s.config.Skin.Text,
		s.config.Skin.MutedText,
		s.config.Skin.Border,
		s.config.Skin.HighlightText,
		s.config.Skin.Theme,
		s.config.CandidatePageSize,
		sanitizePayloadField(s.config.CandidateLayout),
		s.config.ShowCandidateComments,
	)
}

func (s *AppState) imeMode(w http.ResponseWriter, r *http.Request) {
	session := s.sessionForRequest(r)
	if r.Method == http.MethodGet {
		writeJSON(w, session.State())
		return
	}
	var req modeRequest
	query := r.URL.Query()
	if raw := strings.TrimSpace(query.Get("mode")); raw != "" {
		req.Mode = raw
	}
	if raw := strings.TrimSpace(query.Get("toggle")); raw != "" {
		req.Toggle = raw == "1" || strings.EqualFold(raw, "true")
	}
	if req.Mode == "" && !req.Toggle {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if req.Mode == "" && !req.Toggle {
		req.Mode = engine.DefaultConfig().Mode
	}
	if req.Mode != "" && !req.Toggle && normalizeModeValue(req.Mode, "") == "" {
		http.Error(w, "mode must be zh, en, or toggle", http.StatusBadRequest)
		return
	}
	var state engine.State
	if req.Toggle || strings.EqualFold(strings.TrimSpace(req.Mode), "toggle") {
		state = session.ToggleMode()
	} else {
		state = session.SetMode(req.Mode)
	}
	writeJSON(w, state)
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
		if err := s.loadLocalDictionariesIntoLocked(session); err != nil {
			log.Printf("warning: could not load dictionaries into session %s: %v", id, err)
		}
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
	check, err := s.checkLatestDictionaries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, check)
}

func composeAgentResponse(input string, context string) agentResponse {
	lower := strings.ToLower(input)
	response := agentResponse{
		Input:   input,
		Context: strings.TrimSpace(context),
		Actions: []string{"commit", "copy", "open-settings"},
	}
	add := func(intent string, action string, text string) {
		item := agentCandidate{
			Text:   text,
			Intent: intent,
			Action: action,
			Source: "builtin-agent",
		}
		if response.Context != "" {
			item.Context = response.Context
		}
		response.Items = append(response.Items, item)
		response.Candidates = append(response.Candidates, text)
	}
	switch {
	case lower == "/rewrite" || strings.HasPrefix(lower, "/rewrite "):
		text := commandText(input, "/rewrite")
		if text == "" {
			text = response.Context
		}
		add("rewrite", "agent.rewrite.polish", "请润色这段文字："+text)
		add("rewrite", "agent.rewrite.concise", "把下面内容改得更自然、更简洁："+text)
	case lower == "/translate" || strings.HasPrefix(lower, "/translate "):
		text := commandText(input, "/translate")
		if text == "" {
			text = response.Context
		}
		add("translate", "agent.translate.zh", "请把这段内容翻译成中文："+text)
		add("translate", "agent.translate.en", "请把这段内容翻译成英文："+text)
	case lower == "/ask" || strings.HasPrefix(lower, "/ask "):
		text := commandText(input, "/ask")
		if text == "" {
			text = response.Context
		}
		add("ask", "agent.ask.answer", "请回答："+text)
		add("ask", "agent.ask.steps", "请分步骤分析："+text)
	default:
		prefix := "作为输入法 agent，请处理："
		if response.Context != "" {
			prefix = "结合当前上下文，作为输入法 agent，请处理："
		}
		add("compose", "agent.compose", prefix+input)
	}
	return response
}

func commandText(input string, command string) string {
	if len(input) <= len(command) {
		return ""
	}
	return strings.TrimSpace(input[len(command):])
}

func (s *AppState) applyUpdates(w http.ResponseWriter, _ *http.Request) {
	result, err := s.applyLatestDictionaries(true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, result)
}

func (s *AppState) checkLatestDictionaries() (updateCheck, error) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	manifest, manifestURL, err := s.fetchManifest(config.Update.ManifestURLs)
	if err != nil {
		return updateCheck{}, err
	}
	current := config.Update.InstalledVersion
	return updateCheck{
		CurrentVersion:  current,
		LatestVersion:   manifest.Version,
		UpdateAvailable: manifest.Version != "" && manifest.Version != current,
		ManifestURL:     manifestURL,
		Manifest:        manifest,
	}, nil
}

func (s *AppState) applyLatestDictionaries(force bool) (updateApplyResult, error) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	manifest, manifestURL, err := s.fetchManifest(config.Update.ManifestURLs)
	if err != nil {
		return updateApplyResult{}, err
	}
	if !force && manifest.Version != "" && manifest.Version == config.Update.InstalledVersion {
		return updateApplyResult{
			OK:          true,
			ManifestURL: manifestURL,
			Version:     manifest.Version,
			Applied:     []string{},
		}, nil
	}
	if len(manifest.Dictionaries) == 0 {
		return updateApplyResult{}, errors.New("manifest has no dictionaries")
	}

	dir := s.dictionaryDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return updateApplyResult{}, err
	}

	type downloadedDictionary struct {
		item dictionaryDescriptor
		data []byte
	}
	downloaded := make([]downloadedDictionary, 0, len(manifest.Dictionaries))
	applied := make([]string, 0, len(manifest.Dictionaries))
	for _, item := range manifest.Dictionaries {
		if config.Language != "" && item.Language != config.Language {
			continue
		}
		rawData, err := s.downloadDictionaryWithConfig(config, item)
		if err != nil {
			return updateApplyResult{}, err
		}
		if item.SHA256 != "" {
			sum := sha256.Sum256(rawData)
			if !strings.EqualFold(fmt.Sprintf("%x", sum[:]), item.SHA256) {
				return updateApplyResult{}, errors.New("dictionary sha256 mismatch")
			}
		}
		data, err := decodeDownloadedDictionary(item, rawData)
		if err != nil {
			return updateApplyResult{}, err
		}
		if item.ContentSHA256 != "" {
			sum := sha256.Sum256(data)
			if !strings.EqualFold(fmt.Sprintf("%x", sum[:]), item.ContentSHA256) {
				return updateApplyResult{}, errors.New("dictionary content sha256 mismatch")
			}
		}
		downloaded = append(downloaded, downloadedDictionary{item: item, data: data})
	}
	if len(downloaded) == 0 {
		return updateApplyResult{}, fmt.Errorf("manifest has no dictionary for language %s", config.Language)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, file := range downloaded {
		item := file.item
		filePath := filepath.Join(dir, item.Language+"."+item.Version+".json")
		if err := writeFileAtomic(filePath, file.data, 0o644); err != nil {
			return updateApplyResult{}, err
		}
		loaded, err := s.loadDictionaryIntoSessionsLocked(file.data)
		if err != nil {
			return updateApplyResult{}, err
		}
		applied = append(applied, loaded.Language+"@"+loaded.Version)
	}

	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), manifestData, 0o644)
	s.config.Update.InstalledVersion = manifest.Version
	if err := s.saveLocked(); err != nil {
		return updateApplyResult{}, err
	}
	return updateApplyResult{
		OK:          true,
		ManifestURL: manifestURL,
		Version:     manifest.Version,
		Applied:     applied,
	}, nil
}

func (s *AppState) runDictionaryAutoUpdater() {
	timer := time.NewTimer(dictionaryAutoUpdateInitialDelay)
	defer timer.Stop()
	for {
		<-timer.C
		s.runDictionaryAutoUpdateOnce()
		timer.Reset(dictionaryAutoUpdateInterval)
	}
}

func (s *AppState) runDictionaryAutoUpdateOnce() {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()
	if !config.Update.AutoCheck {
		return
	}
	check, err := s.checkLatestDictionaries()
	if err != nil {
		log.Printf("dictionary auto update check failed: %v", err)
		return
	}
	if !check.UpdateAvailable {
		log.Printf("dictionary auto update check: current=%s latest=%s", check.CurrentVersion, check.LatestVersion)
		return
	}
	log.Printf("dictionary update available: current=%s latest=%s manifest=%s", check.CurrentVersion, check.LatestVersion, check.ManifestURL)
	if !config.Update.AutoApply {
		return
	}
	result, err := s.applyLatestDictionaries(false)
	if err != nil {
		log.Printf("dictionary auto update apply failed: %v", err)
		return
	}
	log.Printf("dictionary auto update applied: version=%s applied=%s", result.Version, strings.Join(result.Applied, ","))
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

func (s *AppState) downloadDictionaryWithConfig(config engine.Config, item dictionaryDescriptor) ([]byte, error) {
	urls := dictionaryURLs(config, item.URL)
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

func decodeDownloadedDictionary(item dictionaryDescriptor, data []byte) ([]byte, error) {
	compression := strings.ToLower(strings.TrimSpace(item.Compression))
	if compression == "" && (strings.HasSuffix(strings.ToLower(item.URL), ".gz") || hasGzipHeader(data)) {
		compression = "gzip"
	}
	switch compression {
	case "":
		return data, nil
	case "gzip", "gz":
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	default:
		return nil, fmt.Errorf("unsupported dictionary compression %q", item.Compression)
	}
}

func hasGzipHeader(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b
}

func dictionaryURLs(config engine.Config, rawURL string) []string {
	urls := []string{rawURL}
	name := filepath.Base(strings.ReplaceAll(rawURL, "\\", "/"))
	for _, base := range config.Update.MirrorBaseURLs {
		base = strings.TrimRight(strings.TrimSpace(base), "/")
		if base == "" {
			continue
		}
		urls = append([]string{base + "/" + name}, urls...)
	}
	return urls
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (s *AppState) loadLocalDictionariesIntoLocked(target *engine.Engine) error {
	dir := s.dictionaryDir()
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	gzFiles, err := filepath.Glob(filepath.Join(dir, "*.json.gz"))
	if err != nil {
		return err
	}
	files = append(files, gzFiles...)
	for _, file := range files {
		if isDictionaryMetadataFile(filepath.Base(file)) {
			continue
		}
		f, err := os.Open(file)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		_, loadErr := target.LoadDictionary(f)
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

func (s *AppState) loadLocalDictionariesLocked() error {
	return s.loadLocalDictionariesIntoLocked(s.engine)
}

func (s *AppState) loadDictionaryIntoSessionsLocked(data []byte) (engine.DictionaryFile, error) {
	loaded, err := s.engine.LoadDictionary(bytes.NewReader(data))
	if err != nil {
		return engine.DictionaryFile{}, err
	}
	for id, session := range s.sessions {
		if session == nil || session == s.engine {
			continue
		}
		if _, err := session.LoadDictionary(bytes.NewReader(data)); err != nil {
			return engine.DictionaryFile{}, fmt.Errorf("load dictionary into session %s: %w", id, err)
		}
	}
	return loaded, nil
}

func isDictionaryMetadataFile(name string) bool {
	name = strings.TrimSuffix(name, ".gz")
	return name == "manifest.json" || name == "dictionary-manifest.json"
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

func wordbookResponse(scores map[string]int) map[string]any {
	if scores == nil {
		scores = map[string]int{}
	}
	return map[string]any{
		"userScores": scores,
		"count":      len(scores),
		"updatedAt":  time.Now().UTC(),
	}
}

func (s *AppState) replaceUserScoresLocked(scores map[string]int) {
	for _, session := range s.sessions {
		if session != nil {
			session.ReplaceUserScores(scores)
		}
	}
}

func (s *AppState) saveUserScores(scores map[string]int) error {
	if len(scores) == 0 {
		return nil
	}
	path := s.userScoresPath()
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
	return s.writeUserScores(merged)
}

func (s *AppState) writeUserScores(scores map[string]int) error {
	path := s.userScoresPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	normalized := make(map[string]int, len(scores))
	for key, value := range scores {
		if key == "" || value <= 0 {
			continue
		}
		normalized[key] = value
	}
	store := userScoreStore{
		Version:   1,
		UpdatedAt: time.Now().UTC(),
		Scores:    normalized,
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
		if origin := r.Header.Get("Origin"); isAllowedLocalOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "content-type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func isAllowedLocalOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"http://[::1]:5173",
		"wails://wails",
	} {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func setupFileLogging() (*os.File, string, error) {
	logPath, err := daemonLogFile()
	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, logPath, err
	}
	rotateLogIfLarge(logPath, 4*1024*1024)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, logPath, err
	}
	log.SetOutput(io.MultiWriter(os.Stderr, file))
	return file, logPath, nil
}

func daemonLogFile() (string, error) {
	if override := strings.TrimSpace(os.Getenv("SHURUFA233_LOG")); override != "" {
		return override, nil
	}
	if base := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); base != "" {
		return filepath.Join(base, "shurufa233-daemon.log"), nil
	}
	base, err := os.UserCacheDir()
	if err == nil && strings.TrimSpace(base) != "" {
		return filepath.Join(base, "shurufa233-daemon.log"), nil
	}
	return filepath.Join(os.TempDir(), "shurufa233-daemon.log"), nil
}

func rotateLogIfLarge(path string, maxBytes int64) {
	info, err := os.Stat(path)
	if err != nil || info.Size() <= maxBytes {
		return
	}
	backup := path + ".1"
	_ = os.Remove(backup)
	_ = os.Rename(path, backup)
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
	if config.MaxCandidates < defaults.MaxCandidates {
		config.MaxCandidates = defaults.MaxCandidates
	}
	if config.CandidatePageSize <= 0 {
		config.CandidatePageSize = defaults.CandidatePageSize
	}
	if config.CandidatePageSize < 3 {
		config.CandidatePageSize = 3
	}
	if config.CandidatePageSize > 9 {
		config.CandidatePageSize = 9
	}
	config.CandidateLayout = normalizeCandidateLayoutValue(config.CandidateLayout, defaults.CandidateLayout)
	if config.Language == "" {
		config.Language = defaults.Language
	}
	if config.Mode == "" {
		config.Mode = defaults.Mode
	}
	config.Mode = normalizeModeValue(config.Mode, defaults.Mode)
	config.Punctuation = normalizePunctuationValue(config.Punctuation, defaults.Punctuation)
	config.DoublePinyinScheme = normalizeDoublePinyinSchemeValue(config.DoublePinyinScheme, defaults.DoublePinyinScheme)
	if config.Skin.FontFamily == "" {
		config.Skin = defaults.Skin
	}
	if config.Skin.Surface == "" {
		config.Skin.Surface = defaults.Skin.Surface
	}
	if !isHexColor(config.Skin.Surface) {
		config.Skin.Surface = defaults.Skin.Surface
	}
	if !isHexColor(config.Skin.Accent) {
		config.Skin.Accent = defaults.Skin.Accent
	}
	if config.Skin.Text == "" {
		config.Skin.Text = defaults.Skin.Text
	}
	if !isHexColor(config.Skin.Text) {
		config.Skin.Text = defaults.Skin.Text
	}
	if config.Skin.MutedText == "" {
		config.Skin.MutedText = defaults.Skin.MutedText
	}
	if !isHexColor(config.Skin.MutedText) {
		config.Skin.MutedText = defaults.Skin.MutedText
	}
	if config.Skin.Border == "" {
		config.Skin.Border = defaults.Skin.Border
	}
	if !isHexColor(config.Skin.Border) {
		config.Skin.Border = defaults.Skin.Border
	}
	if config.Skin.HighlightText == "" {
		config.Skin.HighlightText = defaults.Skin.HighlightText
	}
	if !isHexColor(config.Skin.HighlightText) {
		config.Skin.HighlightText = defaults.Skin.HighlightText
	}
	config.Skin.Text = ensureReadableColor(config.Skin.Text, config.Skin.Surface, 4.5)
	config.Skin.MutedText = ensureReadableColor(config.Skin.MutedText, config.Skin.Surface, 3.0)
	config.Skin.HighlightText = ensureReadableColor(config.Skin.HighlightText, config.Skin.Accent, 4.5)
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

func normalizeCandidateLayoutValue(layout string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(layout)) {
	case "", "horizontal", "wechat", "microsoft":
		if strings.TrimSpace(layout) == "" {
			return fallback
		}
		return "horizontal"
	case "vertical", "rime":
		return "vertical"
	case "auto":
		return "auto"
	default:
		return fallback
	}
}

func normalizeModeValue(mode string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "zh", "en":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return fallback
	}
}

func normalizePunctuationValue(mode string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "full", "half":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		if fallback == "" {
			return "full"
		}
		return fallback
	}
}

func normalizeDoublePinyinSchemeValue(scheme string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "", "xiaohe", "flypy":
		return "xiaohe"
	case "microsoft", "ms", "sogou":
		return "microsoft"
	default:
		if fallback == "" {
			return "xiaohe"
		}
		return fallback
	}
}

type rgbColor struct {
	r float64
	g float64
	b float64
}

func isHexColor(value string) bool {
	_, ok := parseHexColor(value)
	return ok
}

func parseHexColor(value string) (rgbColor, bool) {
	if len(value) != 7 || value[0] != '#' {
		return rgbColor{}, false
	}
	r, err := strconv.ParseUint(value[1:3], 16, 8)
	if err != nil {
		return rgbColor{}, false
	}
	g, err := strconv.ParseUint(value[3:5], 16, 8)
	if err != nil {
		return rgbColor{}, false
	}
	b, err := strconv.ParseUint(value[5:7], 16, 8)
	if err != nil {
		return rgbColor{}, false
	}
	return rgbColor{r: float64(r), g: float64(g), b: float64(b)}, true
}

func ensureReadableColor(foreground, background string, minRatio float64) string {
	if contrastRatio(foreground, background) >= minRatio {
		return foreground
	}
	blackRatio := contrastRatio("#111827", background)
	whiteRatio := contrastRatio("#ffffff", background)
	if whiteRatio > blackRatio {
		return "#ffffff"
	}
	return "#111827"
}

func contrastRatio(foreground, background string) float64 {
	fg, ok := parseHexColor(foreground)
	if !ok {
		return 0
	}
	bg, ok := parseHexColor(background)
	if !ok {
		return 0
	}
	l1 := relativeLuminance(fg) + 0.05
	l2 := relativeLuminance(bg) + 0.05
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return l1 / l2
}

func relativeLuminance(color rgbColor) float64 {
	return 0.2126*linearRGB(color.r) + 0.7152*linearRGB(color.g) + 0.0722*linearRGB(color.b)
}

func linearRGB(value float64) float64 {
	value = value / 255
	if value <= 0.03928 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}
