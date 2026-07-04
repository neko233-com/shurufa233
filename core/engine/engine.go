package engine

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
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
	abbr          map[string][]Entry
	user          map[string]int
	buffer        string
	maxReadingLen int
}

const maxPrefixEntries = 256
const fuzzyCandidatePenalty = 120
const fuzzyVariantLimit = 64
const doublePinyinVariantLimit = 64
const abbreviationCandidatePenalty = 600
const segmentedCandidatePenalty = 9000
const segmentedPiecePenalty = 220
const dynamicCandidateWeightBase = 8800

var nowFunc = time.Now

func DefaultConfig() Config {
	return Config{
		MaxCandidates: 42,
		FuzzyInitials: []string{
			"zh=z",
			"ch=c",
			"sh=s",
		},
		DoublePinyin:       false,
		DoublePinyinScheme: "xiaohe",
		Language:           "zh-CN",
		Mode:               "zh",
		Punctuation:        "full",
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
	config.DoublePinyinScheme = normalizeDoublePinyinScheme(config.DoublePinyinScheme)
	config.Mode = normalizeMode(config.Mode)
	e := &Engine{
		config: config,
		dict:   make(map[string][]Entry),
		prefix: make(map[string][]Entry),
		abbr:   make(map[string][]Entry),
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
	config.DoublePinyinScheme = normalizeDoublePinyinScheme(config.DoublePinyinScheme)
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
	e.abbr = make(map[string][]Entry, len(e.dict))
	e.maxReadingLen = 0
	for reading, entries := range e.dict {
		if len(reading) > e.maxReadingLen {
			e.maxReadingLen = len(reading)
		}
		abbreviations := abbreviationsForReading(reading)
		for _, entry := range entries {
			for i := 1; i <= len(reading); i++ {
				prefix := reading[:i]
				if len(e.prefix[prefix]) < maxPrefixEntries {
					e.prefix[prefix] = append(e.prefix[prefix], entry)
				}
			}
			for _, abbr := range abbreviations {
				if len(e.abbr[abbr]) < maxPrefixEntries {
					e.abbr[abbr] = append(e.abbr[abbr], entry)
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
	for key := range e.abbr {
		sortEntries(e.abbr[key])
	}
}

func (e *Engine) LoadDictionary(reader io.Reader) (DictionaryFile, error) {
	file, err := DecodeDictionary(reader)
	if err != nil {
		return file, err
	}
	e.AddEntries(file.Entries)
	return file, nil
}

func DecodeDictionary(reader io.Reader) (DictionaryFile, error) {
	var file DictionaryFile
	data, err := io.ReadAll(reader)
	if err != nil {
		return file, err
	}
	data, err = decompressDictionaryData(data)
	if err != nil {
		return file, err
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if err := json.Unmarshal(data, &file); err != nil {
		return file, err
	}
	return file, nil
}

func decompressDictionaryData(data []byte) ([]byte, error) {
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		return data, nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (e *Engine) InputKey(key rune) State {
	e.mu.Lock()
	defer e.mu.Unlock()
	if unicode.IsLetter(key) || key == ';' {
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
	e.buffer = normalizeInputCode(input)
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

func (e *Engine) SelectChar(index int, side string) (State, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	candidates := e.candidatesLocked()
	if index < 0 || index >= len(candidates) {
		return e.stateLocked(""), errors.New("candidate index out of range")
	}
	text, err := candidateChar(candidates[index].Text, side)
	if err != nil {
		return e.stateLocked(""), err
	}
	e.buffer = ""
	return e.stateLocked(text), nil
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

func (e *Engine) ReplaceUserScores(scores map[string]int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.user = make(map[string]int, len(scores))
	for key, value := range scores {
		if key == "" || value <= 0 {
			continue
		}
		e.user[key] = value
	}
}

func (e *Engine) DeleteUserScore(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.user, key)
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
		candidates = append(candidates, Candidate{
			Text:      entry.Text,
			Reading:   entry.Reading,
			Kind:      entry.Kind,
			Source:    entry.Source,
			Weight:    entry.Weight,
			UserScore: e.entryUserScoreLocked(entry),
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
	inputCode := normalizeInputCode(reading)
	reading = normalizeReading(reading)
	readings := e.lookupReadingsLocked(inputCode)
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

	appendEntries(dynamicEntriesForInput(inputCode, nowFunc()), 0)
	for _, item := range readings {
		exactEntries := e.dict[item.reading]
		appendEntries(exactEntries, item.penalty)
		for _, variant := range e.fuzzyReadingsLocked(item.reading) {
			appendEntries(e.dict[variant], item.penalty+fuzzyCandidatePenalty)
		}
		if len(exactEntries) > 0 {
			segmented, weight := e.segmentBestLocked(item.reading)
			if segmented == "" || segmented == reading {
				continue
			}
			appendEntries([]Entry{{
				Reading: item.reading,
				Text:    segmented,
				Kind:    "phrase",
				Source:  "segmenter",
				Weight:  weight,
			}}, item.penalty)
		}
	}
	if len(exact) > 0 {
		return exact
	}

	seen = map[string]int{}
	for _, item := range readings {
		appendEntries(e.abbr[item.reading], item.penalty+abbreviationCandidatePenalty)
		for _, variant := range e.fuzzyReadingsLocked(item.reading) {
			appendEntries(e.abbr[variant], item.penalty+fuzzyCandidatePenalty+abbreviationCandidatePenalty)
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
			segmented, weight := e.segmentBestLocked(variant)
			if segmented != "" && segmented != reading {
				penalty := item.penalty
				if variant != item.reading {
					penalty += fuzzyCandidatePenalty
				}
				return []Entry{{
					Reading: variant,
					Text:    segmented,
					Kind:    "phrase",
					Source:  "segmenter",
					Weight:  max(1, weight-penalty),
				}}
			}
		}
	}
	return nil
}

func dynamicEntriesForInput(input string, now time.Time) []Entry {
	input = normalizeReading(input)
	if input == "" {
		return nil
	}
	add := func(texts ...string) []Entry {
		out := make([]Entry, 0, len(texts))
		seen := map[string]bool{}
		for index, text := range texts {
			text = strings.TrimSpace(text)
			if text == "" || seen[text] {
				continue
			}
			seen[text] = true
			out = append(out, Entry{
				Reading: input,
				Text:    text,
				Kind:    "dynamic",
				Source:  "builtin-datetime",
				Weight:  dynamicCandidateWeightBase - index,
			})
		}
		return out
	}
	switch input {
	case "rq", "date":
		return add(
			now.Format("2006-01-02"),
			strconv.Itoa(now.Year())+"年"+strconv.Itoa(int(now.Month()))+"月"+strconv.Itoa(now.Day())+"日",
			now.Format("2006/01/02"),
		)
	case "sj", "time":
		return add(now.Format("15:04"), now.Format("15:04:05"))
	case "xq", "week":
		weekday := chineseWeekday(now.Weekday())
		return add("星期"+weekday, "周"+weekday)
	case "dt", "datetime":
		return add(
			now.Format("2006-01-02 15:04"),
			strconv.Itoa(now.Year())+"年"+strconv.Itoa(int(now.Month()))+"月"+strconv.Itoa(now.Day())+"日 "+now.Format("15:04"),
		)
	case "ts", "timestamp":
		return add(strconv.FormatInt(now.Unix(), 10))
	default:
		return nil
	}
}

func chineseWeekday(day time.Weekday) string {
	switch day {
	case time.Monday:
		return "一"
	case time.Tuesday:
		return "二"
	case time.Wednesday:
		return "三"
	case time.Thursday:
		return "四"
	case time.Friday:
		return "五"
	case time.Saturday:
		return "六"
	default:
		return "日"
	}
}

func candidateChar(text string, side string) (string, error) {
	runes := []rune(text)
	if len(runes) == 0 {
		return "", errors.New("candidate text is empty")
	}
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "first", "head", "left":
		return string(runes[0]), nil
	case "last", "tail", "right":
		return string(runes[len(runes)-1]), nil
	default:
		return "", errors.New("candidate char side must be first or last")
	}
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
		for _, decoded := range decodeDoublePinyin(reading, e.config.DoublePinyinScheme) {
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

func (e *Engine) segmentBestLocked(reading string) (string, int) {
	type segmentState struct {
		text   string
		score  int
		pieces int
		ok     bool
	}
	states := make([]segmentState, len(reading)+1)
	states[0] = segmentState{ok: true}
	for i := 0; i < len(reading); i++ {
		if !states[i].ok {
			continue
		}
		end := len(reading)
		if e.maxReadingLen > 0 && i+e.maxReadingLen < end {
			end = i + e.maxReadingLen
		}
		for j := i + 1; j <= end; j++ {
			part := reading[i:j]
			entries := e.dict[part]
			if len(entries) == 0 {
				continue
			}
			best := e.bestEntryLocked(entries)
			score := states[i].score + e.entryScoreLocked(best) - segmentedPiecePenalty
			pieces := states[i].pieces + 1
			if !states[j].ok || score > states[j].score ||
				(score == states[j].score && pieces < states[j].pieces) {
				states[j] = segmentState{
					text:   states[i].text + best.Text,
					score:  score,
					pieces: pieces,
					ok:     true,
				}
			}
		}
	}
	best := states[len(reading)]
	if !best.ok || best.pieces < 2 {
		return "", 0
	}
	return best.text, max(1, best.score-segmentedCandidatePenalty)
}

func (e *Engine) bestEntryLocked(entries []Entry) Entry {
	best := entries[0]
	bestScore := e.entryScoreLocked(best)
	for _, entry := range entries[1:] {
		score := e.entryScoreLocked(entry)
		if score > bestScore || (score == bestScore && len([]rune(entry.Text)) > len([]rune(best.Text))) {
			best = entry
			bestScore = score
		}
	}
	return best
}

func (e *Engine) entryScoreLocked(entry Entry) int {
	return entry.Weight + e.entryUserScoreLocked(entry)
}

func (e *Engine) entryUserScoreLocked(entry Entry) int {
	return e.user[entry.Reading+"|"+entry.Text]
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

func normalizeInputCode(input string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(input) {
		if r >= 'a' && r <= 'z' || r == ';' {
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

func normalizeDoublePinyinScheme(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "", "xiaohe", "flypy":
		return "xiaohe"
	case "microsoft", "ms", "sogou":
		return "microsoft"
	default:
		return "xiaohe"
	}
}

func abbreviationsForReading(reading string) []string {
	parts := segmentPinyinReading(reading)
	if len(parts) < 2 {
		return nil
	}
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			return nil
		}
		builder.WriteByte(part[0])
	}
	abbr := builder.String()
	if len(abbr) < 2 || abbr == reading {
		return nil
	}
	return []string{abbr}
}

func segmentPinyinReading(reading string) []string {
	reading = normalizeReading(reading)
	if reading == "" {
		return nil
	}
	type state struct {
		parts []string
		ok    bool
	}
	states := make([]state, len(reading)+1)
	states[0] = state{ok: true}
	for i := 0; i < len(reading); i++ {
		if !states[i].ok {
			continue
		}
		end := len(reading)
		if i+maxPinyinSyllableLen < end {
			end = i + maxPinyinSyllableLen
		}
		for j := end; j > i; j-- {
			part := reading[i:j]
			if !pinyinSyllables[part] {
				continue
			}
			next := append(append([]string{}, states[i].parts...), part)
			if !states[j].ok || len(next) < len(states[j].parts) {
				states[j] = state{parts: next, ok: true}
			}
		}
	}
	if !states[len(reading)].ok {
		return nil
	}
	return states[len(reading)].parts
}

var pinyinSyllables, maxPinyinSyllableLen = buildPinyinSyllables()

func buildPinyinSyllables() (map[string]bool, int) {
	initials := []string{
		"", "b", "p", "m", "f", "d", "t", "n", "l", "g", "k", "h",
		"j", "q", "x", "zh", "ch", "sh", "r", "z", "c", "s", "y", "w",
	}
	finals := []string{
		"a", "ai", "an", "ang", "ao",
		"e", "ei", "en", "eng", "er",
		"o", "ong", "ou",
		"i", "ia", "ian", "iang", "iao", "ie", "in", "ing", "iong", "iu",
		"u", "ua", "uai", "uan", "uang", "ue", "ui", "un", "uo",
		"v", "van", "ve", "vn",
	}
	syllables := map[string]bool{
		"m": true, "n": true, "ng": true,
	}
	maxLen := 0
	for _, initial := range initials {
		for _, final := range finals {
			syllable := initial + final
			if syllable == "" {
				continue
			}
			syllables[syllable] = true
			if len(syllable) > maxLen {
				maxLen = len(syllable)
			}
		}
	}
	for syllable := range syllables {
		if len(syllable) > maxLen {
			maxLen = len(syllable)
		}
	}
	return syllables, maxLen
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

var microsoftInitials = map[byte]string{
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

var microsoftFinals = map[byte][]string{
	'a': []string{"a"},
	'b': []string{"ou"},
	'c': []string{"iao"},
	'd': []string{"iang", "uang"},
	'e': []string{"e"},
	'f': []string{"en"},
	'g': []string{"eng"},
	'h': []string{"ang"},
	'i': []string{"i"},
	'j': []string{"an"},
	'k': []string{"ao"},
	'l': []string{"ai"},
	'm': []string{"ian"},
	'n': []string{"in"},
	'o': []string{"o", "uo"},
	'p': []string{"un"},
	'q': []string{"iu"},
	'r': []string{"uan", "er"},
	's': []string{"ong", "iong"},
	't': []string{"ue", "ve"},
	'u': []string{"u"},
	'v': []string{"ui", "ue"},
	'w': []string{"ia", "ua"},
	'x': []string{"ie"},
	'y': []string{"uai", "v"},
	'z': []string{"ei"},
	';': []string{"ing"},
}

func decodeDoublePinyin(input string, scheme string) []string {
	input = normalizeInputCode(input)
	if input == "" {
		return nil
	}
	initials := xiaoheInitials
	finals := xiaoheFinals
	zeroInitial := byte(0)
	if normalizeDoublePinyinScheme(scheme) == "microsoft" {
		initials = microsoftInitials
		finals = microsoftFinals
		zeroInitial = 'o'
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
		if zeroInitial == 0 {
			if values, ok := finals[input[pos]]; ok {
				for _, final := range values {
					walk(pos+1, append(parts, normalizeDoublePinyinSyllable("", final)))
				}
			}
		} else if input[pos] == zeroInitial && pos+1 < len(input) {
			if values, ok := finals[input[pos+1]]; ok {
				for _, final := range values {
					walk(pos+2, append(parts, normalizeDoublePinyinSyllable("", final)))
				}
			}
		}
		if pos+1 >= len(input) {
			return
		}
		initial, ok := initials[input[pos]]
		if !ok {
			return
		}
		for _, final := range finals[input[pos+1]] {
			walk(pos+2, append(parts, normalizeDoublePinyinSyllable(initial, final)))
		}
	}
	walk(0, nil)
	return out
}

func decodeXiaoheDoublePinyin(input string) []string {
	return decodeDoublePinyin(input, "xiaohe")
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
