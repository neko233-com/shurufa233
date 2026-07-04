package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neko233-com/shurufa233/core/engine"
)

//export ShurufaAbiVersion
func ShurufaAbiVersion() *C.char {
	return C.CString(abiVersion)
}

//export ShurufaCapabilities
func ShurufaCapabilities() *C.char {
	return jsonCString(map[string]any{
		"ok":         true,
		"abiVersion": abiVersion,
		"features":   abiFeatureList,
		"updatedAt":  time.Now().UTC(),
	})
}

//export ShurufaState
func ShurufaState(id C.uint64_t) *C.char {
	return jsonCString(getSession(uint64(id)).State())
}

//export ShurufaCandidatePayloadV2
func ShurufaCandidatePayloadV2(id C.uint64_t, start C.int, limit C.int) *C.char {
	return jsonCString(buildCandidatePayloadV2(getSession(uint64(id)), int(start), int(limit)))
}

//export ShurufaAssociate
func ShurufaAssociate(id C.uint64_t, payload *C.char) *C.char {
	req, err := decodeExtensionCommandPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	return jsonCString(getSession(uint64(id)).Associate(firstNonEmpty(req.Context, req.Input, req.Text), req.Limit))
}

//export ShurufaCandidateAction
func ShurufaCandidateAction(id C.uint64_t, payload *C.char) *C.char {
	req, err := decodeExtensionCommandPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	return jsonCString(executeCandidateAction(getSession(uint64(id)), req))
}

//export ShurufaCatalogJSON
func ShurufaCatalogJSON(id C.uint64_t, payload *C.char) *C.char {
	req, err := decodeExtensionCommandPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	return jsonCString(getSession(uint64(id)).CatalogEntries(engine.CatalogRequest{
		Kind:  req.Kind,
		Query: firstNonEmpty(req.Query, req.Input, req.Reading),
		Limit: req.Limit,
	}))
}

//export ShurufaReverseLookupJSON
func ShurufaReverseLookupJSON(id C.uint64_t, payload *C.char) *C.char {
	req, err := decodeExtensionCommandPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	return jsonCString(getSession(uint64(id)).ReverseLookup(engine.ReverseLookupRequest{
		Query: firstNonEmpty(req.Query, req.Text, req.Input, req.Context),
		Limit: req.Limit,
	}))
}

//export ShurufaConfigJSON
func ShurufaConfigJSON() *C.char {
	return jsonCString(configEnvelope())
}

//export ShurufaApplyConfigJSON
func ShurufaApplyConfigJSON(payload *C.char) *C.char {
	var config engine.Config
	if err := json.Unmarshal([]byte(C.GoString(payload)), &config); err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	config = normalizeConfig(config)
	return jsonCString(applyConfigEnvelope(config))
}

//export ShurufaSchemaPresetsJSON
func ShurufaSchemaPresetsJSON() *C.char {
	config := loadConfig()
	return jsonCString(map[string]any{
		"ok":        true,
		"selected":  config.Schema,
		"schemas":   engine.BuiltinSchemaPresets(),
		"config":    config,
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaApplySchemaJSON
func ShurufaApplySchemaJSON(payload *C.char) *C.char {
	req, err := decodeExtensionCommandPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	config := loadConfig()
	next, ok := engine.ApplySchemaPresetConfig(config, firstNonEmpty(req.ID, req.Schema, req.Input, req.Text))
	if !ok {
		return jsonCString(errorEnvelope("unknown schema id"))
	}
	next = normalizeConfig(next)
	return jsonCString(applyConfigEnvelope(next))
}

//export ShurufaApplyRimeCustomJSON
func ShurufaApplyRimeCustomJSON(id C.uint64_t, payload *C.char) *C.char {
	return jsonCString(applyRimeCustomPayload(getSession(uint64(id)), C.GoString(payload)))
}

//export ShurufaSwitchesJSON
func ShurufaSwitchesJSON(id C.uint64_t) *C.char {
	session := getSession(uint64(id))
	return jsonCString(map[string]any{
		"ok":        true,
		"switches":  engine.SwitchOptions(session.Config()),
		"config":    session.Config(),
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaApplySwitchJSON
func ShurufaApplySwitchJSON(id C.uint64_t, payload *C.char) *C.char {
	req, err := decodeExtensionCommandPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	session := getSession(uint64(id))
	value := false
	if req.Value != nil {
		value = *req.Value
	}
	toggle := req.Value == nil || strings.EqualFold(req.Action, "toggle")
	next, option, ok := engine.ApplySwitch(session.Config(), firstNonEmpty(req.ID, req.Switch, req.Schema, req.Input, req.Text), value, toggle)
	if !ok {
		return jsonCString(errorEnvelope("unknown switch id"))
	}
	session.Configure(next)
	return jsonCString(map[string]any{
		"ok":        true,
		"switch":    option,
		"switches":  engine.SwitchOptions(next),
		"config":    next,
		"state":     session.State(),
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaAppRulesJSON
func ShurufaAppRulesJSON(id C.uint64_t) *C.char {
	session := getSession(uint64(id))
	return jsonCString(map[string]any{
		"ok":        true,
		"rules":     engine.NormalizeAppRules(session.Config().AppRules),
		"config":    session.Config(),
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaResolveAppContextJSON
func ShurufaResolveAppContextJSON(id C.uint64_t, payload *C.char) *C.char {
	req, err := decodeExtensionCommandPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
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
	return jsonCString(engine.ResolveAppContext(getSession(uint64(id)).Config(), context))
}

//export ShurufaProfileJSON
func ShurufaProfileJSON(id C.uint64_t) *C.char {
	return jsonCString(buildProfileBundle(getSession(uint64(id))))
}

//export ShurufaImportProfileJSON
func ShurufaImportProfileJSON(id C.uint64_t, payload *C.char) *C.char {
	bundle, err := decodeProfileBundle(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	return jsonCString(importProfileBundle(getSession(uint64(id)), bundle))
}

//export ShurufaReloadConfig
func ShurufaReloadConfig() *C.char {
	config := loadConfig()
	return jsonCString(applyConfigEnvelope(config))
}

//export ShurufaReloadDictionaries
func ShurufaReloadDictionaries() *C.char {
	return jsonCString(reloadDictionariesEnvelope())
}

//export ShurufaDictionaryManifestJSON
func ShurufaDictionaryManifestJSON() *C.char {
	return jsonCString(dictionaryManifestEnvelope())
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

func dictionaryManifestEnvelope() map[string]any {
	dir, err := dictionaryDir()
	if err != nil {
		return map[string]any{
			"ok":        false,
			"error":     err.Error(),
			"updatedAt": time.Now().UTC(),
		}
	}
	for _, name := range []string{"manifest.json", "dictionary-manifest.json"} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var manifest any
		if err := json.Unmarshal(data, &manifest); err != nil {
			return map[string]any{
				"ok":        false,
				"error":     err.Error(),
				"updatedAt": time.Now().UTC(),
			}
		}
		return map[string]any{
			"ok":        true,
			"path":      path,
			"manifest":  manifest,
			"updatedAt": time.Now().UTC(),
		}
	}
	return map[string]any{
		"ok":        true,
		"path":      "",
		"manifest":  nil,
		"updatedAt": time.Now().UTC(),
	}
}

//export ShurufaUserScoresJSON
func ShurufaUserScoresJSON(id C.uint64_t) *C.char {
	scores := getSession(uint64(id)).UserScores()
	return jsonCString(map[string]any{
		"ok":         true,
		"userScores": scores,
		"count":      len(scores),
		"updatedAt":  time.Now().UTC(),
	})
}

//export ShurufaUserPhrasesJSON
func ShurufaUserPhrasesJSON(id C.uint64_t) *C.char {
	phrases := getSession(uint64(id)).UserPhrases()
	return jsonCString(map[string]any{
		"ok":        true,
		"phrases":   phrases,
		"entries":   phrases,
		"count":     len(phrases),
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaUserRejectsJSON
func ShurufaUserRejectsJSON(id C.uint64_t) *C.char {
	rejects := getSession(uint64(id)).UserRejects()
	return jsonCString(map[string]any{
		"ok":        true,
		"rejects":   rejects,
		"entries":   rejects,
		"count":     len(rejects),
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaUserPinsJSON
func ShurufaUserPinsJSON(id C.uint64_t) *C.char {
	pins := getSession(uint64(id)).UserPins()
	return jsonCString(map[string]any{
		"ok":        true,
		"pins":      pins,
		"entries":   pins,
		"count":     len(pins),
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaImportUserScoresJSON
func ShurufaImportUserScoresJSON(id C.uint64_t, payload *C.char) *C.char {
	scores, err := decodeUserScoresPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	session := getSession(uint64(id))
	session.ImportUserScores(scores)
	persistUserScores(session.UserScores())
	return jsonCString(map[string]any{
		"ok":        true,
		"imported":  len(scores),
		"total":     len(session.UserScores()),
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaImportUserPhrasesJSON
func ShurufaImportUserPhrasesJSON(id C.uint64_t, payload *C.char) *C.char {
	req, err := decodeExtensionCommandPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	entries := req.Entries
	if len(entries) == 0 {
		entries = req.Phrases
	}
	session := getSession(uint64(id))
	if req.Merge {
		merged := session.UserPhrases()
		merged = append(merged, entries...)
		entries = merged
	}
	session.ReplaceUserPhrases(entries)
	phrases := session.UserPhrases()
	persistUserPhrases(phrases)
	return jsonCString(map[string]any{
		"ok":        true,
		"imported":  len(entries),
		"total":     len(phrases),
		"phrases":   phrases,
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaImportUserRejectsJSON
func ShurufaImportUserRejectsJSON(id C.uint64_t, payload *C.char) *C.char {
	req, err := decodeExtensionCommandPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	entries := req.Entries
	if len(entries) == 0 {
		entries = req.Rejects
	}
	session := getSession(uint64(id))
	if req.Merge {
		merged := session.UserRejects()
		merged = append(merged, entries...)
		entries = merged
	}
	session.ReplaceUserRejects(entries)
	rejects := session.UserRejects()
	persistUserRejects(rejects)
	return jsonCString(map[string]any{
		"ok":        true,
		"imported":  len(entries),
		"total":     len(rejects),
		"rejects":   rejects,
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaImportUserPinsJSON
func ShurufaImportUserPinsJSON(id C.uint64_t, payload *C.char) *C.char {
	req, err := decodeExtensionCommandPayload(C.GoString(payload))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	entries := req.Entries
	if len(entries) == 0 {
		entries = req.Pins
	}
	session := getSession(uint64(id))
	if req.Merge {
		merged := session.UserPins()
		merged = append(merged, entries...)
		entries = merged
	}
	session.ReplaceUserPins(entries)
	pins := session.UserPins()
	persistUserPins(pins)
	return jsonCString(map[string]any{
		"ok":        true,
		"imported":  len(entries),
		"total":     len(pins),
		"pins":      pins,
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaCommitText
func ShurufaCommitText(id C.uint64_t, reading *C.char, text *C.char) *C.char {
	normalizedReading := normalizeABIReading(C.GoString(reading))
	committedText := strings.TrimSpace(C.GoString(text))
	if normalizedReading == "" || committedText == "" {
		return jsonCString(errorEnvelope("reading and text are required"))
	}
	session := getSession(uint64(id))
	key := normalizedReading + "|" + committedText
	session.ImportUserScores(map[string]int{key: 25})
	persistUserScores(session.UserScores())
	return jsonCString(map[string]any{
		"ok":        true,
		"learned":   key,
		"state":     session.State(),
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaAgentCompose
func ShurufaAgentCompose(input *C.char, context *C.char) *C.char {
	return jsonCString(composeAgentABI(C.GoString(input), C.GoString(context)))
}

func configureActiveSessions(config engine.Config) int {
	sessions := activeSessions()
	for _, session := range sessions {
		session.Configure(config)
	}
	return len(sessions)
}

func activeSessions() []*engine.Engine {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	out := make([]*engine.Engine, 0, len(sessions))
	for _, session := range sessions {
		if session != nil {
			out = append(out, session)
		}
	}
	return out
}
