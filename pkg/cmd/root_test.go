/*
Copyright © 2026 Andy Barilla me@andybarilla.com
*/
package cmd

import (
	"bytes"
	"context"
	"gonzo/pkg/gonzo"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// mockRunner implements gonzo.Runner for testing.
type mockRunner struct {
	model         string
	quiet         bool
	maxIterations int
	branch        bool
	tests         bool
	pr            bool
	response      string
	err           error
	// Captured values
	capturedPrompt string
	generateCalled bool
}

func (m *mockRunner) Generate(ctx context.Context, prompt string) (string, error) {
	m.capturedPrompt = prompt
	m.generateCalled = true
	return m.response, m.err
}

// mockRunnerFactory creates a factory function that returns a mock runner and captures options.
func mockRunnerFactory(mock *mockRunner) func(model string, quiet bool, maxIter int, branch bool, tests bool, pr bool) gonzo.Runner {
	return func(model string, quiet bool, maxIter int, branch bool, tests bool, pr bool) gonzo.Runner {
		mock.model = model
		mock.quiet = quiet
		mock.maxIterations = maxIter
		mock.branch = branch
		mock.tests = tests
		mock.pr = pr
		return mock
	}
}

func executeCommandC(root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	c, err = root.ExecuteC()

	return c, buf.String(), err
}

func TestRunClaudePrompt_WithArgs(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	defer func() { newRunner = originalNewRunner }()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "hello", "world")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.capturedPrompt != "hello world" {
		t.Errorf("expected prompt 'hello world', got %q", mock.capturedPrompt)
	}

	output := strings.TrimSpace(buf.String())
	if output != "mocked response" {
		t.Errorf("expected output 'mocked response', got %q", output)
	}
}

func TestRunClaudePrompt_WithPipedStdin(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalStdin := os.Stdin
	defer func() {
		newRunner = originalNewRunner
		os.Stdin = originalStdin
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Create a pipe to simulate stdin
	stdinR, stdinW, _ := os.Pipe()
	os.Stdin = stdinR

	// Write to the pipe in a goroutine
	go func() {
		_, _ = stdinW.WriteString("piped input\n")
		_ = stdinW.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.capturedPrompt != "piped input" {
		t.Errorf("expected prompt 'piped input', got %q", mock.capturedPrompt)
	}
}

func TestRunClaudePrompt_NoInput_ShowsHelp(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	defer func() {
		newRunner = originalNewRunner
	}()

	mock := &mockRunner{}
	newRunner = mockRunnerFactory(mock)

	_, output, err := executeCommandC(rootCmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.generateCalled {
		t.Error("Generate should not be called when no input is provided")
	}

	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected help output containing 'Usage:', got %q", output)
	}
}

func TestRunClaudePrompt_ArgsOverridePipe(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalStdin := os.Stdin
	defer func() {
		newRunner = originalNewRunner
		os.Stdin = originalStdin
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Create a pipe with data (simulating piped stdin)
	stdinR, stdinW, _ := os.Pipe()
	os.Stdin = stdinR

	go func() {
		_, _ = stdinW.WriteString("piped input\n")
		_ = stdinW.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "args", "input")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Args should take precedence over piped stdin
	if mock.capturedPrompt != "args input" {
		t.Errorf("expected prompt 'args input', got %q", mock.capturedPrompt)
	}
}

func TestRunClaudePrompt_MultilineStdin(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalStdin := os.Stdin
	defer func() {
		newRunner = originalNewRunner
		os.Stdin = originalStdin
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Create a pipe with multiline input
	stdinR, stdinW, _ := os.Pipe()
	os.Stdin = stdinR

	go func() {
		_, _ = stdinW.WriteString("line one\nline two\nline three\n")
		_ = stdinW.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPrompt := "line one\nline two\nline three"
	if mock.capturedPrompt != expectedPrompt {
		t.Errorf("expected prompt %q, got %q", expectedPrompt, mock.capturedPrompt)
	}
}

func TestRunClaudePrompt_DefaultModel(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalModel := llmModel
	defer func() {
		newRunner = originalNewRunner
		llmModel = originalModel
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset model to default
	llmModel = ModelClaudeOpus
	_, _, err := executeCommandC(rootCmd, "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedModel := "claude-opus-4-5"
	if mock.model != expectedModel {
		t.Errorf("expected default model %q, got %q", expectedModel, mock.model)
	}
}

func TestRunClaudePrompt_ModelFlag(t *testing.T) {
	tests := []struct {
		name          string
		flagValue     string
		expectedModel string
	}{
		{"haiku", "claude-haiku-4-5", "claude-haiku-4-5"},
		{"sonnet", "claude-sonnet-4-5", "claude-sonnet-4-5"},
		{"opus", "claude-opus-4-5", "claude-opus-4-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original and restore after test
			originalNewRunner := newRunner
			originalModel := llmModel
			defer func() {
				newRunner = originalNewRunner
				llmModel = originalModel
			}()

			mock := &mockRunner{response: "mocked response"}
			newRunner = mockRunnerFactory(mock)

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			_, _, err := executeCommandC(rootCmd, "--model", tt.flagValue, "test prompt")

			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if mock.model != tt.expectedModel {
				t.Errorf("expected model %q, got %q", tt.expectedModel, mock.model)
			}
		})
	}
}

func TestRunClaudePrompt_ModelFlagShort(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalModel := llmModel
	defer func() {
		newRunner = originalNewRunner
		llmModel = originalModel
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "-m", "claude-haiku-4-5", "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedModel := "claude-haiku-4-5"
	if mock.model != expectedModel {
		t.Errorf("expected model %q, got %q", expectedModel, mock.model)
	}
}

func TestRunClaudePrompt_InvalidModel(t *testing.T) {
	// Save original and restore after test
	originalModel := llmModel
	defer func() {
		llmModel = originalModel
	}()

	_, output, err := executeCommandC(rootCmd, "--model", "invalid-model", "test prompt")

	if err == nil {
		t.Error("expected error for invalid model")
	}

	if !strings.Contains(output, "invalid") || !strings.Contains(output, "model") {
		t.Errorf("expected error message about invalid model, got %q", output)
	}
}

func TestRunClaudePrompt_DefaultMaxIterations(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalMaxIterations := maxIterations
	defer func() {
		newRunner = originalNewRunner
		maxIterations = originalMaxIterations
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset to default (flag default is 10)
	maxIterations = 10
	_, _, err := executeCommandC(rootCmd, "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMaxIterations := 10
	if mock.maxIterations != expectedMaxIterations {
		t.Errorf("expected default maxIterations %d, got %d", expectedMaxIterations, mock.maxIterations)
	}
}

func TestRunClaudePrompt_MaxIterationsFlag(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalMaxIterations := maxIterations
	defer func() {
		newRunner = originalNewRunner
		maxIterations = originalMaxIterations
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "--max-iterations", "25", "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMaxIterations := 25
	if mock.maxIterations != expectedMaxIterations {
		t.Errorf("expected maxIterations %d, got %d", expectedMaxIterations, mock.maxIterations)
	}
}

func TestRunClaudePrompt_MaxIterationsFlagShort(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalMaxIterations := maxIterations
	defer func() {
		newRunner = originalNewRunner
		maxIterations = originalMaxIterations
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "-i", "5", "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMaxIterations := 5
	if mock.maxIterations != expectedMaxIterations {
		t.Errorf("expected maxIterations %d, got %d", expectedMaxIterations, mock.maxIterations)
	}
}

func TestRunClaudePrompt_InvalidMaxIterations(t *testing.T) {
	// Save original and restore after test
	originalMaxIterations := maxIterations
	defer func() {
		maxIterations = originalMaxIterations
	}()

	_, output, err := executeCommandC(rootCmd, "--max-iterations", "not-a-number", "test prompt")

	if err == nil {
		t.Error("expected error for invalid max-iterations")
	}

	if !strings.Contains(output, "invalid") {
		t.Errorf("expected error message about invalid value, got %q", output)
	}
}

func TestRunClaudePrompt_DefaultBranch(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalBranch := branch
	defer func() {
		newRunner = originalNewRunner
		branch = originalBranch
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset to default (flag default is true)
	branch = true
	_, _, err := executeCommandC(rootCmd, "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.branch {
		t.Errorf("expected default branch true, got %v", mock.branch)
	}
}

func TestRunClaudePrompt_BranchFlag(t *testing.T) {
	tests := []struct {
		name           string
		flagValue      string
		expectedBranch bool
	}{
		{"branch true", "true", true},
		{"branch false", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original and restore after test
			originalNewRunner := newRunner
			originalBranch := branch
			defer func() {
				newRunner = originalNewRunner
				branch = originalBranch
			}()

			mock := &mockRunner{response: "mocked response"}
			newRunner = mockRunnerFactory(mock)

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			_, _, err := executeCommandC(rootCmd, "--branch="+tt.flagValue, "test prompt")

			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if mock.branch != tt.expectedBranch {
				t.Errorf("expected branch %v, got %v", tt.expectedBranch, mock.branch)
			}
		})
	}
}

func TestRunClaudePrompt_BranchFlagShort(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalBranch := branch
	defer func() {
		newRunner = originalNewRunner
		branch = originalBranch
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "-b=false", "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.branch {
		t.Errorf("expected branch false, got %v", mock.branch)
	}
}

func TestRunClaudePrompt_DefaultTests(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalTests := tests
	defer func() {
		newRunner = originalNewRunner
		tests = originalTests
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset to default (flag default is true)
	tests = true
	_, _, err := executeCommandC(rootCmd, "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.tests {
		t.Errorf("expected default tests true, got %v", mock.tests)
	}
}

func TestRunClaudePrompt_TestsFlag(t *testing.T) {
	testCases := []struct {
		name          string
		flagValue     string
		expectedTests bool
	}{
		{"tests true", "true", true},
		{"tests false", "false", false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Save original and restore after test
			originalNewRunner := newRunner
			originalTests := tests
			defer func() {
				newRunner = originalNewRunner
				tests = originalTests
			}()

			mock := &mockRunner{response: "mocked response"}
			newRunner = mockRunnerFactory(mock)

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			_, _, err := executeCommandC(rootCmd, "--tests="+tt.flagValue, "test prompt")

			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if mock.tests != tt.expectedTests {
				t.Errorf("expected tests %v, got %v", tt.expectedTests, mock.tests)
			}
		})
	}
}

func TestRunClaudePrompt_TestsFlagShort(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalTests := tests
	defer func() {
		newRunner = originalNewRunner
		tests = originalTests
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "-t=false", "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.tests {
		t.Errorf("expected tests false, got %v", mock.tests)
	}
}

func TestRunClaudePrompt_DefaultPR(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalPR := pr
	defer func() {
		newRunner = originalNewRunner
		pr = originalPR
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset to default (flag default is false)
	pr = false
	_, _, err := executeCommandC(rootCmd, "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.pr {
		t.Errorf("expected default pr false, got %v", mock.pr)
	}
}

func TestRunClaudePrompt_PRFlag(t *testing.T) {
	testCases := []struct {
		name       string
		flagValue  string
		expectedPR bool
	}{
		{"pr true", "true", true},
		{"pr false", "false", false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Save original and restore after test
			originalNewRunner := newRunner
			originalPR := pr
			defer func() {
				newRunner = originalNewRunner
				pr = originalPR
			}()

			mock := &mockRunner{response: "mocked response"}
			newRunner = mockRunnerFactory(mock)

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			_, _, err := executeCommandC(rootCmd, "--pr="+tt.flagValue, "test prompt")

			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if mock.pr != tt.expectedPR {
				t.Errorf("expected pr %v, got %v", tt.expectedPR, mock.pr)
			}
		})
	}
}

func TestRunClaudePrompt_PRFlagShort(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	originalPR := pr
	defer func() {
		newRunner = originalNewRunner
		pr = originalPR
	}()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "-p", "test prompt")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.pr {
		t.Errorf("expected pr true, got %v", mock.pr)
	}
}

func TestRunClaudePrompt_WithFeatureFile(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	defer func() { newRunner = originalNewRunner }()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Create a temporary file with feature content
	tmpDir := t.TempDir()
	featureFile := filepath.Join(tmpDir, "feature.txt")
	featureContent := "implement a login form with email and password fields"
	if err := os.WriteFile(featureFile, []byte(featureContent), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, featureFile)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The feature should be read from the file
	if mock.capturedPrompt != featureContent {
		t.Errorf("expected prompt from file %q, got %q", featureContent, mock.capturedPrompt)
	}
}

func TestRunClaudePrompt_WithFeatureFileMultiline(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	defer func() { newRunner = originalNewRunner }()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Create a temporary file with multiline feature content
	tmpDir := t.TempDir()
	featureFile := filepath.Join(tmpDir, "feature.md")
	featureContent := "# Feature: User Login\n\n## Description\nImplement a login form with:\n- Email field\n- Password field\n- Remember me checkbox"
	if err := os.WriteFile(featureFile, []byte(featureContent), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, featureFile)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The feature should be read from the file (trimmed)
	expectedContent := strings.TrimSpace(featureContent)
	if mock.capturedPrompt != expectedContent {
		t.Errorf("expected prompt from file %q, got %q", expectedContent, mock.capturedPrompt)
	}
}

func TestRunClaudePrompt_NonExistentFileTreatedAsFeature(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	defer func() { newRunner = originalNewRunner }()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Pass a non-existent file path - should be treated as a feature string
	nonExistentPath := "/path/to/nonexistent/file.txt"
	_, _, err := executeCommandC(rootCmd, nonExistentPath)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The argument should be treated as a feature string since the file doesn't exist
	if mock.capturedPrompt != nonExistentPath {
		t.Errorf("expected prompt %q, got %q", nonExistentPath, mock.capturedPrompt)
	}
}

func TestRunClaudePrompt_MultipleArgsTreatedAsFeature(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	defer func() { newRunner = originalNewRunner }()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Create a temporary file - but it won't be used since we have multiple args
	tmpDir := t.TempDir()
	featureFile := filepath.Join(tmpDir, "feature.txt")
	if err := os.WriteFile(featureFile, []byte("file content"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Multiple args should be joined as a feature, not read from file
	_, _, err := executeCommandC(rootCmd, featureFile, "extra", "args")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Multiple args should be joined, not read from file
	expectedPrompt := featureFile + " extra args"
	if mock.capturedPrompt != expectedPrompt {
		t.Errorf("expected prompt %q, got %q", expectedPrompt, mock.capturedPrompt)
	}
}

func TestRunClaudePrompt_DirectoryNotReadAsFile(t *testing.T) {
	// Save original and restore after test
	originalNewRunner := newRunner
	defer func() { newRunner = originalNewRunner }()

	mock := &mockRunner{response: "mocked response"}
	newRunner = mockRunnerFactory(mock)

	// Create a temporary directory
	tmpDir := t.TempDir()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Pass a directory path - should be treated as a feature string, not read as file
	_, _, err := executeCommandC(rootCmd, tmpDir)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The directory path should be treated as a feature string
	if mock.capturedPrompt != tmpDir {
		t.Errorf("expected prompt %q, got %q", tmpDir, mock.capturedPrompt)
	}
}

func TestReadFeatureFromFile(t *testing.T) {
	t.Run("reads regular file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		content := "  test content with whitespace  "
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		result, err := readFeatureFromFile(filePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "test content with whitespace"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := readFeatureFromFile("/nonexistent/path/file.txt")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("returns error for directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := readFeatureFromFile(tmpDir)
		if err == nil {
			t.Error("expected error for directory")
		}
	})
}
