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
	mu     sync.RWMutex
	config engine.Config
	engine *engine.Engine
	path   string
	client *http.Client
}

type previewRequest struct {
	Input string `json:"input"`
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

func main() {
	configPath, err := configFile()
	if err != nil {
		log.Fatal(err)
	}

	state := &AppState{
		path: configPath,
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
