package manager

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLock_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "installed.json")

	l := Lock{
		Skills: map[string]Entry{
			"my-skill": {URL: "https://github.com/u/my-skill", InstalledAt: time.Now().UTC().Truncate(time.Second)},
		},
	}

	if err := l.write(path); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := readLock(path)
	if err != nil {
		t.Fatalf("readLock: %v", err)
	}

	if len(got.Skills) != 1 {
		t.Fatalf("want 1 skill, got %d", len(got.Skills))
	}
	e := got.Skills["my-skill"]
	if e.URL != "https://github.com/u/my-skill" {
		t.Errorf("URL = %q, want https://github.com/u/my-skill", e.URL)
	}
	if !e.InstalledAt.Equal(l.Skills["my-skill"].InstalledAt) {
		t.Errorf("InstalledAt mismatch")
	}
}

func TestReadLock_MissingFile(t *testing.T) {
	dir := t.TempDir()
	l, err := readLock(filepath.Join(dir, "nonexistent.json"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if l.Skills == nil {
		t.Error("Skills map should be initialized")
	}
	if len(l.Skills) != 0 {
		t.Errorf("expected empty Skills, got %v", l.Skills)
	}
}

func TestLock_WriteMkdirAll(t *testing.T) {
	dir := t.TempDir()
	// Path with a sub-directory that doesn't exist yet.
	path := filepath.Join(dir, "deep", "nested", "installed.json")

	l := Lock{Skills: map[string]Entry{}}
	if err := l.write(path); err != nil {
		t.Fatalf("write with nested path: %v", err)
	}

	got, err := readLock(path)
	if err != nil {
		t.Fatalf("readLock: %v", err)
	}
	if got.Skills == nil {
		t.Error("Skills map should be initialized after roundtrip")
	}
}
