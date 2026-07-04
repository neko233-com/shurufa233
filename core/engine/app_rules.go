package engine

import (
	"path/filepath"
	"sort"
	"strings"
)

func BuiltinAppRules() []AppRule {
	return []AppRule{
		{
			ID:                "password-field-ascii",
			Name:              "密码框英文直通",
			Description:       "密码框、安全输入框、登录框默认英文直通，不显示中文候选，不记录学习。",
			PasswordField:     true,
			Mode:              "en",
			Punctuation:       "half",
			DisableCandidates: true,
			DisableLearning:   true,
			Priority:          1000,
		},
		{
			ID:          "terminal-ascii",
			Name:        "终端/命令行英文优先",
			Description: "PowerShell、Windows Terminal、cmd、SSH 等命令输入场景默认英文和半角标点。",
			ProcessNames: []string{
				"windowsterminal.exe",
				"powershell.exe",
				"pwsh.exe",
				"cmd.exe",
				"wt.exe",
				"ssh.exe",
			},
			Terminal:    true,
			Mode:        "en",
			Punctuation: "half",
			Priority:    800,
		},
		{
			ID:          "game-performance-ascii",
			Name:        "游戏/电竞性能模式",
			Description: "游戏和全屏低延迟输入场景默认英文直通、半角标点，并关闭候选干扰。",
			ProcessNames: []string{
				"cs2.exe",
				"valorant.exe",
				"leagueclient.exe",
				"league of legends.exe",
				"overwatch.exe",
				"fortniteclient-win64-shipping.exe",
				"steam.exe",
				"wegame.exe",
				"mumunxmain.exe",
			},
			ExeContains: []string{
				"\\steamapps\\common\\",
				"\\wegame\\",
				"\\tencent games\\",
				"\\riot games\\",
				"\\mumu\\",
			},
			GameMode:          true,
			Mode:              "en",
			Punctuation:       "half",
			DisableCandidates: true,
			Priority:          700,
		},
		{
			ID:          "coding-half-punctuation",
			Name:        "IDE/代码编辑半角标点",
			Description: "IDE 和代码编辑器保持当前中英文模式，但默认半角标点，避免中文符号进入代码。",
			ProcessNames: []string{
				"code.exe",
				"cursor.exe",
				"rider64.exe",
				"idea64.exe",
				"goland64.exe",
				"webstorm64.exe",
				"devenv.exe",
			},
			Punctuation: "half",
			Priority:    400,
		},
	}
}

func NormalizeAppRules(rules []AppRule) []AppRule {
	if len(rules) == 0 {
		rules = BuiltinAppRules()
	}
	out := make([]AppRule, 0, len(rules))
	seen := map[string]bool{}
	for _, rule := range rules {
		rule = normalizeAppRule(rule)
		if rule.ID == "" {
			continue
		}
		if seen[rule.ID] {
			continue
		}
		seen[rule.ID] = true
		out = append(out, rule)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].ID < out[j].ID
	})
	if len(out) == 0 {
		return BuiltinAppRules()
	}
	return out
}

func ResolveAppContext(config Config, context AppContext) AppContextDecision {
	config = NormalizeSwitchConfig(config)
	config.AppRules = NormalizeAppRules(config.AppRules)
	context = normalizeAppContext(context)
	decision := AppContextDecision{
		OK:              true,
		Context:         context,
		Config:          config,
		Mode:            config.Mode,
		Punctuation:     config.Punctuation,
		CandidateLayout: config.CandidateLayout,
		Reason:          "default",
	}
	for _, rule := range config.AppRules {
		if !appRuleMatches(rule, context) {
			continue
		}
		next := config
		if rule.Mode != "" {
			next.Mode = normalizeMode(rule.Mode)
		}
		if rule.Punctuation != "" {
			next.Punctuation = normalizePunctuation(rule.Punctuation)
		}
		if rule.CandidateLayout != "" {
			next.CandidateLayout = normalizeCandidateLayout(rule.CandidateLayout)
		}
		decision.Matched = true
		matched := rule
		decision.Rule = &matched
		decision.Config = next
		decision.Mode = next.Mode
		decision.Punctuation = next.Punctuation
		decision.CandidateLayout = next.CandidateLayout
		decision.DisableCandidates = rule.DisableCandidates
		decision.DisableLearning = rule.DisableLearning
		decision.Reason = rule.ID
		return decision
	}
	return decision
}

func normalizeAppRule(rule AppRule) AppRule {
	rule.ID = strings.ToLower(strings.TrimSpace(rule.ID))
	rule.ID = strings.ReplaceAll(rule.ID, " ", "-")
	rule.Name = strings.TrimSpace(rule.Name)
	if rule.Name == "" {
		rule.Name = rule.ID
	}
	rule.Description = strings.TrimSpace(rule.Description)
	rule.ProcessNames = normalizeMatchList(rule.ProcessNames, true)
	rule.ExeContains = normalizeMatchList(rule.ExeContains, false)
	rule.WindowTitle = normalizeMatchList(rule.WindowTitle, false)
	rule.WindowClass = normalizeMatchList(rule.WindowClass, false)
	if rule.Mode != "" {
		rule.Mode = normalizeMode(rule.Mode)
	}
	if rule.Punctuation != "" {
		rule.Punctuation = normalizePunctuation(rule.Punctuation)
	}
	if rule.CandidateLayout != "" {
		rule.CandidateLayout = normalizeCandidateLayout(rule.CandidateLayout)
	}
	return rule
}

func normalizeAppContext(context AppContext) AppContext {
	context.ProcessName = normalizeProcessName(firstNonEmpty(context.ProcessName, context.ExePath))
	context.ExePath = strings.ToLower(strings.TrimSpace(context.ExePath))
	context.WindowTitle = strings.ToLower(strings.TrimSpace(context.WindowTitle))
	context.WindowClass = strings.ToLower(strings.TrimSpace(context.WindowClass))
	context.CompositionMode = strings.ToLower(strings.TrimSpace(context.CompositionMode))
	return context
}

func appRuleMatches(rule AppRule, context AppContext) bool {
	if rule.PasswordField && !context.PasswordField {
		return false
	}
	if rule.Terminal && !context.Terminal && !matchesAny(rule.ProcessNames, context.ProcessName) {
		return false
	}
	if rule.GameMode && !context.GameMode && !matchesAny(rule.ProcessNames, context.ProcessName) && !containsAny(context.ExePath, rule.ExeContains) {
		return false
	}
	hasMatcher := rule.PasswordField || rule.Terminal || rule.GameMode ||
		len(rule.ProcessNames) > 0 || len(rule.ExeContains) > 0 ||
		len(rule.WindowTitle) > 0 || len(rule.WindowClass) > 0
	if !hasMatcher {
		return false
	}
	if len(rule.ProcessNames) > 0 && !matchesAny(rule.ProcessNames, context.ProcessName) && !rule.Terminal && !rule.GameMode {
		return false
	}
	if len(rule.ExeContains) > 0 && !containsAny(context.ExePath, rule.ExeContains) && !rule.GameMode {
		return false
	}
	if len(rule.WindowTitle) > 0 && !containsAny(context.WindowTitle, rule.WindowTitle) {
		return false
	}
	if len(rule.WindowClass) > 0 && !containsAny(context.WindowClass, rule.WindowClass) {
		return false
	}
	return true
}

func normalizeMatchList(values []string, processNames bool) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		if processNames {
			value = normalizeProcessName(value)
		} else {
			value = strings.ToLower(strings.TrimSpace(value))
		}
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func normalizeProcessName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "/")
	value = filepath.Base(value)
	value = strings.Trim(value, `"'`)
	return value
}

func matchesAny(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsAny(target string, needles []string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return false
	}
	for _, needle := range needles {
		if needle != "" && strings.Contains(target, needle) {
			return true
		}
	}
	return false
}
