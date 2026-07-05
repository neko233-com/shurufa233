package engine

import "strings"

func NormalizeSync(sync Sync) Sync {
	defaults := defaultSyncConfig()
	if strings.TrimSpace(sync.Provider) == "" {
		sync.Provider = defaults.Provider
	}
	sync.Provider = strings.ToLower(strings.TrimSpace(sync.Provider))
	switch sync.Provider {
	case "local", "directory":
		sync.Provider = defaults.Provider
	case "local-directory", "github", "webdav", "none":
	default:
		sync.Provider = defaults.Provider
	}
	sync.Directory = strings.TrimSpace(sync.Directory)
	sync.RemoteURL = strings.TrimSpace(sync.RemoteURL)
	sync.MirrorBaseURLs = normalizeSyncStringList(sync.MirrorBaseURLs)
	switch strings.ToLower(strings.TrimSpace(sync.ConflictPolicy)) {
	case "replace-local", "local-wins", "remote-wins":
		sync.ConflictPolicy = strings.ToLower(strings.TrimSpace(sync.ConflictPolicy))
	case "", "merge-newer", "merge":
		sync.ConflictPolicy = defaults.ConflictPolicy
	default:
		sync.ConflictPolicy = defaults.ConflictPolicy
	}
	return sync
}

func defaultSyncConfig() Sync {
	return Sync{
		Enabled:        false,
		Provider:       "local-directory",
		MirrorBaseURLs: []string{},
		AutoExport:     false,
		AutoImport:     false,
		ConflictPolicy: "merge-newer",
	}
}

func normalizeSyncStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
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
