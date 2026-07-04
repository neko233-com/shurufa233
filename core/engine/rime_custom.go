package engine

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type RimeCustomResult struct {
	OK        bool      `json:"ok"`
	Config    Config    `json:"config"`
	Schema    string    `json:"schema,omitempty"`
	Applied   []string  `json:"applied"`
	Warnings  []string  `json:"warnings,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func ApplyRimeCustomYAML(base Config, data []byte) (RimeCustomResult, error) {
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return RimeCustomResult{}, err
	}
	if len(root) == 0 {
		return RimeCustomResult{}, fmt.Errorf("empty rime custom yaml")
	}
	patch := root
	if raw, ok := rimeLookup(root, "patch"); ok {
		if mapped, ok := rimeMap(raw); ok {
			patch = mapped
		}
	}
	config := NormalizeSwitchConfig(base)
	applied := []string{}
	warnings := []string{}
	explicitSchema := ""

	if raw, ok := rimeLookup(patch, "schema_list"); ok {
		schema, appliedSchema, warning := applyRimeSchemaList(&config, raw)
		if appliedSchema != "" {
			explicitSchema = appliedSchema
			applied = append(applied, "schema_list:"+schema)
		}
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}
	if raw, ok := rimeLookup(patch, "schema/schema_id"); ok && explicitSchema == "" {
		schema := strings.TrimSpace(rimeString(raw))
		if schema != "" {
			if next, ok := ApplySchemaPresetConfig(config, schema); ok {
				config = next
				explicitSchema = NormalizeSchemaID(schema)
				applied = append(applied, "schema/schema_id:"+schema)
			} else {
				warnings = append(warnings, "unsupported schema: "+schema)
			}
		}
	}
	if raw, ok := rimeLookup(patch, "menu/page_size"); ok {
		if value, ok := rimeInt(raw); ok {
			config.CandidatePageSize = value
			applied = append(applied, fmt.Sprintf("menu/page_size:%d", value))
		} else {
			warnings = append(warnings, "menu/page_size is not a number")
		}
	}
	if raw, ok := rimeLookup(patch, "style/horizontal"); ok {
		if value, ok := rimeBool(raw); ok {
			if value {
				config.CandidateLayout = "horizontal"
				applied = append(applied, "style/horizontal:true")
			} else {
				config.CandidateLayout = "vertical"
				applied = append(applied, "style/horizontal:false")
			}
		}
	}
	if raw, ok := rimeLookup(patch, "style/vertical"); ok {
		if value, ok := rimeBool(raw); ok && value {
			config.CandidateLayout = "vertical"
			applied = append(applied, "style/vertical:true")
		}
	}
	if raw, ok := rimeLookup(patch, "switches"); ok {
		next, appliedSwitches, switchWarnings := applyRimeSwitches(config, raw)
		config = next
		applied = append(applied, appliedSwitches...)
		warnings = append(warnings, switchWarnings...)
	}
	if raw, ok := rimeLookup(patch, "translator/enable_sentence"); ok {
		if value, ok := rimeBool(raw); ok {
			config.Associations = value
			applied = append(applied, fmt.Sprintf("translator/enable_sentence:%t", value))
		}
	}
	if raw, ok := rimeLookup(patch, "punctuator/import_preset"); ok {
		preset := strings.ToLower(strings.TrimSpace(rimeString(raw)))
		switch preset {
		case "symbols", "symbol":
			applied = append(applied, "punctuator/import_preset:symbols")
		case "alternative", "ascii":
			config.Punctuation = "half"
			applied = append(applied, "punctuator/import_preset:"+preset)
		case "", "default":
			applied = append(applied, "punctuator/import_preset:default")
		default:
			warnings = append(warnings, "unsupported punctuator preset: "+preset)
		}
	}
	if raw, ok := rimeLookup(patch, "punctuator/half_shape"); ok {
		if _, ok := rimeMap(raw); ok {
			config.Punctuation = "half"
			applied = append(applied, "punctuator/half_shape")
		}
	}
	if raw, ok := rimeLookup(patch, "key_binder/import_preset"); ok {
		preset := strings.ToLower(strings.TrimSpace(rimeString(raw)))
		switch preset {
		case "", "default", "alternative":
			config.KeyProfile = "rime"
			config.BracketPageKeys = true
			config.MinusEqualPageKeys = true
			config.CommaPeriodPageKeys = true
			config.SemicolonQuickSelect = false
			config.QuoteQuickSelect = false
			applied = append(applied, "key_binder/import_preset:"+rimeFirstNonEmpty(preset, "default"))
		default:
			warnings = append(warnings, "unsupported key_binder preset: "+preset)
		}
	}
	if raw, ok := rimeLookup(patch, "key_binder/bindings"); ok {
		next, bindingApplied := applyRimeKeyBindings(config, raw)
		config = next
		applied = append(applied, bindingApplied...)
	}
	if raw, ok := rimeLookup(patch, "ascii_composer/switch_key/Shift_L"); ok {
		behavior := strings.ToLower(strings.TrimSpace(rimeString(raw)))
		config.KeyProfile = "custom"
		config.ShiftToggleMode = behavior != "noop"
		applied = append(applied, "ascii_composer/switch_key/Shift_L:"+behavior)
	}
	if raw, ok := rimeLookup(patch, "ascii_composer/switch_key/Shift_R"); ok {
		behavior := strings.ToLower(strings.TrimSpace(rimeString(raw)))
		config.KeyProfile = "custom"
		config.ShiftToggleMode = config.ShiftToggleMode || behavior != "noop"
		applied = append(applied, "ascii_composer/switch_key/Shift_R:"+behavior)
	}

	config = NormalizeSwitchConfig(config)
	if explicitSchema != "" {
		config.Schema = explicitSchema
	}
	return RimeCustomResult{
		OK:        true,
		Config:    config,
		Schema:    config.Schema,
		Applied:   uniqueStrings(applied),
		Warnings:  uniqueStrings(warnings),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func applyRimeSchemaList(config *Config, raw any) (string, string, string) {
	for _, item := range rimeSlice(raw) {
		schema := ""
		if mapped, ok := rimeMap(item); ok {
			schema = rimeString(mapped["schema"])
		} else {
			schema = rimeString(item)
		}
		schema = strings.TrimSpace(schema)
		if schema == "" {
			continue
		}
		next, ok := ApplySchemaPresetConfig(*config, schema)
		if ok {
			*config = next
			return schema, NormalizeSchemaID(schema), ""
		}
	}
	return "", "", "schema_list contains no supported schema"
}

func applyRimeSwitches(config Config, raw any) (Config, []string, []string) {
	applied := []string{}
	warnings := []string{}
	for _, item := range rimeSlice(raw) {
		mapped, ok := rimeMap(item)
		if !ok {
			continue
		}
		name := strings.TrimSpace(rimeString(mapped["name"]))
		id := NormalizeSwitchID(name)
		if id == "" {
			warnings = append(warnings, "unsupported switch: "+name)
			continue
		}
		value := false
		if rawReset, ok := mapped["reset"]; ok {
			if resetBool, ok := rimeBool(rawReset); ok {
				value = resetBool
			} else if resetInt, ok := rimeInt(rawReset); ok {
				value = resetInt != 0
			}
		}
		next, option, ok := ApplySwitch(config, id, value, false)
		if !ok {
			warnings = append(warnings, "unsupported switch: "+name)
			continue
		}
		config = next
		applied = append(applied, "switches/"+option.ID)
	}
	return config, applied, warnings
}

func applyRimeKeyBindings(config Config, raw any) (Config, []string) {
	applied := []string{}
	for _, item := range rimeSlice(raw) {
		mapped, ok := rimeMap(item)
		if !ok {
			continue
		}
		accept := strings.ToLower(strings.TrimSpace(rimeString(mapped["accept"])))
		send := strings.ToLower(strings.TrimSpace(rimeString(mapped["send"])))
		if accept == "" && send == "" {
			continue
		}
		config.KeyProfile = "custom"
		switch accept {
		case "bracketleft", "bracketright":
			config.BracketPageKeys = true
			applied = append(applied, "key_binder/bindings:brackets")
		case "minus", "equal":
			config.MinusEqualPageKeys = true
			applied = append(applied, "key_binder/bindings:-=")
		case "comma", "period":
			if strings.Contains(send, "page_") || strings.Contains(send, "page") {
				config.CommaPeriodPageKeys = true
				applied = append(applied, "key_binder/bindings:,.")
			}
		case "semicolon":
			if strings.Contains(send, "2") || strings.Contains(send, "second") {
				config.SemicolonQuickSelect = true
				applied = append(applied, "key_binder/bindings:semicolon")
			}
		case "apostrophe":
			if strings.Contains(send, "3") || strings.Contains(send, "third") {
				config.QuoteQuickSelect = true
				applied = append(applied, "key_binder/bindings:apostrophe")
			}
		}
	}
	return config, uniqueStrings(applied)
}

func rimeLookup(root map[string]any, path string) (any, bool) {
	if value, ok := root[path]; ok {
		return value, true
	}
	parts := strings.Split(path, "/")
	var current any = root
	for _, part := range parts {
		mapped, ok := rimeMap(current)
		if !ok {
			return nil, false
		}
		value, ok := mapped[part]
		if !ok {
			return nil, false
		}
		current = value
	}
	return current, true
}

func rimeMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[fmt.Sprint(key)] = value
		}
		return out, true
	default:
		return nil, false
	}
}

func rimeSlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func rimeString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func rimeBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case int:
		return typed != 0, true
	case int64:
		return typed != 0, true
	case float64:
		return typed != 0, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "yes", "on", "1":
			return true, true
		case "false", "no", "off", "0":
			return false, true
		}
	}
	return false, false
}

func rimeInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &parsed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func rimeFirstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
