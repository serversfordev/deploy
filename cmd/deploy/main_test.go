package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/serversfordev/deploy/internal/config"
)

var (
	workingDir string
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deploy test suite")
}

var _ = BeforeSuite(func() {
	var err error

	workingDir, err = os.Getwd()
	Expect(err).NotTo(HaveOccurred())

	workingDir = filepath.Join(workingDir, "testdata")
	Expect(err).NotTo(HaveOccurred())

	err = os.RemoveAll(workingDir)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Deploy", func() {
	Context("init command", func() {
		It("should create a new application deployment structure", func() {
			// create a new test environment
			env, err := NewTestEnv(workingDir, "init-test-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(env.Dir).To(BeADirectory())

			// change working directory to the test environment
			err = os.Chdir(env.Dir)
			Expect(err).NotTo(HaveOccurred())

			// run the init command
			err = app.Run([]string{"deploy", "init", "-n", "app"})
			Expect(err).NotTo(HaveOccurred())

			// check that the application structure was created
			Expect(filepath.Join(env.Dir, "app")).To(BeADirectory())
			Expect(filepath.Join(env.Dir, "app", "releases")).To(BeADirectory())
			Expect(filepath.Join(env.Dir, "app", "shared")).To(BeADirectory())
			Expect(filepath.Join(env.Dir, "app", "logs")).To(BeADirectory())
			Expect(filepath.Join(env.Dir, "app", "config.toml")).To(BeAnExistingFile())

			// check that the config file was created with the correct content
			generatedConfig, err := config.Load(filepath.Join(env.Dir, "app", "config.toml"))
			Expect(err).NotTo(HaveOccurred())

			defaultConfig := config.Default()
			Expect(generatedConfig).To(Equal(defaultConfig))
		})
	})

	Context("deploy command", func() {
		It("should successfully deploy", func() {
			// create a new test environment
			env, err := NewTestEnv(workingDir, "deploy-test-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(env.Dir).To(BeADirectory())

			// change working directory to the test environment
			err = os.Chdir(env.Dir)
			Expect(err).NotTo(HaveOccurred())

			// init the app
			_, err = env.InitApp()
			Expect(err).NotTo(HaveOccurred())

			// commit a file
			err = env.CommitFile("test1.txt")
			Expect(err).NotTo(HaveOccurred())

			// start the deployment
			err = app.Run([]string{"deploy", "start", "-f", filepath.Join(env.Dir, "app", "config.toml")})
			Expect(err).NotTo(HaveOccurred())
		})

		// force
		// release lock on error
		// release lock on success
		// rollback
		// hooks
	})
})

type testEnv struct {
	Dir string
}

func NewTestEnv(baseDir string, name string) (*testEnv, error) {
	// create a test directory
	dir := filepath.Join(baseDir, name)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	// create a git repository
	gitDir := filepath.Join(dir, "repo")
	err = os.MkdirAll(gitDir, 0755)
	if err != nil {
		return nil, err
	}

	err = runGitCommand(gitDir, "init")
	if err != nil {
		return nil, err
	}

	err = runGitCommand(gitDir, "branch", "-m", "main")
	if err != nil {
		return nil, err
	}

	return &testEnv{
		Dir: dir,
	}, nil
}

func (t testEnv) InitApp() (string, error) {
	err := app.Run([]string{"deploy", "init", "-n", "app"})
	if err != nil {
		return "", err
	}

	return filepath.Join(t.Dir, "app"), nil
}

func (t testEnv) CommitFile(filename string) error {
	repoDir := filepath.Join(t.Dir, "repo")
	filePath := filepath.Join(repoDir, filename)
	err := os.WriteFile(filePath, []byte("test content"), 0644)
	if err != nil {
		return err
	}

	err = runGitCommand(repoDir, "add", filename)
	if err != nil {
		return err
	}

	err = runGitCommand(repoDir, "commit", "-m", "add "+filename)
	if err != nil {
		return err
	}
	return nil
}

func (t testEnv) CreateHooks() error {
	hookLogFile := filepath.Join(t.Dir, "repo", ".deploy", "hooks.log")

	hookDir := filepath.Join(t.Dir, "repo", ".deploy", "hooks")
	err := os.MkdirAll(hookDir, 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(hookDir, "build"), []byte(fmt.Sprintf("#!/bin/sh\necho 'build' >> %s\n", hookLogFile)), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(hookDir, "deploy"), []byte(fmt.Sprintf("#!/bin/sh\necho 'deploy' >> %s\n", hookLogFile)), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(hookDir, "post_deploy"), []byte(fmt.Sprintf("#!/bin/sh\necho 'post_deploy' >> %s\n", hookLogFile)), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(hookDir, "verify"), []byte(fmt.Sprintf("#!/bin/sh\necho 'verify' >> %s\n", hookLogFile)), 0755)
	if err != nil {
		return err
	}

	return nil
}

func runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %w\n%s", err, string(output))
	}

	return nil
}
