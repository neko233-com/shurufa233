package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	scoresMu     sync.Mutex
	persistOnce  sync.Once
	persistQueue chan map[string]int
)

type userScoreStore struct {
	Version   int            `json:"version"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Scores    map[string]int `json:"scores"`
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
	copied := copyScores(scores)
	persistOnce.Do(startUserScorePersistWorker)
	select {
	case persistQueue <- copied:
	default:
		go persistUserScoresSync(copied)
	}
}

func startUserScorePersistWorker() {
	persistQueue = make(chan map[string]int, 32)
	go func() {
		var pending map[string]int
		var timer *time.Timer
		var timerC <-chan time.Time
		for {
			select {
			case scores := <-persistQueue:
				pending = mergeScoreMaps(pending, scores)
				if timer == nil {
					timer = time.NewTimer(150 * time.Millisecond)
				} else {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(150 * time.Millisecond)
				}
				timerC = timer.C
			case <-timerC:
				persistUserScoresSync(pending)
				pending = nil
				timer = nil
				timerC = nil
			}
		}
	}()
}

func copyScores(scores map[string]int) map[string]int {
	copied := make(map[string]int, len(scores))
	for key, value := range scores {
		copied[key] = value
	}
	return copied
}

func mergeScoreMaps(left map[string]int, right map[string]int) map[string]int {
	if len(left) == 0 {
		return copyScores(right)
	}
	for key, value := range right {
		if value > left[key] {
			left[key] = value
		}
	}
	return left
}

func persistUserScoresSync(scores map[string]int) {
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
	merged := copyScores(scores)
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
