package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var apiBase = "http://127.0.0.1:23333"

type candidate struct {
	Text      string `json:"text"`
	Reading   string `json:"reading"`
	Kind      string `json:"kind,omitempty"`
	Source    string `json:"source,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Weight    int    `json:"weight"`
	UserScore int    `json:"userScore"`
	Pinned    bool   `json:"pinned,omitempty"`
}

type candidatePageItem struct {
	Index        int    `json:"index"`
	DisplayIndex int    `json:"displayIndex"`
	Text         string `json:"text"`
	Reading      string `json:"reading"`
	Kind         string `json:"kind,omitempty"`
	Source       string `json:"source,omitempty"`
	Comment      string `json:"comment,omitempty"`
	Weight       int    `json:"weight"`
	UserScore    int    `json:"userScore"`
	Pinned       bool   `json:"pinned,omitempty"`
	Score        int    `json:"score"`
}

type engineState struct {
	Buffer     string      `json:"buffer"`
	Mode       string      `json:"mode"`
	Candidates []candidate `json:"candidates"`
	Committed  string      `json:"committed,omitempty"`
}

type candidateActionRequest struct {
	Action       string `json:"action,omitempty"`
	Context      string `json:"context,omitempty"`
	Index        int    `json:"index,omitempty"`
	DisplayIndex int    `json:"displayIndex,omitempty"`
	Start        int    `json:"start,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	PageSize     int    `json:"pageSize,omitempty"`
	Delta        int    `json:"delta,omitempty"`
	Side         string `json:"side,omitempty"`
}

type candidateActionResponse struct {
	OK         bool         `json:"ok"`
	Action     string       `json:"action"`
	Start      int          `json:"start"`
	Limit      int          `json:"limit"`
	Total      int          `json:"total"`
	Committed  string       `json:"committed,omitempty"`
	State      engineState  `json:"state"`
	Rejected   *phraseEntry `json:"rejected,omitempty"`
	Pinned     *phraseEntry `json:"pinned,omitempty"`
	Candidates struct {
		OK        bool                `json:"ok"`
		Start     int                 `json:"start"`
		Limit     int                 `json:"limit"`
		Total     int                 `json:"total"`
		Items     []candidatePageItem `json:"items"`
		UpdatedAt string              `json:"updatedAt"`
	} `json:"candidates"`
	UpdatedAt string `json:"updatedAt"`
}

type updateCheck struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ManifestURL     string `json:"manifestUrl"`
}

type dictionarySourceResponse struct {
	Sources  []dictionarySource `json:"sources"`
	Selected string             `json:"selected"`
}

type dictionarySource struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Kind           string          `json:"kind"`
	Description    string          `json:"description"`
	Homepage       string          `json:"homepage"`
	License        string          `json:"license"`
	Installable    bool            `json:"installable"`
	ManifestURLs   []string        `json:"manifestUrls"`
	MirrorBaseURLs []string        `json:"mirrorBaseUrls"`
	RawSources     []dictionaryRaw `json:"rawSources"`
	ConvertCommand string          `json:"convertCommand"`
	SyncCommand    string          `json:"syncCommand"`
}

type dictionaryRaw struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Role  string `json:"role"`
}

type schemaResponse struct {
	Selected string         `json:"selected"`
	Schemas  []schemaPreset `json:"schemas"`
	Config   configPayload  `json:"config,omitempty"`
}

type schemaPreset struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Kind                   string   `json:"kind"`
	RimeID                 string   `json:"rimeId,omitempty"`
	Description            string   `json:"description"`
	Tags                   []string `json:"tags,omitempty"`
	Language               string   `json:"language"`
	DoublePinyin           bool     `json:"doublePinyin"`
	DoublePinyinScheme     string   `json:"doublePinyinScheme,omitempty"`
	DictionarySourcePreset string   `json:"dictionarySourcePreset,omitempty"`
}

type configPayload struct {
	Schema             string `json:"schema,omitempty"`
	DoublePinyin       bool   `json:"doublePinyin"`
	DoublePinyinScheme string `json:"doublePinyinScheme"`
	CandidatePageSize  int    `json:"candidatePageSize,omitempty"`
	CandidateLayout    string `json:"candidateLayout,omitempty"`
	Punctuation        string `json:"punctuation,omitempty"`
	KeyProfile         string `json:"keyProfile,omitempty"`
}

type switchOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RimeName    string `json:"rimeName"`
	Description string `json:"description"`
	Value       bool   `json:"value"`
	On          string `json:"on"`
	Off         string `json:"off"`
	ConfigField string `json:"configField"`
}

type switchResponse struct {
	OK       bool           `json:"ok"`
	Selected *switchOption  `json:"selected,omitempty"`
	Switches []switchOption `json:"switches"`
	Config   configPayload  `json:"config,omitempty"`
}

type rimeCustomResult struct {
	OK       bool          `json:"ok"`
	Config   configPayload `json:"config,omitempty"`
	Schema   string        `json:"schema,omitempty"`
	Applied  []string      `json:"applied,omitempty"`
	Warnings []string      `json:"warnings,omitempty"`
}

type appRule struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	ProcessNames      []string `json:"processNames,omitempty"`
	ExeContains       []string `json:"exeContains,omitempty"`
	WindowTitle       []string `json:"windowTitle,omitempty"`
	WindowClass       []string `json:"windowClass,omitempty"`
	PasswordField     bool     `json:"passwordField,omitempty"`
	Terminal          bool     `json:"terminal,omitempty"`
	GameMode          bool     `json:"gameMode,omitempty"`
	Mode              string   `json:"mode,omitempty"`
	Punctuation       string   `json:"punctuation,omitempty"`
	CandidateLayout   string   `json:"candidateLayout,omitempty"`
	DisableCandidates bool     `json:"disableCandidates,omitempty"`
	DisableLearning   bool     `json:"disableLearning,omitempty"`
	Priority          int      `json:"priority,omitempty"`
}

type appRuleResponse struct {
	OK    bool      `json:"ok"`
	Rules []appRule `json:"rules"`
}

type appContext struct {
	ProcessName   string `json:"processName,omitempty"`
	ExePath       string `json:"exePath,omitempty"`
	WindowTitle   string `json:"windowTitle,omitempty"`
	WindowClass   string `json:"windowClass,omitempty"`
	PasswordField bool   `json:"passwordField,omitempty"`
	Terminal      bool   `json:"terminal,omitempty"`
	GameMode      bool   `json:"gameMode,omitempty"`
}

type appContextDecision struct {
	OK                bool       `json:"ok"`
	Matched           bool       `json:"matched"`
	Rule              *appRule   `json:"rule,omitempty"`
	Context           appContext `json:"context"`
	Mode              string     `json:"mode"`
	Punctuation       string     `json:"punctuation"`
	CandidateLayout   string     `json:"candidateLayout"`
	DisableCandidates bool       `json:"disableCandidates,omitempty"`
	DisableLearning   bool       `json:"disableLearning,omitempty"`
	Reason            string     `json:"reason,omitempty"`
}

type wordbookResponse struct {
	UserScores map[string]int `json:"userScores"`
	Count      int            `json:"count"`
	UpdatedAt  string         `json:"updatedAt"`
}

type phraseEntry struct {
	Reading string `json:"reading"`
	Text    string `json:"text"`
	Kind    string `json:"kind,omitempty"`
	Source  string `json:"source,omitempty"`
	Comment string `json:"comment,omitempty"`
	Weight  int    `json:"weight,omitempty"`
}

type phraseResponse struct {
	Phrases   []phraseEntry `json:"phrases"`
	Entries   []phraseEntry `json:"entries"`
	Count     int           `json:"count"`
	UpdatedAt string        `json:"updatedAt"`
}

type rejectResponse struct {
	Rejects   []phraseEntry `json:"rejects"`
	Entries   []phraseEntry `json:"entries"`
	Count     int           `json:"count"`
	UpdatedAt string        `json:"updatedAt"`
}

type pinResponse struct {
	Pins      []phraseEntry `json:"pins"`
	Entries   []phraseEntry `json:"entries"`
	Count     int           `json:"count"`
	UpdatedAt string        `json:"updatedAt"`
}

type profileBundle struct {
	OK         bool           `json:"ok,omitempty"`
	Version    int            `json:"version"`
	Product    string         `json:"product"`
	ExportedAt string         `json:"exportedAt"`
	Config     map[string]any `json:"config,omitempty"`
	UserScores map[string]int `json:"userScores,omitempty"`
	Phrases    []phraseEntry  `json:"phrases,omitempty"`
	Rejects    []phraseEntry  `json:"rejects,omitempty"`
	Pins       []phraseEntry  `json:"pins,omitempty"`
	Merge      bool           `json:"merge,omitempty"`
	Counts     map[string]int `json:"counts,omitempty"`
}

type catalogResponse struct {
	Kind      string        `json:"kind"`
	Query     string        `json:"query,omitempty"`
	Count     int           `json:"count"`
	Entries   []phraseEntry `json:"entries"`
	UpdatedAt string        `json:"updatedAt"`
}

type reverseLookupResponse struct {
	Query     string        `json:"query"`
	Count     int           `json:"count"`
	Entries   []phraseEntry `json:"entries"`
	UpdatedAt string        `json:"updatedAt"`
}

type agentResponse struct {
	Input      string           `json:"input"`
	Context    string           `json:"context,omitempty"`
	Candidates []string         `json:"candidates"`
	Items      []agentCandidate `json:"items"`
	Actions    []string         `json:"actions"`
}

type agentCandidate struct {
	Text    string `json:"text"`
	Intent  string `json:"intent"`
	Action  string `json:"action"`
	Source  string `json:"source"`
	Context string `json:"context,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	client := &http.Client{Timeout: 8 * time.Second}
	var err error
	switch os.Args[1] {
	case "status":
		err = status(client)
	case "preview":
		err = preview(client, strings.Join(os.Args[2:], " "))
	case "associate", "associations", "predict":
		err = associate(client, os.Args[2:])
	case "update-check":
		err = updateCheckCmd(client)
	case "update-apply":
		err = updateApply(client)
	case "update-sources":
		err = updateSources(client)
	case "update-source":
		err = updateSource(client, os.Args[2:])
	case "schemas":
		err = schemas(client)
	case "schema":
		err = schema(client, os.Args[2:])
	case "rime":
		err = rimeCustom(client, os.Args[2:])
	case "mode":
		err = mode(client, os.Args[2:])
	case "switches":
		err = switches(client)
	case "switch":
		err = applySwitch(client, os.Args[2:])
	case "app-rules":
		err = appRules(client)
	case "app-context":
		err = appContextCmd(client, os.Args[2:])
	case "wordbook":
		err = wordbook(client, os.Args[2:])
	case "phrases", "phrase":
		err = phrases(client, os.Args[2:])
	case "rejects", "reject":
		err = rejects(client, os.Args[2:])
	case "pins", "pin":
		err = pins(client, os.Args[2:])
	case "profile":
		err = profile(client, os.Args[2:])
	case "catalog", "symbols", "symbol":
		err = catalog(client, os.Args[2:])
	case "reverse", "lookup", "fancha":
		err = reverseLookup(client, os.Args[2:])
	case "agent":
		err = agent(client, os.Args[2:])
	case "candidates", "candidate-action":
		err = candidateAction(client, os.Args[2:])
	case "repl":
		err = repl(client)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`shurufa233 CLI

Usage:
  shurufa-imecli status
  shurufa-imecli preview nihao
  shurufa-imecli associate "你好"
  shurufa-imecli mode [zh|en|toggle]
  shurufa-imecli switches
  shurufa-imecli switch <ascii_mode|ascii_punct|simplification|candidate_comments|associations|vertical_candidates> [on|off|toggle]
  shurufa-imecli app-rules
  shurufa-imecli app-context resolve --process WeGame.exe [--title TITLE] [--class CLASS] [--path EXE] [--password] [--terminal] [--game]
  shurufa-imecli wordbook list
  shurufa-imecli wordbook export
  shurufa-imecli wordbook import user-wordbook.json [--replace]
  shurufa-imecli wordbook delete "nihao|你好"
  shurufa-imecli wordbook clear
  shurufa-imecli phrases list
  shurufa-imecli phrases add msd "马上到！" [weight]
  shurufa-imecli phrases import user-phrases.json [--replace]
  shurufa-imecli phrases export
  shurufa-imecli phrases delete "msd|马上到！"
  shurufa-imecli phrases clear
  shurufa-imecli rejects list
  shurufa-imecli rejects add ceshi "错词"
  shurufa-imecli rejects import user-rejects.json [--replace]
  shurufa-imecli rejects delete "ceshi|错词"
  shurufa-imecli rejects clear
  shurufa-imecli pins list
  shurufa-imecli pins add nihao "你好"
  shurufa-imecli pins import user-pins.json [--replace]
  shurufa-imecli pins export
  shurufa-imecli pins delete "nihao|你好"
  shurufa-imecli pins clear
  shurufa-imecli profile export [profile.json]
  shurufa-imecli profile import profile.json [--replace]
  shurufa-imecli symbols [all|emoji|kaomoji|symbol|agent] [query] [--limit N]
  shurufa-imecli reverse "你好" [--limit N]
  shurufa-imecli update-sources
  shurufa-imecli update-source shurufa233-github
  shurufa-imecli schemas
  shurufa-imecli schema [current|apply <id>]
  shurufa-imecli rime import default.custom.yaml
  shurufa-imecli update-check
  shurufa-imecli update-apply
  shurufa-imecli candidates nihao [view|next-page|prev-page|select|pin|forget|first-char|last-char] [--start N] [--limit N] [--display-index N] [--index N]
  shurufa-imecli agent "/rewrite hello"
  shurufa-imecli agent --context "selected text" "/rewrite"
  shurufa-imecli repl`)
}

func status(client *http.Client) error {
	body, err := get(client, "/health")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func preview(client *http.Client, input string) error {
	if input == "" {
		return fmt.Errorf("missing preview input")
	}
	var state engineState
	if err := postJSON(client, "/engine/preview", map[string]string{"input": input}, &state); err != nil {
		return err
	}
	fmt.Printf("buffer: %s\n", state.Buffer)
	for i, item := range state.Candidates {
		meta := ""
		if item.Kind != "" {
			meta = " kind=" + item.Kind
			if item.Source != "" {
				meta += " source=" + item.Source
			}
		}
		if item.Comment != "" {
			meta += " comment=" + item.Comment
		}
		if item.Pinned {
			meta += " pinned=true"
		}
		fmt.Printf("%d. %s [%s] score=%d%s\n", i+1, item.Text, item.Reading, item.Weight+item.UserScore, meta)
	}
	return nil
}

func associate(client *http.Client, args []string) error {
	context := strings.TrimSpace(strings.Join(args, " "))
	if context == "" {
		return fmt.Errorf("missing association context")
	}
	var state engineState
	if err := postJSON(client, "/engine/associate", map[string]any{"context": context}, &state); err != nil {
		return err
	}
	fmt.Printf("context: %s\n", context)
	for i, item := range state.Candidates {
		meta := ""
		if item.Kind != "" {
			meta = " kind=" + item.Kind
			if item.Source != "" {
				meta += " source=" + item.Source
			}
		}
		if item.Comment != "" {
			meta += " comment=" + item.Comment
		}
		if item.Pinned {
			meta += " pinned=true"
		}
		fmt.Printf("%d. %s [%s] score=%d%s\n", i+1, item.Text, item.Reading, item.Weight+item.UserScore, meta)
	}
	return nil
}

func candidateAction(client *http.Client, args []string) error {
	input, request, err := parseCandidateActionArgs(args)
	if err != nil {
		return err
	}
	if isAssociationAction(request.Action) {
		if request.Context == "" {
			request.Context = input
		}
	} else {
		var previewState engineState
		if err := postJSON(client, "/engine/preview", map[string]string{"input": input}, &previewState); err != nil {
			return err
		}
	}
	var response candidateActionResponse
	if err := postJSON(client, "/ime/candidate-action", request, &response); err != nil {
		return err
	}
	printCandidateAction(response)
	return nil
}

func printCandidateAction(response candidateActionResponse) {
	if response.Committed != "" {
		fmt.Printf("committed: %s\n", response.Committed)
	}
	if response.Rejected != nil {
		fmt.Printf("rejected: %s|%s\n", response.Rejected.Reading, response.Rejected.Text)
	}
	if response.Pinned != nil {
		fmt.Printf("pinned: %s|%s\n", response.Pinned.Reading, response.Pinned.Text)
	}
	fmt.Printf("action=%s range=%d-%d/%d\n", response.Action, response.Start+1, response.Start+len(response.Candidates.Items), response.Total)
	for _, item := range response.Candidates.Items {
		meta := ""
		if item.Kind != "" {
			meta = " kind=" + item.Kind
			if item.Source != "" {
				meta += " source=" + item.Source
			}
		}
		if item.Comment != "" {
			meta += " comment=" + item.Comment
		}
		if item.Pinned {
			meta += " pinned=true"
		}
		fmt.Printf("%d. %s [%s] score=%d index=%d%s\n", item.DisplayIndex, item.Text, item.Reading, item.Score, item.Index, meta)
	}
}

func parseCandidateActionArgs(args []string) (string, candidateActionRequest, error) {
	var input []string
	request := candidateActionRequest{Action: "view"}
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if strings.HasPrefix(arg, "--") {
			value := ""
			if split := strings.IndexByte(arg, '='); split >= 0 {
				value = arg[split+1:]
				arg = arg[:split]
			} else if i+1 < len(args) {
				value = args[i+1]
				i++
			}
			if err := applyCandidateActionOption(&request, arg, value); err != nil {
				return "", request, err
			}
			continue
		}
		if isCandidateActionName(arg) {
			request.Action = strings.ToLower(arg)
			continue
		}
		input = append(input, arg)
	}
	joined := strings.TrimSpace(strings.Join(input, " "))
	if joined == "" && !isAssociationAction(request.Action) && request.Context == "" {
		return "", request, fmt.Errorf("missing candidate input")
	}
	return joined, request, nil
}

func applyCandidateActionOption(request *candidateActionRequest, option string, value string) error {
	option = strings.TrimSpace(strings.ToLower(option))
	value = strings.TrimSpace(value)
	switch option {
	case "--side":
		request.Side = value
		return nil
	case "--context", "--input":
		request.Context = value
		return nil
	case "--action":
		if !isCandidateActionName(value) {
			return fmt.Errorf("unknown candidate action %q", value)
		}
		request.Action = strings.ToLower(value)
		return nil
	case "--index":
		return setCandidateActionInt(value, &request.Index)
	case "--display-index", "--display":
		return setCandidateActionInt(value, &request.DisplayIndex)
	case "--start":
		return setCandidateActionInt(value, &request.Start)
	case "--limit":
		return setCandidateActionInt(value, &request.Limit)
	case "--page-size":
		return setCandidateActionInt(value, &request.PageSize)
	case "--delta":
		return setCandidateActionInt(value, &request.Delta)
	default:
		return fmt.Errorf("unknown candidate option %s", option)
	}
}

func setCandidateActionInt(value string, target *int) error {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("invalid integer %q", value)
	}
	*target = parsed
	return nil
}

func isCandidateActionName(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "view", "page", "candidates", "candidate-page",
		"next", "next-page", "page-next",
		"prev", "previous", "previous-page", "prev-page", "page-prev",
		"home", "first-page", "end", "last-page",
		"select", "commit", "commit-candidate",
		"forget", "reject", "delete-candidate", "hide-candidate",
		"pin", "pin-candidate", "favorite", "top",
		"associate", "association", "associations", "predict", "suggest",
		"first-char", "commit-first-char",
		"last-char", "commit-last-char",
		"select-char", "commit-char", "commit-candidate-char":
		return true
	default:
		return false
	}
}

func isAssociationAction(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "associate", "association", "associations", "predict", "suggest":
		return true
	default:
		return false
	}
}

func updateCheckCmd(client *http.Client) error {
	var check updateCheck
	if err := getJSON(client, "/updates/check", &check); err != nil {
		return err
	}
	fmt.Printf("current=%s latest=%s available=%v\n", check.CurrentVersion, check.LatestVersion, check.UpdateAvailable)
	fmt.Println(check.ManifestURL)
	return nil
}

func updateApply(client *http.Client) error {
	body, err := post(client, "/updates/apply", nil)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func updateSources(client *http.Client) error {
	var response dictionarySourceResponse
	if err := getJSON(client, "/updates/sources", &response); err != nil {
		return err
	}
	for _, source := range response.Sources {
		marker := " "
		if source.ID == response.Selected {
			marker = "*"
		}
		installable := "source-only"
		if source.Installable {
			installable = "installable"
		}
		fmt.Printf("%s %s [%s] %s license=%s\n", marker, source.ID, source.Kind, installable, source.License)
		fmt.Printf("  %s\n", source.Name)
		fmt.Printf("  %s\n", source.Homepage)
		if len(source.ManifestURLs) > 0 {
			fmt.Printf("  manifest: %s\n", strings.Join(source.ManifestURLs, ", "))
		}
		for _, raw := range source.RawSources {
			fmt.Printf("  raw: %s %s %s\n", raw.Role, raw.Label, raw.URL)
		}
		if source.ConvertCommand != "" {
			fmt.Printf("  convert: %s\n", source.ConvertCommand)
		}
		if source.SyncCommand != "" {
			fmt.Printf("  sync: %s\n", source.SyncCommand)
		}
	}
	return nil
}

func updateSource(client *http.Client, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("missing update source id")
	}
	body, err := postJSONBytes(client, "/updates/source", map[string]string{"id": strings.TrimSpace(args[0])})
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func schemas(client *http.Client) error {
	var response schemaResponse
	if err := getJSON(client, "/schemas", &response); err != nil {
		return err
	}
	printSchemas(response)
	return nil
}

func schema(client *http.Client, args []string) error {
	if len(args) == 0 || strings.EqualFold(args[0], "current") {
		var response schemaResponse
		if err := getJSON(client, "/schemas", &response); err != nil {
			return err
		}
		fmt.Printf("schema=%s doublePinyin=%v scheme=%s\n", response.Selected, response.Config.DoublePinyin, response.Config.DoublePinyinScheme)
		return nil
	}
	action := strings.ToLower(strings.TrimSpace(args[0]))
	if action == "list" || action == "ls" {
		return schemas(client)
	}
	if action == "apply" || action == "set" || action == "use" {
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("missing schema id")
		}
		var response schemaResponse
		if err := postJSON(client, "/schemas/apply", map[string]string{"id": strings.TrimSpace(args[1])}, &response); err != nil {
			return err
		}
		fmt.Printf("schema=%s doublePinyin=%v scheme=%s\n", response.Selected, response.Config.DoublePinyin, response.Config.DoublePinyinScheme)
		return nil
	}
	var response schemaResponse
	if err := postJSON(client, "/schemas/apply", map[string]string{"id": strings.TrimSpace(args[0])}, &response); err != nil {
		return err
	}
	fmt.Printf("schema=%s doublePinyin=%v scheme=%s\n", response.Selected, response.Config.DoublePinyin, response.Config.DoublePinyinScheme)
	return nil
}

func printSchemas(response schemaResponse) {
	for _, schema := range response.Schemas {
		marker := " "
		if schema.ID == response.Selected {
			marker = "*"
		}
		scheme := "full"
		if schema.DoublePinyin {
			scheme = "double:" + schema.DoublePinyinScheme
		}
		fmt.Printf("%s %s [%s] %s\n", marker, schema.ID, schema.Kind, scheme)
		fmt.Printf("  %s\n", schema.Name)
		if schema.RimeID != "" {
			fmt.Printf("  rime: %s\n", schema.RimeID)
		}
		if schema.DictionarySourcePreset != "" {
			fmt.Printf("  dictionary: %s\n", schema.DictionarySourcePreset)
		}
		fmt.Printf("  %s\n", schema.Description)
	}
}

func rimeCustom(client *http.Client, args []string) error {
	action := "import"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	switch action {
	case "import", "apply", "custom":
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("missing Rime custom yaml file")
		}
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		var result rimeCustomResult
		if err := postJSON(client, "/rime/custom", map[string]string{"yaml": string(data)}, &result); err != nil {
			return err
		}
		fmt.Printf("schema=%s doublePinyin=%v scheme=%s pageSize=%d layout=%s punctuation=%s keyProfile=%s\n",
			valueOr(result.Schema, result.Config.Schema),
			result.Config.DoublePinyin,
			result.Config.DoublePinyinScheme,
			result.Config.CandidatePageSize,
			result.Config.CandidateLayout,
			result.Config.Punctuation,
			result.Config.KeyProfile,
		)
		if len(result.Applied) > 0 {
			fmt.Printf("applied=%s\n", strings.Join(result.Applied, ", "))
		}
		if len(result.Warnings) > 0 {
			fmt.Printf("warnings=%s\n", strings.Join(result.Warnings, ", "))
		}
		return nil
	default:
		return fmt.Errorf("unknown rime action %q", action)
	}
}

func mode(client *http.Client, args []string) error {
	if len(args) == 0 {
		var state engineState
		if err := getJSON(client, "/ime/mode", &state); err != nil {
			return err
		}
		fmt.Println(state.Mode)
		return nil
	}
	next := strings.TrimSpace(strings.ToLower(strings.Join(args, " ")))
	payload := map[string]any{"mode": next}
	if next == "toggle" {
		payload = map[string]any{"toggle": true}
	}
	var state engineState
	if err := postJSON(client, "/ime/mode", payload, &state); err != nil {
		return err
	}
	fmt.Println(state.Mode)
	return nil
}

func switches(client *http.Client) error {
	var response switchResponse
	if err := getJSON(client, "/switches", &response); err != nil {
		return err
	}
	printSwitches(response.Switches)
	return nil
}

func applySwitch(client *http.Client, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("missing switch id")
	}
	payload := map[string]any{"id": strings.TrimSpace(args[0])}
	if len(args) > 1 {
		switch strings.ToLower(strings.TrimSpace(args[1])) {
		case "on", "true", "1", "enable", "enabled":
			payload["value"] = true
		case "off", "false", "0", "disable", "disabled":
			payload["value"] = false
		case "toggle", "":
			payload["action"] = "toggle"
		default:
			return fmt.Errorf("unknown switch value %q", args[1])
		}
	} else {
		payload["action"] = "toggle"
	}
	var response switchResponse
	if err := postJSON(client, "/switches/apply", payload, &response); err != nil {
		return err
	}
	if response.Selected != nil {
		value := response.Selected.Off
		if response.Selected.Value {
			value = response.Selected.On
		}
		fmt.Printf("%s=%v (%s)\n", response.Selected.ID, response.Selected.Value, value)
		return nil
	}
	printSwitches(response.Switches)
	return nil
}

func printSwitches(items []switchOption) {
	for _, item := range items {
		value := item.Off
		if item.Value {
			value = item.On
		}
		fmt.Printf("%s=%v %s [%s]\n", item.ID, item.Value, value, item.ConfigField)
		if item.RimeName != "" && item.RimeName != item.ID {
			fmt.Printf("  rime: %s\n", item.RimeName)
		}
		fmt.Printf("  %s\n", item.Description)
	}
}

func appRules(client *http.Client) error {
	var response appRuleResponse
	if err := getJSON(client, "/app-rules", &response); err != nil {
		return err
	}
	for _, rule := range response.Rules {
		fmt.Printf("%s [%s] mode=%s punct=%s priority=%d\n", rule.ID, rule.Name, valueOr(rule.Mode, "keep"), valueOr(rule.Punctuation, "keep"), rule.Priority)
		if rule.DisableCandidates {
			fmt.Println("  disableCandidates=true")
		}
		if rule.DisableLearning {
			fmt.Println("  disableLearning=true")
		}
		if len(rule.ProcessNames) > 0 {
			fmt.Printf("  process=%s\n", strings.Join(rule.ProcessNames, ","))
		}
		if len(rule.ExeContains) > 0 {
			fmt.Printf("  pathContains=%s\n", strings.Join(rule.ExeContains, ","))
		}
		if rule.Description != "" {
			fmt.Printf("  %s\n", rule.Description)
		}
	}
	return nil
}

func appContextCmd(client *http.Client, args []string) error {
	if len(args) == 0 || strings.EqualFold(args[0], "help") {
		return fmt.Errorf("usage: shurufa-imecli app-context resolve --process WeGame.exe [--path EXE] [--title TITLE] [--class CLASS] [--password] [--terminal] [--game]")
	}
	if strings.EqualFold(args[0], "resolve") {
		args = args[1:]
	}
	context, err := parseAppContextArgs(args)
	if err != nil {
		return err
	}
	var decision appContextDecision
	if err := postJSON(client, "/app-context/resolve", context, &decision); err != nil {
		return err
	}
	fmt.Printf("matched=%v mode=%s punctuation=%s layout=%s disableCandidates=%v disableLearning=%v reason=%s\n",
		decision.Matched,
		decision.Mode,
		decision.Punctuation,
		decision.CandidateLayout,
		decision.DisableCandidates,
		decision.DisableLearning,
		decision.Reason,
	)
	if decision.Rule != nil {
		fmt.Printf("rule=%s [%s]\n", decision.Rule.ID, decision.Rule.Name)
	}
	return nil
}

func parseAppContextArgs(args []string) (appContext, error) {
	var context appContext
	for i := 0; i < len(args); i++ {
		arg := args[i]
		valueArg := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for %s", arg)
			}
			i++
			return args[i], nil
		}
		switch arg {
		case "--process", "-p":
			value, err := valueArg()
			if err != nil {
				return context, err
			}
			context.ProcessName = value
		case "--path":
			value, err := valueArg()
			if err != nil {
				return context, err
			}
			context.ExePath = value
		case "--title", "-t":
			value, err := valueArg()
			if err != nil {
				return context, err
			}
			context.WindowTitle = value
		case "--class":
			value, err := valueArg()
			if err != nil {
				return context, err
			}
			context.WindowClass = value
		case "--password":
			context.PasswordField = true
		case "--terminal":
			context.Terminal = true
		case "--game":
			context.GameMode = true
		default:
			if strings.HasPrefix(arg, "--process=") {
				context.ProcessName = strings.TrimPrefix(arg, "--process=")
			} else if strings.HasPrefix(arg, "--path=") {
				context.ExePath = strings.TrimPrefix(arg, "--path=")
			} else if strings.HasPrefix(arg, "--title=") {
				context.WindowTitle = strings.TrimPrefix(arg, "--title=")
			} else if strings.HasPrefix(arg, "--class=") {
				context.WindowClass = strings.TrimPrefix(arg, "--class=")
			} else if context.ProcessName == "" {
				context.ProcessName = arg
			} else {
				return context, fmt.Errorf("unknown app-context argument %q", arg)
			}
		}
	}
	if context.ProcessName == "" && context.ExePath == "" && context.WindowTitle == "" && context.WindowClass == "" &&
		!context.PasswordField && !context.Terminal && !context.GameMode {
		return context, fmt.Errorf("missing app context matcher")
	}
	return context, nil
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func wordbook(client *http.Client, args []string) error {
	action := "list"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	switch action {
	case "list":
		var response wordbookResponse
		if err := getJSON(client, "/wordbook", &response); err != nil {
			return err
		}
		printWordbook(response.UserScores)
		return nil
	case "export":
		var response wordbookResponse
		if err := getJSON(client, "/wordbook", &response); err != nil {
			return err
		}
		data, err := json.MarshalIndent(map[string]any{"userScores": response.UserScores}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	case "import":
		if len(args) == 0 {
			return fmt.Errorf("missing import file")
		}
		scores, err := readWordbookFile(args[0])
		if err != nil {
			return err
		}
		payload := map[string]any{"userScores": scores, "merge": true}
		if len(args) > 1 && args[1] == "--replace" {
			payload["merge"] = false
		}
		var response wordbookResponse
		if err := putJSON(client, "/wordbook", payload, &response); err != nil {
			return err
		}
		fmt.Printf("imported=%d\n", response.Count)
		return nil
	case "delete":
		if len(args) == 0 {
			return fmt.Errorf("missing wordbook key")
		}
		var response wordbookResponse
		if err := deleteJSON(client, "/wordbook?key="+urlQueryEscape(args[0]), &response); err != nil {
			return err
		}
		fmt.Printf("remaining=%d\n", response.Count)
		return nil
	case "clear":
		var response wordbookResponse
		if err := deleteJSON(client, "/wordbook", &response); err != nil {
			return err
		}
		fmt.Printf("remaining=%d\n", response.Count)
		return nil
	default:
		return fmt.Errorf("unknown wordbook action %q", action)
	}
}

func printWordbook(scores map[string]int) {
	keys := make([]string, 0, len(scores))
	for key := range scores {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if scores[keys[i]] == scores[keys[j]] {
			return keys[i] < keys[j]
		}
		return scores[keys[i]] > scores[keys[j]]
	})
	for _, key := range keys {
		fmt.Printf("%s\t%d\n", key, scores[key])
	}
}

func readWordbookFile(path string) (map[string]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrapped struct {
		UserScores map[string]int `json:"userScores"`
		Scores     map[string]int `json:"scores"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && (wrapped.UserScores != nil || wrapped.Scores != nil) {
		if wrapped.UserScores != nil {
			return wrapped.UserScores, nil
		}
		return wrapped.Scores, nil
	}
	var scores map[string]int
	if err := json.Unmarshal(data, &scores); err != nil {
		return nil, err
	}
	return scores, nil
}

func phrases(client *http.Client, args []string) error {
	action := "list"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	switch action {
	case "list":
		var response phraseResponse
		if err := getJSON(client, "/phrases", &response); err != nil {
			return err
		}
		printPhrases(response.Phrases)
		return nil
	case "export":
		var response phraseResponse
		if err := getJSON(client, "/phrases", &response); err != nil {
			return err
		}
		data, err := json.MarshalIndent(map[string]any{"entries": response.Phrases}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: phrases add <reading> <text> [weight]")
		}
		entry := phraseEntry{Reading: args[0], Text: args[1]}
		if len(args) > 2 {
			weight, err := strconv.Atoi(args[2])
			if err != nil {
				return err
			}
			entry.Weight = weight
		}
		var response phraseResponse
		if err := putJSON(client, "/phrases", map[string]any{"entries": []phraseEntry{entry}, "merge": true}, &response); err != nil {
			return err
		}
		fmt.Printf("phrases=%d\n", response.Count)
		return nil
	case "import":
		if len(args) == 0 {
			return fmt.Errorf("missing import file")
		}
		entries, err := readPhraseFile(args[0])
		if err != nil {
			return err
		}
		payload := map[string]any{"entries": entries, "merge": true}
		if len(args) > 1 && args[1] == "--replace" {
			payload["merge"] = false
		}
		var response phraseResponse
		if err := putJSON(client, "/phrases", payload, &response); err != nil {
			return err
		}
		fmt.Printf("phrases=%d\n", response.Count)
		return nil
	case "delete":
		if len(args) == 0 {
			return fmt.Errorf("missing phrase key")
		}
		var response phraseResponse
		if err := deleteJSON(client, "/phrases?key="+urlQueryEscape(args[0]), &response); err != nil {
			return err
		}
		fmt.Printf("phrases=%d\n", response.Count)
		return nil
	case "clear":
		var response phraseResponse
		if err := deleteJSON(client, "/phrases", &response); err != nil {
			return err
		}
		fmt.Printf("phrases=%d\n", response.Count)
		return nil
	default:
		return fmt.Errorf("unknown phrases action %q", action)
	}
}

func printPhrases(entries []phraseEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Reading == entries[j].Reading {
			return entries[i].Text < entries[j].Text
		}
		return entries[i].Reading < entries[j].Reading
	})
	for _, entry := range entries {
		fmt.Printf("%s|%s\t%d\t%s\n", entry.Reading, entry.Text, entry.Weight, entry.Comment)
	}
}

func readPhraseFile(path string) ([]phraseEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrapped struct {
		Entries []phraseEntry `json:"entries"`
		Phrases []phraseEntry `json:"phrases"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && (wrapped.Entries != nil || wrapped.Phrases != nil) {
		if wrapped.Entries != nil {
			return wrapped.Entries, nil
		}
		return wrapped.Phrases, nil
	}
	var entries []phraseEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func rejects(client *http.Client, args []string) error {
	action := "list"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	switch action {
	case "list":
		var response rejectResponse
		if err := getJSON(client, "/rejects", &response); err != nil {
			return err
		}
		printPhrases(rejectEntries(response))
		return nil
	case "export":
		var response rejectResponse
		if err := getJSON(client, "/rejects", &response); err != nil {
			return err
		}
		data, err := json.MarshalIndent(map[string]any{"entries": rejectEntries(response)}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: rejects add <reading> <text>")
		}
		entry := phraseEntry{Reading: args[0], Text: args[1], Comment: "已屏蔽"}
		var response rejectResponse
		if err := putJSON(client, "/rejects", map[string]any{"entries": []phraseEntry{entry}, "merge": true}, &response); err != nil {
			return err
		}
		fmt.Printf("rejects=%d\n", response.Count)
		return nil
	case "import":
		if len(args) == 0 {
			return fmt.Errorf("missing import file")
		}
		entries, err := readPhraseFile(args[0])
		if err != nil {
			return err
		}
		payload := map[string]any{"entries": entries, "merge": true}
		if len(args) > 1 && args[1] == "--replace" {
			payload["merge"] = false
		}
		var response rejectResponse
		if err := putJSON(client, "/rejects", payload, &response); err != nil {
			return err
		}
		fmt.Printf("rejects=%d\n", response.Count)
		return nil
	case "delete":
		if len(args) == 0 {
			return fmt.Errorf("missing reject key")
		}
		var response rejectResponse
		if err := deleteJSON(client, "/rejects?key="+urlQueryEscape(args[0]), &response); err != nil {
			return err
		}
		fmt.Printf("rejects=%d\n", response.Count)
		return nil
	case "clear":
		var response rejectResponse
		if err := deleteJSON(client, "/rejects", &response); err != nil {
			return err
		}
		fmt.Printf("rejects=%d\n", response.Count)
		return nil
	default:
		return fmt.Errorf("unknown rejects action %q", action)
	}
}

func rejectEntries(response rejectResponse) []phraseEntry {
	if response.Rejects != nil {
		return response.Rejects
	}
	return response.Entries
}

func pins(client *http.Client, args []string) error {
	action := "list"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	switch action {
	case "list":
		var response pinResponse
		if err := getJSON(client, "/pins", &response); err != nil {
			return err
		}
		printPhrases(pinEntries(response))
		return nil
	case "export":
		var response pinResponse
		if err := getJSON(client, "/pins", &response); err != nil {
			return err
		}
		data, err := json.MarshalIndent(map[string]any{"entries": pinEntries(response)}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: pins add <reading> <text> [weight]")
		}
		entry := phraseEntry{Reading: args[0], Text: args[1], Comment: "已置顶"}
		if len(args) > 2 {
			weight, err := strconv.Atoi(args[2])
			if err != nil {
				return err
			}
			entry.Weight = weight
		}
		var response pinResponse
		if err := putJSON(client, "/pins", map[string]any{"entries": []phraseEntry{entry}, "merge": true}, &response); err != nil {
			return err
		}
		fmt.Printf("pins=%d\n", response.Count)
		return nil
	case "import":
		if len(args) == 0 {
			return fmt.Errorf("missing import file")
		}
		entries, err := readPhraseFile(args[0])
		if err != nil {
			return err
		}
		payload := map[string]any{"entries": entries, "merge": true}
		if len(args) > 1 && args[1] == "--replace" {
			payload["merge"] = false
		}
		var response pinResponse
		if err := putJSON(client, "/pins", payload, &response); err != nil {
			return err
		}
		fmt.Printf("pins=%d\n", response.Count)
		return nil
	case "delete":
		if len(args) == 0 {
			return fmt.Errorf("missing pin key")
		}
		var response pinResponse
		if err := deleteJSON(client, "/pins?key="+urlQueryEscape(args[0]), &response); err != nil {
			return err
		}
		fmt.Printf("pins=%d\n", response.Count)
		return nil
	case "clear":
		var response pinResponse
		if err := deleteJSON(client, "/pins", &response); err != nil {
			return err
		}
		fmt.Printf("pins=%d\n", response.Count)
		return nil
	default:
		return fmt.Errorf("unknown pins action %q", action)
	}
}

func pinEntries(response pinResponse) []phraseEntry {
	if response.Pins != nil {
		return response.Pins
	}
	return response.Entries
}

func profile(client *http.Client, args []string) error {
	action := "export"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	switch action {
	case "export", "backup":
		var bundle profileBundle
		if err := getJSON(client, "/profile", &bundle); err != nil {
			return err
		}
		data, err := json.MarshalIndent(bundle, "", "  ")
		if err != nil {
			return err
		}
		if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
			if err := os.WriteFile(args[0], data, 0o644); err != nil {
				return err
			}
			fmt.Printf("exported=%s\n", args[0])
			return nil
		}
		fmt.Println(string(data))
		return nil
	case "import", "restore":
		if len(args) == 0 {
			return fmt.Errorf("missing profile file")
		}
		bundle, err := readProfileFile(args[0])
		if err != nil {
			return err
		}
		bundle.Merge = true
		if len(args) > 1 && args[1] == "--replace" {
			bundle.Merge = false
		}
		var response profileBundle
		if err := putJSON(client, "/profile", bundle, &response); err != nil {
			return err
		}
		fmt.Printf("profile imported scores=%d phrases=%d rejects=%d pins=%d\n",
			response.Counts["userScores"],
			response.Counts["phrases"],
			response.Counts["rejects"],
			response.Counts["pins"],
		)
		return nil
	default:
		return fmt.Errorf("unknown profile action %q", action)
	}
}

func readProfileFile(path string) (profileBundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return profileBundle{}, err
	}
	var wrapped struct {
		Profile *profileBundle `json:"profile"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && wrapped.Profile != nil {
		return *wrapped.Profile, nil
	}
	var bundle profileBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return profileBundle{}, err
	}
	return bundle, nil
}

func catalog(client *http.Client, args []string) error {
	kind, query, limit, err := parseCatalogArgs(args)
	if err != nil {
		return err
	}
	values := url.Values{}
	if kind != "" {
		values.Set("kind", kind)
	}
	if query != "" {
		values.Set("q", query)
	}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	path := "/catalog"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var response catalogResponse
	if err := getJSON(client, path, &response); err != nil {
		return err
	}
	printCatalog(response)
	return nil
}

func parseCatalogArgs(args []string) (string, string, int, error) {
	kind := "all"
	limit := 80
	var query []string
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if arg == "--limit" || arg == "-n" {
			if i+1 >= len(args) {
				return "", "", 0, fmt.Errorf("%s requires a value", arg)
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				return "", "", 0, err
			}
			limit = value
			i++
			continue
		}
		if strings.HasPrefix(arg, "--limit=") {
			value, err := strconv.Atoi(strings.TrimPrefix(arg, "--limit="))
			if err != nil {
				return "", "", 0, err
			}
			limit = value
			continue
		}
		if isCatalogKindArg(arg) && len(query) == 0 && kind == "all" {
			kind = strings.ToLower(arg)
			continue
		}
		query = append(query, arg)
	}
	return kind, strings.TrimSpace(strings.Join(query, " ")), limit, nil
}

func isCatalogKindArg(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "all", "emoji", "kaomoji", "symbol", "symbols", "agent", "ai":
		return true
	default:
		return false
	}
}

func printCatalog(response catalogResponse) {
	fmt.Printf("kind=%s query=%s count=%d\n", response.Kind, response.Query, response.Count)
	for _, entry := range response.Entries {
		meta := entry.Kind
		if entry.Comment != "" {
			meta += "/" + entry.Comment
		}
		fmt.Printf("%s\t%s\t%s\t%d\t%s\n", entry.Reading, entry.Text, meta, entry.Weight, entry.Source)
	}
}

func reverseLookup(client *http.Client, args []string) error {
	query, limit, err := parseReverseLookupArgs(args)
	if err != nil {
		return err
	}
	if query == "" {
		return fmt.Errorf("missing reverse lookup text")
	}
	values := url.Values{"q": []string{query}}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	var response reverseLookupResponse
	if err := getJSON(client, "/engine/reverse?"+values.Encode(), &response); err != nil {
		return err
	}
	printReverseLookup(response)
	return nil
}

func parseReverseLookupArgs(args []string) (string, int, error) {
	limit := 20
	var query []string
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if arg == "--limit" || arg == "-n" {
			if i+1 >= len(args) {
				return "", 0, fmt.Errorf("%s requires a value", arg)
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				return "", 0, err
			}
			limit = value
			i++
			continue
		}
		if strings.HasPrefix(arg, "--limit=") {
			value, err := strconv.Atoi(strings.TrimPrefix(arg, "--limit="))
			if err != nil {
				return "", 0, err
			}
			limit = value
			continue
		}
		query = append(query, arg)
	}
	return strings.TrimSpace(strings.Join(query, " ")), limit, nil
}

func printReverseLookup(response reverseLookupResponse) {
	fmt.Printf("query=%s count=%d\n", response.Query, response.Count)
	for _, entry := range response.Entries {
		meta := entry.Kind
		if entry.Comment != "" {
			meta += "/" + entry.Comment
		}
		fmt.Printf("%s\t%s\t%s\t%d\t%s\n", entry.Text, entry.Reading, meta, entry.Weight, entry.Source)
	}
}

func agent(client *http.Client, args []string) error {
	input, context := parseAgentArgs(args)
	if input == "" {
		return fmt.Errorf("missing agent input")
	}
	var response agentResponse
	payload := map[string]string{"input": input}
	if context != "" {
		payload["context"] = context
	}
	if err := postJSON(client, "/agent/compose", payload, &response); err != nil {
		return err
	}
	if len(response.Items) > 0 {
		for i, item := range response.Items {
			fmt.Printf("%d. %s [%s/%s]\n", i+1, item.Text, item.Intent, item.Action)
		}
		return nil
	}
	for i, item := range response.Candidates {
		fmt.Printf("%d. %s\n", i+1, item)
	}
	return nil
}

func parseAgentArgs(args []string) (string, string) {
	var context string
	var input []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--context", "-c":
			if i+1 < len(args) {
				context = args[i+1]
				i++
			}
		default:
			input = append(input, args[i])
		}
	}
	return strings.TrimSpace(strings.Join(input, " ")), strings.TrimSpace(context)
}

func repl(client *http.Client) error {
	fmt.Println("shurufa233 CLI REPL. Type pinyin, /agent text, /quit.")
	var line string
	for {
		fmt.Print("> ")
		if _, err := fmt.Scanln(&line); err != nil {
			return nil
		}
		if line == "/quit" {
			return nil
		}
		if strings.HasPrefix(line, "/agent") {
			args := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, "/agent")))
			if err := agent(client, args); err != nil {
				fmt.Println(err)
			}
			continue
		}
		if err := preview(client, line); err != nil {
			fmt.Println(err)
		}
	}
}

func getJSON(client *http.Client, path string, out any) error {
	body, err := get(client, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func postJSON(client *http.Client, path string, payload any, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body, err := post(client, path, data)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func postJSONBytes(client *http.Client, path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return post(client, path, data)
}

func putJSON(client *http.Client, path string, payload any, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body, err := request(client, http.MethodPut, path, data)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func get(client *http.Client, path string) ([]byte, error) {
	resp, err := client.Get(apiBase + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readResponse(resp)
}

func post(client *http.Client, path string, body []byte) ([]byte, error) {
	return request(client, http.MethodPost, path, body)
}

func deleteJSON(client *http.Client, path string, out any) error {
	body, err := request(client, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func request(client *http.Client, method string, path string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, apiBase+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readResponse(resp)
}

func urlQueryEscape(value string) string {
	return url.QueryEscape(value)
}

func readResponse(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s returned HTTP %d: %s", resp.Request.URL.Path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
