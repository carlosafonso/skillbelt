package manager

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"

	"github.com/carlosafonso/skillbelt/internal/config"
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

func (m *Manager) Install(rawURL string) (err error) {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git not found in PATH — install git and try again")
	}

	// 1. Process Lock Process Synchronization
	if err := os.MkdirAll(filepath.Dir(m.cfg.LockFile), 0o755); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}
	fileLock := flock.New(m.cfg.LockFile + ".lock")
	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("acquire install lock: %w", err)
	}
	defer func() {
		_ = fileLock.Unlock()
	}()

	pURL := normalizeURL(rawURL)

	// 2. Setup Transactional Rollback Stack (LIFO)
	var rollbacks []func()
	success := false
	defer func() {
		if !success {
			for i := len(rollbacks) - 1; i >= 0; i-- {
				rollbacks[i]()
			}
		}
	}()

	lock, err := readLock(m.cfg.LockFile)
	if err != nil {
		return fmt.Errorf("read lock: %w", err)
	}
	if _, exists := lock.Skills[pURL.name]; exists {
		return fmt.Errorf("%q is already installed — use `skillbelt update %s` to update it", pURL.name, pURL.name)
	}

	repoDir := filepath.Join(m.cfg.ReposDir, pURL.name)
	if err := os.MkdirAll(m.cfg.ReposDir, 0o755); err != nil {
		return fmt.Errorf("create repos dir: %w", err)
	}

	// Git clone/checkout phase
	if pURL.subdir != "" {
		if err := gitCloneSparse(pURL.cloneURL, repoDir, pURL.branch, pURL.subdir); err != nil {
			return fmt.Errorf("clone %s: %w", pURL.cloneURL, err)
		}
	} else {
		if err := gitClone(pURL.cloneURL, repoDir); err != nil {
			return fmt.Errorf("clone %s: %w", pURL.cloneURL, err)
		}
	}
	// Register rollback for git clone
	rollbacks = append(rollbacks, func() {
		_ = os.RemoveAll(repoDir)
	})

	if err := os.MkdirAll(m.cfg.SkillsDir, 0o755); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}
	symlinkTarget := repoDir
	if pURL.subdir != "" {
		symlinkTarget = filepath.Join(repoDir, pURL.subdir)
	}
	link := filepath.Join(m.cfg.SkillsDir, pURL.name)
	if err := os.Symlink(symlinkTarget, link); err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}
	// Register rollback for symlink
	rollbacks = append(rollbacks, func() {
		_ = os.Remove(link)
	})

	lock.Skills[pURL.name] = Entry{URL: pURL.sourceURL, InstalledAt: time.Now().UTC()}
	if err := lock.write(m.cfg.LockFile); err != nil {
		return fmt.Errorf("write lock: %w", err)
	}

	success = true
	fmt.Printf("installed %s (%s)\n", pURL.name, pURL.sourceURL)
	return nil
}

func (m *Manager) Remove(name string, purge bool) error {
	// Process Lock Process Synchronization
	if err := os.MkdirAll(filepath.Dir(m.cfg.LockFile), 0o755); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}
	fileLock := flock.New(m.cfg.LockFile + ".lock")
	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("acquire remove lock: %w", err)
	}
	defer func() {
		_ = fileLock.Unlock()
	}()

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

	// Process Lock Process Synchronization
	if err := os.MkdirAll(filepath.Dir(m.cfg.LockFile), 0o755); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}
	fileLock := flock.New(m.cfg.LockFile + ".lock")
	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("acquire update lock: %w", err)
	}
	defer func() {
		_ = fileLock.Unlock()
	}()

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
	// Readers do not strictly need exclusive locks since lock.write is atomic (via tempfile + rename),
	// meaning os.ReadFile will always get a consistent representation of the JSON data.
	lock, err := readLock(m.cfg.LockFile)
	if err != nil {
		return nil, fmt.Errorf("read lock: %w", err)
	}

	entries := make([]ListEntry, 0, len(lock.Skills))
	for name, e := range lock.Skills {
		link := filepath.Join(m.cfg.SkillsDir, name)
		// Use os.Stat instead of Lstat so that we verify the target directory actually exists.
		// If the symlink exists but points to a deleted directory, Linked will be false.
		_, statErr := os.Stat(link)
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

type parsedURL struct {
	cloneURL  string
	sourceURL string
	name      string
	subdir    string
	branch    string
}

func normalizeURL(raw string) parsedURL {
	raw = strings.TrimRight(raw, "/")

	var withScheme string
	if !strings.HasPrefix(raw, "https://") && !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "git@") {
		withScheme = "https://" + raw
	} else {
		withScheme = raw
	}

	// Detect GitHub subdirectory URL: https://github.com/{owner}/{repo}/tree/{branch}/{path...}
	if strings.HasPrefix(withScheme, "https://github.com/") || strings.HasPrefix(withScheme, "http://github.com/") {
		schemeless := withScheme[strings.Index(withScheme, "://")+3:]
		parts := strings.Split(schemeless, "/")
		// parts: ["github.com", owner, repo, "tree", branch, subdir...]
		if len(parts) >= 6 && parts[3] == "tree" {
			subdir := strings.Join(parts[5:], "/")
			return parsedURL{
				cloneURL:  "https://github.com/" + parts[1] + "/" + parts[2],
				sourceURL: withScheme,
				name:      parts[len(parts)-1],
				subdir:    subdir,
				branch:    parts[4],
			}
		}
	}

	// Non-subdir URL: derive name from last path segment, stripping .git suffix.
	parts := strings.Split(raw, "/")
	name := strings.TrimSuffix(parts[len(parts)-1], ".git")
	return parsedURL{
		cloneURL:  withScheme,
		sourceURL: withScheme,
		name:      name,
	}
}

func gitClone(url, dest string) error {
	cmd := exec.Command("git", "clone", "--depth=1", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitCloneSparse(repoURL, dest, branch, subdir string) error {
	args := []string{"clone", "--depth=1", "--sparse"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, repoURL, dest)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("git", "-C", dest, "sparse-checkout", "set", subdir)
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
