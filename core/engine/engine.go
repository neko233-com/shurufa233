package engine

import (
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

type Engine struct {
	mu     sync.RWMutex
	config Config
	dict   map[string][]Entry
	user   map[string]int
	buffer string
}

func DefaultConfig() Config {
	return Config{
		MaxCandidates: 7,
		FuzzyInitials: []string{
			"zh=z",
			"ch=c",
			"sh=s",
		},
		DoublePinyin: false,
		Language:     "zh-CN",
		Mode:         "zh",
		Skin: Skin{
			FontFamily: "Microsoft YaHei UI",
			FontSize:   15,
			Accent:     "#2563eb",
			Theme:      "system",
		},
		Update: Update{
			Channel: "stable",
			ManifestURLs: []string{
				"https://github.com/neko233-com/shurufa233/releases/latest/download/dictionary-manifest.json",
			},
			MirrorBaseURLs:   []string{},
			AutoCheck:        true,
			AutoApply:        false,
			InstalledVersion: "builtin",
		},
	}
}

func New(config Config) *Engine {
	if config.MaxCandidates <= 0 {
		config = DefaultConfig()
	}
	return &Engine{
		config: config,
		dict:   defaultDictionary(),
		user:   make(map[string]int),
	}
}

func (e *Engine) Configure(config Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if config.MaxCandidates <= 0 {
		config.MaxCandidates = DefaultConfig().MaxCandidates
	}
	e.config = config
}

func (e *Engine) AddEntries(entries []Entry) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, entry := range entries {
		entry.Reading = normalizeReading(entry.Reading)
		if entry.Reading == "" || entry.Text == "" {
			continue
		}
		e.dict[entry.Reading] = append(e.dict[entry.Reading], entry)
	}
}

func (e *Engine) LoadDictionary(reader io.Reader) (DictionaryFile, error) {
	var file DictionaryFile
	if err := json.NewDecoder(reader).Decode(&file); err != nil {
		return file, err
	}
	e.AddEntries(file.Entries)
	return file, nil
}

func (e *Engine) InputKey(key rune) State {
	e.mu.Lock()
	defer e.mu.Unlock()
	if unicode.IsLetter(key) {
		e.buffer += strings.ToLower(string(key))
	}
	return e.stateLocked("")
}

func (e *Engine) Backspace() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.buffer != "" {
		runes := []rune(e.buffer)
		e.buffer = string(runes[:len(runes)-1])
	}
	return e.stateLocked("")
}

func (e *Engine) Clear() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.buffer = ""
	return e.stateLocked("")
}

func (e *Engine) Preview(input string) State {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.buffer = normalizeReading(input)
	return e.stateLocked("")
}

func (e *Engine) Select(index int) (State, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	candidates := e.candidatesLocked()
	if index < 0 || index >= len(candidates) {
		return e.stateLocked(""), errors.New("candidate index out of range")
	}
	selected := candidates[index]
	e.user[selected.Reading+"|"+selected.Text] += 25
	e.buffer = ""
	return e.stateLocked(selected.Text), nil
}

func (e *Engine) State() State {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stateLocked("")
}

func (e *Engine) UserScores() map[string]int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	copyScores := make(map[string]int, len(e.user))
	for key, value := range e.user {
		copyScores[key] = value
	}
	return copyScores
}

func (e *Engine) stateLocked(committed string) State {
	return State{
		Buffer:     e.buffer,
		Candidates: e.candidatesLocked(),
		Committed:  committed,
		UpdatedAt:  time.Now().UTC(),
	}
}

func (e *Engine) candidatesLocked() []Candidate {
	if e.buffer == "" {
		return nil
	}
	if e.config.Mode == "en" {
		return []Candidate{{
			Text:    e.buffer,
			Reading: e.buffer,
			Weight:  1,
		}}
	}

	entries := e.lookupLocked(e.buffer)
	candidates := make([]Candidate, 0, len(entries))
	for _, entry := range entries {
		scoreKey := entry.Reading + "|" + entry.Text
		candidates = append(candidates, Candidate{
			Text:      entry.Text,
			Reading:   entry.Reading,
			Weight:    entry.Weight,
			UserScore: e.user[scoreKey],
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i].Weight + candidates[i].UserScore
		right := candidates[j].Weight + candidates[j].UserScore
		if left == right {
			return len([]rune(candidates[i].Text)) > len([]rune(candidates[j].Text))
		}
		return left > right
	})

	max := e.config.MaxCandidates
	if max <= 0 {
		max = DefaultConfig().MaxCandidates
	}
	if len(candidates) > max {
		candidates = candidates[:max]
	}
	return candidates
}

func (e *Engine) lookupLocked(reading string) []Entry {
	reading = normalizeReading(reading)
	exact := append([]Entry{}, e.dict[reading]...)
	if len(exact) > 0 {
		return exact
	}

	var prefixMatches []Entry
	for key, entries := range e.dict {
		if strings.HasPrefix(key, reading) {
			prefixMatches = append(prefixMatches, entries...)
		}
	}
	if len(prefixMatches) > 0 {
		return prefixMatches
	}

	segmented := e.segmentGreedyLocked(reading)
	if segmented != "" && segmented != reading {
		return []Entry{{Reading: reading, Text: segmented, Weight: 3000}}
	}
	return nil
}

func (e *Engine) segmentGreedyLocked(reading string) string {
	var out strings.Builder
	for i := 0; i < len(reading); {
		bestEnd := -1
		var best Entry
		for j := len(reading); j > i; j-- {
			part := reading[i:j]
			entries := e.dict[part]
			if len(entries) == 0 {
				continue
			}
			bestEnd = j
			best = entries[0]
			break
		}
		if bestEnd == -1 {
			return ""
		}
		out.WriteString(best.Text)
		i = bestEnd
	}
	return out.String()
}

func normalizeReading(input string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(input) {
		if r >= 'a' && r <= 'z' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
