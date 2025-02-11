package config

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Source SourceConfig `toml:"source"`
	Deploy DeployConfig `toml:"deploy"`
}

type SourceConfig struct {
	Provider string    `toml:"provider"`
	Git      GitConfig `toml:"git,omitempty"`
}

type GitConfig struct {
	Repo   string `toml:"repo"`
	Branch string `toml:"branch"`
}

type DeployConfig struct {
	KeepReleases int          `toml:"keep_releases"`
	Jitter       JitterConfig `toml:"jitter"`
	Shared       SharedConfig `toml:"shared"`
}

type JitterConfig struct {
	Min int `toml:"min"`
	Max int `toml:"max"`
}

type SharedConfig struct {
	Dirs  []string `toml:"dirs"`
	Files []string `toml:"files"`
}

func Default() *Config {
	c := &Config{}

	c.Source.Provider = "git"
	c.Source.Git.Repo = ""
	c.Source.Git.Branch = "main"

	c.Deploy.KeepReleases = 3
	c.Deploy.Jitter.Min = 5
	c.Deploy.Jitter.Max = 10
	c.Deploy.Shared.Dirs = []string{}
	c.Deploy.Shared.Files = []string{}

	return c
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}
