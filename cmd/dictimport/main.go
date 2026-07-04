package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/neko233-com/shurufa233/core/engine"
)

const customPhraseWeightBase = 50000

func main() {
	language := flag.String("language", "zh-CN", "dictionary language")
	version := flag.String("version", "rime-import", "dictionary version")
	source := flag.String("source", "rime", "source label")
	outPath := flag.String("out", "", "output JSON path, stdout when empty")
	gzipOutput := flag.Bool("gzip", false, "write gzip-compressed JSON output; also enabled when -out ends with .gz")
	includeImports := flag.Bool("imports", true, "resolve Rime import_tables recursively")
	missingImports := flag.String("missing-imports", "error", "missing import_tables policy: error, warn, or skip")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: shurufa-dictimport [flags] file.dict.yaml ...")
		os.Exit(2)
	}

	var entries []engine.Entry
	collector := newRimeCollector(*source, *includeImports, *missingImports, os.Stderr)
	for _, path := range flag.Args() {
		parsed, err := collector.collect(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		entries = append(entries, parsed...)
	}

	dictionary := engine.DictionaryFile{
		Language: *language,
		Version:  *version,
		Entries:  mergeEntries(entries),
	}
	data, err := json.MarshalIndent(dictionary, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := writeDictionaryOutput(*outPath, data, *gzipOutput || strings.HasSuffix(strings.ToLower(*outPath), ".gz")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeDictionaryOutput(path string, data []byte, gzipEnabled bool) error {
	data = append(data, '\n')
	if gzipEnabled {
		compressed, err := gzipData(data)
		if err != nil {
			return err
		}
		data = compressed
	}
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func gzipData(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type rimeCollector struct {
	source         string
	includeImports bool
	missingPolicy  string
	warnTo         io.Writer
	visited        map[string]bool
}

func newRimeCollector(source string, includeImports bool, missingPolicy string, warnTo io.Writer) *rimeCollector {
	return &rimeCollector{
		source:         source,
		includeImports: includeImports,
		missingPolicy:  strings.ToLower(strings.TrimSpace(missingPolicy)),
		warnTo:         warnTo,
		visited:        make(map[string]bool),
	}
}

func (collector *rimeCollector) collect(path string) ([]engine.Entry, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return collector.collectResolved(resolved)
}

func (collector *rimeCollector) collectResolved(path string) ([]engine.Entry, error) {
	cleanPath := filepath.Clean(path)
	if collector.visited[cleanPath] {
		return nil, nil
	}
	collector.visited[cleanPath] = true

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, err
	}
	entries, imports, parseErr := parseRimeDocument(file, collector.source)
	closeErr := file.Close()
	if parseErr != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, parseErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, closeErr)
	}

	if !collector.includeImports {
		return entries, nil
	}
	var out []engine.Entry
	for _, table := range imports {
		importPath, ok, err := resolveImportTable(filepath.Dir(cleanPath), table)
		if err != nil {
			return nil, fmt.Errorf("%s import %q: %w", cleanPath, table, err)
		}
		if !ok {
			if err := collector.handleMissingImport(cleanPath, table); err != nil {
				return nil, err
			}
			continue
		}
		imported, err := collector.collectResolved(importPath)
		if err != nil {
			return nil, err
		}
		out = append(out, imported...)
	}
	out = append(out, entries...)
	return out, nil
}

func (collector *rimeCollector) handleMissingImport(path string, table string) error {
	message := fmt.Sprintf("%s imports missing table %q", path, table)
	switch collector.missingPolicy {
	case "skip":
		return nil
	case "warn":
		if collector.warnTo != nil {
			fmt.Fprintf(collector.warnTo, "warning: %s\n", message)
		}
		return nil
	case "", "error":
		return fmt.Errorf("%s", message)
	default:
		return fmt.Errorf("unknown -missing-imports value %q", collector.missingPolicy)
	}
}

func parseRimeDictionary(reader io.Reader, source string) ([]engine.Entry, error) {
	entries, _, err := parseRimeDocument(reader, source)
	return entries, err
}

func parseRimeDocument(reader io.Reader, source string) ([]engine.Entry, []string, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	inBody := false
	sawYAMLHeader := false
	var entries []engine.Entry
	var imports []string
	importList := false
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "---" {
			sawYAMLHeader = true
			importList = false
			continue
		}
		if !inBody {
			if line == "..." {
				inBody = true
				importList = false
				continue
			}
			nextImports, ok := parseImportTablesLine(line)
			if ok {
				imports = append(imports, nextImports...)
				importList = len(nextImports) == 0
				continue
			}
			if importList {
				if table, ok := parseImportTableItem(line); ok {
					imports = append(imports, table)
					continue
				}
				importList = false
			}
			if sawYAMLHeader {
				continue
			}
			entry, ok := parseRimeEntry(line, source)
			if ok {
				entry.Weight = customPhraseWeight(entry.Weight)
				entries = append(entries, entry)
			}
			continue
		}
		entry, ok := parseRimeEntry(line, source)
		if ok {
			entries = append(entries, entry)
		}
	}
	return entries, imports, scanner.Err()
}

func parseImportTablesLine(line string) ([]string, bool) {
	if line != "import_tables:" && !strings.HasPrefix(line, "import_tables:") {
		return nil, false
	}
	value := strings.TrimSpace(strings.TrimPrefix(line, "import_tables:"))
	if value == "" {
		return nil, true
	}
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
		if value == "" {
			return nil, true
		}
		parts := strings.Split(value, ",")
		imports := make([]string, 0, len(parts))
		for _, part := range parts {
			if table := normalizeImportTable(part); table != "" {
				imports = append(imports, table)
			}
		}
		return imports, true
	}
	if table := normalizeImportTable(value); table != "" {
		return []string{table}, true
	}
	return nil, true
}

func parseImportTableItem(line string) (string, bool) {
	if !strings.HasPrefix(line, "-") {
		return "", false
	}
	table := normalizeImportTable(strings.TrimSpace(strings.TrimPrefix(line, "-")))
	return table, table != ""
}

func normalizeImportTable(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	if index := strings.Index(value, "#"); index >= 0 {
		value = strings.TrimSpace(value[:index])
	}
	value = strings.Trim(value, `"'`)
	if value == "" {
		return ""
	}
	return filepath.Clean(filepath.FromSlash(value))
}

func resolveImportTable(baseDir string, table string) (string, bool, error) {
	if filepath.IsAbs(table) || filepath.VolumeName(table) != "" || strings.HasPrefix(table, "/") || strings.HasPrefix(table, `\`) {
		return "", false, fmt.Errorf("absolute import path is not allowed")
	}
	cleanTable := filepath.Clean(table)
	if cleanTable == "." || strings.HasPrefix(cleanTable, ".."+string(filepath.Separator)) || cleanTable == ".." {
		return "", false, fmt.Errorf("import path escapes dictionary directory")
	}
	candidates := []string{filepath.Join(baseDir, cleanTable)}
	if filepath.Ext(cleanTable) == "" {
		candidates = append(candidates,
			filepath.Join(baseDir, cleanTable+".dict.yaml"),
			filepath.Join(baseDir, cleanTable+".yaml"),
		)
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			resolved, err := filepath.Abs(candidate)
			if err != nil {
				return "", false, err
			}
			return filepath.Clean(resolved), true, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", false, err
		}
	}
	return "", false, nil
}

func parseRimeEntry(line string, source string) (engine.Entry, bool) {
	fields, ok := splitRimeEntryFields(line)
	if !ok {
		return engine.Entry{}, false
	}
	if len(fields) < 2 {
		return engine.Entry{}, false
	}
	text := strings.TrimSpace(fields[0])
	reading := normalizeRimeReading(stripRimeInlineComment(fields[1]))
	if text == "" || reading == "" {
		return engine.Entry{}, false
	}
	weight := 1000
	if len(fields) >= 3 {
		if parsed, err := strconv.Atoi(strings.TrimSpace(stripRimeInlineComment(fields[2]))); err == nil && parsed > 0 {
			weight = parsed
		}
	}
	return engine.Entry{
		Reading: reading,
		Text:    text,
		Source:  source,
		Weight:  weight,
	}, true
}

func splitRimeEntryFields(line string) ([]string, bool) {
	if strings.Contains(line, "\t") {
		return strings.Split(line, "\t"), true
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil, false
	}
	codeIndex := len(fields) - 1
	var weight string
	if parsed, err := strconv.Atoi(stripRimeInlineComment(fields[codeIndex])); err == nil && parsed > 0 {
		weight = fields[codeIndex]
		codeIndex--
	}
	if codeIndex < 1 {
		return nil, false
	}
	text := strings.Join(fields[:codeIndex], " ")
	if strings.HasPrefix(text, "-") || strings.HasSuffix(text, ":") {
		return nil, false
	}
	out := []string{text, fields[codeIndex]}
	if weight != "" {
		out = append(out, weight)
	}
	return out, true
}

func customPhraseWeight(weight int) int {
	if weight <= 0 {
		weight = 1000
	}
	return customPhraseWeightBase + weight
}

func stripRimeInlineComment(value string) string {
	if index := strings.Index(value, "#"); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

var rimeReadingReplacer = strings.NewReplacer(
	"u:", "v",
	"U:", "v",
	"ü", "v",
	"Ü", "v",
	"ǖ", "v",
	"Ǘ", "v",
	"ǘ", "v",
	"Ǚ", "v",
	"ǚ", "v",
	"Ǜ", "v",
	"ǜ", "v",
	"Ǖ", "v",
	"ā", "a",
	"á", "a",
	"ǎ", "a",
	"à", "a",
	"Ā", "a",
	"Á", "a",
	"Ǎ", "a",
	"À", "a",
	"ē", "e",
	"é", "e",
	"ě", "e",
	"è", "e",
	"Ē", "e",
	"É", "e",
	"Ě", "e",
	"È", "e",
	"ī", "i",
	"í", "i",
	"ǐ", "i",
	"ì", "i",
	"Ī", "i",
	"Í", "i",
	"Ǐ", "i",
	"Ì", "i",
	"ō", "o",
	"ó", "o",
	"ǒ", "o",
	"ò", "o",
	"Ō", "o",
	"Ó", "o",
	"Ǒ", "o",
	"Ò", "o",
	"ū", "u",
	"ú", "u",
	"ǔ", "u",
	"ù", "u",
	"Ū", "u",
	"Ú", "u",
	"Ǔ", "u",
	"Ù", "u",
)

func normalizeRimeReading(value string) string {
	value = rimeReadingReplacer.Replace(value)
	var builder strings.Builder
	for _, r := range strings.ToLower(value) {
		if r >= 'a' && r <= 'z' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func mergeEntries(entries []engine.Entry) []engine.Entry {
	seen := make(map[string]int, len(entries))
	out := make([]engine.Entry, 0, len(entries))
	for _, entry := range entries {
		key := entry.Reading + "\x00" + entry.Text
		if index, ok := seen[key]; ok {
			if entry.Weight > out[index].Weight {
				out[index].Weight = entry.Weight
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, entry)
	}
	return out
}
