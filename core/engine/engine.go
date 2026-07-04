package engine

import (
	"bytes"
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
	mu            sync.RWMutex
	config        Config
	dict          map[string][]Entry
	prefix        map[string][]Entry
	user          map[string]int
	buffer        string
	maxReadingLen int
}

const maxPrefixEntries = 256
const fuzzyCandidatePenalty = 120
const fuzzyVariantLimit = 64
const doublePinyinVariantLimit = 64

func DefaultConfig() Config {
	return Config{
		MaxCandidates: 42,
		FuzzyInitials: []string{
			"zh=z",
			"ch=c",
			"sh=s",
		},
		DoublePinyin: false,
		Language:     "zh-CN",
		Mode:         "zh",
		Skin: Skin{
			FontFamily:    "Microsoft YaHei UI",
			FontSize:      15,
			Accent:        "#2563eb",
			Surface:       "#ffffff",
			Text:          "#111827",
			MutedText:     "#64748b",
			Border:        "#d1d5db",
			HighlightText: "#ffffff",
			Theme:         "system",
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
	config.Mode = normalizeMode(config.Mode)
	e := &Engine{
		config: config,
		dict:   make(map[string][]Entry),
		prefix: make(map[string][]Entry),
		user:   make(map[string]int),
	}
	e.AddEntries(defaultEntries)
	return e
}

func (e *Engine) Configure(config Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if config.MaxCandidates <= 0 {
		config.MaxCandidates = DefaultConfig().MaxCandidates
	}
	config.Mode = normalizeMode(config.Mode)
	e.config = config
}

func (e *Engine) AddEntries(entries []Entry) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.addEntriesLocked(entries)
}

func (e *Engine) addEntriesLocked(entries []Entry) {
	for _, entry := range entries {
		entry.Reading = normalizeReading(entry.Reading)
		if entry.Reading == "" || entry.Text == "" {
			continue
		}
		merged := false
		for i := range e.dict[entry.Reading] {
			if e.dict[entry.Reading][i].Text == entry.Text {
				if entry.Weight > e.dict[entry.Reading][i].Weight {
					e.dict[entry.Reading][i].Weight = entry.Weight
				}
				merged = true
				break
			}
		}
		if !merged {
			e.dict[entry.Reading] = append(e.dict[entry.Reading], entry)
		}
	}
	e.rebuildPrefixLocked()
	e.sortIndexLocked()
}

func (e *Engine) rebuildPrefixLocked() {
	e.prefix = make(map[string][]Entry, len(e.dict)*2)
	e.maxReadingLen = 0
	for reading, entries := range e.dict {
		if len(reading) > e.maxReadingLen {
			e.maxReadingLen = len(reading)
		}
		for _, entry := range entries {
			for i := 1; i <= len(reading); i++ {
				prefix := reading[:i]
				if len(e.prefix[prefix]) < maxPrefixEntries {
					e.prefix[prefix] = append(e.prefix[prefix], entry)
				}
			}
		}
	}
}

func (e *Engine) sortIndexLocked() {
	for key := range e.dict {
		sortEntries(e.dict[key])
	}
	for key := range e.prefix {
		sortEntries(e.prefix[key])
	}
}

func (e *Engine) LoadDictionary(reader io.Reader) (DictionaryFile, error) {
	var file DictionaryFile
	data, err := io.ReadAll(reader)
	if err != nil {
		return file, err
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if err := json.Unmarshal(data, &file); err != nil {
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

func (e *Engine) SetMode(mode string) State {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config.Mode = normalizeMode(mode)
	e.buffer = ""
	return e.stateLocked("")
}

func (e *Engine) ToggleMode() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	if normalizeMode(e.config.Mode) == "en" {
		e.config.Mode = "zh"
	} else {
		e.config.Mode = "en"
	}
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

func (e *Engine) ImportUserScores(scores map[string]int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(scores) == 0 {
		return
	}
	if e.user == nil {
		e.user = make(map[string]int, len(scores))
	}
	for key, value := range scores {
		if key == "" || value <= 0 {
			continue
		}
		e.user[key] = value
	}
}

func (e *Engine) stateLocked(committed string) State {
	return State{
		Buffer:     e.buffer,
		Mode:       normalizeMode(e.config.Mode),
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
			Kind:      entry.Kind,
			Source:    entry.Source,
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
	readings := e.lookupReadingsLocked(reading)
	var exact []Entry
	seen := map[string]int{}
	appendEntries := func(entries []Entry, penalty int) {
		for _, entry := range entries {
			if entry.Text == "" {
				continue
			}
			next := entry
			if penalty > 0 {
				next.Weight = max(1, next.Weight-penalty)
			}
			key := next.Text + "\x00" + next.Kind + "\x00" + next.Source
			if previous, ok := seen[key]; ok {
				if next.Weight > exact[previous].Weight {
					exact[previous] = next
				}
				continue
			}
			seen[key] = len(exact)
			exact = append(exact, next)
		}
	}

	for _, item := range readings {
		appendEntries(e.dict[item.reading], item.penalty)
		for _, variant := range e.fuzzyReadingsLocked(item.reading) {
			appendEntries(e.dict[variant], item.penalty+fuzzyCandidatePenalty)
		}
	}
	if len(exact) > 0 {
		return exact
	}

	seen = map[string]int{}
	for _, item := range readings {
		appendEntries(e.prefix[item.reading], item.penalty)
		for _, variant := range e.fuzzyReadingsLocked(item.reading) {
			appendEntries(e.prefix[variant], item.penalty+fuzzyCandidatePenalty)
		}
	}
	if len(exact) > 0 {
		return exact
	}

	for _, item := range readings {
		for _, variant := range append([]string{item.reading}, e.fuzzyReadingsLocked(item.reading)...) {
			segmented := e.segmentGreedyLocked(variant)
			if segmented != "" && segmented != reading {
				penalty := item.penalty
				if variant != item.reading {
					penalty += fuzzyCandidatePenalty
				}
				return []Entry{{Reading: variant, Text: segmented, Weight: max(1, 3000-penalty)}}
			}
		}
	}
	return nil
}

type lookupReading struct {
	reading string
	penalty int
}

func (e *Engine) lookupReadingsLocked(reading string) []lookupReading {
	seen := map[string]bool{}
	items := make([]lookupReading, 0, 4)
	add := func(value string, penalty int) {
		value = normalizeReading(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		items = append(items, lookupReading{reading: value, penalty: penalty})
	}
	add(reading, 0)
	if e.config.DoublePinyin {
		for _, decoded := range decodeXiaoheDoublePinyin(reading) {
			add(decoded, 0)
		}
	}
	return items
}

func (e *Engine) fuzzyReadingsLocked(reading string) []string {
	rules := fuzzyRules(e.config.FuzzyInitials)
	if len(rules) == 0 || reading == "" {
		return nil
	}
	seen := map[string]bool{reading: true}
	queue := []string{reading}
	var out []string
	for head := 0; head < len(queue) && len(seen) < fuzzyVariantLimit; head++ {
		current := queue[head]
		for _, rule := range rules {
			for start := 0; start < len(current) && len(seen) < fuzzyVariantLimit; {
				index := strings.Index(current[start:], rule.from)
				if index < 0 {
					break
				}
				index += start
				next := current[:index] + rule.to + current[index+len(rule.from):]
				start = index + len(rule.to)
				if next == "" || seen[next] || len(next) > e.maxReadingLen+4 {
					continue
				}
				seen[next] = true
				queue = append(queue, next)
				out = append(out, next)
			}
		}
	}
	return out
}

func (e *Engine) segmentGreedyLocked(reading string) string {
	var out strings.Builder
	for i := 0; i < len(reading); {
		bestEnd := -1
		var best Entry
		end := len(reading)
		if e.maxReadingLen > 0 && i+e.maxReadingLen < end {
			end = i + e.maxReadingLen
		}
		for j := end; j > i; j-- {
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

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "en":
		return "en"
	default:
		return "zh"
	}
}

var xiaoheInitials = map[byte]string{
	'b': "b",
	'p': "p",
	'm': "m",
	'f': "f",
	'd': "d",
	't': "t",
	'n': "n",
	'l': "l",
	'g': "g",
	'k': "k",
	'h': "h",
	'j': "j",
	'q': "q",
	'x': "x",
	'v': "zh",
	'i': "ch",
	'u': "sh",
	'r': "r",
	'z': "z",
	'c': "c",
	's': "s",
	'y': "y",
	'w': "w",
}

var xiaoheFinals = map[byte][]string{
	'a': []string{"a"},
	'b': []string{"in"},
	'c': []string{"iao"},
	'd': []string{"ai"},
	'e': []string{"e"},
	'f': []string{"en"},
	'g': []string{"eng"},
	'h': []string{"ang"},
	'i': []string{"i"},
	'j': []string{"an"},
	'k': []string{"ao"},
	'l': []string{"ing"},
	'm': []string{"ian"},
	'n': []string{"iang", "uang"},
	'o': []string{"o", "uo"},
	'p': []string{"ie"},
	'q': []string{"iu"},
	'r': []string{"uan", "er"},
	's': []string{"ong", "iong"},
	't': []string{"ue", "ve"},
	'u': []string{"u"},
	'v': []string{"v", "ui"},
	'w': []string{"ei"},
	'x': []string{"ia", "ua"},
	'y': []string{"un"},
	'z': []string{"ou"},
}

func decodeXiaoheDoublePinyin(input string) []string {
	input = normalizeReading(input)
	if input == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	var walk func(pos int, parts []string)
	walk = func(pos int, parts []string) {
		if len(out) >= doublePinyinVariantLimit {
			return
		}
		if pos == len(input) {
			value := strings.Join(parts, "")
			if value != "" && value != input && !seen[value] {
				seen[value] = true
				out = append(out, value)
			}
			return
		}
		if finals, ok := xiaoheFinals[input[pos]]; ok {
			for _, final := range finals {
				walk(pos+1, append(parts, normalizeDoublePinyinSyllable("", final)))
			}
		}
		if pos+1 >= len(input) {
			return
		}
		initial, ok := xiaoheInitials[input[pos]]
		if !ok {
			return
		}
		for _, final := range xiaoheFinals[input[pos+1]] {
			walk(pos+2, append(parts, normalizeDoublePinyinSyllable(initial, final)))
		}
	}
	walk(0, nil)
	return out
}

func normalizeDoublePinyinSyllable(initial string, final string) string {
	if initial == "j" || initial == "q" || initial == "x" || initial == "y" {
		final = strings.ReplaceAll(final, "v", "u")
	}
	if initial == "" {
		final = strings.ReplaceAll(final, "v", "u")
	}
	return initial + final
}

type fuzzyRule struct {
	from string
	to   string
}

func fuzzyRules(items []string) []fuzzyRule {
	seen := map[string]bool{}
	var rules []fuzzyRule
	for _, item := range items {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		left := normalizeReading(parts[0])
		right := normalizeReading(parts[1])
		if left == "" || right == "" || left == right {
			continue
		}
		for _, pair := range [][2]string{{left, right}, {right, left}} {
			key := pair[0] + "=" + pair[1]
			if seen[key] {
				continue
			}
			seen[key] = true
			rules = append(rules, fuzzyRule{from: pair[0], to: pair[1]})
		}
	}
	sort.SliceStable(rules, func(i, j int) bool {
		return len(rules[i].from) > len(rules[j].from)
	})
	return rules
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Weight == entries[j].Weight {
			return len([]rune(entries[i].Text)) > len([]rune(entries[j].Text))
		}
		return entries[i].Weight > entries[j].Weight
	})
}
