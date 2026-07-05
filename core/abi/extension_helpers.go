package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	"skin-presets-json",
	"apply-skin-preset-json",
	"rime-custom-yaml",
	"reverse-lookup-json",
	"user-scores-json",
	"user-phrases-json",
	"rime-custom-phrase-text",
	"user-rejects-json",
	"user-pins-json",
	"commit-text",
	"agent-compose",
	"agent-config-json",
	"apply-agent-config-json",
	"profile-sync-json",
	"apply-sync-config-json",
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
	"apply-app-rules-json",
	"profile-bundle-json",
	"user-data-delete-json",
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

type agentComposeResult = engine.AgentComposeResponse

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
	Index       int                        `json:"index,omitempty"`
	PageDelta   int                        `json:"pageDelta,omitempty"`
	MoveDelta   int                        `json:"moveDelta,omitempty"`
	Committed   string                     `json:"committed,omitempty"`
	PassThrough string                     `json:"passThrough,omitempty"`
	Reason      string                     `json:"reason,omitempty"`
	Decision    *engine.AppContextDecision `json:"decision,omitempty"`
	State       engine.State               `json:"state"`
	Candidates  candidatePayloadV2         `json:"candidates"`
	UpdatedAt   time.Time                  `json:"updatedAt"`
}

type keyEventPunctuationState struct {
	mu              sync.Mutex
	doubleQuoteOpen bool
	singleQuoteOpen bool
}

var keyEventPunctuationStates sync.Map

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
	Preset       string             `json:"preset,omitempty"`
	YAML         string             `json:"yaml,omitempty"`
	Format       string             `json:"format,omitempty"`
	Data         string             `json:"data,omitempty"`
	Reading      string             `json:"reading,omitempty"`
	Text         string             `json:"text,omitempty"`
	Kind         string             `json:"kind,omitempty"`
	Query        string             `json:"query,omitempty"`
	Config       *engine.Config     `json:"config,omitempty"`
	Agent        *engine.Agent      `json:"agent,omitempty"`
	Sync         *engine.Sync       `json:"sync,omitempty"`
	AppContext   *engine.AppContext `json:"appContext,omitempty"`
	Rules        []engine.AppRule   `json:"rules,omitempty"`
	UserScores   map[string]int     `json:"userScores,omitempty"`
	Scores       map[string]int     `json:"scores,omitempty"`
	Entries      []engine.Entry     `json:"entries,omitempty"`
	Phrases      []engine.Entry     `json:"phrases,omitempty"`
	Rejects      []engine.Entry     `json:"rejects,omitempty"`
	Pins         []engine.Entry     `json:"pins,omitempty"`
	Merge        bool               `json:"merge,omitempty"`
	Directory    string             `json:"directory,omitempty"`
	Path         string             `json:"path,omitempty"`
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

func applyAppRulesPayload(session *engine.Engine, req extensionCommandPayload) map[string]any {
	config := session.Config()
	config.AppRules = req.Rules
	config = normalizeConfig(config)
	session.Configure(config)
	persisted := true
	var persistError string
	if err := persistConfig(config); err != nil {
		persisted = false
		persistError = err.Error()
	}
	return map[string]any{
		"ok":              true,
		"rules":           engine.NormalizeAppRules(config.AppRules),
		"config":          config,
		"persisted":       persisted,
		"persistError":    persistError,
		"sessionsUpdated": 1,
		"updatedAt":       time.Now().UTC(),
	}
}

func configEnvelope() map[string]any {
	return map[string]any{
		"ok":        true,
		"config":    loadConfig(),
		"updatedAt": time.Now().UTC(),
	}
}

func applyConfigEnvelope(config engine.Config) map[string]any {
	config = normalizeConfig(config)
	updated := configureActiveSessions(config)
	persisted := true
	var persistError string
	if err := persistConfig(config); err != nil {
		persisted = false
		persistError = err.Error()
	}
	return map[string]any{
		"ok":              true,
		"config":          config,
		"persisted":       persisted,
		"persistError":    persistError,
		"sessionsUpdated": updated,
		"updatedAt":       time.Now().UTC(),
	}
}

func reloadDictionariesEnvelope() map[string]any {
	groups := loadLocalDictionaryEntries()
	sessions := activeSessions()
	entryCount := 0
	for _, group := range groups {
		entryCount += len(group)
	}
	for _, session := range sessions {
		for _, group := range groups {
			session.AddEntries(group)
		}
	}
	return map[string]any{
		"ok":               true,
		"dictionaryGroups": len(groups),
		"entries":          entryCount,
		"sessionsUpdated":  len(sessions),
		"updatedAt":        time.Now().UTC(),
	}
}

func agentConfigEnvelope() map[string]any {
	config := loadConfig()
	return map[string]any{
		"ok":        true,
		"agent":     engine.NormalizeAgent(config.Agent),
		"config":    config,
		"updatedAt": time.Now().UTC(),
	}
}

func applyAgentConfigPayload(req extensionCommandPayload) map[string]any {
	config := loadConfig()
	if req.Agent != nil {
		config.Agent = *req.Agent
	}
	config = normalizeConfig(config)
	return applyConfigEnvelope(config)
}

func syncConfigEnvelope() map[string]any {
	config := loadConfig()
	directory := resolveSyncDirectory(config, "")
	path := filepath.Join(directory, "shurufa233-profile.json")
	_, statErr := os.Stat(path)
	return map[string]any{
		"ok":         true,
		"sync":       engine.NormalizeSync(config.Sync),
		"directory":  directory,
		"bundlePath": path,
		"exists":     statErr == nil,
		"updatedAt":  time.Now().UTC(),
	}
}

func applySyncConfigPayload(req extensionCommandPayload) map[string]any {
	config := loadConfig()
	if req.Sync != nil {
		config.Sync = *req.Sync
	}
	if strings.TrimSpace(req.Directory) != "" {
		config.Sync.Directory = strings.TrimSpace(req.Directory)
	}
	config = normalizeConfig(config)
	result := applyConfigEnvelope(config)
	result["sync"] = config.Sync
	result["directory"] = resolveSyncDirectory(config, "")
	return result
}

func exportProfileSyncPayload(session *engine.Engine, req extensionCommandPayload) any {
	config := loadConfig()
	path := resolveSyncProfilePath(config, req)
	bundle := buildProfileBundle(session)
	if err := writeJSONAtomic(path, bundle); err != nil {
		return errorEnvelope(err.Error())
	}
	return map[string]any{
		"ok":        true,
		"exported":  true,
		"path":      path,
		"profile":   bundle,
		"updatedAt": time.Now().UTC(),
	}
}

func importProfileSyncPayload(session *engine.Engine, req extensionCommandPayload) any {
	config := loadConfig()
	path := resolveSyncProfilePath(config, req)
	data, err := os.ReadFile(path)
	if err != nil {
		return errorEnvelope(err.Error())
	}
	bundle, err := decodeProfileBundle(string(data))
	if err != nil {
		return errorEnvelope(err.Error())
	}
	if config.Sync.ConflictPolicy != "replace-local" {
		bundle.Merge = true
	}
	if req.Merge {
		bundle.Merge = true
	}
	result := importProfileBundle(session, bundle)
	result["path"] = path
	return result
}

func resolveSyncProfilePath(config engine.Config, req extensionCommandPayload) string {
	if strings.TrimSpace(req.Path) != "" {
		return filepath.Clean(strings.TrimSpace(req.Path))
	}
	return filepath.Join(resolveSyncDirectory(config, req.Directory), "shurufa233-profile.json")
}

func resolveSyncDirectory(config engine.Config, override string) string {
	if strings.TrimSpace(override) != "" {
		return filepath.Clean(strings.TrimSpace(override))
	}
	if strings.TrimSpace(config.Sync.Directory) != "" {
		return filepath.Clean(strings.TrimSpace(config.Sync.Directory))
	}
	configPath, err := configFile()
	if err != nil {
		return filepath.Join(os.TempDir(), "shurufa233", "sync")
	}
	return filepath.Join(filepath.Dir(configPath), "sync")
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		if retryErr := os.Rename(tmp, path); retryErr != nil {
			_ = os.Remove(tmp)
			return retryErr
		}
	}
	return nil
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
	return engine.ComposeAgent(loadConfig(), engine.AgentComposeRequest{Input: input, Context: context})
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

	if quickIndex, ok := keyEventQuickSelectIndex(session, key, character, before); ok {
		state, err := session.Select(quickIndex)
		if err != nil {
			return buildKeyEventResult(session, "error", key, character, "", "", err.Error(), decision, true, req.Start, limit)
		}
		persistUserScores(session.UserScores())
		return buildKeyEventResultFromStateWithNavigation("commit-candidate", key, character, quickIndex, 0, 0, state.Committed, "", "quick-select", decision, true, 0, limit, state)
	}
	if moveDelta, ok := keyEventMoveDelta(key, req); ok && len(before.Candidates) > 0 {
		return buildKeyEventResultFromStateWithNavigation("move-selection", key, character, req.Index, 0, moveDelta, "", "", "", decision, true, req.Start, limit, before)
	}
	if pageDelta, ok := keyEventPageDelta(session, key, character, req, before); ok {
		return buildKeyEventResultFromStateWithNavigation("page-candidates", key, character, req.Index, pageDelta, 0, "", "", "", decision, true, req.Start, limit, before)
	}
	if index, ok := keyEventCandidateIndex(key, req.Start); ok && len(before.Candidates) > index {
		state, err := session.Select(index)
		if err != nil {
			return buildKeyEventResult(session, "error", key, character, "", "", err.Error(), decision, true, req.Start, limit)
		}
		persistUserScores(session.UserScores())
		return buildKeyEventResultFromState("commit-candidate", key, character, state.Committed, "", "", decision, true, 0, limit, state)
	}
	if punctuation, label, ok := keyEventPunctuation(session, req, key, character); ok {
		if len(before.Candidates) > 0 {
			index := req.Index
			state, err := session.Select(index)
			if err != nil {
				return buildKeyEventResult(session, "error", key, character, "", "", err.Error(), decision, true, req.Start, limit)
			}
			persistUserScores(session.UserScores())
			return buildKeyEventResultFromState("commit-candidate-punctuation", key, character, state.Committed+punctuation, "", label, decision, true, 0, limit, state)
		}
		if before.Buffer != "" {
			committed := before.Buffer + punctuation
			state := session.Clear()
			return buildKeyEventResultFromState("commit-raw-punctuation", key, character, committed, "", label, decision, true, 0, limit, state)
		}
		if strings.EqualFold(session.Config().Punctuation, "half") {
			return buildKeyEventResult(session, "pass-through", key, character, "", punctuation, "half-punctuation", decision, false, req.Start, limit)
		}
		return buildKeyEventResult(session, "commit-punctuation", key, character, punctuation, "", label, decision, true, req.Start, limit)
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
	return buildKeyEventResultFromStateWithNavigation(action, key, character, 0, 0, 0, committed, passThrough, reason, decision, handled, start, limit, state)
}

func buildKeyEventResultFromStateWithNavigation(action string, key string, character string, index int, pageDelta int, moveDelta int, committed string, passThrough string, reason string, decision *engine.AppContextDecision, handled bool, start int, limit int, state engine.State) keyEventResult {
	if committed == "" {
		committed = state.Committed
	}
	return keyEventResult{
		OK:          true,
		Handled:     handled,
		Action:      action,
		Key:         key,
		Character:   character,
		Index:       index,
		PageDelta:   pageDelta,
		MoveDelta:   moveDelta,
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

func keyEventQuickSelectIndex(session *engine.Engine, key string, character string, state engine.State) (int, bool) {
	if len(state.Candidates) == 0 {
		return 0, false
	}
	config := session.Config()
	label := keyEventPunctuationLabel(extensionCommandPayload{}, key, character)
	switch label {
	case ";":
		if config.DoublePinyin && strings.EqualFold(config.DoublePinyinScheme, "microsoft") {
			return 0, false
		}
		if !config.SemicolonQuickSelect || len(state.Candidates) < 2 {
			return 0, false
		}
		return 1, true
	case "'":
		if !config.QuoteQuickSelect || len(state.Candidates) < 3 {
			return 0, false
		}
		return 2, true
	default:
		return 0, false
	}
}

func keyEventMoveDelta(key string, req extensionCommandPayload) (int, bool) {
	switch key {
	case "right", "down", "arrowright", "arrowdown":
		return 1, true
	case "left", "up", "arrowleft", "arrowup":
		return -1, true
	case "tab":
		if req.Shift {
			return -1, true
		}
		return 1, true
	default:
		return 0, false
	}
}

func keyEventPageDelta(session *engine.Engine, key string, character string, req extensionCommandPayload, state engine.State) (int, bool) {
	if len(state.Candidates) <= keyEventLimit(session, req) {
		return 0, false
	}
	config := session.Config()
	label := keyEventPunctuationLabel(req, key, character)
	if config.BracketPageKeys {
		switch label {
		case "[":
			return -1, true
		case "]":
			return 1, true
		}
	}
	if config.MinusEqualPageKeys {
		switch label {
		case "-":
			return -1, true
		case "=":
			return 1, true
		}
		switch key {
		case "pageup", "prior":
			return -1, true
		case "pagedown", "next":
			return 1, true
		}
	}
	if config.CommaPeriodPageKeys {
		switch label {
		case ",":
			return -1, true
		case ".":
			return 1, true
		}
	}
	return 0, false
}

func keyEventLimit(session *engine.Engine, req extensionCommandPayload) int {
	limit := req.Limit
	if limit <= 0 {
		limit = req.PageSize
	}
	if limit <= 0 {
		limit = session.Config().CandidatePageSize
	}
	if limit <= 0 {
		return 7
	}
	return limit
}

func keyEventPunctuation(session *engine.Engine, req extensionCommandPayload, key string, character string) (string, string, bool) {
	label := keyEventPunctuationLabel(req, key, character)
	if label == "" {
		return "", "", false
	}
	config := session.Config()
	if label == ";" && config.DoublePinyin && strings.EqualFold(config.DoublePinyinScheme, "microsoft") {
		return "", "", false
	}
	if strings.EqualFold(config.Punctuation, "half") {
		if value := punctuationShapeValue(config.PunctuationHalfShape, label); value != "" {
			return value, label, true
		}
		if value := defaultHalfPunctuation(label); value != "" {
			return value, label, true
		}
		return "", "", false
	}
	if value := punctuationShapeValue(config.PunctuationFullShape, label); value != "" {
		return value, label, true
	}
	if value := defaultFullPunctuation(session, label); value != "" {
		return value, label, true
	}
	return "", "", false
}

func keyEventPunctuationLabel(req extensionCommandPayload, key string, character string) string {
	source := character
	if source == "" {
		source = key
	}
	if source == "" {
		return ""
	}
	if req.Shift {
		switch source {
		case ",":
			return "<"
		case ".":
			return ">"
		case ";":
			return ":"
		case "/":
			return "?"
		case "[":
			return "{"
		case "]":
			return "}"
		case "'":
			return "\""
		case "-":
			return "_"
		case "1":
			return "!"
		case "6":
			return "^"
		case "9":
			return "("
		case "0":
			return ")"
		}
	}
	switch source {
	case ",", "<", ".", ">", ";", ":", "/", "?", "[", "{", "]", "}", "'", "\"", "-", "_", "=", "+", "!", "^", "(", ")":
		return source
	default:
		return ""
	}
}

func punctuationShapeValue(shape map[string][]string, label string) string {
	if len(shape) == 0 {
		return ""
	}
	values := shape[label]
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func defaultFullPunctuation(session *engine.Engine, label string) string {
	switch label {
	case ",":
		return "，"
	case "<":
		return "《"
	case ".":
		return "。"
	case ">":
		return "》"
	case ";":
		return "；"
	case ":":
		return "："
	case "/":
		return "、"
	case "?":
		return "？"
	case "[":
		return "「"
	case "{":
		return "【"
	case "]":
		return "」"
	case "}":
		return "】"
	case "'":
		state := punctuationStateForSession(session)
		state.mu.Lock()
		defer state.mu.Unlock()
		state.singleQuoteOpen = !state.singleQuoteOpen
		if state.singleQuoteOpen {
			return "‘"
		}
		return "’"
	case "\"":
		state := punctuationStateForSession(session)
		state.mu.Lock()
		defer state.mu.Unlock()
		state.doubleQuoteOpen = !state.doubleQuoteOpen
		if state.doubleQuoteOpen {
			return "“"
		}
		return "”"
	case "-":
		return "-"
	case "_":
		return "——"
	case "!":
		return "！"
	case "^":
		return "……"
	case "(":
		return "（"
	case ")":
		return "）"
	default:
		return ""
	}
}

func defaultHalfPunctuation(label string) string {
	switch label {
	case ",", "<", ".", ">", ";", ":", "/", "?", "[", "{", "]", "}", "'", "\"", "-", "_", "!", "^", "(", ")":
		return label
	default:
		return ""
	}
}

func punctuationStateForSession(session *engine.Engine) *keyEventPunctuationState {
	if existing, ok := keyEventPunctuationStates.Load(session); ok {
		return existing.(*keyEventPunctuationState)
	}
	state := &keyEventPunctuationState{}
	actual, _ := keyEventPunctuationStates.LoadOrStore(session, state)
	return actual.(*keyEventPunctuationState)
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

func isRimeCustomPhraseFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "rime-custom-phrase", "custom-phrase", "custom_phrase", "custom_phrase.txt", "rime":
		return true
	default:
		return false
	}
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
	case "agent-config-json", "agent-config":
		return agentConfigEnvelope(), true
	case "apply-agent-config-json", "apply-agent-config":
		return applyAgentConfigPayload(req), true
	case "sync-config-json", "profile-sync-json", "sync-json", "sync":
		return syncConfigEnvelope(), true
	case "apply-sync-config-json", "apply-sync-config":
		return applySyncConfigPayload(req), true
	case "export-profile-sync-json", "profile-sync-export", "sync-export":
		return exportProfileSyncPayload(session, req), true
	case "import-profile-sync-json", "profile-sync-import", "sync-import":
		return importProfileSyncPayload(session, req), true
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
	case "skin-presets-json", "skins", "skins-json":
		return map[string]any{
			"ok":        true,
			"selected":  session.Config().Skin.Theme,
			"presets":   engine.BuiltinSkinPresets(),
			"config":    session.Config(),
			"updatedAt": session.State().UpdatedAt,
		}, true
	case "apply-skin-preset-json", "apply-skin-preset", "skin-preset":
		next, ok := engine.ApplySkinPresetConfig(session.Config(), firstNonEmpty(req.ID, req.Preset, req.Input, req.Text))
		if !ok {
			return errorEnvelope("unknown skin preset id"), true
		}
		session.Configure(next)
		result := applyConfigEnvelope(next)
		result["selected"] = next.Skin.Theme
		result["presets"] = engine.BuiltinSkinPresets()
		result["state"] = session.State()
		return result, true
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
	case "apply-app-rules-json", "put-app-rules-json", "set-app-rules-json":
		return applyAppRulesPayload(session, req), true
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
	case "rime-custom-phrase-text", "user-phrases-rime-text", "export-rime-custom-phrases":
		phrases := session.UserPhrases()
		return map[string]any{
			"ok":        true,
			"format":    "rime-custom-phrase",
			"data":      engine.FormatRimeCustomPhrases(phrases),
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
		if len(entries) == 0 && isRimeCustomPhraseFormat(req.Format) && strings.TrimSpace(req.Data) != "" {
			parsed, err := engine.ParseRimeCustomPhrases([]byte(req.Data))
			if err != nil {
				return errorEnvelope(err.Error()), true
			}
			entries = parsed
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
	case "import-rime-custom-phrases", "import-rime-custom-phrase-text":
		entries, err := engine.ParseRimeCustomPhrases([]byte(req.Data))
		if err != nil {
			return errorEnvelope(err.Error()), true
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
