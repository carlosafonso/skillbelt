package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	SkillsDir string
	ReposDir  string
	LockFile  string
}

func Load() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	skillbeltHome := filepath.Join(home, ".skillbelt")
	if v := os.Getenv("SKILLBELT_HOME"); v != "" {
		skillbeltHome = v
	}

	skillsDir := filepath.Join(home, ".gemini", "config", "skills")
	if v := os.Getenv("SKILLBELT_SKILLS_DIR"); v != "" {
		skillsDir = v
	}

	return Config{
		SkillsDir: skillsDir,
		ReposDir:  filepath.Join(skillbeltHome, "repos"),
		LockFile:  filepath.Join(skillbeltHome, "installed.json"),
	}, nil
}
