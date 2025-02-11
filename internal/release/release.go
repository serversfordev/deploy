package release

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ErrNoCurrentRelease indicates that there is no current release symlink
var ErrNoCurrentRelease = fmt.Errorf("no current release exists")

func UpdateCurrent(appDir string, releaseDir string) error {
	currentSymlink := filepath.Join(appDir, "current")
	previousSymlink := filepath.Join(appDir, "previous")

	// Set previous symlink if current exists
	if currentSymlinkTarget, err := os.Readlink(currentSymlink); err == nil {
		os.Remove(previousSymlink)
		os.Symlink(currentSymlinkTarget, previousSymlink)
	}

	tmpLink := currentSymlink + ".tmp"
	if err := os.Symlink(releaseDir, tmpLink); err != nil {
		return fmt.Errorf("failed to create temporary symlink: %w", err)
	}
	if err := os.Rename(tmpLink, currentSymlink); err != nil {
		os.Remove(tmpLink)
		return fmt.Errorf("failed to update current symlink: %w", err)
	}

	return nil
}

func Rollback(appDir string) error {
	previousSymlink := filepath.Join(appDir, "previous")

	previousTarget, err := os.Readlink(previousSymlink)
	if err != nil {
		return fmt.Errorf("failed to read previous symlink: %w", err)
	}

	if err := UpdateCurrent(appDir, previousTarget); err != nil {
		return fmt.Errorf("failed to rollback to previous release: %w", err)
	}

	return nil
}

func NewRelease(appDir string, revisionID string) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	releaseDir := filepath.Join(appDir, "releases", timestamp)
	if err := os.MkdirAll(releaseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create release directory: %w", err)
	}

	revisionFile := filepath.Join(releaseDir, "REVISION")
	if err := os.WriteFile(revisionFile, []byte(revisionID), 0644); err != nil {
		return "", fmt.Errorf("failed to write revision file: %w", err)
	}

	return releaseDir, nil
}

func CurrentRevision(appDir string) (string, error) {
	currentSymlink := filepath.Join(appDir, "current")
	if _, err := os.Readlink(currentSymlink); err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoCurrentRelease
		}
		return "", fmt.Errorf("failed to read current symlink: %w", err)
	}

	revisionFile := filepath.Join(appDir, "current", "REVISION")
	revision, err := os.ReadFile(revisionFile)
	if err != nil {
		return "", fmt.Errorf("failed to read revision file: %w", err)
	}

	return string(revision), nil
}

func CleanupRelease(releaseDir string) error {
	return os.RemoveAll(releaseDir)
}

func CleanupOldReleases(appDir string, keep int) error {
	releasesDir := filepath.Join(appDir, "releases")
	releases, err := os.ReadDir(releasesDir)
	if err != nil {
		return err
	}

	type releaseDirInfo struct {
		name    string
		created time.Time
	}

	var releaseDirInfos []releaseDirInfo
	for _, releaseDir := range releases {
		if releaseDir.IsDir() {
			path := filepath.Join(releasesDir, releaseDir.Name())

			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("failed to get file info for %s: %w", path, err)
			}

			releaseDirInfos = append(releaseDirInfos, releaseDirInfo{
				name:    releaseDir.Name(),
				created: info.ModTime(),
			})
		}
	}

	sort.Slice(releaseDirInfos, func(i, j int) bool {
		return releaseDirInfos[i].created.Before(releaseDirInfos[j].created)
	})

	if len(releaseDirInfos) <= keep {
		return nil
	}

	for _, releaseDirInfo := range releaseDirInfos[:len(releaseDirInfos)-keep] {
		if err := os.RemoveAll(filepath.Join(releasesDir, releaseDirInfo.name)); err != nil {
			return fmt.Errorf("failed to remove release %s: %w", releaseDirInfo.name, err)
		}
	}

	return nil
}
