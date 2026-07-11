package engine

import (
	"sort"
	"strings"
)

var keyBindingCatalog = []KeyBinding{
	{Action: "toggle-mode", Name: "切换中英文", Description: "在中文/英文直通模式之间切换。", Scope: "global"},
	{Action: "commit-selection", Name: "上屏当前候选", Description: "上屏高亮候选；无候选时提交原始输入。", Scope: "composition"},
	{Action: "clear-composition", Name: "清空输入", Description: "取消当前拼音串和候选。", Scope: "composition"},
	{Action: "backspace", Name: "删除一个字符", Description: "删除拼音缓冲区最后一个字符。", Scope: "composition"},
	{Action: "move-selection-prev", Name: "候选上移", Description: "把高亮移动到上一个候选。", Scope: "candidate"},
	{Action: "move-selection-next", Name: "候选下移", Description: "把高亮移动到下一个候选。", Scope: "candidate"},
	{Action: "move-selection-first", Name: "移到首候选", Description: "把高亮移动到第一个候选。", Scope: "candidate"},
	{Action: "move-selection-last", Name: "移到末候选", Description: "把高亮移动到最后一个候选。", Scope: "candidate"},
	{Action: "page-prev", Name: "上一页候选", Description: "翻到上一页候选。", Scope: "candidate"},
	{Action: "page-next", Name: "下一页候选", Description: "翻到下一页候选。", Scope: "candidate"},
	{Action: "quick-select-2", Name: "二候选快捷上屏", Description: "直接上屏第 2 个候选。", Scope: "candidate"},
	{Action: "quick-select-3", Name: "三候选快捷上屏", Description: "直接上屏第 3 个候选。", Scope: "candidate"},
	{Action: "select-candidate-1", Name: "选择第 1 候选", Description: "用数字键直接上屏候选。", Scope: "candidate"},
	{Action: "select-candidate-2", Name: "选择第 2 候选", Description: "用数字键直接上屏候选。", Scope: "candidate"},
	{Action: "select-candidate-3", Name: "选择第 3 候选", Description: "用数字键直接上屏候选。", Scope: "candidate"},
	{Action: "select-candidate-4", Name: "选择第 4 候选", Description: "用数字键直接上屏候选。", Scope: "candidate"},
	{Action: "select-candidate-5", Name: "选择第 5 候选", Description: "用数字键直接上屏候选。", Scope: "candidate"},
	{Action: "select-candidate-6", Name: "选择第 6 候选", Description: "用数字键直接上屏候选。", Scope: "candidate"},
	{Action: "select-candidate-7", Name: "选择第 7 候选", Description: "用数字键直接上屏候选。", Scope: "candidate"},
	{Action: "select-candidate-8", Name: "选择第 8 候选", Description: "用数字键直接上屏候选。", Scope: "candidate"},
	{Action: "select-candidate-9", Name: "选择第 9 候选", Description: "用数字键直接上屏候选。", Scope: "candidate"},
	{Action: "commit-first-char", Name: "首字上屏", Description: "只上屏当前候选的第一个字。", Scope: "candidate"},
	{Action: "commit-last-char", Name: "末字上屏", Description: "只上屏当前候选的最后一个字。", Scope: "candidate"},
}

func KeyBindingCatalog() []KeyBinding {
	out := make([]KeyBinding, len(keyBindingCatalog))
	copy(out, keyBindingCatalog)
	for i := range out {
		out[i].Keys = nil
		out[i].Enabled = true
	}
	return out
}

func DefaultKeyBindings(config Config) []KeyBinding {
	bindings := cloneKeyBindingCatalog()
	set := func(action string, keys ...string) {
		if binding := findKeyBinding(bindings, action); binding != nil {
			binding.Keys = normalizeShortcutList(keys)
			binding.Enabled = true
		}
	}
	set("toggle-mode", boolKeys(config.ShiftToggleMode, "Shift")...)
	set("commit-selection", "Space", "Enter")
	set("clear-composition", "Escape")
	set("backspace", "Backspace")
	set("move-selection-prev", "ArrowLeft", "ArrowUp", "Shift+Tab")
	set("move-selection-next", "ArrowRight", "ArrowDown", "Tab")
	set("move-selection-first", "Home")
	set("move-selection-last", "End")
	set("page-prev", defaultPrevPageKeys(config)...)
	set("page-next", defaultNextPageKeys(config)...)
	set("quick-select-2", boolKeys(config.SemicolonQuickSelect && !(config.DoublePinyin && strings.EqualFold(config.DoublePinyinScheme, "microsoft")), ";")...)
	set("quick-select-3", boolKeys(config.QuoteQuickSelect, "'")...)
	for i := 1; i <= 9; i++ {
		set("select-candidate-"+string(rune('0'+i)), string(rune('0'+i)))
	}
	set("commit-first-char")
	set("commit-last-char")
	return bindings
}

func NormalizeKeyBindings(config Config) []KeyBinding {
	if NormalizeKeyProfile(config.KeyProfile) != "custom" || len(config.KeyBindings) == 0 {
		return DefaultKeyBindings(config)
	}
	defaults := DefaultKeyBindings(config)
	byAction := map[string]KeyBinding{}
	for _, binding := range defaults {
		byAction[binding.Action] = binding
	}
	for _, binding := range config.KeyBindings {
		action := normalizeShortcutAction(binding.Action)
		if action == "" {
			continue
		}
		base, ok := byAction[action]
		if !ok {
			continue
		}
		base.Keys = normalizeShortcutList(binding.Keys)
		base.Enabled = binding.Enabled
		if strings.TrimSpace(binding.Name) != "" {
			base.Name = strings.TrimSpace(binding.Name)
		}
		if strings.TrimSpace(binding.Description) != "" {
			base.Description = strings.TrimSpace(binding.Description)
		}
		byAction[action] = base
	}
	out := cloneKeyBindingCatalog()
	for i := range out {
		out[i] = byAction[out[i].Action]
	}
	return out
}

func KeyBindingConflicts(config Config) []KeyBindingConflict {
	bindings := NormalizeKeyBindings(config)
	keyToActions := map[string][]string{}
	for _, binding := range bindings {
		if !binding.Enabled {
			continue
		}
		for _, key := range binding.Keys {
			key = NormalizeShortcutKey(key)
			if key == "" {
				continue
			}
			keyToActions[key] = append(keyToActions[key], binding.Action)
		}
	}
	var conflicts []KeyBindingConflict
	for key, actions := range keyToActions {
		actions = uniqueStrings(actions)
		if len(actions) > 1 {
			conflicts = append(conflicts, KeyBindingConflict{
				Key:     key,
				Actions: actions,
				Level:   "error",
				Message: "同一个按键绑定了多个输入法动作。",
			})
			continue
		}
		if message := reservedShortcutMessage(key); message != "" {
			conflicts = append(conflicts, KeyBindingConflict{
				Key:     key,
				Actions: actions,
				Level:   "warning",
				Message: message,
			})
		}
	}
	sort.SliceStable(conflicts, func(i, j int) bool {
		if conflicts[i].Level != conflicts[j].Level {
			return conflicts[i].Level < conflicts[j].Level
		}
		return conflicts[i].Key < conflicts[j].Key
	})
	return conflicts
}

// BlockingKeyBindingConflicts returns only ambiguous bindings. Warnings such
// as Ctrl+C or Win combinations remain configurable because users may accept
// their host-app tradeoff, but one keystroke must never resolve to two IME
// actions.
func BlockingKeyBindingConflicts(config Config) []KeyBindingConflict {
	conflicts := KeyBindingConflicts(config)
	blocking := make([]KeyBindingConflict, 0, len(conflicts))
	for _, conflict := range conflicts {
		if conflict.Level == "error" {
			blocking = append(blocking, conflict)
		}
	}
	return blocking
}

func KeyBindingConflictMessage(config Config) string {
	conflicts := BlockingKeyBindingConflicts(config)
	if len(conflicts) == 0 {
		return ""
	}
	items := make([]string, 0, len(conflicts))
	for _, conflict := range conflicts {
		items = append(items, conflict.Key+" ("+strings.Join(conflict.Actions, ", ")+")")
	}
	return "conflicting key bindings: " + strings.Join(items, "; ")
}

func ShortcutActionForStroke(config Config, stroke KeyStroke) string {
	key := NormalizeKeyStroke(stroke)
	if key == "" {
		return ""
	}
	bindings := NormalizeKeyBindings(config)
	for _, binding := range bindings {
		if !binding.Enabled {
			continue
		}
		for _, candidate := range binding.Keys {
			if NormalizeShortcutKey(candidate) == key {
				return binding.Action
			}
		}
	}
	return ""
}

func NormalizeKeyStroke(stroke KeyStroke) string {
	key := firstNonEmpty(stroke.Character, stroke.Key)
	return normalizeShortcutParts(key, stroke.Ctrl, stroke.Alt, stroke.Shift, stroke.Meta, stroke.Modifiers)
}

func NormalizeShortcutKey(value string) string {
	return normalizeShortcutParts(value, false, false, false, false, nil)
}

func normalizeShortcutParts(value string, ctrl bool, alt bool, shift bool, meta bool, modifiers []string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "+")
	key := strings.TrimSpace(parts[len(parts)-1])
	for _, part := range parts[:len(parts)-1] {
		switch normalizeShortcutToken(part) {
		case "Ctrl":
			ctrl = true
		case "Alt":
			alt = true
		case "Shift":
			shift = true
		case "Win":
			meta = true
		}
	}
	for _, modifier := range modifiers {
		switch normalizeShortcutToken(modifier) {
		case "Ctrl":
			ctrl = true
		case "Alt":
			alt = true
		case "Shift":
			shift = true
		case "Win":
			meta = true
		}
	}
	key = normalizeShortcutToken(key)
	if key == "" {
		return ""
	}
	if key == "Ctrl" {
		ctrl = false
	}
	if key == "Alt" {
		alt = false
	}
	if key == "Shift" {
		shift = false
	}
	if key == "Win" {
		meta = false
	}
	var out []string
	if ctrl {
		out = append(out, "Ctrl")
	}
	if alt {
		out = append(out, "Alt")
	}
	if shift {
		out = append(out, "Shift")
	}
	if meta {
		out = append(out, "Win")
	}
	out = append(out, key)
	return strings.Join(out, "+")
}

func cloneKeyBindingCatalog() []KeyBinding {
	out := make([]KeyBinding, len(keyBindingCatalog))
	copy(out, keyBindingCatalog)
	for i := range out {
		out[i].Enabled = true
	}
	return out
}

func findKeyBinding(bindings []KeyBinding, action string) *KeyBinding {
	for i := range bindings {
		if bindings[i].Action == action {
			return &bindings[i]
		}
	}
	return nil
}

func boolKeys(enabled bool, keys ...string) []string {
	if !enabled {
		return nil
	}
	return keys
}

func defaultPrevPageKeys(config Config) []string {
	keys := []string{}
	if config.BracketPageKeys {
		keys = append(keys, "[")
	}
	if config.MinusEqualPageKeys {
		keys = append(keys, "PageUp", "-")
	}
	if config.CommaPeriodPageKeys {
		keys = append(keys, ",")
	}
	return keys
}

func defaultNextPageKeys(config Config) []string {
	keys := []string{}
	if config.BracketPageKeys {
		keys = append(keys, "]")
	}
	if config.MinusEqualPageKeys {
		keys = append(keys, "PageDown", "=")
	}
	if config.CommaPeriodPageKeys {
		keys = append(keys, ".")
	}
	return keys
}

func normalizeShortcutList(keys []string) []string {
	out := make([]string, 0, len(keys))
	seen := map[string]bool{}
	for _, key := range keys {
		normalized := NormalizeShortcutKey(key)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	return out
}

func normalizeShortcutAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	action = strings.ReplaceAll(action, "_", "-")
	switch action {
	case "toggle-mode", "commit-selection", "clear-composition", "backspace",
		"move-selection-prev", "move-selection-next", "move-selection-first", "move-selection-last", "page-prev", "page-next",
		"quick-select-2", "quick-select-3", "commit-first-char", "commit-last-char":
		return action
	}
	if strings.HasPrefix(action, "select-candidate-") {
		suffix := strings.TrimPrefix(action, "select-candidate-")
		if len(suffix) == 1 && suffix[0] >= '1' && suffix[0] <= '9' {
			return action
		}
	}
	return ""
}

func normalizeShortcutToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	switch lower {
	case "control", "ctrl":
		return "Ctrl"
	case "menu", "option", "alt":
		return "Alt"
	case "shift":
		return "Shift"
	case "meta", "cmd", "command", "super", "windows", "win":
		return "Win"
	case "spacebar", "space", " ":
		return "Space"
	case "return", "enter", "\r", "\n":
		return "Enter"
	case "esc", "escape":
		return "Escape"
	case "backspace", "bksp":
		return "Backspace"
	case "tab":
		return "Tab"
	case "left", "arrowleft":
		return "ArrowLeft"
	case "right", "arrowright":
		return "ArrowRight"
	case "up", "arrowup":
		return "ArrowUp"
	case "down", "arrowdown":
		return "ArrowDown"
	case "pageup", "prior", "pgup":
		return "PageUp"
	case "pagedown", "next", "pgdn":
		return "PageDown"
	case "del", "delete":
		return "Delete"
	case "home":
		return "Home"
	case "end":
		return "End"
	default:
		runes := []rune(value)
		if len(runes) == 1 {
			return strings.ToUpper(value)
		}
		return value
	}
}

func reservedShortcutMessage(key string) string {
	switch key {
	case "Alt+Tab", "Alt+F4", "Ctrl+Alt+Delete", "Ctrl+Shift", "Win+Space":
		return "这是 Windows 常见系统快捷键，建议换一个组合。"
	}
	if strings.HasPrefix(key, "Win+") {
		return "Win 组合键通常由 Windows 全局占用，可能无法稳定被输入法接管。"
	}
	switch key {
	case "Ctrl+C", "Ctrl+V", "Ctrl+X", "Ctrl+Z", "Ctrl+Y", "Ctrl+A", "Ctrl+S", "Ctrl+F", "Ctrl+P", "Ctrl+W", "Ctrl+N", "Ctrl+T":
		return "这是常见应用快捷键，绑定后可能影响宿主程序。"
	}
	return ""
}
