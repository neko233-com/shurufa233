package engine

import "time"

type Config struct {
	MaxCandidates         int      `json:"maxCandidates"`
	Schema                string   `json:"schema,omitempty"`
	CandidatePageSize     int      `json:"candidatePageSize"`
	CandidateLayout       string   `json:"candidateLayout"`
	ShowCandidateComments bool     `json:"showCandidateComments"`
	FuzzyInitials         []string `json:"fuzzyInitials"`
	DoublePinyin          bool     `json:"doublePinyin"`
	DoublePinyinScheme    string   `json:"doublePinyinScheme"`
	Language              string   `json:"language"`
	Mode                  string   `json:"mode"`
	Punctuation           string   `json:"punctuation"`
	Script                string   `json:"script"`
	Associations          bool     `json:"associations"`
	KeyProfile            string   `json:"keyProfile"`
	ShiftToggleMode       bool     `json:"shiftToggleMode"`
	SemicolonQuickSelect  bool     `json:"semicolonQuickSelect"`
	QuoteQuickSelect      bool     `json:"quoteQuickSelect"`
	BracketPageKeys       bool     `json:"bracketPageKeys"`
	MinusEqualPageKeys    bool     `json:"minusEqualPageKeys"`
	CommaPeriodPageKeys   bool     `json:"commaPeriodPageKeys"`
	Skin                  Skin     `json:"skin"`
	Update                Update   `json:"update"`
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
	Pinned    bool   `json:"pinned,omitempty"`
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

type CatalogRequest struct {
	Kind  string `json:"kind,omitempty"`
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type CatalogResponse struct {
	Kind      string    `json:"kind"`
	Query     string    `json:"query,omitempty"`
	Count     int       `json:"count"`
	Entries   []Entry   `json:"entries"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ReverseLookupRequest struct {
	Query string `json:"query,omitempty"`
	Text  string `json:"text,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type ReverseLookupResponse struct {
	Query     string    `json:"query"`
	Count     int       `json:"count"`
	Entries   []Entry   `json:"entries"`
	UpdatedAt time.Time `json:"updatedAt"`
}

const (
	UserPhraseKind   = "phrase"
	UserPhraseSource = "user-phrase"
	UserRejectSource = "user-reject"
	UserPinSource    = "user-pin"
)

type Update struct {
	SourcePreset     string   `json:"sourcePreset,omitempty"`
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

type SchemaPreset struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Kind                   string   `json:"kind"`
	RimeID                 string   `json:"rimeId,omitempty"`
	Description            string   `json:"description"`
	Tags                   []string `json:"tags,omitempty"`
	Language               string   `json:"language"`
	DoublePinyin           bool     `json:"doublePinyin"`
	DoublePinyinScheme     string   `json:"doublePinyinScheme,omitempty"`
	FuzzyInitials          []string `json:"fuzzyInitials,omitempty"`
	Punctuation            string   `json:"punctuation,omitempty"`
	KeyProfile             string   `json:"keyProfile,omitempty"`
	ShiftToggleMode        bool     `json:"shiftToggleMode"`
	SemicolonQuickSelect   bool     `json:"semicolonQuickSelect"`
	QuoteQuickSelect       bool     `json:"quoteQuickSelect"`
	BracketPageKeys        bool     `json:"bracketPageKeys"`
	MinusEqualPageKeys     bool     `json:"minusEqualPageKeys"`
	CommaPeriodPageKeys    bool     `json:"commaPeriodPageKeys"`
	CandidateLayout        string   `json:"candidateLayout,omitempty"`
	ShowCandidateComments  bool     `json:"showCandidateComments"`
	DictionarySourcePreset string   `json:"dictionarySourcePreset,omitempty"`
}
