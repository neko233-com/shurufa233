package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBuildPlanDefaultsToRimeIceArtifact(t *testing.T) {
	plan, err := buildPlan(options{
		Preset:       "rime-ice-source",
		WorkDir:      filepath.Join("tmp", "dicts"),
		OutDir:       filepath.Join("out", "dicts"),
		ManifestPath: filepath.Join("out", "dicts", "manifest.json"),
	}, time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if plan.SourceLabel != "rime-ice" {
		t.Fatalf("source label = %q", plan.SourceLabel)
	}
	if plan.Version != "rime-ice-2026.07.05" {
		t.Fatalf("version = %q", plan.Version)
	}
	if !strings.HasSuffix(plan.ArtifactPath, filepath.Join("out", "dicts", "zh-CN.rime-ice-2026.07.05.json.gz")) {
		t.Fatalf("artifact path = %q", plan.ArtifactPath)
	}
	if plan.RepoSlug != "iDvel/rime-ice" {
		t.Fatalf("repo slug = %q", plan.RepoSlug)
	}
}

func TestCloneURLsPreferMirrorsAndDedupe(t *testing.T) {
	got := cloneURLs("https://github.com/iDvel/rime-ice.git", []string{
		"https://mirror.example/{repo}.git",
		"https://proxy.example/{url}",
		"https://github.com/iDvel/rime-ice.git",
	})
	want := []string{
		"https://mirror.example/iDvel/rime-ice.git",
		"https://proxy.example/https://github.com/iDvel/rime-ice.git",
		"https://github.com/iDvel/rime-ice.git",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("clone URLs = %#v, want %#v", got, want)
	}
}

func TestInputFilesForPreset(t *testing.T) {
	got := inputFilesForPreset("rime-ice-source")
	want := []string{"rime_ice.dict.yaml", "symbols_v.yaml", "symbols_caps_v.yaml", "opencc/emoji.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rime ice inputs = %#v, want %#v", got, want)
	}
	if got := inputFilesForPreset("shurufa233-github"); len(got) != 0 {
		t.Fatalf("manifest source should not have raw input files: %#v", got)
	}
}

func TestRepositoryURLAndSlug(t *testing.T) {
	raw, err := repositoryURLForTest("https://github.com/rime/rime-luna-pinyin")
	if err != nil {
		t.Fatal(err)
	}
	if raw != "https://github.com/rime/rime-luna-pinyin.git" {
		t.Fatalf("repo url = %q", raw)
	}
	slug, err := repositorySlug(raw)
	if err != nil {
		t.Fatal(err)
	}
	if slug != "rime/rime-luna-pinyin" {
		t.Fatalf("slug = %q", slug)
	}
}

func repositoryURLForTest(homepage string) (string, error) {
	plan, err := buildPlan(options{
		Preset:       "rime-luna-source",
		WorkDir:      "cache",
		OutDir:       "out",
		ManifestPath: filepath.Join("out", "manifest.json"),
		Version:      "test",
	}, time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		return "", err
	}
	plan.Source.Homepage = homepage
	return repositoryURL(plan.Source)
}
