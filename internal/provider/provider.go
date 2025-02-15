package provider

import (
	"fmt"

	"github.com/serversfordev/deploy/internal/config"
	"github.com/serversfordev/deploy/internal/provider/git"
)

type Provider interface {
	Init() error
	GetRevision() (string, error)
	Clone(targetDir string) error
}

func New(cfg *config.Config, appDir string) (Provider, error) {
	switch cfg.Source.Provider {
	case "git":
		return git.New(cfg, appDir), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Source.Provider)
	}
}
