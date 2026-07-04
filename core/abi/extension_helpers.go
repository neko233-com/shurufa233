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
	"user-scores-json",
	"user-phrases-json",
	"user-rejects-json",
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
	State      engine.State       `json:"state"`
	Candidates candidatePayloadV2 `json:"candidates"`
	UpdatedAt  time.Time          `json:"updatedAt"`
}

type extensionCommandPayload struct {
	Input        string           `json:"input,omitempty"`
	Context      string           `json:"context,omitempty"`
	Action       string           `json:"action,omitempty"`
	Mode         string           `json:"mode,omitempty"`
	Toggle       bool             `json:"toggle,omitempty"`
	Index        int              `json:"index,omitempty"`
	DisplayIndex int              `json:"displayIndex,omitempty"`
	Start        int              `json:"start,omitempty"`
	Limit        int              `json:"limit,omitempty"`
	PageSize     int              `json:"pageSize,omitempty"`
	Delta        int              `json:"delta,omitempty"`
	Side         string           `json:"side,omitempty"`
	Reading      string           `json:"reading,omitempty"`
	Text         string           `json:"text,omitempty"`
	Kind         string           `json:"kind,omitempty"`
	Query        string           `json:"query,omitempty"`
	Config       *engine.Config   `json:"config,omitempty"`
	UserScores   map[string]int   `json:"userScores,omitempty"`
	Scores       map[string]int   `json:"scores,omitempty"`
	Entries      []engine.Entry   `json:"entries,omitempty"`
	Phrases      []engine.Entry   `json:"phrases,omitempty"`
	Rejects      []engine.Entry   `json:"rejects,omitempty"`
	Merge        bool             `json:"merge,omitempty"`
	Raw          *json.RawMessage `json:"raw,omitempty"`
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
			Score:        candidate.Weight + candidate.UserScore,
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

func buildCandidateActionResult(session *engine.Engine, action string, start int, limit int, committed string) candidateActionResult {
	return buildCandidateActionResultWithState(action, start, limit, committed, session.State())
}

func buildCandidateActionResultWithState(action string, start int, limit int, committed string, state engine.State) candidateActionResult {
	return buildCandidateActionResultFull(action, start, limit, committed, state, nil)
}

func buildCandidateActionResultWithRejected(action string, start int, limit int, state engine.State, rejected engine.Entry) candidateActionResult {
	return buildCandidateActionResultFull(action, start, limit, "", state, &rejected)
}

func buildCandidateActionResultFull(action string, start int, limit int, committed string, state engine.State, rejected *engine.Entry) candidateActionResult {
	candidates := buildCandidatePayloadV2FromState(state, start, limit)
	return candidateActionResult{
		OK:         true,
		Action:     action,
		Start:      candidates.Start,
		Limit:      candidates.Limit,
		Total:      candidates.Total,
		Committed:  committed,
		Rejected:   rejected,
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
	case "candidate-action", "candidate-action-json":
		return executeCandidateAction(session, req), true
	case "catalog", "catalog-json", "symbols", "symbols-json":
		return session.CatalogEntries(engine.CatalogRequest{
			Kind:  req.Kind,
			Query: firstNonEmpty(req.Query, req.Input, req.Reading),
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
