package gonzo

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"
)

// commandContext is a variable that wraps exec.CommandContext for testing.
// Tests can replace this to mock command execution.
var commandContext = exec.CommandContext

const CLAUDE_CODE_CLI = "claude"
const CLAUDE_HAIKU = "claude-haiku-4-5"
const CLAUDE_SONNET = "claude-sonnet-4-5"
const CLAUDE_OPUS = "claude-opus-4-5"

//go:embed prompts
var promptLib embed.FS

// ClaudeGenerate sends a prompt to the Claude API and returns the generated response.
func ClaudeGenerate(ctx context.Context, model string, prompt string, quiet bool) (string, error) {
	systemPrompt, err := promptLib.ReadFile("prompts/PARAM_TASK_RUNNER.md")

	logInfo(quiet, "Starting Gonzo")
	logInfo(quiet, "  Model: %s", model)

	err = ensureProgressFileExists()
	if err != nil {
		return "", fmt.Errorf("failed to ensure progress file exists: %w", err)
	}

	cmd := commandContext(
		ctx,
		CLAUDE_CODE_CLI,
		"--dangerously-skip-permissions",
		"--print",
		"--model",
		model,
		"--system-prompt",
		string(systemPrompt),
		prompt)
	out, err := cmd.Output()

	logInfo(quiet, "Task completed!")
	return string(out), err
}

func ensureProgressFileExists() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	progressFile := filepath.Join(dir, "progress.txt")

	if _, err := os.Stat(progressFile); errors.Is(err, os.ErrNotExist) {
		t, err := template.ParseFS(promptLib, "prompts/progress.tmpl")
		if err != nil {
			return fmt.Errorf("failed to read progress template: %w", err)
		}

		f, err := os.Create(progressFile)
		if err != nil {
			return fmt.Errorf("failed to create progress file: %w", err)
		}
		defer func() { Swallow(f.Close()) }()
		err = t.ExecuteTemplate(f, "progress.tmpl", struct {
			Now time.Time
		}{
			Now: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to write to progress file: %w", err)
		}
	}
	return nil
}

func logInfo(quiet bool, format string, args ...interface{}) {
	if !quiet {
		fmt.Printf(format+"\n", args...)
	}
}
