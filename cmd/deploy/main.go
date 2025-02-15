package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/serversfordev/deploy/internal/config"
	"github.com/serversfordev/deploy/internal/deployer"
	"github.com/serversfordev/deploy/internal/logger"
	"github.com/serversfordev/deploy/internal/provider"
	"github.com/serversfordev/deploy/internal/utils"
)

var app = &cli.App{
	Name:  "deploy",
	Usage: "a simple application deployment tool",
	Commands: []*cli.Command{
		{
			Name:   "init",
			Usage:  "initialize new application deployment structure",
			Action: initCommand,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Aliases:  []string{"n"},
					Usage:    "application name",
					Required: true,
				},
			},
		},
		{
			Name:   "start",
			Usage:  "start deployment process",
			Action: startCommand,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "file",
					Aliases: []string{"f"},
					Usage:   "path to app.yaml configuration file",
				},
				&cli.BoolFlag{
					Name:  "force",
					Usage: "force deployment even if no changes detected",
					Value: false,
				},
			},
		},
	},
}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func initCommand(c *cli.Context) error {
	appName := c.String("name")
	appName = utils.NormalizeAppName(appName)

	appDir, err := utils.InitializeAppStructure(appName)
	if err != nil {
		return fmt.Errorf("failed to initialize app structure: %w", err)
	}

	fmt.Printf("successfully initialized deployment structure for %s under %s\n", appName, appDir)

	return nil
}

func startCommand(c *cli.Context) error {
	var configPath string
	if c.String("file") != "" {
		configPath = c.String("file")
	} else {
		configPath = "config.toml"
	}

	configPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	appDir, err := filepath.Abs(filepath.Dir(configPath))
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	logger, err := logger.New(appDir)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	p, err := provider.New(cfg, appDir)
	if err != nil {
		return err
	}

	ctx := deployer.Context{
		Logger:   logger,
		Config:   cfg,
		Provider: p,
		AppDir:   appDir,
		Force:    c.Bool("force"),
	}

	deployer := deployer.New()
	err = deployer.Execute(&ctx)
	if err != nil {
		return err
	}

	return nil
}
