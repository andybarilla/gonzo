package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetViper resets Viper to a clean state between tests
func resetViper() {
	viper.Reset()
}

func TestInit_DefaultValues(t *testing.T) {
	resetViper()

	err := Init()
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	tests := []struct {
		key      string
		expected interface{}
		getter   func() interface{}
	}{
		{KeyModel, DefaultModel, func() interface{} { return GetModel() }},
		{KeyMaxIterations, DefaultMaxIterations, func() interface{} { return GetMaxIterations() }},
		{KeyQuiet, DefaultQuiet, func() interface{} { return GetQuiet() }},
		{KeyBranch, DefaultBranch, func() interface{} { return GetBranch() }},
		{KeyTests, DefaultTests, func() interface{} { return GetTests() }},
		{KeyPR, DefaultPR, func() interface{} { return GetPR() }},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := tt.getter()
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestInit_EnvironmentVariables(t *testing.T) {
	resetViper()

	// Set environment variables
	envVars := map[string]string{
		"GONZO_MODEL":          "claude-haiku-4-5",
		"GONZO_MAX_ITERATIONS": "25",
		"GONZO_QUIET":          "true",
		"GONZO_BRANCH":         "false",
		"GONZO_TESTS":          "false",
		"GONZO_PR":             "true",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	err := Init()
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	tests := []struct {
		name     string
		expected interface{}
		getter   func() interface{}
	}{
		{"model", "claude-haiku-4-5", func() interface{} { return GetModel() }},
		{"max-iterations", 25, func() interface{} { return GetMaxIterations() }},
		{"quiet", true, func() interface{} { return GetQuiet() }},
		{"branch", false, func() interface{} { return GetBranch() }},
		{"tests", false, func() interface{} { return GetTests() }},
		{"pr", true, func() interface{} { return GetPR() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.getter()
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestInit_ConfigFile(t *testing.T) {
	resetViper()

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gonzo.yaml")

	configContent := `model: claude-sonnet-4-5
max-iterations: 15
quiet: true
branch: false
tests: false
pr: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Change to the temp directory so viper finds the config
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	err := Init()
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	tests := []struct {
		name     string
		expected interface{}
		getter   func() interface{}
	}{
		{"model", "claude-sonnet-4-5", func() interface{} { return GetModel() }},
		{"max-iterations", 15, func() interface{} { return GetMaxIterations() }},
		{"quiet", true, func() interface{} { return GetQuiet() }},
		{"branch", false, func() interface{} { return GetBranch() }},
		{"tests", false, func() interface{} { return GetTests() }},
		{"pr", true, func() interface{} { return GetPR() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.getter()
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}

	// Verify the config file was used
	if ConfigFileUsed() == "" {
		t.Error("expected ConfigFileUsed() to return a path")
	}
}

func TestInit_EnvOverridesConfigFile(t *testing.T) {
	resetViper()

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gonzo.yaml")

	configContent := `model: claude-sonnet-4-5
max-iterations: 15
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set environment variable that should override config file
	os.Setenv("GONZO_MODEL", "claude-opus-4-5")
	os.Setenv("GONZO_MAX_ITERATIONS", "50")
	defer func() {
		os.Unsetenv("GONZO_MODEL")
		os.Unsetenv("GONZO_MAX_ITERATIONS")
	}()

	// Change to the temp directory so viper finds the config
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	err := Init()
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	// Environment variable should override config file
	if got := GetModel(); got != "claude-opus-4-5" {
		t.Errorf("expected model to be overridden by env var, got %v", got)
	}
	if got := GetMaxIterations(); got != 50 {
		t.Errorf("expected max-iterations to be overridden by env var, got %v", got)
	}
}

func TestBindFlags(t *testing.T) {
	resetViper()

	// Create a test command with flags
	cmd := &cobra.Command{Use: "test"}
	cmd.PersistentFlags().String(KeyModel, DefaultModel, "model")
	cmd.PersistentFlags().Int(KeyMaxIterations, DefaultMaxIterations, "max iterations")
	cmd.PersistentFlags().Bool(KeyQuiet, DefaultQuiet, "quiet mode")
	cmd.PersistentFlags().Bool(KeyBranch, DefaultBranch, "branch")
	cmd.PersistentFlags().Bool(KeyTests, DefaultTests, "tests")
	cmd.PersistentFlags().Bool(KeyPR, DefaultPR, "pr")

	// Set a flag value
	cmd.PersistentFlags().Set(KeyModel, "claude-haiku-4-5")
	cmd.PersistentFlags().Set(KeyMaxIterations, "42")

	err := Init()
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	err = BindFlags(cmd)
	if err != nil {
		t.Fatalf("BindFlags() returned error: %v", err)
	}

	// After binding, Viper should see the flag values
	if got := viper.GetString(KeyModel); got != "claude-haiku-4-5" {
		t.Errorf("expected model from flag binding, got %v", got)
	}
	if got := viper.GetInt(KeyMaxIterations); got != 42 {
		t.Errorf("expected max-iterations from flag binding, got %v", got)
	}
}

func TestAllSettings(t *testing.T) {
	resetViper()

	err := Init()
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	settings := AllSettings()

	// Check that all keys are present
	expectedKeys := []string{KeyModel, KeyMaxIterations, KeyQuiet, KeyBranch, KeyTests, KeyPR}
	for _, key := range expectedKeys {
		if _, ok := settings[key]; !ok {
			t.Errorf("expected key %q in AllSettings()", key)
		}
	}
}

func TestInit_NoConfigFile(t *testing.T) {
	resetViper()

	// Change to a directory with no config file
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Init should not fail if no config file exists
	err := Init()
	if err != nil {
		t.Fatalf("Init() should not fail when no config file exists: %v", err)
	}

	// Should still have default values
	if got := GetModel(); got != DefaultModel {
		t.Errorf("expected default model, got %v", got)
	}
}
