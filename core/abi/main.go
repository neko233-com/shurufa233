package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/neko233-com/shurufa233/core/engine"
)

var (
	sessionsMu sync.Mutex
	nextID     uint64 = 1
	sessions          = map[uint64]*engine.Engine{}
	scoresMu   sync.Mutex
)

type userScoreStore struct {
	Version   int            `json:"version"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Scores    map[string]int `json:"scores"`
}

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
	defer sessionsMu.Unlock()
	delete(sessions, uint64(id))
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
		out.WriteString(strconv.Itoa(candidate.Weight + candidate.UserScore))
		out.WriteByte('\t')
		out.WriteString(sanitizePayloadField(candidate.Kind))
		out.WriteByte('\t')
		out.WriteString(sanitizePayloadField(candidate.Source))
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

//export ShurufaFree
func ShurufaFree(value *C.char) {
	C.free(unsafe.Pointer(value))
}

func getSession(id uint64) *engine.Engine {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	session := sessions[id]
	if session == nil {
		session = newEngine()
		sessions[id] = session
	}
	return session
}

func newEngine() *engine.Engine {
	session := engine.New(engine.DefaultConfig())
	for _, entry := range loadLocalDictionaryEntries() {
		session.AddEntries(entry)
	}
	session.ImportUserScores(loadUserScores())
	return session
}

func userScoresPath() (string, error) {
	if override := os.Getenv("SHURUFA233_USER_SCORES"); override != "" {
		return override, nil
	}
	base := os.Getenv("APPDATA")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "shurufa233", "user-scores.json"), nil
}

func loadUserScores() map[string]int {
	scoresMu.Lock()
	defer scoresMu.Unlock()
	path, err := userScoresPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var store userScoreStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil
	}
	return store.Scores
}

func persistUserScores(scores map[string]int) {
	if len(scores) == 0 {
		return
	}
	scoresMu.Lock()
	defer scoresMu.Unlock()
	path, err := userScoresPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	merged := make(map[string]int, len(scores))
	for key, value := range scores {
		merged[key] = value
	}
	if data, err := os.ReadFile(path); err == nil {
		var existing userScoreStore
		if json.Unmarshal(data, &existing) == nil {
			for key, value := range existing.Scores {
				if value > merged[key] {
					merged[key] = value
				}
			}
		}
	}
	store := userScoreStore{
		Version:   1,
		UpdatedAt: time.Now().UTC(),
		Scores:    merged,
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			_ = os.Remove(path)
			_ = os.Rename(tmp, path)
		} else {
			_ = os.Remove(tmp)
		}
	}
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
