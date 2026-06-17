package manager

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Lock struct {
	Skills map[string]Entry `json:"skills"`
}

type Entry struct {
	URL         string    `json:"url"`
	InstalledAt time.Time `json:"installed_at"`
}

func readLock(path string) (Lock, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Lock{Skills: map[string]Entry{}}, nil
	}
	if err != nil {
		return Lock{}, err
	}
	var l Lock
	if err := json.Unmarshal(data, &l); err != nil {
		return Lock{}, err
	}
	if l.Skills == nil {
		l.Skills = map[string]Entry{}
	}
	return l, nil
}

func (l *Lock) write(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	// Atomic write: write to temp file then rename.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
