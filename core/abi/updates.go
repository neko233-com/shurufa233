package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neko233-com/shurufa233/core/engine"
)

type abiDictionaryManifest struct {
	Version      string                    `json:"version"`
	Channel      string                    `json:"channel"`
	GeneratedAt  string                    `json:"generatedAt"`
	Source       *abiSourceProvenance      `json:"source,omitempty"`
	Dictionaries []abiDictionaryDescriptor `json:"dictionaries"`
}

type abiSourceProvenance struct {
	Preset         string `json:"preset,omitempty"`
	URL            string `json:"url,omitempty"`
	Commit         string `json:"commit,omitempty"`
	License        string `json:"license,omitempty"`
	ConvertCommand string `json:"convertCommand,omitempty"`
}

type abiDictionaryDescriptor struct {
	Language      string               `json:"language"`
	Version       string               `json:"version"`
	URL           string               `json:"url"`
	SHA256        string               `json:"sha256,omitempty"`
	Compression   string               `json:"compression,omitempty"`
	ContentSHA256 string               `json:"contentSha256,omitempty"`
	Source        *abiSourceProvenance `json:"source,omitempty"`
}

type abiUpdateCheck struct {
	OK              bool                  `json:"ok"`
	CurrentVersion  string                `json:"currentVersion"`
	LatestVersion   string                `json:"latestVersion"`
	UpdateAvailable bool                  `json:"updateAvailable"`
	ManifestURL     string                `json:"manifestUrl,omitempty"`
	Manifest        abiDictionaryManifest `json:"manifest,omitempty"`
	UpdatedAt       time.Time             `json:"updatedAt"`
}

type abiUpdateApplyResult struct {
	OK          bool      `json:"ok"`
	ManifestURL string    `json:"manifestUrl"`
	Version     string    `json:"version"`
	Applied     []string  `json:"applied"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func dictionaryUpdateCheckPayload(req extensionCommandPayload) any {
	config := dictionaryUpdateConfigFromPayload(req)
	manifest, manifestURL, err := fetchDictionaryUpdateManifest(config, http.DefaultClient)
	if err != nil {
		return errorEnvelope(err.Error())
	}
	current := config.Update.InstalledVersion
	return abiUpdateCheck{
		OK:              true,
		CurrentVersion:  current,
		LatestVersion:   manifest.Version,
		UpdateAvailable: manifest.Version != "" && manifest.Version != current,
		ManifestURL:     manifestURL,
		Manifest:        manifest,
		UpdatedAt:       time.Now().UTC(),
	}
}

func dictionaryUpdateApplyPayload(req extensionCommandPayload) any {
	config := dictionaryUpdateConfigFromPayload(req)
	manifest, manifestURL, err := fetchDictionaryUpdateManifest(config, http.DefaultClient)
	if err != nil {
		return errorEnvelope(err.Error())
	}
	force := req.Force == nil || *req.Force
	if !force && manifest.Version != "" && manifest.Version == config.Update.InstalledVersion {
		return abiUpdateApplyResult{
			OK:          true,
			ManifestURL: manifestURL,
			Version:     manifest.Version,
			Applied:     []string{},
			UpdatedAt:   time.Now().UTC(),
		}
	}
	if len(manifest.Dictionaries) == 0 {
		return errorEnvelope("manifest has no dictionaries")
	}
	dir, err := dictionaryDir()
	if err != nil {
		return errorEnvelope(err.Error())
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errorEnvelope(err.Error())
	}

	type downloadedDictionary struct {
		item abiDictionaryDescriptor
		data []byte
	}
	downloaded := make([]downloadedDictionary, 0, len(manifest.Dictionaries))
	for _, item := range manifest.Dictionaries {
		if config.Language != "" && item.Language != config.Language {
			continue
		}
		rawData, err := downloadDictionaryArtifact(config, item, http.DefaultClient)
		if err != nil {
			return errorEnvelope(err.Error())
		}
		if item.SHA256 != "" {
			sum := sha256.Sum256(rawData)
			if !strings.EqualFold(fmt.Sprintf("%x", sum[:]), item.SHA256) {
				return errorEnvelope("dictionary sha256 mismatch")
			}
		}
		data, err := decodeDictionaryArtifact(item, rawData)
		if err != nil {
			return errorEnvelope(err.Error())
		}
		if item.ContentSHA256 != "" {
			sum := sha256.Sum256(data)
			if !strings.EqualFold(fmt.Sprintf("%x", sum[:]), item.ContentSHA256) {
				return errorEnvelope("dictionary content sha256 mismatch")
			}
		}
		downloaded = append(downloaded, downloadedDictionary{item: item, data: data})
	}
	if len(downloaded) == 0 {
		return errorEnvelope(fmt.Sprintf("manifest has no dictionary for language %s", config.Language))
	}

	applied := make([]string, 0, len(downloaded))
	sessions := activeSessions()
	for _, file := range downloaded {
		item := file.item
		filePath := filepath.Join(dir, item.Language+"."+item.Version+".json")
		if err := writeFileAtomic(filePath, file.data, 0o644); err != nil {
			return errorEnvelope(err.Error())
		}
		loaded, err := engine.DecodeDictionary(bytes.NewReader(file.data))
		if err != nil {
			return errorEnvelope(err.Error())
		}
		for _, session := range sessions {
			if _, err := session.LoadDictionary(bytes.NewReader(file.data)); err != nil {
				return errorEnvelope(err.Error())
			}
		}
		applied = append(applied, loaded.Language+"@"+loaded.Version)
	}

	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	if err := writeFileAtomic(filepath.Join(dir, "manifest.json"), manifestData, 0o644); err != nil {
		return errorEnvelope(err.Error())
	}
	config.Update.InstalledVersion = manifest.Version
	config = normalizeConfig(config)
	configureActiveSessions(config)
	if err := persistConfig(config); err != nil {
		return errorEnvelope(err.Error())
	}
	return abiUpdateApplyResult{
		OK:          true,
		ManifestURL: manifestURL,
		Version:     manifest.Version,
		Applied:     applied,
		UpdatedAt:   time.Now().UTC(),
	}
}

func dictionaryUpdateConfigFromPayload(req extensionCommandPayload) engine.Config {
	config := loadConfig()
	if len(req.ManifestURLs) > 0 {
		config.Update.ManifestURLs = append([]string(nil), req.ManifestURLs...)
	}
	if req.MirrorBaseURLs != nil {
		config.Update.MirrorBaseURLs = append([]string(nil), req.MirrorBaseURLs...)
	}
	return normalizeConfig(config)
}

func fetchDictionaryUpdateManifest(config engine.Config, client *http.Client) (abiDictionaryManifest, string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	var lastErr error
	for _, url := range mirroredUpdateURLs(config, config.Update.ManifestURLs...) {
		resp, err := client.Get(url)
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
		var manifest abiDictionaryManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			lastErr = err
			continue
		}
		return manifest, url, nil
	}
	if lastErr == nil {
		lastErr = errors.New("no update manifest URLs configured")
	}
	return abiDictionaryManifest{}, "", lastErr
}

func downloadDictionaryArtifact(config engine.Config, item abiDictionaryDescriptor, client *http.Client) ([]byte, error) {
	if client == nil {
		client = http.DefaultClient
	}
	var lastErr error
	for _, url := range mirroredUpdateURLs(config, item.URL) {
		resp, err := client.Get(url)
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
	if lastErr == nil {
		lastErr = errors.New("no dictionary URLs configured")
	}
	return nil, lastErr
}

func decodeDictionaryArtifact(item abiDictionaryDescriptor, data []byte) ([]byte, error) {
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

func mirroredUpdateURLs(config engine.Config, rawURLs ...string) []string {
	out := make([]string, 0, len(rawURLs)*(len(config.Update.MirrorBaseURLs)+1))
	seen := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		out = append(out, value)
	}
	for _, rawURL := range rawURLs {
		for _, mirror := range config.Update.MirrorBaseURLs {
			add(renderMirrorUpdateURL(mirror, rawURL))
		}
		add(rawURL)
	}
	return out
}

func renderMirrorUpdateURL(mirror string, rawURL string) string {
	mirror = strings.TrimSpace(mirror)
	rawURL = strings.TrimSpace(rawURL)
	if mirror == "" || rawURL == "" {
		return ""
	}
	name := filepath.Base(strings.ReplaceAll(rawURL, "\\", "/"))
	if strings.Contains(mirror, "{url}") || strings.Contains(mirror, "{file}") || strings.Contains(mirror, "{filename}") || strings.Contains(mirror, "{name}") {
		out := strings.ReplaceAll(mirror, "{url}", rawURL)
		out = strings.ReplaceAll(out, "{file}", name)
		out = strings.ReplaceAll(out, "{filename}", name)
		out = strings.ReplaceAll(out, "{name}", name)
		return out
	}
	return strings.TrimRight(mirror, "/") + "/" + name
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
