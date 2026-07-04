package main

import (
	"sync"

	"github.com/neko233-com/shurufa233/core/engine"
)

var (
	sessionsMu sync.Mutex
	nextID     uint64 = 1
	sessions          = map[uint64]*engine.Engine{}
)

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
	session := engine.New(loadConfig())
	for _, entry := range loadLocalDictionaryEntries() {
		session.AddEntries(entry)
	}
	session.AddUserPhrases(loadUserPhrases())
	session.AddUserRejects(loadUserRejects())
	session.AddUserPins(loadUserPins())
	session.ImportUserScores(loadUserScores())
	return session
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
