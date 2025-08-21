package config

import (
	"os"
	"path/filepath"
)

// FindEnvTest searches for the nearest file
// If filename is empty, it searches for .env
func FindEnvTest(filename string) (string, error) {
	if filename == "" {
		filename = ".env"
	}
	startDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	curr := startDir
	for {
		candidate := filepath.Join(curr, filename)
		if _, err = os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return "", os.ErrNotExist
}
