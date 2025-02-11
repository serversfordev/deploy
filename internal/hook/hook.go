package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Hook string

const (
	HookClone      Hook = "clone"
	HookBuild      Hook = "build"
	HookDeploy     Hook = "deploy"
	HookPostDeploy Hook = "post_deploy"
	HookVerify     Hook = "verify"
)

func ExecuteHook(releaseDir string, hook Hook) error {
	hookPath := filepath.Join(releaseDir, ".deploy", "hooks", string(hook))

	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command(hookPath)
	cmd.Dir = releaseDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute hook %s: %w", hook, err)
	}

	return nil
}
