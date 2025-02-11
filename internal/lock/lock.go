package lock

import (
	"fmt"
	"os"
	"path/filepath"
)

const lockFileName = "deploy.lock"

// Acquire creates a lock file to prevent concurrent deployments.
// Returns an error if the lock file already exists or cannot be created.
func Acquire(appDir string) error {
	lockFile := filepath.Join(appDir, lockFileName)
	if _, err := os.Stat(lockFile); err == nil {
		return fmt.Errorf("deployment already in progress, lock file exists")
	}
	return os.WriteFile(lockFile, []byte{}, 0644)
}

// Release removes the lock file.
// Returns an error if the lock file cannot be removed.
func Release(appDir string) error {
	return os.Remove(filepath.Join(appDir, lockFileName))
}
