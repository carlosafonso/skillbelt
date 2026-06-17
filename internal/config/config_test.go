package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("SKILLBELT_HOME", "")
	t.Setenv("SKILLBELT_SKILLS_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	home, _ := os.UserHomeDir()

	if !strings.HasPrefix(cfg.SkillsDir, home) {
		t.Errorf("SkillsDir %q should be under home dir", cfg.SkillsDir)
	}
	if !strings.Contains(cfg.SkillsDir, ".agents") {
		t.Errorf("SkillsDir %q should contain .agents", cfg.SkillsDir)
	}
	if !strings.HasSuffix(cfg.SkillsDir, filepath.Join(".agents", "skills")) {
		t.Errorf("SkillsDir %q should end with .agents/skills", cfg.SkillsDir)
	}
	if !strings.Contains(cfg.ReposDir, ".skillbelt") {
		t.Errorf("ReposDir %q should contain .skillbelt", cfg.ReposDir)
	}
	if !strings.HasSuffix(cfg.LockFile, "installed.json") {
		t.Errorf("LockFile %q should end with installed.json", cfg.LockFile)
	}
}

func TestLoad_SkillsDirOverride(t *testing.T) {
	t.Setenv("SKILLBELT_SKILLS_DIR", "/tmp/custom-skills")
	t.Setenv("SKILLBELT_HOME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.SkillsDir != "/tmp/custom-skills" {
		t.Errorf("SkillsDir = %q, want /tmp/custom-skills", cfg.SkillsDir)
	}
}

func TestLoad_HomeOverride(t *testing.T) {
	t.Setenv("SKILLBELT_HOME", "/tmp/custom-skillbelt")
	t.Setenv("SKILLBELT_SKILLS_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.ReposDir != "/tmp/custom-skillbelt/repos" {
		t.Errorf("ReposDir = %q, want /tmp/custom-skillbelt/repos", cfg.ReposDir)
	}
	if cfg.LockFile != "/tmp/custom-skillbelt/installed.json" {
		t.Errorf("LockFile = %q, want /tmp/custom-skillbelt/installed.json", cfg.LockFile)
	}
}
