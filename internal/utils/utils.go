package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/zsoltkacsandi/deploy/internal/config"
)

// NormalizeAppName sanitizes the input string to contain only letters, digits,
// underscores, hyphens, and dots, removing any other characters.
func NormalizeAppName(input string) string {
	var builder strings.Builder

	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			builder.WriteRune(r)
		}
	}

	return strings.TrimSpace(builder.String())
}

// InitializeAppStructure creates the necessary directory structure and config file for a new application.
// It returns the path to the created base directory and any error encountered.
func InitializeAppStructure(appName string) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	baseDir := filepath.Join(currentDir, appName)
	dirs := []string{
		filepath.Join(baseDir, "releases"),
		filepath.Join(baseDir, "shared"),
		filepath.Join(baseDir, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	cfg := config.Default()
	var tomlBuffer strings.Builder
	if err := toml.NewEncoder(&tomlBuffer).Encode(cfg); err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	tomlPath := filepath.Join(baseDir, "config.toml")
	if err := os.WriteFile(tomlPath, []byte(tomlBuffer.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return baseDir, nil
}
