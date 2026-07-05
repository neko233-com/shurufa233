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
	"unicode"
	"unicode/utf8"

	"github.com/neko233-com/shurufa233/core/engine"
)

const customPhraseWeightBase = 50000
const rimeSymbolWeightBase = 48000
const openCCWeightBase = 47000

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
	readingsByText map[string][]string
}

func newRimeCollector(source string, includeImports bool, missingPolicy string, warnTo io.Writer) *rimeCollector {
	return &rimeCollector{
		source:         source,
		includeImports: includeImports,
		missingPolicy:  strings.ToLower(strings.TrimSpace(missingPolicy)),
		warnTo:         warnTo,
		visited:        make(map[string]bool),
		readingsByText: make(map[string][]string),
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
	entries, imports, parseErr := parseRimeDocumentWithHints(file, collector.source, rimeParseHints{
		ReadingsByText: collector.readingsByText,
	})
	closeErr := file.Close()
	if parseErr != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, parseErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, closeErr)
	}

	if !collector.includeImports {
		collector.indexReadings(entries)
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
	collector.indexReadings(out)
	return out, nil
}

func (collector *rimeCollector) indexReadings(entries []engine.Entry) {
	for _, entry := range entries {
		text := strings.TrimSpace(entry.Text)
		reading := normalizeRimeReading(entry.Reading)
		if text == "" || reading == "" {
			continue
		}
		if !containsString(collector.readingsByText[text], reading) {
			collector.readingsByText[text] = append(collector.readingsByText[text], reading)
		}
	}
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
	return parseRimeDocumentWithHints(reader, source, rimeParseHints{})
}

type rimeParseHints struct {
	ReadingsByText map[string][]string
}

func parseRimeDocumentWithHints(reader io.Reader, source string, hints rimeParseHints) ([]engine.Entry, []string, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	inBody := false
	sawYAMLHeader := false
	columns := defaultRimeColumns()
	var entries []engine.Entry
	var imports []string
	importList := false
	columnList := false
	symbolList := rimeSymbolListState{}
	symbolSection := false
	symbolSectionIndent := 0
	for scanner.Scan() {
		rawLine := strings.TrimPrefix(scanner.Text(), "\ufeff")
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		indent := leadingYAMLIndent(rawLine)
		if symbolSection && indent <= symbolSectionIndent && !isRimeSymbolSectionLine(line) {
			symbolSection = false
		}
		if line == "---" {
			sawYAMLHeader = true
			importList = false
			columnList = false
			symbolList = rimeSymbolListState{}
			symbolSection = false
			continue
		}
		if !inBody {
			if line == "..." {
				inBody = true
				importList = false
				columnList = false
				symbolList = rimeSymbolListState{}
				symbolSection = false
				continue
			}
			if isRimeSymbolSectionLine(line) {
				symbolSection = true
				symbolSectionIndent = indent
				continue
			}
			if symbolList.active {
				if entry, ok := parseRimeSymbolListItem(line, source, symbolList.reading, symbolList.index); ok {
					entries = append(entries, entry)
					symbolList.index++
					continue
				}
				symbolList = rimeSymbolListState{}
			}
			nextImports, ok := parseImportTablesLine(line)
			if ok {
				imports = append(imports, nextImports...)
				importList = len(nextImports) == 0
				columnList = false
				continue
			}
			if importList {
				if table, ok := parseImportTableItem(line); ok {
					imports = append(imports, table)
					continue
				}
				importList = false
			}
			nextColumns, ok := parseColumnsLine(line)
			if ok {
				if len(nextColumns) > 0 {
					columns = columnsFromNames(nextColumns)
				} else {
					columns = blankRimeColumns()
				}
				columnList = len(nextColumns) == 0
				continue
			}
			if columnList {
				if column, ok := parseColumnItem(line); ok {
					columns = columns.withColumn(column)
					continue
				}
				columnList = false
			}
			if nextSymbolList, ok := parseRimeSymbolListStart(line, symbolSection); ok {
				symbolList = nextSymbolList
				continue
			}
			if symbolEntries, ok := parseRimeSymbolLine(line, source, symbolSection); ok {
				entries = append(entries, symbolEntries...)
				continue
			}
			if openCCEntries, ok := parseOpenCCTextLine(line, source, hints); ok {
				entries = append(entries, openCCEntries...)
				continue
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
		nextEntries, ok := parseRimeEntriesWithColumns(line, source, columns, hints)
		if ok {
			entries = append(entries, nextEntries...)
		}
	}
	return entries, imports, scanner.Err()
}

func parseOpenCCTextLine(line string, source string, hints rimeParseHints) ([]engine.Entry, bool) {
	if !strings.Contains(line, "\t") {
		return nil, false
	}
	fields := strings.Split(line, "\t")
	if len(fields) < 2 {
		return nil, false
	}
	key := strings.TrimSpace(fields[0])
	value := strings.TrimSpace(strings.Join(fields[1:], " "))
	if key == "" || value == "" || !strings.HasPrefix(value, key) {
		return nil, false
	}
	readings := readingsForOpenCCKey(key, hints)
	if len(readings) == 0 {
		return nil, false
	}
	values := splitOpenCCValues(value)
	if len(values) == 0 {
		return nil, false
	}
	var entries []engine.Entry
	for valueIndex, text := range values {
		if text == key || !looksLikeOpenCCSupplement(text) {
			continue
		}
		for _, reading := range readings {
			entries = append(entries, engine.Entry{
				Reading: reading,
				Text:    text,
				Kind:    symbolKind(text),
				Source:  source,
				Comment: key,
				Weight:  openCCWeightBase - valueIndex,
			})
		}
	}
	return entries, len(entries) > 0
}

func readingsForOpenCCKey(key string, hints rimeParseHints) []string {
	var readings []string
	if reading := normalizeRimeReading(key); reading != "" {
		readings = append(readings, reading)
	}
	for _, reading := range hints.ReadingsByText[key] {
		reading = normalizeRimeReading(reading)
		if reading != "" && !containsString(readings, reading) {
			readings = append(readings, reading)
		}
	}
	return readings
}

func splitOpenCCValues(value string) []string {
	fields := strings.Fields(value)
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			values = append(values, field)
		}
	}
	return values
}

func looksLikeOpenCCSupplement(text string) bool {
	kind := symbolKind(text)
	if kind == "emoji" || kind == "kaomoji" {
		return true
	}
	for _, r := range text {
		if unicode.IsSymbol(r) || unicode.IsPunct(r) {
			return true
		}
	}
	return false
}

type rimeSymbolListState struct {
	active  bool
	reading string
	index   int
}

func parseRimeSymbolListStart(line string, allowVMode bool) (rimeSymbolListState, bool) {
	key, value, ok := splitYAMLKeyValue(line)
	if !ok || strings.TrimSpace(value) != "" {
		return rimeSymbolListState{}, false
	}
	key = strings.TrimSpace(strings.Trim(key, `"'`))
	reading := normalizeRimeSymbolReading(key, allowVMode)
	if reading == "" {
		return rimeSymbolListState{}, false
	}
	return rimeSymbolListState{active: true, reading: reading}, true
}

func parseRimeSymbolListItem(line string, source string, reading string, index int) (engine.Entry, bool) {
	if !strings.HasPrefix(line, "-") {
		return engine.Entry{}, false
	}
	text := cleanYAMLScalar(stripYAMLInlineComment(strings.TrimSpace(strings.TrimPrefix(line, "-"))))
	if text == "" {
		return engine.Entry{}, false
	}
	return engine.Entry{
		Reading: reading,
		Text:    text,
		Kind:    symbolKind(text),
		Source:  source,
		Comment: symbolComment(reading, text),
		Weight:  rimeSymbolWeightBase - index,
	}, true
}

func parseRimeSymbolLine(line string, source string, allowVMode bool) ([]engine.Entry, bool) {
	key, value, ok := splitYAMLKeyValue(line)
	if !ok {
		return nil, false
	}
	key = strings.TrimSpace(strings.Trim(key, `"'`))
	reading := normalizeRimeSymbolReading(key, allowVMode)
	if reading == "" {
		return nil, false
	}
	values := parseYAMLFlowValues(value)
	if len(values) == 0 {
		return nil, false
	}
	entries := make([]engine.Entry, 0, len(values))
	for index, text := range values {
		text = cleanYAMLScalar(text)
		if text == "" {
			continue
		}
		entries = append(entries, engine.Entry{
			Reading: reading,
			Text:    text,
			Kind:    symbolKind(text),
			Source:  source,
			Comment: symbolComment(reading, text),
			Weight:  rimeSymbolWeightBase - index,
		})
	}
	return entries, len(entries) > 0
}

func normalizeRimeSymbolReading(key string, allowVMode bool) string {
	key = strings.TrimSpace(strings.Trim(key, `"'`))
	if strings.HasPrefix(key, "/") {
		return normalizeRimeReading(strings.TrimLeft(key, "/"))
	}
	if allowVMode && strings.HasPrefix(strings.ToLower(key), "v") && len([]rune(key)) > 1 {
		return normalizeRimeReading(string([]rune(key)[1:]))
	}
	return ""
}

func isRimeSymbolSectionLine(line string) bool {
	key, value, ok := splitYAMLKeyValue(line)
	if !ok || strings.TrimSpace(value) != "" {
		return false
	}
	key = strings.TrimSpace(strings.Trim(key, `"'`))
	return key == "symbols" || strings.HasSuffix(key, "/symbols") ||
		strings.Contains(key, "punctuator/symbols")
}

func leadingYAMLIndent(line string) int {
	count := 0
	for _, r := range line {
		if r != ' ' && r != '\t' {
			return count
		}
		count++
	}
	return count
}

func splitYAMLKeyValue(line string) (string, string, bool) {
	var quote rune
	for index, r := range line {
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == ':' {
			return strings.TrimSpace(line[:index]), strings.TrimSpace(line[index+len(":"):]), true
		}
	}
	return "", "", false
}

func parseYAMLFlowValues(value string) []string {
	value = stripYAMLInlineComment(value)
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	}
	if value == "" {
		return nil
	}
	var values []string
	var builder strings.Builder
	var quote rune
	escaped := false
	for _, r := range value {
		if escaped {
			builder.WriteRune(r)
			escaped = false
			continue
		}
		if quote != 0 {
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
				builder.WriteRune(r)
				continue
			}
			builder.WriteRune(r)
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
			builder.WriteRune(r)
		case ',':
			if item := strings.TrimSpace(builder.String()); item != "" {
				values = append(values, item)
			}
			builder.Reset()
		default:
			builder.WriteRune(r)
		}
	}
	if item := strings.TrimSpace(builder.String()); item != "" {
		values = append(values, item)
	}
	return values
}

func stripYAMLInlineComment(value string) string {
	var quote rune
	for index, r := range value {
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == '#' {
			return strings.TrimSpace(value[:index])
		}
	}
	return value
}

func cleanYAMLScalar(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		left, leftSize := utf8.DecodeRuneInString(value)
		right, _ := utf8.DecodeLastRuneInString(value)
		if (left == '\'' && right == '\'') || (left == '"' && right == '"') {
			value = value[leftSize : len(value)-leftSize]
		}
	}
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, `\'`, `'`)
	value = strings.ReplaceAll(value, `\\`, `\`)
	return strings.TrimSpace(value)
}

func symbolKind(text string) string {
	for _, r := range text {
		if isEmojiRune(r) {
			return "emoji"
		}
	}
	if strings.ContainsAny(text, "()[]{}<>/\\|_=^;:-~") && len([]rune(text)) >= 3 {
		return "kaomoji"
	}
	return "symbol"
}

func isEmojiRune(r rune) bool {
	return r >= 0x1F000 && r <= 0x1FAFF || r >= 0x2600 && r <= 0x27BF
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

func parseColumnsLine(line string) ([]string, bool) {
	if line != "columns:" && !strings.HasPrefix(line, "columns:") {
		return nil, false
	}
	value := strings.TrimSpace(strings.TrimPrefix(line, "columns:"))
	if value == "" {
		return nil, true
	}
	return normalizeColumnNames(parseYAMLFlowValues(value)), true
}

func parseColumnItem(line string) (string, bool) {
	if !strings.HasPrefix(line, "-") {
		return "", false
	}
	column := normalizeColumnName(strings.TrimSpace(strings.TrimPrefix(line, "-")))
	return column, column != ""
}

func normalizeColumnNames(values []string) []string {
	columns := make([]string, 0, len(values))
	for _, value := range values {
		if column := normalizeColumnName(value); column != "" {
			columns = append(columns, column)
		}
	}
	return columns
}

func normalizeColumnName(value string) string {
	value = cleanYAMLScalar(stripYAMLInlineComment(value))
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "text", "code", "weight", "stem", "comment", "comments", "annotation":
		return value
	default:
		return ""
	}
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
	return parseRimeEntryWithColumns(line, source, defaultRimeColumns())
}

func parseRimeEntryWithColumns(line string, source string, columns rimeColumns) (engine.Entry, bool) {
	entries, ok := parseRimeEntriesWithColumns(line, source, columns, rimeParseHints{})
	if !ok {
		return engine.Entry{}, false
	}
	return entries[0], true
}

func parseRimeEntriesWithColumns(line string, source string, columns rimeColumns, hints rimeParseHints) ([]engine.Entry, bool) {
	columns = columns.normalized()
	fields, ok := splitRimeEntryFields(line)
	if !ok {
		return nil, false
	}
	if len(fields) <= columns.textIndex {
		return nil, false
	}
	text := strings.TrimSpace(fields[columns.textIndex])
	if text == "" {
		return nil, false
	}
	readings := rimeReadingsFromFields(text, fields, columns, hints)
	if len(readings) == 0 {
		return nil, false
	}
	weight := 1000
	if columns.weightIndex >= 0 && len(fields) > columns.weightIndex {
		if parsed, err := strconv.Atoi(strings.TrimSpace(stripRimeInlineComment(fields[columns.weightIndex]))); err == nil && parsed > 0 {
			weight = parsed
		}
	}
	comment := ""
	if columns.commentIndex >= 0 && len(fields) > columns.commentIndex {
		comment = strings.TrimSpace(stripRimeInlineComment(fields[columns.commentIndex]))
	}
	entries := make([]engine.Entry, 0, len(readings))
	for _, reading := range readings {
		entries = append(entries, engine.Entry{
			Reading: reading,
			Text:    text,
			Source:  source,
			Comment: comment,
			Weight:  weight,
		})
	}
	return entries, true
}

func rimeReadingsFromFields(text string, fields []string, columns rimeColumns, hints rimeParseHints) []string {
	if columns.codeIndex >= 0 && len(fields) > columns.codeIndex {
		reading := normalizeRimeReading(stripRimeInlineComment(fields[columns.codeIndex]))
		if reading != "" {
			return []string{reading}
		}
	}
	return inferRimeReadings(text, hints)
}

func inferRimeReadings(text string, hints rimeParseHints) []string {
	if len(hints.ReadingsByText) == 0 {
		return nil
	}
	readings := uniqueNormalizedReadings(hints.ReadingsByText[text])
	if derived := deriveRimeReadingFromRunes(text, hints); derived != "" && !containsString(readings, derived) {
		readings = append(readings, derived)
	}
	return readings
}

func deriveRimeReadingFromRunes(text string, hints rimeParseHints) string {
	var builder strings.Builder
	for _, r := range text {
		readings := uniqueNormalizedReadings(hints.ReadingsByText[string(r)])
		if len(readings) == 0 {
			return ""
		}
		builder.WriteString(readings[0])
	}
	return builder.String()
}

func uniqueNormalizedReadings(values []string) []string {
	readings := make([]string, 0, len(values))
	for _, value := range values {
		reading := normalizeRimeReading(value)
		if reading != "" && !containsString(readings, reading) {
			readings = append(readings, reading)
		}
	}
	return readings
}

type rimeColumns struct {
	textIndex    int
	codeIndex    int
	weightIndex  int
	commentIndex int
	nextIndex    int
}

func defaultRimeColumns() rimeColumns {
	return rimeColumns{
		textIndex:    0,
		codeIndex:    1,
		weightIndex:  2,
		commentIndex: -1,
		nextIndex:    3,
	}
}

func columnsFromNames(names []string) rimeColumns {
	columns := blankRimeColumns()
	for _, name := range names {
		columns = columns.withColumn(name)
	}
	return columns.normalized()
}

func blankRimeColumns() rimeColumns {
	return rimeColumns{
		textIndex:    -1,
		codeIndex:    -1,
		weightIndex:  -1,
		commentIndex: -1,
		nextIndex:    0,
	}
}

func (columns rimeColumns) normalized() rimeColumns {
	if columns.textIndex < 0 {
		columns.textIndex = 0
	}
	return columns
}

func (columns rimeColumns) withColumn(name string) rimeColumns {
	switch normalizeColumnName(name) {
	case "text":
		columns.textIndex = columns.nextIndex
	case "code":
		columns.codeIndex = columns.nextIndex
	case "weight":
		columns.weightIndex = columns.nextIndex
	case "comment", "comments", "annotation":
		columns.commentIndex = columns.nextIndex
	case "stem":
	}
	columns.nextIndex++
	return columns
}

func symbolComment(reading string, text string) string {
	switch symbolKind(text) {
	case "emoji":
		return "表情"
	case "kaomoji":
		return "颜文字"
	default:
		if reading != "" {
			return "符号"
		}
		return ""
	}
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
			if out[index].Comment == "" && entry.Comment != "" {
				out[index].Comment = entry.Comment
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, entry)
	}
	return out
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
