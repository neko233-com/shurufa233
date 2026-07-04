package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/neko233-com/shurufa233/core/engine"
)

func main() {
	language := flag.String("language", "zh-CN", "dictionary language")
	version := flag.String("version", "rime-import", "dictionary version")
	source := flag.String("source", "rime", "source label")
	outPath := flag.String("out", "", "output JSON path, stdout when empty")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: shurufa-dictimport [flags] file.dict.yaml ...")
		os.Exit(2)
	}

	var entries []engine.Entry
	for _, path := range flag.Args() {
		file, err := os.Open(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		parsed, err := parseRimeDictionary(file, *source)
		closeErr := file.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if closeErr != nil {
			fmt.Fprintln(os.Stderr, closeErr)
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
	data = append(data, '\n')
	if *outPath == "" {
		_, _ = os.Stdout.Write(data)
		return
	}
	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseRimeDictionary(reader io.Reader, source string) ([]engine.Entry, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	inBody := false
	var entries []engine.Entry
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !inBody {
			if line == "..." {
				inBody = true
			}
			continue
		}
		entry, ok := parseRimeEntry(line, source)
		if ok {
			entries = append(entries, entry)
		}
	}
	return entries, scanner.Err()
}

func parseRimeEntry(line string, source string) (engine.Entry, bool) {
	fields := strings.Split(line, "\t")
	if len(fields) < 2 {
		return engine.Entry{}, false
	}
	text := strings.TrimSpace(fields[0])
	reading := normalizeRimeReading(fields[1])
	if text == "" || reading == "" {
		return engine.Entry{}, false
	}
	weight := 1000
	if len(fields) >= 3 {
		if parsed, err := strconv.Atoi(strings.TrimSpace(fields[2])); err == nil && parsed > 0 {
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

func normalizeRimeReading(value string) string {
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
