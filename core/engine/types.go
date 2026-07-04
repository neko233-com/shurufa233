package engine

import "time"

type Config struct {
	MaxCandidates int      `json:"maxCandidates"`
	FuzzyInitials []string `json:"fuzzyInitials"`
	DoublePinyin  bool     `json:"doublePinyin"`
	Language      string   `json:"language"`
	Mode          string   `json:"mode"`
	Skin          Skin     `json:"skin"`
	Update        Update   `json:"update"`
}

type Skin struct {
	FontFamily string `json:"fontFamily"`
	FontSize   int    `json:"fontSize"`
	Accent     string `json:"accent"`
	Theme      string `json:"theme"`
}

type Candidate struct {
	Text      string `json:"text"`
	Reading   string `json:"reading"`
	Weight    int    `json:"weight"`
	UserScore int    `json:"userScore"`
}

type State struct {
	Buffer     string      `json:"buffer"`
	Candidates []Candidate `json:"candidates"`
	Committed  string      `json:"committed,omitempty"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

type Entry struct {
	Reading string `json:"reading"`
	Text    string `json:"text"`
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
