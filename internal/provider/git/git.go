package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/serversfordev/deploy/internal/config"
)

type GitProvider struct {
	config *config.Config
	appDir string
}

func New(config *config.Config, appDir string) *GitProvider {
	return &GitProvider{
		config: config,
		appDir: appDir,
	}
}

func (p *GitProvider) Init() error {
	_, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git executable not found in PATH: %w", err)
	}

	if _, err := execGitCommand(p.sourcePath(), "rev-parse", "--git-dir"); err != nil {
		args := []string{"clone", p.config.Source.Git.Repo, p.sourcePath()}
		if _, err := execGitCommand(p.appDir, args...); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	args := []string{"checkout", p.config.Source.Git.Branch}
	if _, err := execGitCommand(p.sourcePath(), args...); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	args = []string{"pull", "origin", p.config.Source.Git.Branch}
	if _, err := execGitCommand(p.sourcePath(), args...); err != nil {
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}

	return nil
}

func (p *GitProvider) Clone(targetDir string) error {
	// Use checkout-index to copy files into target directory
	args := []string{"checkout-index", "--prefix=" + targetDir + "/", "-a", "-f"}
	_, err := execGitCommand(p.sourcePath(), args...)
	if err != nil {
		return fmt.Errorf("failed to copy repository files: %w", err)
	}

	return nil
}

func (p *GitProvider) GetRevision() (string, error) {
	currentHash, err := execGitCommand(p.sourcePath(), "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	currentHash = strings.TrimSpace(currentHash)

	return currentHash, nil
}

func (p *GitProvider) sourcePath() string {
	return filepath.Join(p.appDir, ".git")
}

func execGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git error: %s. %w", stderr.String(), err)
	}

	return stdout.String(), nil
}
