package manager

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/elfonsi/skillbelt/internal/config"
)

type Manager struct {
	cfg config.Config
}

func New(cfg config.Config) *Manager {
	return &Manager{cfg: cfg}
}

type ListEntry struct {
	Name        string
	URL         string
	InstalledAt time.Time
	Linked      bool
}

func (m *Manager) Install(rawURL string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git not found in PATH — install git and try again")
	}

	url, name := normalizeURL(rawURL)

	lock, err := readLock(m.cfg.LockFile)
	if err != nil {
		return fmt.Errorf("read lock: %w", err)
	}
	if _, exists := lock.Skills[name]; exists {
		return fmt.Errorf("%q is already installed — use `skillbelt update %s` to update it", name, name)
	}

	repoDir := filepath.Join(m.cfg.ReposDir, name)
	if err := os.MkdirAll(m.cfg.ReposDir, 0o755); err != nil {
		return fmt.Errorf("create repos dir: %w", err)
	}
	if err := gitClone(url, repoDir); err != nil {
		return fmt.Errorf("clone %s: %w", url, err)
	}

	if err := os.MkdirAll(m.cfg.SkillsDir, 0o755); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}
	link := filepath.Join(m.cfg.SkillsDir, name)
	if err := os.Symlink(repoDir, link); err != nil {
		// Roll back the clone so state stays consistent.
		_ = os.RemoveAll(repoDir)
		return fmt.Errorf("create symlink: %w", err)
	}

	lock.Skills[name] = Entry{URL: url, InstalledAt: time.Now().UTC()}
	if err := lock.write(m.cfg.LockFile); err != nil {
		return fmt.Errorf("write lock: %w", err)
	}

	fmt.Printf("installed %s (%s)\n", name, url)
	return nil
}

func (m *Manager) Remove(name string, purge bool) error {
	lock, err := readLock(m.cfg.LockFile)
	if err != nil {
		return fmt.Errorf("read lock: %w", err)
	}
	if _, exists := lock.Skills[name]; !exists {
		return fmt.Errorf("%q is not installed", name)
	}

	link := filepath.Join(m.cfg.SkillsDir, name)
	if err := os.Remove(link); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove symlink: %w", err)
	}

	if purge {
		repoDir := filepath.Join(m.cfg.ReposDir, name)
		if err := os.RemoveAll(repoDir); err != nil {
			return fmt.Errorf("remove repo: %w", err)
		}
	}

	delete(lock.Skills, name)
	if err := lock.write(m.cfg.LockFile); err != nil {
		return fmt.Errorf("write lock: %w", err)
	}

	if purge {
		fmt.Printf("removed %s (symlink and repo deleted)\n", name)
	} else {
		fmt.Printf("removed %s (symlink deleted; repo kept at %s)\n", name, filepath.Join(m.cfg.ReposDir, name))
	}
	return nil
}

func (m *Manager) Update(name string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git not found in PATH — install git and try again")
	}

	lock, err := readLock(m.cfg.LockFile)
	if err != nil {
		return fmt.Errorf("read lock: %w", err)
	}

	if name != "" {
		if _, exists := lock.Skills[name]; !exists {
			return fmt.Errorf("%q is not installed", name)
		}
		return m.pullOne(name)
	}

	if len(lock.Skills) == 0 {
		fmt.Println("no skills installed")
		return nil
	}
	var errs []string
	for n := range lock.Skills {
		if err := m.pullOne(n); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", n, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("update errors:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

func (m *Manager) List() ([]ListEntry, error) {
	lock, err := readLock(m.cfg.LockFile)
	if err != nil {
		return nil, fmt.Errorf("read lock: %w", err)
	}

	entries := make([]ListEntry, 0, len(lock.Skills))
	for name, e := range lock.Skills {
		link := filepath.Join(m.cfg.SkillsDir, name)
		_, statErr := os.Lstat(link)
		entries = append(entries, ListEntry{
			Name:        name,
			URL:         e.URL,
			InstalledAt: e.InstalledAt,
			Linked:      statErr == nil,
		})
	}
	return entries, nil
}

func (m *Manager) pullOne(name string) error {
	repoDir := filepath.Join(m.cfg.ReposDir, name)
	if err := gitPull(repoDir); err != nil {
		return err
	}
	fmt.Printf("updated %s\n", name)
	return nil
}

func normalizeURL(raw string) (url, name string) {
	// Strip trailing slash.
	raw = strings.TrimRight(raw, "/")

	if !strings.HasPrefix(raw, "https://") && !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "git@") {
		url = "https://" + raw
	} else {
		url = raw
	}

	// Derive name from last path segment, stripping .git suffix.
	parts := strings.Split(strings.TrimRight(raw, "/"), "/")
	name = strings.TrimSuffix(parts[len(parts)-1], ".git")
	return url, name
}

func gitClone(url, dest string) error {
	cmd := exec.Command("git", "clone", "--depth=1", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitPull(repoDir string) error {
	cmd := exec.Command("git", "-C", repoDir, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
