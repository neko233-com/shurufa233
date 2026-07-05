package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/neko233-com/shurufa233/core/engine"
)

type stringList []string

func (list *stringList) String() string {
	return strings.Join(*list, ",")
}

func (list *stringList) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			*list = append(*list, part)
		}
	}
	return nil
}

type options struct {
	Preset         string
	WorkDir        string
	OutDir         string
	ManifestPath   string
	Version        string
	Channel        string
	BaseURL        string
	SourceRef      string
	MissingImports string
	SkipPull       bool
	DictImport     string
	DictManifest   string
	Git            string
	Mirrors        []string
}

type syncPlan struct {
	Source       engine.DictionarySourcePreset
	SourceLabel  string
	RepoURL      string
	RepoSlug     string
	CheckoutDir  string
	Version      string
	ArtifactPath string
	ManifestPath string
}

type syncReport struct {
	Preset       string   `json:"preset"`
	SourceURL    string   `json:"sourceUrl"`
	SourceCommit string   `json:"sourceCommit"`
	Version      string   `json:"version"`
	Artifact     string   `json:"artifact"`
	Manifest     string   `json:"manifest"`
	Inputs       []string `json:"inputs"`
	UpdatedAt    string   `json:"updatedAt"`
}

func main() {
	opts := parseFlags()
	report, err := run(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func parseFlags() options {
	var mirrors stringList
	preset := flag.String("preset", "rime-ice-source", "dictionary source preset id")
	workDir := flag.String("workdir", filepath.Join(".cache", "dictionaries"), "checkout/cache directory for upstream repositories")
	outDir := flag.String("out-dir", filepath.Join("data", "dictionaries"), "output directory for converted dictionary artifacts")
	manifestPath := flag.String("manifest", filepath.Join("data", "dictionaries", "dictionary-manifest.json"), "output shurufa233 dictionary manifest path")
	version := flag.String("version", "", "dictionary release version; defaults to <source>-YYYY.MM.DD")
	channel := flag.String("channel", "stable", "dictionary release channel")
	baseURL := flag.String("base-url", "https://github.com/neko233-com/shurufa233/releases/latest/download", "base URL used in generated manifests")
	sourceRef := flag.String("ref", "", "optional upstream branch, tag, or commit to checkout after fetch/clone")
	missingImports := flag.String("missing-imports", "error", "Rime import_tables missing policy: error, warn, or skip")
	skipPull := flag.Bool("skip-pull", false, "reuse an existing checkout without fetching")
	dictImport := flag.String("dictimport", "", "path to shurufa-dictimport; defaults to sibling executable, PATH, or go run ./cmd/dictimport in a source checkout")
	dictManifest := flag.String("dictmanifest", "", "path to shurufa-dictmanifest; defaults to sibling executable, PATH, or go run ./cmd/dictmanifest in a source checkout")
	gitPath := flag.String("git", "git", "git executable path")
	flag.Var(&mirrors, "mirror-url", "mirror clone URL or template; repeatable. Templates may use {url} and {repo}.")
	flag.Parse()
	return options{
		Preset:         *preset,
		WorkDir:        *workDir,
		OutDir:         *outDir,
		ManifestPath:   *manifestPath,
		Version:        *version,
		Channel:        *channel,
		BaseURL:        *baseURL,
		SourceRef:      *sourceRef,
		MissingImports: *missingImports,
		SkipPull:       *skipPull,
		DictImport:     *dictImport,
		DictManifest:   *dictManifest,
		Git:            *gitPath,
		Mirrors:        mirrors,
	}
}

func run(opts options) (syncReport, error) {
	plan, err := buildPlan(opts, time.Now().UTC())
	if err != nil {
		return syncReport{}, err
	}
	if err := syncRepository(opts, plan); err != nil {
		return syncReport{}, err
	}
	commit, err := gitOutput(opts.Git, plan.CheckoutDir, "rev-parse", "HEAD")
	if err != nil {
		return syncReport{}, err
	}
	inputs, err := existingInputFiles(plan)
	if err != nil {
		return syncReport{}, err
	}
	if err := os.MkdirAll(filepath.Dir(plan.ArtifactPath), 0o755); err != nil {
		return syncReport{}, err
	}
	if err := os.MkdirAll(filepath.Dir(plan.ManifestPath), 0o755); err != nil {
		return syncReport{}, err
	}

	importTool := resolveTool(opts.DictImport, "shurufa-dictimport", "./cmd/dictimport")
	importArgs := []string{
		"-language", "zh-CN",
		"-version", plan.Version,
		"-source", plan.SourceLabel,
		"-missing-imports", opts.MissingImports,
		"-out", plan.ArtifactPath,
	}
	importArgs = append(importArgs, inputs...)
	if err := runTool(importTool, importArgs, "convert Rime dictionaries"); err != nil {
		return syncReport{}, err
	}

	manifestTool := resolveTool(opts.DictManifest, "shurufa-dictmanifest", "./cmd/dictmanifest")
	convertCommand := renderConvertCommand(importTool, importArgs)
	manifestArgs := []string{
		"-version", plan.Version,
		"-channel", strings.TrimSpace(opts.Channel),
		"-source-preset", plan.Source.ID,
		"-source-url", plan.Source.Homepage,
		"-source-commit", commit,
		"-license", plan.Source.License,
		"-convert-command", convertCommand,
		"-base-url", strings.TrimSpace(opts.BaseURL),
		"-out", plan.ManifestPath,
		plan.ArtifactPath,
	}
	if err := runTool(manifestTool, manifestArgs, "write dictionary manifest"); err != nil {
		return syncReport{}, err
	}

	return syncReport{
		Preset:       plan.Source.ID,
		SourceURL:    plan.Source.Homepage,
		SourceCommit: commit,
		Version:      plan.Version,
		Artifact:     plan.ArtifactPath,
		Manifest:     plan.ManifestPath,
		Inputs:       inputs,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func buildPlan(opts options, now time.Time) (syncPlan, error) {
	source, ok := engine.DictionarySourceByID(strings.TrimSpace(opts.Preset))
	if !ok {
		return syncPlan{}, fmt.Errorf("unknown dictionary source preset %q", opts.Preset)
	}
	if source.Installable {
		return syncPlan{}, fmt.Errorf("%s is already a manifest source; choose a rime-source/opencc-source preset", source.ID)
	}
	repoURL, err := repositoryURL(source)
	if err != nil {
		return syncPlan{}, err
	}
	repoSlug, err := repositorySlug(repoURL)
	if err != nil {
		return syncPlan{}, err
	}
	version := strings.TrimSpace(opts.Version)
	sourceLabel := sourceLabelFromPreset(source.ID)
	if version == "" {
		version = sourceLabel + "-" + now.Format("2006.01.02")
	}
	outDir := cleanOrDefault(opts.OutDir, filepath.Join("data", "dictionaries"))
	artifactName := fmt.Sprintf("zh-CN.%s.json.gz", safeFilename(version))
	return syncPlan{
		Source:       source,
		SourceLabel:  sourceLabel,
		RepoURL:      repoURL,
		RepoSlug:     repoSlug,
		CheckoutDir:  filepath.Join(cleanOrDefault(opts.WorkDir, filepath.Join(".cache", "dictionaries")), safeFilename(repoSlug)),
		Version:      version,
		ArtifactPath: filepath.Join(outDir, artifactName),
		ManifestPath: cleanOrDefault(opts.ManifestPath, filepath.Join("data", "dictionaries", "dictionary-manifest.json")),
	}, nil
}

func syncRepository(opts options, plan syncPlan) error {
	if opts.SkipPull {
		if _, err := os.Stat(filepath.Join(plan.CheckoutDir, ".git")); err != nil {
			return fmt.Errorf("-skip-pull requested but %s is not a git checkout", plan.CheckoutDir)
		}
		return nil
	}
	if _, err := os.Stat(filepath.Join(plan.CheckoutDir, ".git")); err == nil {
		if err := gitRun(opts.Git, plan.CheckoutDir, "fetch", "--tags", "origin"); err != nil {
			return err
		}
		if strings.TrimSpace(opts.SourceRef) != "" {
			return gitRun(opts.Git, plan.CheckoutDir, "checkout", strings.TrimSpace(opts.SourceRef))
		}
		return gitRun(opts.Git, plan.CheckoutDir, "pull", "--ff-only")
	}
	if err := os.MkdirAll(filepath.Dir(plan.CheckoutDir), 0o755); err != nil {
		return err
	}
	var lastErr error
	for _, cloneURL := range cloneURLs(plan.RepoURL, opts.Mirrors) {
		_ = os.RemoveAll(plan.CheckoutDir)
		args := []string{"clone", cloneURL, plan.CheckoutDir}
		if err := runCommand(opts.Git, args, "", "clone "+cloneURL); err != nil {
			lastErr = err
			continue
		}
		if strings.TrimSpace(opts.SourceRef) != "" {
			return gitRun(opts.Git, plan.CheckoutDir, "checkout", strings.TrimSpace(opts.SourceRef))
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("no clone URLs configured")
	}
	return lastErr
}

func existingInputFiles(plan syncPlan) ([]string, error) {
	relative := inputFilesForPreset(plan.Source.ID)
	if len(relative) == 0 {
		return nil, fmt.Errorf("no sync file plan for source preset %s", plan.Source.ID)
	}
	var out []string
	for index, rel := range relative {
		path := filepath.Join(plan.CheckoutDir, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			if index == 0 {
				return nil, fmt.Errorf("required upstream file missing: %s", path)
			}
			continue
		}
		out = append(out, path)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no upstream input files found under %s", plan.CheckoutDir)
	}
	return out, nil
}

func inputFilesForPreset(id string) []string {
	switch id {
	case "rime-luna-source":
		return []string{"luna_pinyin.dict.yaml"}
	case "rime-ice-source":
		return []string{"rime_ice.dict.yaml", "symbols_v.yaml", "symbols_caps_v.yaml", "opencc/emoji.txt"}
	case "rime-emoji-source":
		return []string{"opencc/emoji_word.txt", "opencc/emoji_category.txt"}
	default:
		return nil
	}
}

func repositoryURL(source engine.DictionarySourcePreset) (string, error) {
	homepage := strings.TrimSpace(source.Homepage)
	if homepage == "" {
		return "", fmt.Errorf("source %s has no homepage", source.ID)
	}
	if strings.HasSuffix(homepage, ".git") {
		return homepage, nil
	}
	parsed, err := url.Parse(homepage)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("source homepage is not an absolute URL: %s", homepage)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + ".git"
	return parsed.String(), nil
}

func cloneURLs(canonical string, mirrors []string) []string {
	out := make([]string, 0, len(mirrors)+1)
	for _, mirror := range mirrors {
		if rendered := renderMirrorURL(mirror, canonical); rendered != "" {
			out = append(out, rendered)
		}
	}
	out = append(out, canonical)
	return dedupeStrings(out)
}

func renderMirrorURL(template string, canonical string) string {
	template = strings.TrimSpace(template)
	if template == "" {
		return ""
	}
	repo, _ := repositorySlug(canonical)
	if strings.Contains(template, "{url}") || strings.Contains(template, "{repo}") {
		rendered := strings.ReplaceAll(template, "{url}", canonical)
		rendered = strings.ReplaceAll(rendered, "{repo}", repo)
		return rendered
	}
	return template
}

func repositorySlug(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	path := strings.Trim(parsed.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("cannot infer owner/repo from %s", rawURL)
	}
	return parts[len(parts)-2] + "/" + parts[len(parts)-1], nil
}

func sourceLabelFromPreset(id string) string {
	return strings.TrimSuffix(strings.TrimSpace(id), "-source")
}

func safeFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "dictionary"
	}
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '.', r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('-')
		}
	}
	out := strings.Trim(builder.String(), "-.")
	if out == "" {
		return "dictionary"
	}
	return out
}

func cleanOrDefault(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return filepath.Clean(value)
}

func resolveTool(override string, executable string, goPackage string) []string {
	if strings.TrimSpace(override) != "" {
		return []string{strings.TrimSpace(override)}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, executable+exeSuffix())
		if _, err := os.Stat(candidate); err == nil {
			return []string{candidate}
		}
	}
	if path, err := exec.LookPath(executable); err == nil {
		return []string{path}
	}
	if root := findModuleRoot(); root != "" {
		return []string{"go", "-C", root, "run", goPackage}
	}
	return []string{executable}
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			return ""
		}
		dir = next
	}
}

func renderConvertCommand(tool []string, args []string) string {
	parts := append([]string{}, tool...)
	parts = append(parts, args...)
	for index, part := range parts {
		if strings.ContainsAny(part, " \t\"") {
			parts[index] = `"` + strings.ReplaceAll(part, `"`, `\"`) + `"`
		}
	}
	return strings.Join(parts, " ")
}

func runTool(tool []string, args []string, label string) error {
	if len(tool) == 0 {
		return errors.New("empty tool command")
	}
	name := tool[0]
	fullArgs := append([]string{}, tool[1:]...)
	fullArgs = append(fullArgs, args...)
	return runCommand(name, fullArgs, "", label)
}

func gitRun(git string, dir string, args ...string) error {
	return runCommand(git, args, dir, "git "+strings.Join(args, " "))
}

func gitOutput(git string, dir string, args ...string) (string, error) {
	command := exec.Command(git, args...)
	command.Dir = dir
	out, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func runCommand(name string, args []string, dir string, label string) error {
	command := exec.Command(name, args...)
	if dir != "" {
		command.Dir = dir
	}
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", label, err)
	}
	return nil
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
