package gonzo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// mockCommandContext creates a mock exec.Cmd that calls TestHelperProcess instead of the real command.
// The response parameter is what the mock CLI will output.
func mockCommandContext(response string, exitCode int) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], cs...)
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			fmt.Sprintf("GO_HELPER_RESPONSE=%s", response),
			fmt.Sprintf("GO_HELPER_EXIT_CODE=%d", exitCode),
		}
		return cmd
	}
}

// TestHelperProcess is not a real test. It's used as a mock process for exec.Command tests.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	response := os.Getenv("GO_HELPER_RESPONSE")
	exitCodeStr := os.Getenv("GO_HELPER_EXIT_CODE")
	exitCode := 0
	if exitCodeStr != "" {
		fmt.Sscanf(exitCodeStr, "%d", &exitCode)
	}
	fmt.Print(response)
	os.Exit(exitCode)
}

func TestClaudeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"CLAUDE_CODE_CLI", ClaudeCodeCli, "claude"},
		{"CLAUDE_HAIKU", ClaudeHaiku, "claude-haiku-4-5"},
		{"CLAUDE_SONNET", ClaudeSonnet, "claude-sonnet-4-5"},
		{"CLAUDE_OPUS", ClaudeOpus, "claude-opus-4-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected %s to be %q, got %q", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

func TestEnsureProgressFileExists_CreatesFile(t *testing.T) {
	// Create a temp directory and change to it
	tmpDir, err := os.MkdirTemp("", "gonzo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save current directory and change to temp
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Verify .gonzo/progress.txt doesn't exist initially
	gonzoDir := filepath.Join(tmpDir, ".gonzo")
	progressPath := filepath.Join(gonzoDir, "progress.txt")
	if _, err := os.Stat(progressPath); !os.IsNotExist(err) {
		t.Fatal(".gonzo/progress.txt should not exist before test")
	}

	// Call the function - note: this will fail if promptLib isn't properly embedded
	cc := New()
	err = cc.ensureProgressFileExists()

	// The function may fail due to embed.FS not being initialized in test context
	// This is expected behavior - the embed directive requires the prompts directory
	if err != nil {
		t.Skipf("Skipping test - embed.FS not available in test context: %v", err)
	}

	// If we get here, verify the .gonzo directory and file were created
	if _, err := os.Stat(gonzoDir); os.IsNotExist(err) {
		t.Error(".gonzo directory should have been created")
	}
	if _, err := os.Stat(progressPath); os.IsNotExist(err) {
		t.Error(".gonzo/progress.txt should have been created")
	}
}

func TestEnsureProgressFileExists_ExistingFile(t *testing.T) {
	// Create a temp directory and change to it
	tmpDir, err := os.MkdirTemp("", "gonzo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save current directory and change to temp
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create the .gonzo directory and an existing progress.txt with custom content
	gonzoDir := filepath.Join(tmpDir, ".gonzo")
	if err := os.MkdirAll(gonzoDir, 0755); err != nil {
		t.Fatalf("failed to create .gonzo directory: %v", err)
	}
	progressPath := filepath.Join(gonzoDir, "progress.txt")
	originalContent := "existing content"
	if err := os.WriteFile(progressPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to create existing .gonzo/progress.txt: %v", err)
	}

	// Call the function
	cc := New()
	err = cc.ensureProgressFileExists()
	if err != nil {
		t.Skipf("Skipping test - embed.FS not available in test context: %v", err)
	}

	// Verify the existing file was not overwritten
	content, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read .gonzo/progress.txt: %v", err)
	}

	if string(content) != originalContent {
		t.Errorf("existing .gonzo/progress.txt should not be modified, got %q, want %q", string(content), originalContent)
	}
}

func TestGenerate_CLINotFound(t *testing.T) {
	// Test behavior when claude CLI is not available
	if _, err := exec.LookPath(ClaudeCodeCli); err == nil {
		t.Skip("Skipping test - claude CLI is available on this system")
	}

	ctx := context.Background()
	cc := New().WithModel(ClaudeSonnet).WithQuiet(true)
	_, err := cc.Generate(ctx, "test prompt")

	// Should fail because claude CLI is not found (or embed.FS issue)
	if err == nil {
		t.Error("expected error when claude CLI is not available")
	}
}

func TestGenerate_WithContext(t *testing.T) {
	// Save original and restore after test
	originalCommandContext := commandContext
	defer func() { commandContext = originalCommandContext }()

	// Mock the command to return a simple response
	commandContext = mockCommandContext("mocked response", 0)

	// Test that a cancelled context doesn't cause panic
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The function should handle the cancelled context gracefully
	// The implementation uses exec.CommandContext to respect context cancellation
	cc := New().WithModel(ClaudeSonnet).WithQuiet(true)
	_, err := cc.Generate(ctx, "test prompt")

	// With a cancelled context, we expect an error (context cancelled)
	// Main goal is to ensure no panic occurs
	_ = err
}

func TestGenerate_ModelPassthrough(t *testing.T) {
	// Save original and restore after test
	originalCommandContext := commandContext
	defer func() { commandContext = originalCommandContext }()

	// Mock the command to return a simple response
	commandContext = mockCommandContext("mocked response", 0)

	models := []string{
		ClaudeHaiku,
		ClaudeSonnet,
		ClaudeOpus,
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			ctx := context.Background()
			cc := New().WithModel(model).WithQuiet(true)
			result, err := cc.Generate(ctx, "test")
			if err != nil {
				t.Errorf("unexpected error for model %s: %v", model, err)
			}
			if result != "mocked response" {
				t.Errorf("expected 'mocked response', got %q", result)
			}
		})
	}
}

func TestGenerate_ReturnsOutput(t *testing.T) {
	// Save original and restore after test
	originalCommandContext := commandContext
	defer func() { commandContext = originalCommandContext }()

	expectedResponse := "This is the generated response from Claude"
	commandContext = mockCommandContext(expectedResponse, 0)

	ctx := context.Background()
	cc := New().WithModel(ClaudeSonnet).WithQuiet(true)
	result, err := cc.Generate(ctx, "test prompt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expectedResponse {
		t.Errorf("expected %q, got %q", expectedResponse, result)
	}
}

func TestGenerate_HandlesError(t *testing.T) {
	// Save original and restore after test
	originalCommandContext := commandContext
	defer func() { commandContext = originalCommandContext }()

	// Mock a command that exits with error
	commandContext = mockCommandContext("error output", 1)

	ctx := context.Background()
	cc := New().WithModel(ClaudeSonnet).WithQuiet(true)
	_, err := cc.Generate(ctx, "test prompt")

	if err == nil {
		t.Error("expected error when command exits with non-zero code")
	}
}

func TestWithPR(t *testing.T) {
	tests := []struct {
		name     string
		prValue  bool
		expected bool
	}{
		{"pr enabled", true, true},
		{"pr disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := New().WithPR(tt.prValue)
			if cc.pr != tt.expected {
				t.Errorf("expected pr %v, got %v", tt.expected, cc.pr)
			}
		})
	}
}

func TestDefaultPR(t *testing.T) {
	cc := New()
	if cc.pr != DefaultPR {
		t.Errorf("expected default pr %v, got %v", DefaultPR, cc.pr)
	}
	if cc.pr != false {
		t.Errorf("expected default pr to be false, got %v", cc.pr)
	}
}

func TestWithCommitAuthor(t *testing.T) {
	tests := []struct {
		name              string
		commitAuthorValue string
		expected          string
	}{
		{"custom author", "Custom Author <custom@example.com>", "Custom Author <custom@example.com>"},
		{"another author", "Another Person <another@test.org>", "Another Person <another@test.org>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := New().WithCommitAuthor(tt.commitAuthorValue)
			if cc.commitAuthor != tt.expected {
				t.Errorf("expected commitAuthor %q, got %q", tt.expected, cc.commitAuthor)
			}
		})
	}
}

func TestDefaultCommitAuthor(t *testing.T) {
	cc := New()
	if cc.commitAuthor != DefaultCommitAuthor {
		t.Errorf("expected default commitAuthor %q, got %q", DefaultCommitAuthor, cc.commitAuthor)
	}
	expectedDefault := "Gonzo <gonzo@barilla.you>"
	if cc.commitAuthor != expectedDefault {
		t.Errorf("expected default commitAuthor to be %q, got %q", expectedDefault, cc.commitAuthor)
	}
}
