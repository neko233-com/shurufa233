package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neko233-com/shurufa233/core/engine"
)

type manifestOptions struct {
	Version     string
	Channel     string
	BaseURL     string
	GeneratedAt string
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

func main() {
	version := flag.String("version", "", "manifest version; defaults to the first dictionary version")
	channel := flag.String("channel", "stable", "release channel")
	baseURL := flag.String("base-url", "", "release base URL, for example https://github.com/owner/repo/releases/latest/download")
	generatedAt := flag.String("generated-at", "", "RFC3339 generated timestamp; defaults to now")
	outPath := flag.String("out", "", "output manifest path, stdout when empty")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: shurufa-dictmanifest [flags] dictionary.json[.gz] ...")
		os.Exit(2)
	}

	manifest, err := buildManifest(flag.Args(), manifestOptions{
		Version:     *version,
		Channel:     *channel,
		BaseURL:     *baseURL,
		GeneratedAt: *generatedAt,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if *outPath == "" {
		if _, err := os.Stdout.Write(data); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildManifest(paths []string, options manifestOptions) (dictionaryManifest, error) {
	manifest := dictionaryManifest{
		Version:      strings.TrimSpace(options.Version),
		Channel:      strings.TrimSpace(options.Channel),
		GeneratedAt:  strings.TrimSpace(options.GeneratedAt),
		Dictionaries: make([]dictionaryDescriptor, 0, len(paths)),
	}
	if manifest.Channel == "" {
		manifest.Channel = "stable"
	}
	if manifest.GeneratedAt == "" {
		manifest.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	for _, path := range paths {
		descriptor, err := describeDictionaryArtifact(path, options.BaseURL)
		if err != nil {
			return dictionaryManifest{}, err
		}
		if manifest.Version == "" {
			manifest.Version = descriptor.Version
		}
		manifest.Dictionaries = append(manifest.Dictionaries, descriptor)
	}
	if manifest.Version == "" {
		manifest.Version = "dictionary-release"
	}
	return manifest, nil
}

func describeDictionaryArtifact(path string, baseURL string) (dictionaryDescriptor, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return dictionaryDescriptor{}, err
	}
	content, compression, err := decodeArtifactContent(path, raw)
	if err != nil {
		return dictionaryDescriptor{}, fmt.Errorf("%s: %w", path, err)
	}
	dictionary, err := engine.DecodeDictionary(bytes.NewReader(raw))
	if err != nil {
		return dictionaryDescriptor{}, fmt.Errorf("%s: %w", path, err)
	}
	if strings.TrimSpace(dictionary.Language) == "" {
		return dictionaryDescriptor{}, fmt.Errorf("%s: dictionary language is empty", path)
	}
	if strings.TrimSpace(dictionary.Version) == "" {
		return dictionaryDescriptor{}, fmt.Errorf("%s: dictionary version is empty", path)
	}
	descriptor := dictionaryDescriptor{
		Language: strings.TrimSpace(dictionary.Language),
		Version:  strings.TrimSpace(dictionary.Version),
		URL:      artifactURL(baseURL, path),
		SHA256:   sha256Hex(raw),
	}
	if compression != "" {
		descriptor.Compression = compression
		descriptor.ContentSHA256 = sha256Hex(content)
	}
	return descriptor, nil
}

func decodeArtifactContent(path string, raw []byte) ([]byte, string, error) {
	if !isGzipArtifact(path, raw) {
		return raw, "", nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}
	return content, "gzip", nil
}

func isGzipArtifact(path string, raw []byte) bool {
	return strings.HasSuffix(strings.ToLower(path), ".gz") || (len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b)
}

func artifactURL(baseURL string, path string) string {
	name := filepath.Base(path)
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return name
	}
	return baseURL + "/" + name
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}
