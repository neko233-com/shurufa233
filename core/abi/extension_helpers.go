package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/neko233-com/shurufa233/core/engine"
)

const abiVersion = "2026.07.05"

var abiFeatureList = []string{
	"candidate-payload-v2",
	"state-json",
	"config-json",
	"reload-config",
	"reload-dictionaries",
	"dictionary-manifest-json",
	"dictionary-source-presets",
	"schema-presets-json",
	"apply-schema-json",
	"rime-custom-yaml",
	"reverse-lookup-json",
	"user-scores-json",
	"user-phrases-json",
	"user-rejects-json",
	"user-pins-json",
	"commit-text",
	"agent-compose",
	"rime-compatible-dictionaries",
	"gzip-dictionaries",
	"abbreviation-candidates",
	"pinyin-separators",
	"rime-symbol-prefix",
	"emoji-kaomoji-symbol-candidates",
	"catalog-json",
	"dynamic-datetime-candidates",
	"candidate-char-commit",
	"candidate-comments",
	"association-candidates",
	"candidate-action-json",
	"extension-command-json",
	"key-behavior-config",
	"rime-switches-json",
	"rime-recognizer-patterns-json",
	"app-context-rules-json",
	"profile-bundle-json",
	"key-event-json",
}

type abiEnvelope struct {
	OK        bool      `json:"ok"`
	Error     string    `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type candidatePayloadV2 struct {
	OK        bool                     `json:"ok"`
	Start     int                      `json:"start"`
	Limit     int                      `json:"limit"`
	Total     int                      `json:"total"`
	Items     []candidatePayloadV2Item `json:"items"`
	UpdatedAt time.Time                `json:"updatedAt"`
}

type candidatePayloadV2Item struct {
	Index        int    `json:"index"`
	DisplayIndex int    `json:"displayIndex"`
	Text         string `json:"text"`
	Reading      string `json:"reading"`
	Kind         string `json:"kind,omitempty"`
	Source       string `json:"source,omitempty"`
	Comment      string `json:"comment,omitempty"`
	Weight       int    `json:"weight"`
	UserScore    int    `json:"userScore"`
	Score        int    `json:"score"`
	Pinned       bool   `json:"pinned,omitempty"`
}

type agentComposeResult struct {
	OK        bool             `json:"ok"`
	Input     string           `json:"input"`
	Context   string           `json:"context,omitempty"`
	Items     []agentCandidate `json:"items"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type agentCandidate struct {
	Intent string `json:"intent"`
	Action string `json:"action"`
	Text   string `json:"text"`
	Source string `json:"source"`
}

type candidateActionResult struct {
	OK         bool               `json:"ok"`
	Action     string             `json:"action"`
	Start      int                `json:"start"`
	Limit      int                `json:"limit"`
	Total      int                `json:"total"`
	Committed  string             `json:"committed,omitempty"`
	Rejected   *engine.Entry      `json:"rejected,omitempty"`
	Pinned     *engine.Entry      `json:"pinned,omitempty"`
	State      engine.State       `json:"state"`
	Candidates candidatePayloadV2 `json:"candidates"`
	UpdatedAt  time.Time          `json:"updatedAt"`
}

type keyEventResult struct {
	OK          bool                       `json:"ok"`
	Handled     bool                       `json:"handled"`
	Action      string                     `json:"action"`
	Key         string                     `json:"key,omitempty"`
	Character   string                     `json:"character,omitempty"`
	Committed   string                     `json:"committed,omitempty"`
	PassThrough string                     `json:"passThrough,omitempty"`
	Reason      string                     `json:"reason,omitempty"`
	Decision    *engine.AppContextDecision `json:"decision,omitempty"`
	State       engine.State               `json:"state"`
	Candidates  candidatePayloadV2         `json:"candidates"`
	UpdatedAt   time.Time                  `json:"updatedAt"`
}

type profileBundle struct {
	OK         bool           `json:"ok,omitempty"`
	Version    int            `json:"version"`
	Product    string         `json:"product"`
	ExportedAt time.Time      `json:"exportedAt"`
	Config     *engine.Config `json:"config,omitempty"`
	UserScores map[string]int `json:"userScores,omitempty"`
	Phrases    []engine.Entry `json:"phrases,omitempty"`
	Rejects    []engine.Entry `json:"rejects,omitempty"`
	Pins       []engine.Entry `json:"pins,omitempty"`
	Merge      bool           `json:"merge,omitempty"`
	Counts     map[string]int `json:"counts,omitempty"`
}

type extensionCommandPayload struct {
	Input        string             `json:"input,omitempty"`
	Context      string             `json:"context,omitempty"`
	Action       string             `json:"action,omitempty"`
	Key          string             `json:"key,omitempty"`
	Character    string             `json:"character,omitempty"`
	Code         int                `json:"code,omitempty"`
	Mode         string             `json:"mode,omitempty"`
	Toggle       bool               `json:"toggle,omitempty"`
	Ctrl         bool               `json:"ctrl,omitempty"`
	Alt          bool               `json:"alt,omitempty"`
	Shift        bool               `json:"shift,omitempty"`
	Meta         bool               `json:"meta,omitempty"`
	Modifiers    []string           `json:"modifiers,omitempty"`
	Index        int                `json:"index,omitempty"`
	DisplayIndex int                `json:"displayIndex,omitempty"`
	Start        int                `json:"start,omitempty"`
	Limit        int                `json:"limit,omitempty"`
	PageSize     int                `json:"pageSize,omitempty"`
	Delta        int                `json:"delta,omitempty"`
	Side         string             `json:"side,omitempty"`
	ID           string             `json:"id,omitempty"`
	Switch       string             `json:"switch,omitempty"`
	Value        *bool              `json:"value,omitempty"`
	Schema       string             `json:"schema,omitempty"`
	YAML         string             `json:"yaml,omitempty"`
	Reading      string             `json:"reading,omitempty"`
	Text         string             `json:"text,omitempty"`
	Kind         string             `json:"kind,omitempty"`
	Query        string             `json:"query,omitempty"`
	Config       *engine.Config     `json:"config,omitempty"`
	AppContext   *engine.AppContext `json:"appContext,omitempty"`
	UserScores   map[string]int     `json:"userScores,omitempty"`
	Scores       map[string]int     `json:"scores,omitempty"`
	Entries      []engine.Entry     `json:"entries,omitempty"`
	Phrases      []engine.Entry     `json:"phrases,omitempty"`
	Rejects      []engine.Entry     `json:"rejects,omitempty"`
	Pins         []engine.Entry     `json:"pins,omitempty"`
	Merge        bool               `json:"merge,omitempty"`
	Raw          *json.RawMessage   `json:"raw,omitempty"`
}

func buildProfileBundle(session *engine.Engine) profileBundle {
	config := session.Config()
	scores := session.UserScores()
	phrases := session.UserPhrases()
	rejects := session.UserRejects()
	pins := session.UserPins()
	return profileBundle{
		OK:         true,
		Version:    1,
		Product:    "shurufa233",
		ExportedAt: time.Now().UTC(),
		Config:     &config,
		UserScores: scores,
		Phrases:    phrases,
		Rejects:    rejects,
		Pins:       pins,
		Counts: map[string]int{
			"userScores": len(scores),
			"phrases":    len(phrases),
			"rejects":    len(rejects),
			"pins":       len(pins),
			"appRules":   len(config.AppRules),
		},
	}
}

func decodeProfileBundle(payload string) (profileBundle, error) {
	var wrapped struct {
		Profile *profileBundle `json:"profile"`
	}
	if err := json.Unmarshal([]byte(payload), &wrapped); err == nil && wrapped.Profile != nil {
		return *wrapped.Profile, nil
	}
	var bundle profileBundle
	if err := json.Unmarshal([]byte(payload), &bundle); err != nil {
		return profileBundle{}, err
	}
	return bundle, nil
}

func importProfileBundle(session *engine.Engine, bundle profileBundle) map[string]any {
	if bundle.Config != nil {
		config := normalizeConfig(*bundle.Config)
		session.Configure(config)
		_ = persistConfig(config)
	}
	if bundle.UserScores != nil {
		if bundle.Merge {
			session.ImportUserScores(bundle.UserScores)
		} else {
			session.ReplaceUserScores(bundle.UserScores)
		}
		persistUserScores(session.UserScores())
	}
	if bundle.Phrases != nil {
		entries := bundle.Phrases
		if bundle.Merge {
			entries = append(session.UserPhrases(), entries...)
		}
		session.ReplaceUserPhrases(entries)
		persistUserPhrases(session.UserPhrases())
	}
	if bundle.Rejects != nil {
		entries := bundle.Rejects
		if bundle.Merge {
			entries = append(session.UserRejects(), entries...)
		}
		session.ReplaceUserRejects(entries)
		persistUserRejects(session.UserRejects())
	}
	if bundle.Pins != nil {
		entries := bundle.Pins
		if bundle.Merge {
			entries = append(session.UserPins(), entries...)
		}
		session.ReplaceUserPins(entries)
		persistUserPins(session.UserPins())
	}
	result := buildProfileBundle(session)
	return map[string]any{
		"ok":        true,
		"imported":  true,
		"profile":   result,
		"counts":    result.Counts,
		"updatedAt": time.Now().UTC(),
	}
}

func applyRimeCustomPayload(session *engine.Engine, payload string) any {
	yamlText, err := decodeRimeCustomText(payload)
	if err != nil {
		return errorEnvelope(err.Error())
	}
	result, err := engine.ApplyRimeCustomYAML(session.Config(), []byte(yamlText))
	if err != nil {
		return errorEnvelope(err.Error())
	}
	config := normalizeConfig(result.Config)
	session.Configure(config)
	_ = persistConfig(config)
	result.Config = config
	result.Schema = config.Schema
	return result
}

func decodeRimeCustomText(payload string) (string, error) {
	req, err := decodeExtensionCommandPayload(payload)
	if err != nil {
		if strings.TrimSpace(payload) == "" {
			return "", err
		}
		return payload, nil
	}
	yamlText := firstNonEmpty(req.YAML, req.Text, req.Input)
	if strings.TrimSpace(yamlText) == "" {
		return "", errMissingRimeCustomYAML()
	}
	return yamlText, nil
}

func errMissingRimeCustomYAML() error {
	return &rimeCustomError{message: "missing rime custom yaml"}
}

type rimeCustomError struct {
	message string
}

func (e *rimeCustomError) Error() string {
	return e.message
}

func buildCandidatePayloadV2(session *engine.Engine, start int, limit int) candidatePayloadV2 {
	return buildCandidatePayloadV2FromState(session.State(), start, limit)
}

func buildCandidatePayloadV2FromState(state engine.State, start int, limit int) candidatePayloadV2 {
	if start < 0 {
		start = 0
	}
	if start > len(state.Candidates) {
		start = len(state.Candidates)
	}
	if limit <= 0 || limit > 64 {
		limit = 64
	}
	end := start + limit
	if end > len(state.Candidates) {
		end = len(state.Candidates)
	}
	items := make([]candidatePayloadV2Item, 0, end-start)
	for i, candidate := range state.Candidates[start:end] {
		index := start + i
		items = append(items, candidatePayloadV2Item{
			Index:        index,
			DisplayIndex: i + 1,
			Text:         candidate.Text,
			Reading:      candidate.Reading,
			Kind:         candidate.Kind,
			Source:       candidate.Source,
			Comment:      candidate.Comment,
			Weight:       candidate.Weight,
			UserScore:    candidate.UserScore,
			Score:        candidateScore(candidate),
			Pinned:       candidate.Pinned,
		})
	}
	return candidatePayloadV2{
		OK:        true,
		Start:     start,
		Limit:     limit,
		Total:     len(state.Candidates),
		Items:     items,
		UpdatedAt: time.Now().UTC(),
	}
}

func candidateScore(candidate engine.Candidate) int {
	score := candidate.Weight + candidate.UserScore
	if candidate.Pinned {
		score += 1000000
	}
	return score
}

func decodeUserScoresPayload(payload string) (map[string]int, error) {
	var wrapped struct {
		UserScores map[string]int `json:"userScores"`
		Scores     map[string]int `json:"scores"`
	}
	if err := json.Unmarshal([]byte(payload), &wrapped); err == nil {
		switch {
		case len(wrapped.UserScores) > 0:
			return wrapped.UserScores, nil
		case len(wrapped.Scores) > 0:
			return wrapped.Scores, nil
		}
	}
	var raw map[string]int
	err := json.Unmarshal([]byte(payload), &raw)
	return raw, err
}

func normalizeABIReading(input string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(input) {
		if r >= 'a' && r <= 'z' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func composeAgentABI(input string, context string) agentComposeResult {
	input = strings.TrimSpace(input)
	context = strings.TrimSpace(context)
	text := input
	if context != "" {
		text = context
	}
	result := agentComposeResult{
		OK:        true,
		Input:     input,
		Context:   context,
		UpdatedAt: time.Now().UTC(),
	}
	add := func(intent string, action string, value string) {
		result.Items = append(result.Items, agentCandidate{
			Intent: intent,
			Action: action,
			Text:   value,
			Source: "builtin-agent",
		})
	}
	lowered := strings.ToLower(input)
	switch {
	case strings.Contains(lowered, "rewrite") || strings.Contains(input, "润色"):
		add("rewrite", "agent.rewrite.polish", "请润色这段文字："+text)
		add("rewrite", "agent.rewrite.concise", "把下面内容改得更自然、更简洁："+text)
	case strings.Contains(lowered, "translate") || strings.Contains(input, "翻译"):
		add("translate", "agent.translate.zh", "请把这段内容翻译成中文："+text)
		add("translate", "agent.translate.en", "请把这段内容翻译成英文："+text)
	default:
		prefix := "作为输入法 agent，请处理："
		if context != "" {
			prefix = "结合当前上下文，作为输入法 agent，请处理："
		}
		add("compose", "agent.compose", prefix+input)
	}
	return result
}

func executeCandidateAction(session *engine.Engine, req extensionCommandPayload) any {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "view"
	}
	limit := req.Limit
	if limit <= 0 {
		limit = req.PageSize
	}
	if limit <= 0 {
		limit = 7
	}
	if limit > 64 {
		limit = 64
	}
	start := req.Start
	if start < 0 {
		start = 0
	}
	total := len(session.State().Candidates)
	if start > total {
		start = total
	}

	switch action {
	case "view", "page", "candidates", "candidate-page":
		return buildCandidateActionResult(session, action, start, limit, "")
	case "associate", "association", "associations", "predict", "suggest":
		state := session.Associate(firstNonEmpty(req.Context, req.Input, req.Text), limit)
		return buildCandidateActionResultWithState(action, 0, limit, "", state)
	case "next", "next-page", "page-next":
		step := limit
		if req.Delta > 0 {
			step = req.Delta * limit
		}
		start += step
		if start >= total {
			start = maxCandidatePageStart(total, limit)
		}
		return buildCandidateActionResult(session, action, start, limit, "")
	case "prev", "previous", "previous-page", "prev-page", "page-prev":
		step := limit
		if req.Delta > 0 {
			step = req.Delta * limit
		}
		start -= step
		if start < 0 {
			start = 0
		}
		return buildCandidateActionResult(session, action, start, limit, "")
	case "home", "first-page":
		return buildCandidateActionResult(session, action, 0, limit, "")
	case "end", "last-page":
		return buildCandidateActionResult(session, action, maxCandidatePageStart(total, limit), limit, "")
	case "select", "commit", "commit-candidate":
		index := candidateActionIndex(req, start)
		state, err := session.Select(index)
		if err != nil {
			return errorEnvelope(err.Error())
		}
		persistUserScores(session.UserScores())
		return buildCandidateActionResultWithState(action, 0, limit, state.Committed, state)
	case "forget", "reject", "delete-candidate", "hide-candidate":
		index := candidateActionIndex(req, start)
		state, rejected, err := session.RejectCandidate(index)
		if err != nil {
			return errorEnvelope(err.Error())
		}
		persistUserScoresReplaceSync(session.UserScores())
		persistUserRejects(session.UserRejects())
		return buildCandidateActionResultWithRejected(action, start, limit, state, rejected)
	case "pin", "pin-candidate", "favorite", "top":
		index := candidateActionIndex(req, start)
		state, pinned, err := session.PinCandidate(index)
		if err != nil {
			return errorEnvelope(err.Error())
		}
		persistUserPins(session.UserPins())
		return buildCandidateActionResultWithPinned(action, start, limit, state, pinned)
	case "first-char", "commit-first-char":
		return executeCandidateCharAction(session, req, start, limit, action, "first")
	case "last-char", "commit-last-char":
		return executeCandidateCharAction(session, req, start, limit, action, "last")
	case "select-char", "commit-char", "commit-candidate-char":
		return executeCandidateCharAction(session, req, start, limit, action, req.Side)
	default:
		return errorEnvelope("unknown candidate action: " + action)
	}
}

func executeCandidateCharAction(session *engine.Engine, req extensionCommandPayload, start int, limit int, action string, side string) any {
	index := candidateActionIndex(req, start)
	state, err := session.SelectChar(index, side)
	if err != nil {
		return errorEnvelope(err.Error())
	}
	return buildCandidateActionResultWithState(action, 0, limit, state.Committed, state)
}

func candidateActionIndex(req extensionCommandPayload, start int) int {
	if req.DisplayIndex > 0 {
		return start + req.DisplayIndex - 1
	}
	return req.Index
}

func executeKeyEvent(session *engine.Engine, req extensionCommandPayload) keyEventResult {
	key := normalizeKeyEventKey(req)
	character := keyEventCharacter(req)
	before := session.State()
	limit := req.Limit
	if limit <= 0 {
		limit = req.PageSize
	}
	if limit <= 0 {
		limit = session.Config().CandidatePageSize
	}

	var decision *engine.AppContextDecision
	if req.AppContext != nil {
		resolved := engine.ResolveAppContext(session.Config(), *req.AppContext)
		decision = &resolved
		if resolved.DisableCandidates || resolved.Mode == "en" {
			return buildKeyEventResult(session, "pass-through", key, character, "", character, "app-context", decision, false, req.Start, limit)
		}
	}
	if hasSystemModifier(req) {
		return buildKeyEventResult(session, "pass-through", key, character, "", character, "system-modifier", decision, false, req.Start, limit)
	}
	if before.Mode == "en" && character != "" {
		return buildKeyEventResult(session, "pass-through", key, character, "", character, "ascii-mode", decision, false, req.Start, limit)
	}

	switch key {
	case "shift", "leftshift", "rightshift":
		if session.Config().ShiftToggleMode && (strings.EqualFold(req.Action, "tap") || strings.EqualFold(req.Action, "up") || req.Toggle) {
			state := session.ToggleMode()
			return buildKeyEventResultFromState("toggle-mode", key, character, "", "", "", decision, true, req.Start, limit, state)
		}
		return buildKeyEventResult(session, "pass-through", key, character, "", "", "shift-down", decision, false, req.Start, limit)
	case "backspace", "bksp":
		if before.Buffer == "" {
			return buildKeyEventResult(session, "pass-through", key, character, "", "", "empty-buffer", decision, false, req.Start, limit)
		}
		return buildKeyEventResultFromState("backspace", key, character, "", "", "", decision, true, req.Start, limit, session.Backspace())
	case "escape", "esc":
		if before.Buffer == "" && len(before.Candidates) == 0 {
			return buildKeyEventResult(session, "pass-through", key, character, "", "", "empty-buffer", decision, false, req.Start, limit)
		}
		return buildKeyEventResultFromState("clear", key, character, "", "", "", decision, true, req.Start, limit, session.Clear())
	case "space", "enter", "return":
		if len(before.Candidates) > 0 {
			index := req.Index
			state, err := session.Select(index)
			if err != nil {
				return buildKeyEventResult(session, "error", key, character, "", "", err.Error(), decision, true, req.Start, limit)
			}
			persistUserScores(session.UserScores())
			return buildKeyEventResultFromState("commit-candidate", key, character, state.Committed, "", "", decision, true, 0, limit, state)
		}
		if before.Buffer != "" {
			committed := before.Buffer
			if key == "space" {
				committed += " "
			}
			state := session.Clear()
			return buildKeyEventResultFromState("commit-raw", key, character, committed, "", "", decision, true, 0, limit, state)
		}
		return buildKeyEventResult(session, "pass-through", key, character, "", character, "empty-buffer", decision, false, req.Start, limit)
	}

	if index, ok := keyEventCandidateIndex(key, req.Start); ok && len(before.Candidates) > index {
		state, err := session.Select(index)
		if err != nil {
			return buildKeyEventResult(session, "error", key, character, "", "", err.Error(), decision, true, req.Start, limit)
		}
		persistUserScores(session.UserScores())
		return buildKeyEventResultFromState("commit-candidate", key, character, state.Committed, "", "", decision, true, 0, limit, state)
	}
	if character != "" {
		runes := []rune(character)
		if len(runes) != 1 || !isABIKeyEventInputRune(runes[0]) {
			return buildKeyEventResult(session, "pass-through", key, character, "", character, "non-composing-character", decision, false, req.Start, limit)
		}
		state := session.InputKey(runes[0])
		return buildKeyEventResultFromState("input", key, character, "", "", "", decision, true, req.Start, limit, state)
	}
	return buildKeyEventResult(session, "pass-through", key, character, "", "", "unhandled-key", decision, false, req.Start, limit)
}

func buildKeyEventResult(session *engine.Engine, action string, key string, character string, committed string, passThrough string, reason string, decision *engine.AppContextDecision, handled bool, start int, limit int) keyEventResult {
	return buildKeyEventResultFromState(action, key, character, committed, passThrough, reason, decision, handled, start, limit, session.State())
}

func buildKeyEventResultFromState(action string, key string, character string, committed string, passThrough string, reason string, decision *engine.AppContextDecision, handled bool, start int, limit int, state engine.State) keyEventResult {
	if committed == "" {
		committed = state.Committed
	}
	return keyEventResult{
		OK:          true,
		Handled:     handled,
		Action:      action,
		Key:         key,
		Character:   character,
		Committed:   committed,
		PassThrough: passThrough,
		Reason:      reason,
		Decision:    decision,
		State:       state,
		Candidates:  buildCandidatePayloadV2FromState(state, start, limit),
		UpdatedAt:   time.Now().UTC(),
	}
}

func normalizeKeyEventKey(req extensionCommandPayload) string {
	key := strings.ToLower(strings.TrimSpace(firstNonEmpty(req.Key, req.Input, req.Text)))
	switch key {
	case " ", "spacebar":
		return "space"
	case "\r", "\n", "ret":
		return "enter"
	case "esc":
		return "escape"
	case "del":
		return "delete"
	default:
		return key
	}
}

func keyEventCharacter(req extensionCommandPayload) string {
	character := firstNonEmpty(req.Character, req.Text)
	if character != "" {
		runes := []rune(character)
		if len(runes) == 1 {
			return character
		}
	}
	input := strings.TrimSpace(req.Input)
	if input != "" {
		runes := []rune(input)
		if len(runes) == 1 {
			return input
		}
	}
	key := strings.TrimSpace(req.Key)
	runes := []rune(key)
	if len(runes) == 1 {
		return key
	}
	if req.Code > 0 && req.Code < 128 {
		r := rune(req.Code)
		if isABIKeyEventInputRune(r) {
			return string(r)
		}
	}
	return ""
}

func hasSystemModifier(req extensionCommandPayload) bool {
	if req.Ctrl || req.Alt || req.Meta {
		return true
	}
	for _, modifier := range req.Modifiers {
		switch strings.ToLower(strings.TrimSpace(modifier)) {
		case "ctrl", "control", "alt", "menu", "meta", "win", "windows", "cmd", "command", "super":
			return true
		}
	}
	return false
}

func keyEventCandidateIndex(key string, start int) (int, bool) {
	if len(key) != 1 || key[0] < '1' || key[0] > '9' {
		return 0, false
	}
	if start < 0 {
		start = 0
	}
	return start + int(key[0]-'1'), true
}

func isABIKeyEventInputRune(r rune) bool {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	switch r {
	case ';', '\'', '/', '`', '@', '.', '-', '_', ':', '?', '&', '=', '%', '+':
		return true
	default:
		return false
	}
}

func buildCandidateActionResult(session *engine.Engine, action string, start int, limit int, committed string) candidateActionResult {
	return buildCandidateActionResultWithState(action, start, limit, committed, session.State())
}

func buildCandidateActionResultWithState(action string, start int, limit int, committed string, state engine.State) candidateActionResult {
	return buildCandidateActionResultFull(action, start, limit, committed, state, nil, nil)
}

func buildCandidateActionResultWithRejected(action string, start int, limit int, state engine.State, rejected engine.Entry) candidateActionResult {
	return buildCandidateActionResultFull(action, start, limit, "", state, &rejected, nil)
}

func buildCandidateActionResultWithPinned(action string, start int, limit int, state engine.State, pinned engine.Entry) candidateActionResult {
	return buildCandidateActionResultFull(action, start, limit, "", state, nil, &pinned)
}

func buildCandidateActionResultFull(action string, start int, limit int, committed string, state engine.State, rejected *engine.Entry, pinned *engine.Entry) candidateActionResult {
	candidates := buildCandidatePayloadV2FromState(state, start, limit)
	return candidateActionResult{
		OK:         true,
		Action:     action,
		Start:      candidates.Start,
		Limit:      candidates.Limit,
		Total:      candidates.Total,
		Committed:  committed,
		Rejected:   rejected,
		Pinned:     pinned,
		State:      state,
		Candidates: candidates,
		UpdatedAt:  time.Now().UTC(),
	}
}

func maxCandidatePageStart(total int, limit int) int {
	if total <= 0 || limit <= 0 {
		return 0
	}
	return ((total - 1) / limit) * limit
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func decodeExtensionCommandPayload(payload string) (extensionCommandPayload, error) {
	var out extensionCommandPayload
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		return out, err
	}
	return out, nil
}

func executeSessionExtensionCommand(session *engine.Engine, command string, payload string) (any, bool) {
	command = strings.ToLower(strings.TrimSpace(command))
	req, err := decodeExtensionCommandPayload(payload)
	if err != nil {
		return errorEnvelope(err.Error()), true
	}
	switch command {
	case "state", "state-json":
		return session.State(), true
	case "preview":
		return session.Preview(req.Input), true
	case "input-key":
		runes := []rune(req.Input)
		if len(runes) == 0 {
			return errorEnvelope("input is required"), true
		}
		return session.InputKey(runes[0]), true
	case "backspace":
		return session.Backspace(), true
	case "clear":
		return session.Clear(), true
	case "mode":
		state := session.State()
		return map[string]any{
			"ok":        true,
			"mode":      state.Mode,
			"state":     state,
			"updatedAt": state.UpdatedAt,
		}, true
	case "set-mode":
		if req.Toggle {
			return session.ToggleMode(), true
		}
		return session.SetMode(req.Mode), true
	case "toggle-mode":
		return session.ToggleMode(), true
	case "candidate-payload-v2":
		return buildCandidatePayloadV2(session, req.Start, req.Limit), true
	case "associate", "associate-json", "association", "predict", "suggest":
		return session.Associate(firstNonEmpty(req.Context, req.Input, req.Text), req.Limit), true
	case "dictionary-sources-json", "dictionary-source-presets", "update-sources":
		return map[string]any{
			"ok":        true,
			"sources":   engine.BuiltinDictionarySources(),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "schema-presets-json", "schemas", "schemas-json":
		return map[string]any{
			"ok":        true,
			"schemas":   engine.BuiltinSchemaPresets(),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "rime-custom-json", "rime-custom-yaml", "apply-rime-custom-json", "apply-rime-custom-yaml":
		return applyRimeCustomPayload(session, payload), true
	case "switches-json", "rime-switches-json", "switches":
		return map[string]any{
			"ok":        true,
			"switches":  engine.SwitchOptions(session.Config()),
			"config":    session.Config(),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "recognizer-json", "recognizer-patterns-json", "rime-recognizer-patterns-json":
		return map[string]any{
			"ok":        true,
			"patterns":  session.Config().RecognizerPatterns,
			"config":    session.Config(),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "apply-switch-json", "apply-switch", "toggle-switch", "switch":
		value := false
		if req.Value != nil {
			value = *req.Value
		}
		toggle := req.Value == nil || strings.EqualFold(req.Action, "toggle") || strings.HasPrefix(command, "toggle")
		next, option, ok := engine.ApplySwitch(session.Config(), firstNonEmpty(req.ID, req.Switch, req.Schema, req.Input, req.Text), value, toggle)
		if !ok {
			return errorEnvelope("unknown switch id"), true
		}
		session.Configure(next)
		return map[string]any{
			"ok":        true,
			"switch":    option,
			"switches":  engine.SwitchOptions(next),
			"config":    next,
			"state":     session.State(),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "app-rules-json", "app-context-rules-json", "app-rules":
		return map[string]any{
			"ok":        true,
			"rules":     engine.NormalizeAppRules(session.Config().AppRules),
			"config":    session.Config(),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "resolve-app-context-json", "app-context-json", "app-context":
		context := engine.AppContext{}
		if req.AppContext != nil {
			context = *req.AppContext
		} else {
			context = engine.AppContext{
				ProcessName: strings.TrimSpace(req.ID),
				ExePath:     strings.TrimSpace(req.Input),
				WindowTitle: strings.TrimSpace(req.Text),
				WindowClass: strings.TrimSpace(req.Kind),
			}
			switch strings.ToLower(strings.TrimSpace(req.Action)) {
			case "password":
				context.PasswordField = true
			case "terminal":
				context.Terminal = true
			case "game":
				context.GameMode = true
			}
		}
		return engine.ResolveAppContext(session.Config(), context), true
	case "profile-json", "profile-bundle-json", "export-profile-json", "user-profile-json":
		return buildProfileBundle(session), true
	case "import-profile-json", "restore-profile-json":
		bundle, err := decodeProfileBundle(payload)
		if err != nil {
			return errorEnvelope(err.Error()), true
		}
		return importProfileBundle(session, bundle), true
	case "candidate-action", "candidate-action-json":
		return executeCandidateAction(session, req), true
	case "key-event", "key-event-json", "process-key-event-json":
		return executeKeyEvent(session, req), true
	case "catalog", "catalog-json", "symbols", "symbols-json":
		return session.CatalogEntries(engine.CatalogRequest{
			Kind:  req.Kind,
			Query: firstNonEmpty(req.Query, req.Input, req.Reading),
			Limit: req.Limit,
		}), true
	case "reverse", "reverse-lookup", "reverse-lookup-json", "lookup-reading":
		return session.ReverseLookup(engine.ReverseLookupRequest{
			Query: firstNonEmpty(req.Query, req.Text, req.Input, req.Context),
			Limit: req.Limit,
		}), true
	case "select", "commit-candidate":
		state, err := session.Select(req.Index)
		if err != nil {
			return errorEnvelope(err.Error()), true
		}
		persistUserScores(session.UserScores())
		return map[string]any{
			"ok":        true,
			"committed": state.Committed,
			"state":     state,
			"updatedAt": state.UpdatedAt,
		}, true
	case "select-candidate-char", "commit-candidate-char":
		state, err := session.SelectChar(req.Index, req.Side)
		if err != nil {
			return errorEnvelope(err.Error()), true
		}
		return map[string]any{
			"ok":        true,
			"committed": state.Committed,
			"state":     state,
			"updatedAt": state.UpdatedAt,
		}, true
	case "user-scores-json", "user-scores":
		scores := session.UserScores()
		return map[string]any{
			"ok":         true,
			"userScores": scores,
			"count":      len(scores),
			"updatedAt":  session.State().UpdatedAt,
		}, true
	case "user-phrases-json", "user-phrases":
		phrases := session.UserPhrases()
		return map[string]any{
			"ok":        true,
			"phrases":   phrases,
			"entries":   phrases,
			"count":     len(phrases),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "user-rejects-json", "user-rejects":
		rejects := session.UserRejects()
		return map[string]any{
			"ok":        true,
			"rejects":   rejects,
			"entries":   rejects,
			"count":     len(rejects),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "user-pins-json", "user-pins", "pins":
		pins := session.UserPins()
		return map[string]any{
			"ok":        true,
			"pins":      pins,
			"entries":   pins,
			"count":     len(pins),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "import-user-scores-json", "import-user-scores":
		scores := req.UserScores
		if scores == nil {
			scores = req.Scores
		}
		if scores == nil {
			var err error
			scores, err = decodeUserScoresPayload(payload)
			if err != nil {
				return errorEnvelope(err.Error()), true
			}
		}
		session.ImportUserScores(scores)
		persistUserScores(session.UserScores())
		return map[string]any{
			"ok":        true,
			"imported":  len(scores),
			"total":     len(session.UserScores()),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "import-user-phrases-json", "import-user-phrases":
		entries := req.Entries
		if len(entries) == 0 {
			entries = req.Phrases
		}
		if req.Merge {
			merged := session.UserPhrases()
			merged = append(merged, entries...)
			entries = merged
		}
		session.ReplaceUserPhrases(entries)
		phrases := session.UserPhrases()
		persistUserPhrases(phrases)
		return map[string]any{
			"ok":        true,
			"imported":  len(entries),
			"total":     len(phrases),
			"phrases":   phrases,
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "delete-user-phrase":
		reading := normalizeABIReading(req.Reading)
		text := strings.TrimSpace(req.Text)
		session.DeleteUserPhrase(reading, text)
		phrases := session.UserPhrases()
		persistUserPhrases(phrases)
		return map[string]any{
			"ok":        true,
			"total":     len(phrases),
			"phrases":   phrases,
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "import-user-rejects-json", "import-user-rejects":
		entries := req.Entries
		if len(entries) == 0 {
			entries = req.Rejects
		}
		if req.Merge {
			merged := session.UserRejects()
			merged = append(merged, entries...)
			entries = merged
		}
		session.ReplaceUserRejects(entries)
		rejects := session.UserRejects()
		persistUserRejects(rejects)
		return map[string]any{
			"ok":        true,
			"imported":  len(entries),
			"total":     len(rejects),
			"rejects":   rejects,
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "delete-user-reject":
		reading := normalizeABIReading(req.Reading)
		text := strings.TrimSpace(req.Text)
		session.DeleteUserReject(reading, text)
		rejects := session.UserRejects()
		persistUserRejects(rejects)
		return map[string]any{
			"ok":        true,
			"total":     len(rejects),
			"rejects":   rejects,
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "import-user-pins-json", "import-user-pins":
		entries := req.Entries
		if len(entries) == 0 {
			entries = req.Pins
		}
		if req.Merge {
			merged := session.UserPins()
			merged = append(merged, entries...)
			entries = merged
		}
		session.ReplaceUserPins(entries)
		pins := session.UserPins()
		persistUserPins(pins)
		return map[string]any{
			"ok":        true,
			"imported":  len(entries),
			"total":     len(pins),
			"pins":      pins,
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "delete-user-pin":
		reading := normalizeABIReading(firstNonEmpty(req.Reading, req.Input))
		text := strings.TrimSpace(req.Text)
		session.DeleteUserPin(reading, text)
		pins := session.UserPins()
		persistUserPins(pins)
		return map[string]any{
			"ok":        true,
			"total":     len(pins),
			"pins":      pins,
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "commit-text":
		reading := normalizeABIReading(req.Reading)
		text := strings.TrimSpace(req.Text)
		if reading == "" || text == "" {
			return errorEnvelope("reading and text are required"), true
		}
		key := reading + "|" + text
		session.ImportUserScores(map[string]int{key: 25})
		persistUserScores(session.UserScores())
		return map[string]any{
			"ok":        true,
			"learned":   key,
			"state":     session.State(),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "agent-compose":
		return composeAgentABI(req.Input, req.Context), true
	default:
		return nil, false
	}
}

func errorEnvelope(message string) abiEnvelope {
	return abiEnvelope{
		OK:        false,
		Error:     message,
		UpdatedAt: time.Now().UTC(),
	}
}
