package engine

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

const rimeCustomPhraseBaseWeight = 50000

func ParseRimeCustomPhrases(data []byte) ([]Entry, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	entries := []Entry{}
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || line == "---" || line == "..." {
			continue
		}
		if strings.Contains(line, ":") && !strings.Contains(line, "\t") {
			key := strings.TrimSpace(strings.SplitN(line, ":", 2)[0])
			switch key {
			case "name", "version", "sort", "use_preset_vocabulary", "columns", "encoder", "import_tables":
				continue
			}
		}
		entry, ok := parseRimeCustomPhraseLine(line)
		if !ok {
			return nil, fmt.Errorf("invalid custom phrase row at line %d", lineNumber)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return normalizeUserPhraseEntries(entries), nil
}

func ParseRimeUserDB(data []byte) (map[string]int, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scores := map[string]int{}
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || line == "---" || line == "..." {
			continue
		}
		reading, text, score, ok := parseRimeUserDBLine(line)
		if !ok {
			return nil, fmt.Errorf("invalid Rime userdb row at line %d", lineNumber)
		}
		key := reading + "|" + text
		scores[key] += score
		if scores[key] > 1000000 {
			scores[key] = 1000000
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return scores, nil
}

func FormatRimeCustomPhrases(entries []Entry) string {
	entries = normalizeUserPhraseEntries(entries)
	var builder strings.Builder
	builder.WriteString("# Rime custom_phrase.txt exported by shurufa233\n")
	builder.WriteString("# columns: text\tcode\tweight\n")
	for _, entry := range entries {
		builder.WriteString(sanitizeRimeCustomPhraseField(entry.Text))
		builder.WriteByte('\t')
		builder.WriteString(sanitizeRimeCustomPhraseField(entry.Reading))
		builder.WriteByte('\t')
		builder.WriteString(strconv.Itoa(exportRimeCustomPhraseWeight(entry.Weight)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func parseRimeUserDBLine(line string) (string, string, int, bool) {
	fields := strings.Fields(line)
	metadataStart := len(fields)
	for i, field := range fields {
		if isRimeUserDBMetadataField(field) {
			metadataStart = i
			break
		}
	}
	if metadataStart < 2 {
		return "", "", 0, false
	}
	text := strings.TrimSpace(fields[metadataStart-1])
	reading := normalizeReading(strings.Join(fields[:metadataStart-1], ""))
	if reading == "" || text == "" {
		return "", "", 0, false
	}
	score := rimeUserDBScore(fields[metadataStart:])
	return reading, text, score, true
}

func isRimeUserDBMetadataField(field string) bool {
	key, _, ok := strings.Cut(strings.TrimSpace(field), "=")
	if !ok {
		return false
	}
	switch key {
	case "c", "d", "t":
		return true
	default:
		return false
	}
}

func rimeUserDBScore(metadata []string) int {
	count := 1.0
	decay := 1.0
	for _, field := range metadata {
		key, value, ok := strings.Cut(strings.TrimSpace(field), "=")
		if !ok || value == "" {
			continue
		}
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || parsed <= 0 {
			continue
		}
		switch key {
		case "c":
			count = parsed
		case "d":
			decay = parsed
		}
	}
	score := int(count*decay*25 + 0.5)
	if score < 1 {
		return 1
	}
	if score > 1000000 {
		return 1000000
	}
	return score
}

func parseRimeCustomPhraseLine(line string) (Entry, bool) {
	fields := splitRimeCustomPhraseFields(line)
	if len(fields) < 2 {
		return Entry{}, false
	}
	text := strings.TrimSpace(fields[0])
	reading := normalizeReading(fields[1])
	if text == "" || reading == "" {
		return Entry{}, false
	}
	weight := rimeCustomPhraseBaseWeight
	if len(fields) > 2 {
		if parsed, ok := parseRimeCustomPhraseWeight(fields[2]); ok {
			weight = importRimeCustomPhraseWeight(parsed)
		}
	}
	return Entry{
		Reading: reading,
		Text:    text,
		Weight:  weight,
		Comment: "Rime custom_phrase",
	}, true
}

func splitRimeCustomPhraseFields(line string) []string {
	if strings.Contains(line, "\t") {
		parts := strings.Split(line, "\t")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			out = append(out, strings.TrimSpace(part))
		}
		return out
	}
	return strings.Fields(line)
}

func parseRimeCustomPhraseWeight(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if index := strings.Index(value, "#"); index >= 0 {
		value = strings.TrimSpace(value[:index])
	}
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func importRimeCustomPhraseWeight(weight int) int {
	if weight >= rimeCustomPhraseBaseWeight {
		return weight
	}
	return rimeCustomPhraseBaseWeight + weight
}

func exportRimeCustomPhraseWeight(weight int) int {
	if weight <= 0 {
		return 1
	}
	if weight > rimeCustomPhraseBaseWeight {
		return weight - rimeCustomPhraseBaseWeight
	}
	return weight
}

func sanitizeRimeCustomPhraseField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}
