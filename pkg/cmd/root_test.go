/*
Copyright © 2026 Andy Barilla me@andybarilla.com
*/
package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// mockClaudeGenerate creates a mock that captures the prompt and model, and returns a canned response.
func mockClaudeGenerate(capturedModel, capturedPrompt *string, response string, err error) func(ctx context.Context, model string, prompt string) (string, error) {
	return func(ctx context.Context, model string, prompt string) (string, error) {
		if capturedModel != nil {
			*capturedModel = model
		}
		if capturedPrompt != nil {
			*capturedPrompt = prompt
		}
		return response, err
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
	originalClaudeGenerate := claudeGenerate
	defer func() { claudeGenerate = originalClaudeGenerate }()

	var capturedPrompt string
	claudeGenerate = mockClaudeGenerate(nil, &capturedPrompt, "mocked response", nil)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "hello", "world")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedPrompt != "hello world" {
		t.Errorf("expected prompt 'hello world', got %q", capturedPrompt)
	}

	output := strings.TrimSpace(buf.String())
	if output != "mocked response" {
		t.Errorf("expected output 'mocked response', got %q", output)
	}
}

func TestRunClaudePrompt_WithPipedStdin(t *testing.T) {
	// Save original and restore after test
	originalClaudeGenerate := claudeGenerate
	originalStdin := os.Stdin
	defer func() {
		claudeGenerate = originalClaudeGenerate
		os.Stdin = originalStdin
	}()

	var capturedPrompt string
	claudeGenerate = mockClaudeGenerate(nil, &capturedPrompt, "mocked response", nil)

	// Create a pipe to simulate stdin
	stdinR, stdinW, _ := os.Pipe()
	os.Stdin = stdinR

	// Write to the pipe in a goroutine
	go func() {
		stdinW.WriteString("piped input\n")
		stdinW.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedPrompt != "piped input" {
		t.Errorf("expected prompt 'piped input', got %q", capturedPrompt)
	}
}

func TestRunClaudePrompt_NoInput_ShowsHelp(t *testing.T) {
	// Save original and restore after test
	originalClaudeGenerate := claudeGenerate
	originalStdin := os.Stdin
	defer func() {
		claudeGenerate = originalClaudeGenerate
		os.Stdin = originalStdin
	}()

	// Track if ClaudeGenerate was called (it shouldn't be)
	generateCalled := false
	claudeGenerate = func(ctx context.Context, model string, prompt string) (string, error) {
		generateCalled = true
		return "", nil
	}

	_, output, err := executeCommandC(rootCmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if generateCalled {
		t.Error("ClaudeGenerate should not be called when no input is provided")
	}

	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected help output containing 'Usage:', got %q", output)
	}
}

func TestRunClaudePrompt_ArgsOverridePipe(t *testing.T) {
	// Save original and restore after test
	originalClaudeGenerate := claudeGenerate
	originalStdin := os.Stdin
	defer func() {
		claudeGenerate = originalClaudeGenerate
		os.Stdin = originalStdin
	}()

	var capturedPrompt string
	claudeGenerate = mockClaudeGenerate(nil, &capturedPrompt, "mocked response", nil)

	// Create a pipe with data (simulating piped stdin)
	stdinR, stdinW, _ := os.Pipe()
	os.Stdin = stdinR

	go func() {
		stdinW.WriteString("piped input\n")
		stdinW.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "args", "input")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Args should take precedence over piped stdin
	if capturedPrompt != "args input" {
		t.Errorf("expected prompt 'args input', got %q", capturedPrompt)
	}
}

func TestRunClaudePrompt_MultilineStdin(t *testing.T) {
	// Save original and restore after test
	originalClaudeGenerate := claudeGenerate
	originalStdin := os.Stdin
	defer func() {
		claudeGenerate = originalClaudeGenerate
		os.Stdin = originalStdin
	}()

	var capturedPrompt string
	claudeGenerate = mockClaudeGenerate(nil, &capturedPrompt, "mocked response", nil)

	// Create a pipe with multiline input
	stdinR, stdinW, _ := os.Pipe()
	os.Stdin = stdinR

	go func() {
		stdinW.WriteString("line one\nline two\nline three\n")
		stdinW.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPrompt := "line one\nline two\nline three"
	if capturedPrompt != expectedPrompt {
		t.Errorf("expected prompt %q, got %q", expectedPrompt, capturedPrompt)
	}
}

func TestRunClaudePrompt_DefaultModel(t *testing.T) {
	// Save original and restore after test
	originalClaudeGenerate := claudeGenerate
	originalModel := llmModel
	defer func() {
		claudeGenerate = originalClaudeGenerate
		llmModel = originalModel
	}()

	var capturedModel string
	claudeGenerate = mockClaudeGenerate(&capturedModel, nil, "mocked response", nil)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset model to default
	llmModel = ModelClaudeOpus
	_, _, err := executeCommandC(rootCmd, "test prompt")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedModel := "claude-opus-4-5"
	if capturedModel != expectedModel {
		t.Errorf("expected default model %q, got %q", expectedModel, capturedModel)
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
			originalClaudeGenerate := claudeGenerate
			originalModel := llmModel
			defer func() {
				claudeGenerate = originalClaudeGenerate
				llmModel = originalModel
			}()

			var capturedModel string
			claudeGenerate = mockClaudeGenerate(&capturedModel, nil, "mocked response", nil)

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			_, _, err := executeCommandC(rootCmd, "--model", tt.flagValue, "test prompt")

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if capturedModel != tt.expectedModel {
				t.Errorf("expected model %q, got %q", tt.expectedModel, capturedModel)
			}
		})
	}
}

func TestRunClaudePrompt_ModelFlagShort(t *testing.T) {
	// Save original and restore after test
	originalClaudeGenerate := claudeGenerate
	originalModel := llmModel
	defer func() {
		claudeGenerate = originalClaudeGenerate
		llmModel = originalModel
	}()

	var capturedModel string
	claudeGenerate = mockClaudeGenerate(&capturedModel, nil, "mocked response", nil)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err := executeCommandC(rootCmd, "-m", "claude-haiku-4-5", "test prompt")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedModel := "claude-haiku-4-5"
	if capturedModel != expectedModel {
		t.Errorf("expected model %q, got %q", expectedModel, capturedModel)
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
