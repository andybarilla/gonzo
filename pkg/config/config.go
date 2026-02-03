// Package config provides configuration management for gonzo using Viper.
// It supports configuration from multiple sources with the following precedence:
// 1. Command-line flags (highest priority)
// 2. Environment variables (GONZO_ prefix)
// 3. Configuration file (~/.gonzo.yaml or ./gonzo.yaml)
// 4. Default values (lowest priority)
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// EnvPrefix is the prefix for environment variables
	EnvPrefix = "GONZO"

	// ConfigName is the name of the config file (without extension)
	ConfigName = "gonzo"

	// ConfigType is the default config file type
	ConfigType = "yaml"
)

// Config keys
const (
	KeyModel         = "model"
	KeyMaxIterations = "max-iterations"
	KeyQuiet         = "quiet"
	KeyBranch        = "branch"
	KeyTests         = "tests"
	KeyPR            = "pr"
	KeyCommitAuthor  = "commit-author"
)

// Default values
const (
	DefaultModel         = "claude-opus-4-5"
	DefaultMaxIterations = 10
	DefaultQuiet         = false
	DefaultBranch        = true
	DefaultTests         = true
	DefaultPR            = true
	DefaultCommitAuthor  = "Gonzo <gonzo@barilla.you>"
)

// Init initializes Viper with defaults, config file, and environment variables.
// This should be called before cobra.Command.Execute() to ensure configuration
// is loaded before flags are parsed.
func Init() error {
	// Set default values
	viper.SetDefault(KeyModel, DefaultModel)
	viper.SetDefault(KeyMaxIterations, DefaultMaxIterations)
	viper.SetDefault(KeyQuiet, DefaultQuiet)
	viper.SetDefault(KeyBranch, DefaultBranch)
	viper.SetDefault(KeyTests, DefaultTests)
	viper.SetDefault(KeyPR, DefaultPR)
	viper.SetDefault(KeyCommitAuthor, DefaultCommitAuthor)

	// Set config file name and type
	viper.SetConfigName(ConfigName)
	viper.SetConfigType(ConfigType)

	// Add config search paths
	// 1. Current directory
	viper.AddConfigPath(".")

	// 2. Home directory
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(home)
		// Also check ~/.config/gonzo/
		viper.AddConfigPath(filepath.Join(home, ".config", "gonzo"))
	}

	// Read config file if it exists (ignore error if not found)
	if err := viper.ReadInConfig(); err != nil {
		// Only return error if it's not a "file not found" error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Set up environment variables
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	return nil
}

// BindFlags binds Cobra flags to Viper configuration.
// This should be called in the cobra command's PersistentPreRunE or PreRunE
// after flags have been defined but before they are used.
func BindFlags(cmd *cobra.Command) error {
	flags := []string{KeyModel, KeyMaxIterations, KeyQuiet, KeyBranch, KeyTests, KeyPR, KeyCommitAuthor}

	for _, flag := range flags {
		if err := viper.BindPFlag(flag, cmd.PersistentFlags().Lookup(flag)); err != nil {
			return fmt.Errorf("error binding flag %s: %w", flag, err)
		}
	}

	return nil
}

// GetModel returns the configured model name
func GetModel() string {
	return viper.GetString(KeyModel)
}

// GetMaxIterations returns the configured max iterations
func GetMaxIterations() int {
	return viper.GetInt(KeyMaxIterations)
}

// GetQuiet returns whether quiet mode is enabled
func GetQuiet() bool {
	return viper.GetBool(KeyQuiet)
}

// GetBranch returns whether branch creation is enabled
func GetBranch() bool {
	return viper.GetBool(KeyBranch)
}

// GetTests returns whether tests should be run
func GetTests() bool {
	return viper.GetBool(KeyTests)
}

// GetPR returns whether PR creation is enabled
func GetPR() bool {
	return viper.GetBool(KeyPR)
}

// GetCommitAuthor returns the configured commit author
func GetCommitAuthor() string {
	return viper.GetString(KeyCommitAuthor)
}

// ConfigFileUsed returns the config file path if one was found and loaded
func ConfigFileUsed() string {
	return viper.ConfigFileUsed()
}

// AllSettings returns all settings as a map
func AllSettings() map[string]interface{} {
	return viper.AllSettings()
}
