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
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	home, _ := os.UserHomeDir()
	wantSkillbeltHome := filepath.Join(home, ".config", "skillbelt")

	if !strings.HasPrefix(cfg.SkillsDir, home) {
		t.Errorf("SkillsDir %q should be under home dir", cfg.SkillsDir)
	}
	if !strings.HasSuffix(cfg.SkillsDir, filepath.Join(".gemini", "config", "skills")) {
		t.Errorf("SkillsDir %q should end with .gemini/config/skills", cfg.SkillsDir)
	}
	if !strings.HasPrefix(cfg.ReposDir, wantSkillbeltHome) {
		t.Errorf("ReposDir %q should be under %s", cfg.ReposDir, wantSkillbeltHome)
	}
	if !strings.HasSuffix(cfg.LockFile, "installed.json") {
		t.Errorf("LockFile %q should end with installed.json", cfg.LockFile)
	}
}

func TestLoad_XDGConfigHomeOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/custom-config")
	t.Setenv("SKILLBELT_HOME", "")
	t.Setenv("SKILLBELT_SKILLS_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	wantSkillbeltHome := "/tmp/custom-config/skillbelt"
	if cfg.ReposDir != wantSkillbeltHome+"/repos" {
		t.Errorf("ReposDir = %q, want %s/repos", cfg.ReposDir, wantSkillbeltHome)
	}
	if cfg.LockFile != wantSkillbeltHome+"/installed.json" {
		t.Errorf("LockFile = %q, want %s/installed.json", cfg.LockFile, wantSkillbeltHome)
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
