package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"sync"
	"unsafe"

	"github.com/neko233-com/shurufa233/core/engine"
)

var (
	sessionsMu sync.Mutex
	nextID     uint64 = 1
	sessions          = map[uint64]*engine.Engine{}
)

func main() {}

//export ShurufaCreateSession
func ShurufaCreateSession() C.uint64_t {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	id := nextID
	nextID++
	sessions[id] = engine.New(engine.DefaultConfig())
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

//export ShurufaClear
func ShurufaClear(id C.uint64_t) *C.char {
	session := getSession(uint64(id))
	return jsonCString(session.Clear())
}

//export ShurufaSelect
func ShurufaSelect(id C.uint64_t, index C.int) *C.char {
	session := getSession(uint64(id))
	state, err := session.Select(int(index))
	if err != nil {
		return C.CString(`{"error":"candidate index out of range"}`)
	}
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
		session = engine.New(engine.DefaultConfig())
		sessions[id] = session
	}
	return session
}

func jsonCString(value any) *C.char {
	data, err := json.Marshal(value)
	if err != nil {
		return C.CString(`{"error":"json marshal failed"}`)
	}
	return C.CString(string(data))
}
