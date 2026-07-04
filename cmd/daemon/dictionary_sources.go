package main

import "github.com/neko233-com/shurufa233/core/engine"

type dictionarySourcePreset = engine.DictionarySourcePreset
type dictionaryRawFile = engine.DictionaryRawSource

type dictionarySourceResponse struct {
	Sources   []dictionarySourcePreset `json:"sources"`
	Selected  string                   `json:"selected"`
	UpdatedAt string                   `json:"updatedAt"`
}

type dictionarySourceRequest struct {
	ID             string   `json:"id"`
	ManifestURLs   []string `json:"manifestUrls,omitempty"`
	MirrorBaseURLs []string `json:"mirrorBaseUrls,omitempty"`
}
