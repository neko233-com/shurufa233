package engine

import (
	"fmt"
	"strconv"
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
	styleApplied, styleWarnings := applyRimeStyleConfig(&config, patch)
	applied = append(applied, styleApplied...)
	warnings = append(warnings, styleWarnings...)
	nextConfig, appOptionsApplied, appOptionsWarnings := applyRimeAppOptions(config, patch)
	config = nextConfig
	applied = append(applied, appOptionsApplied...)
	warnings = append(warnings, appOptionsWarnings...)
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
	if raw, ok := rimeLookup(patch, "style/candidate_list_layout"); ok {
		layout := normalizeCandidateLayout(rimeString(raw))
		config.CandidateLayout = layout
		applied = append(applied, "style/candidate_list_layout:"+layout)
	}
	if raw, ok := rimeLookup(patch, "speller/algebra"); ok {
		next, algebraApplied, algebraWarnings := applyRimeSpellerAlgebra(config, raw)
		config = next
		applied = append(applied, algebraApplied...)
		warnings = append(warnings, algebraWarnings...)
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
	if raw, ok := rimeLookup(patch, "punctuator/full_shape"); ok {
		if shape := rimePunctuationShape(raw); len(shape) > 0 {
			config.PunctuationFullShape = mergePunctuationShape(config.PunctuationFullShape, shape)
			applied = append(applied, "punctuator/full_shape")
		} else {
			warnings = append(warnings, "punctuator/full_shape has no supported entries")
		}
	}
	if raw, ok := rimeLookup(patch, "punctuator/half_shape"); ok {
		if shape := rimePunctuationShape(raw); len(shape) > 0 {
			config.PunctuationHalfShape = mergePunctuationShape(config.PunctuationHalfShape, shape)
			config.Punctuation = "half"
			applied = append(applied, "punctuator/half_shape")
		} else if _, ok := rimeMap(raw); ok {
			config.Punctuation = "half"
			applied = append(applied, "punctuator/half_shape")
		} else {
			warnings = append(warnings, "punctuator/half_shape has no supported entries")
		}
	}
	if raw, ok := rimeLookup(patch, "recognizer/import_preset"); ok {
		preset := strings.ToLower(strings.TrimSpace(rimeString(raw)))
		switch preset {
		case "", "default":
			config.RecognizerPatterns = mergeRecognizerPatterns(config.RecognizerPatterns, DefaultRecognizerPatterns())
			applied = append(applied, "recognizer/import_preset:default")
		default:
			warnings = append(warnings, "unsupported recognizer preset: "+preset)
		}
	}
	if raw, ok := rimeLookup(patch, "recognizer/patterns"); ok {
		if patterns := rimeStringMap(raw); len(patterns) > 0 {
			config.RecognizerPatterns = mergeRecognizerPatterns(config.RecognizerPatterns, patterns)
			applied = append(applied, "recognizer/patterns")
		} else {
			warnings = append(warnings, "recognizer/patterns has no supported entries")
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

func applyRimeStyleConfig(config *Config, patch map[string]any) ([]string, []string) {
	applied := []string{}
	warnings := []string{}
	if raw, ok := rimeLookup(patch, "style/font_face"); ok {
		if font := rimeFontFace(raw); font != "" {
			config.Skin.FontFamily = font
			applied = append(applied, "style/font_face")
		}
	}
	if raw, ok := rimeLookup(patch, "style/font_point"); ok {
		if point, ok := rimeInt(raw); ok {
			config.Skin.FontSize = clampRimeFontPoint(point)
			applied = append(applied, fmt.Sprintf("style/font_point:%d", point))
		} else {
			warnings = append(warnings, "style/font_point is not a number")
		}
	}
	if raw, ok := rimeLookup(patch, "style/color_scheme"); ok {
		scheme := strings.TrimSpace(rimeString(raw))
		if scheme != "" {
			if schemesRaw, ok := rimeLookup(patch, "preset_color_schemes"); ok {
				if mapped, ok := rimeNamedColorScheme(schemesRaw, scheme); ok {
					applyRimeColorScheme(config, scheme, mapped)
					applied = append(applied, "style/color_scheme:"+scheme)
					return uniqueStrings(applied), uniqueStrings(warnings)
				}
			}
			if next, ok := ApplySkinPresetConfig(*config, scheme); ok {
				*config = next
				applied = append(applied, "style/color_scheme:"+scheme)
			} else {
				warnings = append(warnings, "unsupported style/color_scheme: "+scheme)
			}
		}
	}
	return uniqueStrings(applied), uniqueStrings(warnings)
}

func applyRimeAppOptions(config Config, patch map[string]any) (Config, []string, []string) {
	options := collectRimeAppOptions(patch)
	applied := []string{}
	warnings := []string{}
	rules := []AppRule{}
	for app, fields := range options {
		rule, ok, ruleWarnings := rimeAppOptionRule(app, fields)
		warnings = append(warnings, ruleWarnings...)
		if !ok {
			continue
		}
		rules = append(rules, rule)
		applied = append(applied, "app_options:"+app)
	}
	if len(rules) > 0 {
		config.AppRules = mergeRimeAppRules(config.AppRules, rules)
	}
	return config, uniqueStrings(applied), uniqueStrings(warnings)
}

func collectRimeAppOptions(patch map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	if raw, ok := rimeLookup(patch, "app_options"); ok {
		if mapped, ok := rimeMap(raw); ok {
			for app, value := range mapped {
				if fields, ok := rimeMap(value); ok {
					mergeRimeAppOption(out, app, fields)
				}
			}
		}
	}
	for key, value := range patch {
		if !strings.HasPrefix(key, "app_options/") {
			continue
		}
		rest := strings.TrimPrefix(key, "app_options/")
		parts := strings.Split(rest, "/")
		if len(parts) == 1 {
			if fields, ok := rimeMap(value); ok {
				mergeRimeAppOption(out, parts[0], fields)
			}
			continue
		}
		app := strings.TrimSpace(parts[0])
		field := strings.TrimSpace(strings.Join(parts[1:], "/"))
		if app == "" || field == "" {
			continue
		}
		if _, ok := out[app]; !ok {
			out[app] = map[string]any{}
		}
		out[app][field] = value
	}
	return out
}

func mergeRimeAppOption(out map[string]map[string]any, app string, fields map[string]any) {
	app = strings.TrimSpace(app)
	if app == "" {
		return
	}
	if _, ok := out[app]; !ok {
		out[app] = map[string]any{}
	}
	for key, value := range fields {
		key = strings.TrimSpace(key)
		if key != "" {
			out[app][key] = value
		}
	}
}

func rimeAppOptionRule(app string, fields map[string]any) (AppRule, bool, []string) {
	app = strings.TrimSpace(app)
	if app == "" {
		return AppRule{}, false, nil
	}
	rule := AppRule{
		ID:          "rime-app-" + sanitizeRimeAppRuleID(app),
		Name:        "Rime app option " + app,
		Description: "Imported from Rime app_options/" + app,
		Priority:    650,
	}
	if rimeAppOptionLooksLikeProcess(app) {
		rule.ProcessNames = []string{app}
	} else {
		rule.WindowClass = []string{strings.ToLower(app)}
	}
	mapped := false
	if value, ok := rimeOptionBool(fields, "ascii_mode"); ok {
		if value {
			rule.Mode = "en"
		} else {
			rule.Mode = "zh"
		}
		mapped = true
	}
	if value, ok := rimeOptionBool(fields, "ascii_punct", "ascii_punctuation"); ok {
		if value {
			rule.Punctuation = "half"
		} else {
			rule.Punctuation = "full"
		}
		mapped = true
	}
	if value, ok := rimeOptionBool(fields, "disabled", "disable_input_method"); ok && value {
		rule.Mode = "en"
		rule.Punctuation = "half"
		rule.DisableCandidates = true
		rule.DisableLearning = true
		mapped = true
	}
	if value, ok := rimeOptionBool(fields, "disable_candidates", "no_candidates"); ok {
		rule.DisableCandidates = value
		mapped = true
	}
	if value, ok := rimeOptionBool(fields, "disable_learning", "no_user_dict"); ok {
		rule.DisableLearning = value
		mapped = true
	}
	if !mapped {
		return AppRule{}, false, []string{"app_options/" + app + " has no supported fields"}
	}
	return normalizeAppRule(rule), true, nil
}

func rimeOptionBool(fields map[string]any, names ...string) (bool, bool) {
	for _, name := range names {
		if value, ok := fields[name]; ok {
			return rimeBool(value)
		}
	}
	return false, false
}

func rimeAppOptionLooksLikeProcess(app string) bool {
	app = strings.ToLower(strings.TrimSpace(app))
	if app == "" {
		return false
	}
	if strings.HasSuffix(app, ".exe") {
		return true
	}
	return !strings.Contains(app, ".")
}

func sanitizeRimeAppRuleID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "custom"
	}
	return out
}

func mergeRimeAppRules(base []AppRule, imported []AppRule) []AppRule {
	replacements := map[string]AppRule{}
	for _, rule := range imported {
		replacements[rule.ID] = rule
	}
	out := make([]AppRule, 0, len(base)+len(imported))
	for _, rule := range NormalizeAppRules(base) {
		if replacement, ok := replacements[rule.ID]; ok {
			out = append(out, replacement)
			delete(replacements, rule.ID)
			continue
		}
		out = append(out, rule)
	}
	for _, rule := range replacements {
		out = append(out, rule)
	}
	return NormalizeAppRules(out)
}

func rimeFontFace(raw any) string {
	value := strings.TrimSpace(rimeString(raw))
	if value == "" {
		return ""
	}
	for _, part := range strings.Split(value, ",") {
		part = strings.Trim(strings.TrimSpace(part), `"'`)
		if part != "" {
			return part
		}
	}
	return ""
}

func clampRimeFontPoint(point int) int {
	if point < 12 {
		return 12
	}
	if point > 24 {
		return 24
	}
	return point
}

func rimeNamedColorScheme(raw any, scheme string) (map[string]any, bool) {
	schemes, ok := rimeMap(raw)
	if !ok {
		return nil, false
	}
	mapped, ok := rimeMap(schemes[scheme])
	return mapped, ok
}

func applyRimeColorScheme(config *Config, scheme string, mapped map[string]any) {
	if font := rimeFontFace(mapped["font_face"]); font != "" {
		config.Skin.FontFamily = font
	}
	if point, ok := rimeInt(mapped["font_point"]); ok {
		config.Skin.FontSize = clampRimeFontPoint(point)
	}
	if color, ok := rimeColor(mapped["back_color"]); ok {
		config.Skin.Surface = color
	}
	if color, ok := firstRimeColor(mapped, "candidate_text_color", "text_color", "label_color"); ok {
		config.Skin.Text = color
	}
	if color, ok := firstRimeColor(mapped, "comment_text_color", "candidate_comment_text_color"); ok {
		config.Skin.MutedText = color
	}
	if color, ok := rimeColor(mapped["border_color"]); ok {
		config.Skin.Border = color
	}
	if color, ok := firstRimeColor(mapped, "hilited_candidate_back_color", "hilited_back_color"); ok {
		config.Skin.Accent = color
	}
	if color, ok := firstRimeColor(mapped, "hilited_candidate_text_color", "hilited_text_color"); ok {
		config.Skin.HighlightText = color
	}
	if raw, ok := mapped["horizontal"]; ok {
		if horizontal, ok := rimeBool(raw); ok {
			if horizontal {
				config.CandidateLayout = "horizontal"
			} else {
				config.CandidateLayout = "vertical"
			}
		}
	}
	if raw, ok := mapped["inline_preedit"]; ok {
		if inline, ok := rimeBool(raw); ok && inline {
			config.ShowCandidateComments = false
		}
	}
	config.Skin.Theme = "rime-" + sanitizeRimeThemeID(scheme)
}

func firstRimeColor(mapped map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if color, ok := rimeColor(mapped[key]); ok {
			return color, true
		}
	}
	return "", false
}

func rimeColor(raw any) (string, bool) {
	value := strings.TrimSpace(rimeString(raw))
	value = strings.Trim(value, `"'`)
	if value == "" {
		return "", false
	}
	if strings.HasPrefix(value, "#") {
		if len(value) == 7 && isRimeHex(value[1:]) {
			return strings.ToLower(value), true
		}
		return "", false
	}
	base := 10
	if strings.HasPrefix(strings.ToLower(value), "0x") {
		value = value[2:]
		base = 16
	}
	if strings.HasPrefix(value, "$") {
		value = value[1:]
		base = 16
	}
	parsed, err := strconv.ParseInt(value, base, 32)
	if err != nil || parsed < 0 || parsed > 0xffffff {
		return "", false
	}
	// Weasel/Squirrel color schemes commonly encode colors as 0xBBGGRR.
	red := parsed & 0xff
	green := (parsed >> 8) & 0xff
	blue := (parsed >> 16) & 0xff
	return fmt.Sprintf("#%02x%02x%02x", red, green, blue), true
}

func isRimeHex(value string) bool {
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

func sanitizeRimeThemeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			builder.WriteRune(char)
		} else if char == '_' || char == ' ' {
			builder.WriteByte('-')
		}
	}
	if builder.Len() == 0 {
		return "custom"
	}
	return builder.String()
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

func applyRimeSpellerAlgebra(config Config, raw any) (Config, []string, []string) {
	applied := []string{}
	warnings := []string{}
	rules := []string{}
	for _, item := range rimeSlice(raw) {
		rule := strings.TrimSpace(rimeString(item))
		if rule == "" {
			continue
		}
		rules = append(rules, rule)
		if fuzzy := rimeAlgebraFuzzyRule(rule); fuzzy != "" {
			config.FuzzyInitials = appendUnique(config.FuzzyInitials, fuzzy)
			applied = append(applied, "speller/algebra:"+rule)
		}
	}
	if len(rules) > 0 {
		config.SpellerAlgebra = appendUnique(config.SpellerAlgebra, rules...)
		if len(applied) == 0 {
			applied = append(applied, "speller/algebra")
			warnings = append(warnings, "speller/algebra stored; no immediately supported fuzzy derive rule found")
		}
	}
	return config, applied, warnings
}

func rimePunctuationShape(raw any) map[string][]string {
	mapped, ok := rimeMap(raw)
	if !ok {
		return nil
	}
	out := map[string][]string{}
	for key, value := range mapped {
		if key == "" {
			continue
		}
		values := rimePunctuationValues(value)
		if len(values) > 0 {
			out[key] = values
		}
	}
	return out
}

func rimePunctuationValues(raw any) []string {
	switch typed := raw.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	case []any:
		values := []string{}
		for _, item := range typed {
			values = append(values, rimePunctuationValues(item)...)
		}
		return uniqueRawStrings(values)
	case []string:
		return uniqueRawStrings(typed)
	default:
		if mapped, ok := rimeMap(raw); ok {
			for _, field := range []string{"commit", "pair", "text"} {
				if value, ok := mapped[field]; ok {
					if values := rimePunctuationValues(value); len(values) > 0 {
						return values
					}
				}
			}
		}
	}
	value := rimeString(raw)
	if value == "" || value == "<nil>" {
		return nil
	}
	return []string{value}
}

func mergePunctuationShape(base map[string][]string, next map[string][]string) map[string][]string {
	if len(base) == 0 {
		base = map[string][]string{}
	}
	for key, values := range next {
		base[key] = uniqueRawStrings(append(base[key], values...))
	}
	return base
}

func rimeStringMap(raw any) map[string]string {
	mapped, ok := rimeMap(raw)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for key, value := range mapped {
		key = strings.TrimSpace(key)
		text := strings.TrimSpace(rimeString(value))
		if key == "" || text == "" {
			continue
		}
		out[key] = text
	}
	return out
}

func mergeRecognizerPatterns(base map[string]string, next map[string]string) map[string]string {
	if len(base) == 0 {
		base = map[string]string{}
	}
	for key, value := range next {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		base[key] = value
	}
	return base
}

func rimeAlgebraFuzzyRule(rule string) string {
	rule = strings.TrimSpace(rule)
	switch rule {
	case "derive/^([zcs])h/$1/":
		return "zh=z,ch=c,sh=s"
	case "derive/^zh/z/", "derive/^z/zh/":
		return "zh=z"
	case "derive/^ch/c/", "derive/^c/ch/":
		return "ch=c"
	case "derive/^sh/s/", "derive/^s/sh/":
		return "sh=s"
	case "derive/^([nl])ue$/$1ve/", "derive/^([nl])ve$/$1ue/":
		return "ue=ve"
	case "derive/un$/uen/", "derive/uen$/un/":
		return "un=uen"
	case "derive/ui$/uei/", "derive/uei$/ui/":
		return "ui=uei"
	case "derive/iu$/iou/", "derive/iou$/iu/":
		return "iu=iou"
	case "derive/ong$/on/", "derive/on$/ong/":
		return "ong=on"
	case "derive/ao$/oa/", "derive/oa$/ao/":
		return "ao=oa"
	default:
		return ""
	}
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

func uniqueRawStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func appendUnique(values []string, additions ...string) []string {
	return uniqueStrings(append(values, splitCommaRules(additions)...))
}

func splitCommaRules(values []string) []string {
	out := []string{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}
