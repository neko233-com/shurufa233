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
