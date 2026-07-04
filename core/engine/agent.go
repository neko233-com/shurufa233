package engine

import (
	"strings"
	"time"
)

func NormalizeAgent(agent Agent) Agent {
	defaults := defaultAgentConfig()
	if strings.TrimSpace(agent.Provider) == "" {
		agent.Provider = defaults.Provider
	}
	agent.Provider = strings.ToLower(strings.TrimSpace(agent.Provider))
	if agent.Provider == "" {
		agent.Provider = "builtin"
	}
	agent.Endpoint = strings.TrimSpace(agent.Endpoint)
	agent.Model = strings.TrimSpace(agent.Model)
	if agent.Model == "" {
		agent.Model = defaults.Model
	}
	agent.SystemPrompt = strings.TrimSpace(agent.SystemPrompt)
	if agent.SystemPrompt == "" {
		agent.SystemPrompt = defaults.SystemPrompt
	}
	agent.Triggers = normalizeAgentStringList(agent.Triggers, defaults.Triggers)
	agent.Actions = normalizeAgentStringList(agent.Actions, defaults.Actions)
	if agent.TimeoutMs <= 0 {
		agent.TimeoutMs = defaults.TimeoutMs
	}
	if agent.TimeoutMs < 1000 {
		agent.TimeoutMs = 1000
	}
	if agent.TimeoutMs > 120000 {
		agent.TimeoutMs = 120000
	}
	return agent
}

func defaultAgentConfig() Agent {
	return Agent{
		Enabled:      true,
		Provider:     "builtin",
		Model:        "prompt-router",
		SystemPrompt: "作为输入法 agent，优先生成可直接上屏、复制或交给外部 agent 的短指令。",
		Triggers:     []string{"/ask", "/rewrite", "/translate"},
		Actions:      []string{"commit", "copy", "open-settings", "handoff"},
		TimeoutMs:    12000,
	}
}

func ComposeAgent(config Config, req AgentComposeRequest) AgentComposeResponse {
	agent := NormalizeAgent(config.Agent)
	input := strings.TrimSpace(req.Input)
	context := strings.TrimSpace(req.Context)
	response := AgentComposeResponse{
		OK:        true,
		Input:     input,
		Context:   context,
		Provider:  agent.Provider,
		Model:     agent.Model,
		Actions:   append([]string{}, agent.Actions...),
		UpdatedAt: time.Now().UTC(),
	}
	add := func(intent string, action string, text string) {
		item := AgentCandidate{
			Text:     text,
			Intent:   intent,
			Action:   action,
			Source:   "builtin-agent",
			Provider: agent.Provider,
			Model:    agent.Model,
		}
		if context != "" {
			item.Context = context
		}
		response.Items = append(response.Items, item)
		response.Candidates = append(response.Candidates, text)
	}
	lower := strings.ToLower(input)
	switch {
	case lower == "/rewrite" || strings.HasPrefix(lower, "/rewrite ") || strings.Contains(input, "润色"):
		text := agentCommandText(input, "/rewrite")
		if text == "" {
			text = firstNonEmpty(context, input)
		}
		add("rewrite", "agent.rewrite.polish", "请润色这段文字："+text)
		add("rewrite", "agent.rewrite.concise", "把下面内容改得更自然、更简洁："+text)
	case lower == "/translate" || strings.HasPrefix(lower, "/translate ") || strings.Contains(input, "翻译"):
		text := agentCommandText(input, "/translate")
		if text == "" {
			text = firstNonEmpty(context, input)
		}
		add("translate", "agent.translate.zh", "请把这段内容翻译成中文："+text)
		add("translate", "agent.translate.en", "请把这段内容翻译成英文："+text)
	case lower == "/ask" || strings.HasPrefix(lower, "/ask ") || strings.Contains(input, "提问"):
		text := agentCommandText(input, "/ask")
		if text == "" {
			text = firstNonEmpty(context, input)
		}
		add("ask", "agent.ask.answer", "请回答："+text)
		add("ask", "agent.ask.steps", "请分步骤分析："+text)
	default:
		prefix := "作为输入法 agent，请处理："
		if context != "" {
			prefix = "结合当前上下文，作为输入法 agent，请处理："
		}
		add("compose", "agent.compose", prefix+input)
	}
	return response
}

func normalizeAgentStringList(values []string, fallback []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if len(out) == 0 {
		return append([]string{}, fallback...)
	}
	return out
}

func agentCommandText(input string, command string) string {
	if len(input) <= len(command) {
		return ""
	}
	return strings.TrimSpace(input[len(command):])
}
