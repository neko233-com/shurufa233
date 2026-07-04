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
	"commit-text",
	"agent-compose",
	"rime-compatible-dictionaries",
	"gzip-dictionaries",
	"abbreviation-candidates",
	"emoji-kaomoji-symbol-candidates",
	"dynamic-datetime-candidates",
	"candidate-char-commit",
	"candidate-comments",
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

func buildCandidatePayloadV2(session *engine.Engine, start int, limit int) candidatePayloadV2 {
	state := session.State()
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

func errorEnvelope(message string) abiEnvelope {
	return abiEnvelope{
		OK:        false,
		Error:     message,
		UpdatedAt: time.Now().UTC(),
	}
}
