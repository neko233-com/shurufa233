package main

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildManifestForPlainDictionary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zh-CN.test.json")
	dictionary := []byte(`{
		"language": "zh-CN",
		"version": "plain-test",
		"entries": [{ "reading": "ceshi", "text": "测试", "weight": 100 }]
	}`)
	if err := os.WriteFile(path, dictionary, 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := buildManifest([]string{path}, manifestOptions{
		Channel:     "stable",
		BaseURL:     "https://example.com/releases",
		GeneratedAt: "2026-07-05T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "plain-test" {
		t.Fatalf("version = %q", manifest.Version)
	}
	got := manifest.Dictionaries[0]
	if got.Language != "zh-CN" || got.Version != "plain-test" {
		t.Fatalf("descriptor = %#v", got)
	}
	if got.URL != "https://example.com/releases/zh-CN.test.json" {
		t.Fatalf("url = %q", got.URL)
	}
	if got.SHA256 != sha256Hex(dictionary) {
		t.Fatalf("sha256 = %q", got.SHA256)
	}
	if got.Compression != "" || got.ContentSHA256 != "" {
		t.Fatalf("plain descriptor should not contain compression fields: %#v", got)
	}
}

func TestBuildManifestForGzipDictionary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zh-CN.test.json.gz")
	dictionary := []byte(`{
		"language": "zh-CN",
		"version": "gzip-test",
		"entries": [{ "reading": "yasuo", "text": "压缩", "weight": 100 }]
	}`)
	compressed := gzipBytes(t, dictionary)
	if err := os.WriteFile(path, compressed, 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := buildManifest([]string{path}, manifestOptions{
		Version:     "2026.07.05",
		Channel:     "stable",
		BaseURL:     "https://example.com/releases/",
		GeneratedAt: "2026-07-05T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "2026.07.05" {
		t.Fatalf("manifest version = %q", manifest.Version)
	}
	got := manifest.Dictionaries[0]
	if got.URL != "https://example.com/releases/zh-CN.test.json.gz" {
		t.Fatalf("url = %q", got.URL)
	}
	if got.Compression != "gzip" {
		t.Fatalf("compression = %q", got.Compression)
	}
	if got.SHA256 != sha256Hex(compressed) {
		t.Fatalf("sha256 = %q", got.SHA256)
	}
	if got.ContentSHA256 != sha256Hex(dictionary) {
		t.Fatalf("contentSha256 = %q", got.ContentSHA256)
	}
}

func TestBuildManifestIncludesSourceProvenance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zh-CN.rime-ice.json")
	dictionary := []byte(`{
		"language": "zh-CN",
		"version": "rime-ice-test",
		"entries": [{ "reading": "wusong", "text": "雾凇", "weight": 100 }]
	}`)
	if err := os.WriteFile(path, dictionary, 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := buildManifest([]string{path}, manifestOptions{
		Channel:        "stable",
		BaseURL:        "https://example.com/releases",
		GeneratedAt:    "2026-07-05T00:00:00Z",
		SourcePreset:   "rime-ice-source",
		SourceURL:      "https://github.com/iDvel/rime-ice",
		SourceCommit:   "abcdef0",
		SourceLicense:  "GPL-3.0",
		ConvertCommand: "shurufa-dictimport ...",
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Source == nil || manifest.Source.Preset != "rime-ice-source" || manifest.Source.Commit != "abcdef0" {
		t.Fatalf("manifest source = %#v", manifest.Source)
	}
	got := manifest.Dictionaries[0]
	if got.Source == nil || got.Source.URL != "https://github.com/iDvel/rime-ice" || got.Source.License != "GPL-3.0" {
		t.Fatalf("dictionary source = %#v", got.Source)
	}
}

func TestBuildManifestRequiresDictionaryMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte(`{"entries":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := buildManifest([]string{path}, manifestOptions{}); err == nil {
		t.Fatal("expected metadata validation error")
	}
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
