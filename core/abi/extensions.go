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

//export ShurufaConfigJSON
func ShurufaConfigJSON() *C.char {
	return jsonCString(map[string]any{
		"ok":        true,
		"config":    loadConfig(),
		"updatedAt": time.Now().UTC(),
	})
}

//export ShurufaApplyConfigJSON
func ShurufaApplyConfigJSON(payload *C.char) *C.char {
	var config engine.Config
	if err := json.Unmarshal([]byte(C.GoString(payload)), &config); err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	config = normalizeConfig(config)
	updated := configureActiveSessions(config)
	return jsonCString(map[string]any{
		"ok":              true,
		"config":          config,
		"sessionsUpdated": updated,
		"updatedAt":       time.Now().UTC(),
	})
}

//export ShurufaReloadConfig
func ShurufaReloadConfig() *C.char {
	config := loadConfig()
	updated := configureActiveSessions(config)
	return jsonCString(map[string]any{
		"ok":              true,
		"config":          config,
		"sessionsUpdated": updated,
		"updatedAt":       time.Now().UTC(),
	})
}

//export ShurufaReloadDictionaries
func ShurufaReloadDictionaries() *C.char {
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
	return jsonCString(map[string]any{
		"ok":               true,
		"dictionaryGroups": len(groups),
		"entries":          entryCount,
		"sessionsUpdated":  len(sessions),
		"updatedAt":        time.Now().UTC(),
	})
}

//export ShurufaDictionaryManifestJSON
func ShurufaDictionaryManifestJSON() *C.char {
	dir, err := dictionaryDir()
	if err != nil {
		return jsonCString(errorEnvelope(err.Error()))
	}
	for _, name := range []string{"manifest.json", "dictionary-manifest.json"} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var manifest any
		if err := json.Unmarshal(data, &manifest); err != nil {
			return jsonCString(errorEnvelope(err.Error()))
		}
		return jsonCString(map[string]any{
			"ok":        true,
			"path":      path,
			"manifest":  manifest,
			"updatedAt": time.Now().UTC(),
		})
	}
	return jsonCString(map[string]any{
		"ok":        true,
		"path":      "",
		"manifest":  nil,
		"updatedAt": time.Now().UTC(),
	})
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
