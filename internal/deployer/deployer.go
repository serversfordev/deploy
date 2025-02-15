package deployer

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"github.com/serversfordev/deploy/internal/config"
	"github.com/serversfordev/deploy/internal/hook"
	"github.com/serversfordev/deploy/internal/lock"
	"github.com/serversfordev/deploy/internal/logger"
	"github.com/serversfordev/deploy/internal/provider"
	"github.com/serversfordev/deploy/internal/release"
)

type State string

const (
	StateInit          State = "init"
	StateDetectChanges State = "detect_changes"
	StateClone         State = "clone"
	StateBuild         State = "build"
	StateDeploy        State = "deploy"
	StatePostDeploy    State = "post_deploy"
	StateVerify        State = "verify"
	StateError         State = "error"
	StateFinalize      State = "finalize"
	StateEnd           State = "end"
)

var stateTransitions = map[State][]State{
	StateInit:          {StateDetectChanges, StateError},
	StateDetectChanges: {StateClone, StateFinalize, StateError},
	StateClone:         {StateBuild, StateError},
	StateBuild:         {StateDeploy, StateError},
	StateDeploy:        {StatePostDeploy, StateError},
	StatePostDeploy:    {StateVerify, StateError},
	StateVerify:        {StateFinalize, StateError},
	StateError:         {StateFinalize},
	StateFinalize:      {StateEnd},
}

type Context struct {
	Logger        *logger.Logger
	Config        *config.Config
	Provider      provider.Provider
	AppDir        string
	Force         bool
	NewReleaseDir string

	rollbackFuncs []func() error
}

func (ctx *Context) AddRollbackFunc(fn func() error) {
	ctx.rollbackFuncs = append(ctx.rollbackFuncs, fn)
}

func (ctx *Context) ExecuteRollback() {
	// Execute rollback functions in reverse order (LIFO)
	for i := len(ctx.rollbackFuncs) - 1; i >= 0; i-- {
		if err := ctx.rollbackFuncs[i](); err != nil {
			ctx.Logger.Printf("rollback function %d failed: %s", i, err)
		}
	}

	// Clear rollback functions after execution
	ctx.rollbackFuncs = nil
}

type StateHandler func(ctx *Context) (State, error)

var stateHandlers = map[State]StateHandler{
	StateInit: func(ctx *Context) (State, error) {
		ctx.Logger.Printf("initializing")

		ctx.Logger.Printf("acquiring lock")
		if err := lock.Acquire(ctx.AppDir); err != nil {
			ctx.Logger.Printf("failed to acquire lock: %s", err)
			return StateError, nil
		}

		if ctx.Config.Deploy.Jitter.Min != 0 && ctx.Config.Deploy.Jitter.Max != 0 {
			min := float64(ctx.Config.Deploy.Jitter.Min)
			max := float64(ctx.Config.Deploy.Jitter.Max)

			jitterSeconds := min + rand.Float64()*(max-min)

			ctx.Logger.Printf("jittering for %d seconds", int(jitterSeconds))
			time.Sleep(time.Duration(jitterSeconds * float64(time.Second)))
		}

		if err := ctx.Provider.Init(); err != nil {
			ctx.Logger.Printf("failed to initialize provider: %s", err)
			return StateError, nil
		}

		return StateDetectChanges, nil
	},

	StateDetectChanges: func(ctx *Context) (State, error) {
		ctx.Logger.Printf("detecting changes")

		currentRevision, err := release.CurrentRevision(ctx.AppDir)
		if err != nil {
			if err == release.ErrNoCurrentRelease {
				return StateClone, nil
			}

			ctx.Logger.Printf("failed to get current revision: %s", err)
			return StateError, nil
		}

		providerRevision, err := ctx.Provider.GetRevision()
		if err != nil {
			ctx.Logger.Printf("failed to get provider revision: %s", err)
			return StateError, nil
		}

		if currentRevision != providerRevision {
			ctx.Logger.Printf("changes detected")
			return StateClone, nil
		}

		if ctx.Force {
			ctx.Logger.Printf("forcing deployment")
			return StateClone, nil
		}

		ctx.Logger.Printf("no changes detected, skipping deployment")

		return StateFinalize, nil
	},

	StateClone: func(ctx *Context) (State, error) {
		ctx.Logger.Printf("cloning")

		providerRevision, err := ctx.Provider.GetRevision()
		if err != nil {
			ctx.Logger.Printf("failed to get provider revision: %s", err)
			return StateError, nil
		}

		ctx.NewReleaseDir, err = release.NewRelease(ctx.AppDir, providerRevision)
		if err != nil {
			ctx.Logger.Printf("failed to create new release: %s", err)
			return StateError, nil
		}

		// cleaning up in case of failure
		ctx.AddRollbackFunc(func() error {
			ctx.Logger.Printf("cleaning up new release")
			return release.CleanupRelease(ctx.NewReleaseDir)
		})
		ctx.Logger.Printf("new release directory created: %s", providerRevision)

		if err := ctx.Provider.Clone(ctx.NewReleaseDir); err != nil {
			ctx.Logger.Printf("failed to clone provider: %s", err)
			return StateError, nil
		}

		ctx.Logger.Printf("executing clone hook")
		if err := hook.ExecuteHook(ctx.NewReleaseDir, hook.HookClone); err != nil {
			ctx.Logger.Printf("failed to execute clone hook: %s", err)
			return StateError, nil
		}

		ctx.Logger.Printf("linking shared files and dirs")
		for _, dir := range ctx.Config.Deploy.Shared.Dirs {
			sharedDirPath := filepath.Join(ctx.AppDir, "shared", dir)
			releaseDirPath := filepath.Join(ctx.NewReleaseDir, dir)

			_ = os.RemoveAll(releaseDirPath)

			if err := os.Symlink(sharedDirPath, releaseDirPath); err != nil {
				ctx.Logger.Printf("failed to symlink shared dir: %s", err)
				return StateError, nil
			}
		}

		for _, file := range ctx.Config.Deploy.Shared.Files {
			sharedFilePath := filepath.Join(ctx.AppDir, "shared", file)
			releaseFilePath := filepath.Join(ctx.NewReleaseDir, file)

			_ = os.RemoveAll(releaseFilePath)

			if err := os.Symlink(sharedFilePath, releaseFilePath); err != nil {
				ctx.Logger.Printf("failed to symlink shared file: %s", err)
				return StateError, nil
			}
		}

		return StateBuild, nil
	},

	StateBuild: func(ctx *Context) (State, error) {
		ctx.Logger.Printf("executing build hook")
		if err := hook.ExecuteHook(ctx.NewReleaseDir, hook.HookBuild); err != nil {
			ctx.Logger.Printf("failed to execute build hook: %s", err)
			return StateError, nil
		}

		return StateDeploy, nil
	},

	StateDeploy: func(ctx *Context) (State, error) {
		ctx.Logger.Printf("executing deploy hook")
		if err := hook.ExecuteHook(ctx.NewReleaseDir, hook.HookDeploy); err != nil {
			ctx.Logger.Printf("failed to execute deploy hook: %s", err)
			return StateError, nil
		}

		ctx.Logger.Printf("updating current release")
		err := release.UpdateCurrent(ctx.AppDir, ctx.NewReleaseDir)
		if err != nil {
			ctx.Logger.Printf("failed to update current release: %s", err)
			return StateError, nil
		}

		// rollback in case of failure
		ctx.AddRollbackFunc(func() error {
			ctx.Logger.Printf("rolling back")
			return release.Rollback(ctx.AppDir)
		})

		return StatePostDeploy, nil
	},

	StatePostDeploy: func(ctx *Context) (State, error) {
		ctx.Logger.Printf("executing post deploy hook")
		if err := hook.ExecuteHook(ctx.NewReleaseDir, hook.HookPostDeploy); err != nil {
			ctx.Logger.Printf("failed to execute post deploy hook: %s", err)
			return StateError, nil
		}

		return StateVerify, nil
	},

	StateVerify: func(ctx *Context) (State, error) {
		ctx.Logger.Printf("executing verify hook")
		if err := hook.ExecuteHook(ctx.NewReleaseDir, hook.HookVerify); err != nil {
			ctx.Logger.Printf("verification hook returned with a non-zero exit code")

			return StateError, nil
		}

		return StateFinalize, nil
	},

	StateError: func(ctx *Context) (State, error) {
		ctx.Logger.Printf("rolling back")

		ctx.ExecuteRollback()

		return StateFinalize, nil
	},

	StateFinalize: func(ctx *Context) (State, error) {
		ctx.Logger.Printf("finalizing")

		ctx.Logger.Printf("cleaning up old releases")
		if err := release.CleanupOldReleases(ctx.AppDir, ctx.Config.Deploy.KeepReleases); err != nil {
			ctx.Logger.Printf("failed to cleanup old releases: %s", err)
		}

		ctx.Logger.Printf("releasing lock")
		if err := lock.Release(ctx.AppDir); err != nil {
			ctx.Logger.Printf("failed to release lock: %s", err)
		}

		return StateEnd, nil
	},
}

type Deployer struct {
	currentState State
	transitions  map[State][]State
	handlers     map[State]StateHandler
}

func New() *Deployer {
	d := &Deployer{
		currentState: StateInit,
		transitions:  stateTransitions,
		handlers:     stateHandlers,
	}

	return d
}

func (d *Deployer) Execute(ctx *Context) error {
	for {
		if d.currentState == StateEnd {
			break
		}

		handler, exists := d.handlers[d.currentState]
		if !exists {
			return fmt.Errorf("no handler for state: %s", d.currentState)
		}

		nextState, err := handler(ctx)
		if err != nil {
			return err
		}

		if !d.isValidTransition(nextState) {
			return fmt.Errorf("invalid state transition: %s -> %s", d.currentState, nextState)
		}

		d.currentState = nextState
	}

	return nil
}

func (d *Deployer) isValidTransition(next State) bool {
	validStates, exists := d.transitions[d.currentState]
	if !exists {
		return false
	}

	for _, state := range validStates {
		if state == next {
			return true
		}
	}

	return false
}
