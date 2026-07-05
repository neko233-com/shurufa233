package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/neko233-com/shurufa233/core/engine"
)

func main() {}

//export ShurufaCreateSession
func ShurufaCreateSession() C.uint64_t {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	id := nextID
	nextID++
	sessions[id] = newEngine()
	return C.uint64_t(id)
}

//export ShurufaDestroySession
func ShurufaDestroySession(id C.uint64_t) {
	sessionsMu.Lock()
	session := sessions[uint64(id)]
	delete(sessions, uint64(id))
	sessionsMu.Unlock()
	if session != nil {
		persistUserScoresSync(session.UserScores())
		persistUserRejects(session.UserRejects())
		persistUserPins(session.UserPins())
	}
}

//export ShurufaInputKey
func ShurufaInputKey(id C.uint64_t, key C.char) *C.char {
	session := getSession(uint64(id))
	state := session.InputKey(rune(byte(key)))
	return jsonCString(state)
}

//export ShurufaInputKeyFast
func ShurufaInputKeyFast(id C.uint64_t, key C.char) C.int {
	session := getSession(uint64(id))
	state := session.InputKey(rune(byte(key)))
	return C.int(len(state.Candidates))
}

//export ShurufaPreview
func ShurufaPreview(id C.uint64_t, input *C.char) *C.char {
	session := getSession(uint64(id))
	state := session.Preview(C.GoString(input))
	return jsonCString(state)
}

//export ShurufaBackspace
func ShurufaBackspace(id C.uint64_t) *C.char {
	session := getSession(uint64(id))
	return jsonCString(session.Backspace())
}

//export ShurufaBackspaceFast
func ShurufaBackspaceFast(id C.uint64_t) C.int {
	session := getSession(uint64(id))
	state := session.Backspace()
	return C.int(len(state.Candidates))
}

//export ShurufaClear
func ShurufaClear(id C.uint64_t) *C.char {
	session := getSession(uint64(id))
	return jsonCString(session.Clear())
}

//export ShurufaSetMode
func ShurufaSetMode(id C.uint64_t, mode *C.char) *C.char {
	session := getSession(uint64(id))
	return jsonCString(session.SetMode(C.GoString(mode)))
}

//export ShurufaToggleMode
func ShurufaToggleMode(id C.uint64_t) *C.char {
	session := getSession(uint64(id))
	return jsonCString(session.ToggleMode())
}

//export ShurufaMode
func ShurufaMode(id C.uint64_t) *C.char {
	session := getSession(uint64(id))
	return C.CString(session.State().Mode)
}

//export ShurufaCandidateCount
func ShurufaCandidateCount(id C.uint64_t) C.int {
	session := getSession(uint64(id))
	return C.int(len(session.State().Candidates))
}

//export ShurufaCandidateText
func ShurufaCandidateText(id C.uint64_t, index C.int) *C.char {
	session := getSession(uint64(id))
	candidates := session.State().Candidates
	i := int(index)
	if i < 0 || i >= len(candidates) {
		return C.CString("")
	}
	return C.CString(candidates[i].Text)
}

//export ShurufaCandidateReading
func ShurufaCandidateReading(id C.uint64_t, index C.int) *C.char {
	session := getSession(uint64(id))
	candidates := session.State().Candidates
	i := int(index)
	if i < 0 || i >= len(candidates) {
		return C.CString("")
	}
	return C.CString(candidates[i].Reading)
}

//export ShurufaCandidateScore
func ShurufaCandidateScore(id C.uint64_t, index C.int) C.int {
	session := getSession(uint64(id))
	candidates := session.State().Candidates
	i := int(index)
	if i < 0 || i >= len(candidates) {
		return 0
	}
	return C.int(candidates[i].Weight + candidates[i].UserScore)
}

//export ShurufaCandidatePayload
func ShurufaCandidatePayload(id C.uint64_t, limit C.int) *C.char {
	return ShurufaCandidatePayloadRange(id, 0, limit)
}

//export ShurufaCandidatePayloadRange
func ShurufaCandidatePayloadRange(id C.uint64_t, start C.int, limit C.int) *C.char {
	session := getSession(uint64(id))
	candidates := session.State().Candidates
	startIndex := int(start)
	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex > len(candidates) {
		startIndex = len(candidates)
	}
	maxItems := int(limit)
	if maxItems <= 0 || maxItems > 9 {
		maxItems = 9
	}
	if remaining := len(candidates) - startIndex; remaining < maxItems {
		maxItems = remaining
	}
	var out strings.Builder
	for i := 0; i < maxItems; i++ {
		if i > 0 {
			out.WriteByte('\n')
		}
		candidate := candidates[startIndex+i]
		out.WriteString(strconv.Itoa(i + 1))
		out.WriteByte('\t')
		out.WriteString(sanitizePayloadField(candidate.Text))
		out.WriteByte('\t')
		out.WriteString(sanitizePayloadField(candidate.Reading))
		out.WriteByte('\t')
		out.WriteString(strconv.Itoa(candidateScore(candidate)))
		out.WriteByte('\t')
		out.WriteString(sanitizePayloadField(candidate.Kind))
		out.WriteByte('\t')
		out.WriteString(sanitizePayloadField(candidate.Source))
		out.WriteByte('\t')
		out.WriteString(sanitizePayloadField(candidate.Comment))
		out.WriteByte('\t')
		out.WriteString(strconv.FormatBool(candidate.Pinned))
	}
	return C.CString(out.String())
}

//export ShurufaCommitCandidate
func ShurufaCommitCandidate(id C.uint64_t, index C.int) *C.char {
	session := getSession(uint64(id))
	state, err := session.Select(int(index))
	if err != nil {
		return C.CString("")
	}
	persistUserScores(session.UserScores())
	return C.CString(state.Committed)
}

//export ShurufaCommitCandidateChar
func ShurufaCommitCandidateChar(id C.uint64_t, index C.int, side *C.char) *C.char {
	committed, err := commitCandidateChar(uint64(id), int(index), C.GoString(side))
	if err != nil {
		return C.CString("")
	}
	return C.CString(committed)
}

//export ShurufaSelect
func ShurufaSelect(id C.uint64_t, index C.int) *C.char {
	session := getSession(uint64(id))
	state, err := session.Select(int(index))
	if err != nil {
		return C.CString(`{"error":"candidate index out of range"}`)
	}
	persistUserScores(session.UserScores())
	return jsonCString(state)
}

//export ShurufaSelectCandidateChar
func ShurufaSelectCandidateChar(id C.uint64_t, index C.int, side *C.char) *C.char {
	session := getSession(uint64(id))
	state, err := session.SelectChar(int(index), C.GoString(side))
	if err != nil {
		return C.CString(`{"error":"candidate char selection failed"}`)
	}
	return jsonCString(state)
}

//export ShurufaRejectCandidate
func ShurufaRejectCandidate(id C.uint64_t, index C.int) *C.char {
	session := getSession(uint64(id))
	state, rejected, err := session.RejectCandidate(int(index))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	persistUserScoresReplaceSync(session.UserScores())
	persistUserRejects(session.UserRejects())
	return jsonCString(map[string]any{
		"ok":        true,
		"rejected":  rejected,
		"state":     state,
		"updatedAt": state.UpdatedAt,
	})
}

//export ShurufaPinCandidate
func ShurufaPinCandidate(id C.uint64_t, index C.int) *C.char {
	session := getSession(uint64(id))
	state, pinned, err := session.PinCandidate(int(index))
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	persistUserPins(session.UserPins())
	return jsonCString(map[string]any{
		"ok":        true,
		"pinned":    pinned,
		"state":     state,
		"updatedAt": state.UpdatedAt,
	})
}

//export ShurufaExecuteCommand
func ShurufaExecuteCommand(id C.uint64_t, command *C.char, payload *C.char) *C.char {
	return jsonCString(executeExtensionCommand(uint64(id), C.GoString(command), C.GoString(payload)))
}

//export ShurufaFree
func ShurufaFree(value *C.char) {
	C.free(unsafe.Pointer(value))
}

func executeExtensionCommand(id uint64, command string, payload string) any {
	command = strings.ToLower(strings.TrimSpace(command))
	req, err := decodeExtensionCommandPayload(payload)
	if err != nil {
		return errorEnvelope(err.Error())
	}
	session := getSession(id)
	if result, ok := executeSessionExtensionCommand(session, command, payload); ok {
		return result
	}
	switch command {
	case "config-json", "config":
		return configEnvelope()
	case "apply-config-json", "apply-config":
		config, err := configFromCommandPayload(req, payload)
		if err != nil {
			return errorEnvelope(err.Error())
		}
		return applyConfigEnvelope(config)
	case "schema-presets-json", "schemas", "schemas-json":
		config := loadConfig()
		return map[string]any{
			"ok":        true,
			"selected":  config.Schema,
			"schemas":   engine.BuiltinSchemaPresets(),
			"config":    config,
			"updatedAt": time.Now().UTC(),
		}
	case "apply-schema-json", "apply-schema":
		config := loadConfig()
		next, ok := engine.ApplySchemaPresetConfig(config, firstNonEmpty(req.ID, req.Schema, req.Input, req.Text))
		if !ok {
			return errorEnvelope("unknown schema id")
		}
		return applyConfigEnvelope(normalizeConfig(next))
	case "skin-presets-json", "skins", "skins-json":
		config := loadConfig()
		return map[string]any{
			"ok":        true,
			"selected":  config.Skin.Theme,
			"presets":   engine.BuiltinSkinPresets(),
			"config":    config,
			"updatedAt": time.Now().UTC(),
		}
	case "apply-skin-preset-json", "apply-skin-preset", "skin-preset":
		config := loadConfig()
		next, ok := engine.ApplySkinPresetConfig(config, firstNonEmpty(req.ID, req.Preset, req.Input, req.Text))
		if !ok {
			return errorEnvelope("unknown skin preset id")
		}
		return applyConfigEnvelope(normalizeConfig(next))
	case "reload-config":
		return applyConfigEnvelope(loadConfig())
	case "reload-dictionaries":
		return reloadDictionariesEnvelope()
	case "dictionary-manifest-json", "dictionary-manifest":
		return dictionaryManifestEnvelope()
	default:
		return errorEnvelope("unknown extension command: " + command)
	}
}

func configFromCommandPayload(req extensionCommandPayload, payload string) (engine.Config, error) {
	if req.Config != nil {
		return normalizeConfig(*req.Config), nil
	}
	config := engine.Config{}
	if err := json.Unmarshal([]byte(payload), &config); err != nil {
		return config, err
	}
	return normalizeConfig(config), nil
}

func commitCandidateChar(id uint64, index int, side string) (string, error) {
	session := getSession(id)
	state, err := session.SelectChar(index, side)
	if err != nil {
		return "", err
	}
	return state.Committed, nil
}

func jsonCString(value any) *C.char {
	data, err := json.Marshal(value)
	if err != nil {
		return C.CString(`{"error":"json marshal failed"}`)
	}
	return C.CString(string(data))
}

func sanitizePayloadField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
