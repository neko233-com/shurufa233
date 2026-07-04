package engine

import "time"

type Config struct {
	MaxCandidates      int      `json:"maxCandidates"`
	FuzzyInitials      []string `json:"fuzzyInitials"`
	DoublePinyin       bool     `json:"doublePinyin"`
	DoublePinyinScheme string   `json:"doublePinyinScheme"`
	Language           string   `json:"language"`
	Mode               string   `json:"mode"`
	Punctuation        string   `json:"punctuation"`
	Skin               Skin     `json:"skin"`
	Update             Update   `json:"update"`
}

type Skin struct {
	FontFamily    string `json:"fontFamily"`
	FontSize      int    `json:"fontSize"`
	Accent        string `json:"accent"`
	Surface       string `json:"surface"`
	Text          string `json:"text"`
	MutedText     string `json:"mutedText"`
	Border        string `json:"border"`
	HighlightText string `json:"highlightText"`
	Theme         string `json:"theme"`
}

type Candidate struct {
	Text      string `json:"text"`
	Reading   string `json:"reading"`
	Kind      string `json:"kind,omitempty"`
	Source    string `json:"source,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Weight    int    `json:"weight"`
	UserScore int    `json:"userScore"`
}

type State struct {
	Buffer     string      `json:"buffer"`
	Mode       string      `json:"mode"`
	Candidates []Candidate `json:"candidates"`
	Committed  string      `json:"committed,omitempty"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

type Entry struct {
	Reading string `json:"reading"`
	Text    string `json:"text"`
	Kind    string `json:"kind,omitempty"`
	Source  string `json:"source,omitempty"`
	Comment string `json:"comment,omitempty"`
	Weight  int    `json:"weight"`
}

type Update struct {
	Channel          string   `json:"channel"`
	ManifestURLs     []string `json:"manifestUrls"`
	MirrorBaseURLs   []string `json:"mirrorBaseUrls"`
	AutoCheck        bool     `json:"autoCheck"`
	AutoApply        bool     `json:"autoApply"`
	InstalledVersion string   `json:"installedVersion"`
}

type DictionaryFile struct {
	Language string  `json:"language"`
	Version  string  `json:"version"`
	Entries  []Entry `json:"entries"`
}
