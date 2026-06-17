package manager

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/elfonsi/skillbelt/internal/config"
)

// fakeGitDir returns the absolute path to testdata/fakegit so tests can
// prepend it to PATH, ensuring exec.Command("git", ...) hits our stub.
func fakeGitDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// Walk up from internal/manager/ to repo root, then into testdata/fakegit.
	root := filepath.Join(filepath.Dir(file), "..", "..")
	abs, err := filepath.Abs(filepath.Join(root, "testdata", "fakegit"))
	if err != nil {
		t.Fatalf("resolve fakegit dir: %v", err)
	}
	return abs
}

func newTestManager(t *testing.T) (*Manager, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Config{
		SkillsDir: filepath.Join(dir, "skills"),
		ReposDir:  filepath.Join(dir, "repos"),
		LockFile:  filepath.Join(dir, "installed.json"),
	}
	// Inject fake git into PATH.
	t.Setenv("PATH", fakeGitDir(t)+string(os.PathListSeparator)+os.Getenv("PATH"))
	// Each test gets its own log file so calls don't bleed between tests.
	logFile := filepath.Join(dir, "fakegit.log")
	t.Setenv("FAKEGIT_LOG", logFile)
	return New(cfg), logFile
}

func gitLog(t *testing.T, logFile string) []string {
	t.Helper()
	data, err := os.ReadFile(logFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read fakegit log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

// --- Install ---

func TestInstall_Happy(t *testing.T) {
	m, logFile := newTestManager(t)

	if err := m.Install("github.com/user/my-skill"); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Symlink should exist.
	link := filepath.Join(m.cfg.SkillsDir, "my-skill")
	if _, err := os.Lstat(link); err != nil {
		t.Errorf("symlink not created: %v", err)
	}

	// Lock file should record the install.
	lock, err := readLock(m.cfg.LockFile)
	if err != nil {
		t.Fatalf("readLock: %v", err)
	}
	e, ok := lock.Skills["my-skill"]
	if !ok {
		t.Fatal("my-skill not in lock")
	}
	if e.URL != "https://github.com/user/my-skill" {
		t.Errorf("URL = %q, want https://github.com/user/my-skill", e.URL)
	}

	// Fake git should have been called for clone.
	calls := gitLog(t, logFile)
	if len(calls) == 0 {
		t.Error("expected git to be called")
	}
	if !strings.Contains(calls[0], "clone") {
		t.Errorf("expected first git call to be clone, got %q", calls[0])
	}
}

func TestInstall_AlreadyInstalled(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.Install("github.com/user/my-skill"); err != nil {
		t.Fatalf("first install: %v", err)
	}
	err := m.Install("github.com/user/my-skill")
	if err == nil {
		t.Fatal("expected error on second install, got nil")
	}
	if !strings.Contains(err.Error(), "already installed") {
		t.Errorf("error = %q, want 'already installed'", err.Error())
	}
}

func TestInstall_NormalizeURL(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.Install("github.com/u/r"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	lock, _ := readLock(m.cfg.LockFile)
	e := lock.Skills["r"]
	if e.URL != "https://github.com/u/r" {
		t.Errorf("URL = %q, want https://github.com/u/r", e.URL)
	}
}

func TestInstall_StripDotGit(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.Install("github.com/u/r.git"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	lock, _ := readLock(m.cfg.LockFile)
	if _, ok := lock.Skills["r"]; !ok {
		t.Error("expected skill name 'r' (with .git stripped)")
	}
}

func TestInstall_FullHTTPSURL(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.Install("https://github.com/u/skill-x"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	lock, _ := readLock(m.cfg.LockFile)
	if _, ok := lock.Skills["skill-x"]; !ok {
		t.Error("expected skill name 'skill-x'")
	}
}

// --- Remove ---

func TestRemove_Happy(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.Install("github.com/user/my-skill"); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if err := m.Remove("my-skill", false); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Symlink should be gone.
	link := filepath.Join(m.cfg.SkillsDir, "my-skill")
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Errorf("symlink should be removed, got err: %v", err)
	}

	// Clone should still exist.
	repoDir := filepath.Join(m.cfg.ReposDir, "my-skill")
	if _, err := os.Stat(repoDir); err != nil {
		t.Errorf("repo should be kept: %v", err)
	}

	// Lock should not have entry.
	lock, _ := readLock(m.cfg.LockFile)
	if _, ok := lock.Skills["my-skill"]; ok {
		t.Error("my-skill should not be in lock after remove")
	}
}

func TestRemove_Purge(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.Install("github.com/user/my-skill"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if err := m.Remove("my-skill", true); err != nil {
		t.Fatalf("Remove --purge: %v", err)
	}

	repoDir := filepath.Join(m.cfg.ReposDir, "my-skill")
	if _, err := os.Stat(repoDir); !os.IsNotExist(err) {
		t.Errorf("repo should be deleted after purge")
	}
}

func TestRemove_NotInstalled(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.Remove("no-such-skill", false)
	if err == nil {
		t.Fatal("expected error removing non-installed skill")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error = %q, want 'not installed'", err.Error())
	}
}

// --- Update ---

func TestUpdate_OneSkill(t *testing.T) {
	m, logFile := newTestManager(t)

	if err := m.Install("github.com/user/my-skill"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	// Clear the log so we only see the update call.
	os.Remove(logFile)

	if err := m.Update("my-skill"); err != nil {
		t.Fatalf("Update: %v", err)
	}

	calls := gitLog(t, logFile)
	if len(calls) == 0 {
		t.Fatal("expected git pull to be called")
	}
	if !strings.Contains(calls[0], "pull") {
		t.Errorf("expected git pull call, got %q", calls[0])
	}
	repoDir := filepath.Join(m.cfg.ReposDir, "my-skill")
	if !strings.Contains(calls[0], repoDir) {
		t.Errorf("pull call should reference repo dir %q, got %q", repoDir, calls[0])
	}
}

func TestUpdate_All(t *testing.T) {
	m, logFile := newTestManager(t)

	if err := m.Install("github.com/user/skill-a"); err != nil {
		t.Fatalf("install skill-a: %v", err)
	}
	if err := m.Install("github.com/user/skill-b"); err != nil {
		t.Fatalf("install skill-b: %v", err)
	}
	os.Remove(logFile)

	if err := m.Update(""); err != nil {
		t.Fatalf("Update all: %v", err)
	}

	calls := gitLog(t, logFile)
	pullCalls := 0
	for _, c := range calls {
		if strings.Contains(c, "pull") {
			pullCalls++
		}
	}
	if pullCalls != 2 {
		t.Errorf("expected 2 pull calls, got %d (calls: %v)", pullCalls, calls)
	}
}

func TestUpdate_NotInstalled(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.Update("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-installed skill")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error = %q, want 'not installed'", err.Error())
	}
}

func TestUpdate_EmptyNoSkills(t *testing.T) {
	m, _ := newTestManager(t)

	// Should succeed with a friendly message, not an error.
	if err := m.Update(""); err != nil {
		t.Fatalf("Update all with no skills should not error: %v", err)
	}
}

// --- List ---

func TestList_Empty(t *testing.T) {
	m, _ := newTestManager(t)

	entries, err := m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestList_WithEntries(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.Install("github.com/user/skill-a"); err != nil {
		t.Fatalf("install: %v", err)
	}

	entries, err := m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Name != "skill-a" {
		t.Errorf("Name = %q, want skill-a", e.Name)
	}
	if e.URL != "https://github.com/user/skill-a" {
		t.Errorf("URL = %q, want https://github.com/user/skill-a", e.URL)
	}
	if !e.Linked {
		t.Error("Linked should be true for installed skill")
	}
}

func TestList_BrokenSymlink(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.Install("github.com/user/skill-a"); err != nil {
		t.Fatalf("install: %v", err)
	}
	// Manually remove the symlink to simulate a broken state.
	link := filepath.Join(m.cfg.SkillsDir, "skill-a")
	os.Remove(link)

	entries, err := m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Linked {
		t.Error("Linked should be false when symlink is missing")
	}
}

// --- URL normalization (unit tests on the pure function) ---

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		input    string
		wantURL  string
		wantName string
	}{
		{"github.com/u/repo", "https://github.com/u/repo", "repo"},
		{"github.com/u/repo.git", "https://github.com/u/repo.git", "repo"},
		{"https://github.com/u/repo", "https://github.com/u/repo", "repo"},
		{"https://github.com/u/repo.git", "https://github.com/u/repo.git", "repo"},
		{"github.com/u/repo/", "https://github.com/u/repo", "repo"},
		{"git@github.com:u/repo.git", "git@github.com:u/repo.git", "repo"},
	}

	for _, tc := range cases {
		gotURL, gotName := normalizeURL(tc.input)
		if gotURL != tc.wantURL {
			t.Errorf("normalizeURL(%q) url = %q, want %q", tc.input, gotURL, tc.wantURL)
		}
		if gotName != tc.wantName {
			t.Errorf("normalizeURL(%q) name = %q, want %q", tc.input, gotName, tc.wantName)
		}
	}
}

// --- Git not found ---

func TestGitNotFound(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		SkillsDir: filepath.Join(dir, "skills"),
		ReposDir:  filepath.Join(dir, "repos"),
		LockFile:  filepath.Join(dir, "installed.json"),
	}
	// Override PATH to a dir with no git binary.
	t.Setenv("PATH", dir)

	m := New(cfg)
	err := m.Install("github.com/u/skill")
	if err == nil {
		t.Fatal("expected error when git not in PATH")
	}
	if !strings.Contains(err.Error(), "git not found") {
		t.Errorf("error = %q, want 'git not found'", err.Error())
	}
}
