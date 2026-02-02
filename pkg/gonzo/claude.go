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
var commandContext = exec.CommandContext

const ClaudeCodeCli = "claude"
const ClaudeHaiku = "claude-haiku-4-5"
const ClaudeSonnet = "claude-sonnet-4-5"
const ClaudeOpus = "claude-opus-4-5"

const DefaultOptClaudeModel = ClaudeOpus
const DefaultOptQuiet = false

//go:embed prompts
var promptLib embed.FS

// Runner is the interface for generating responses from Claude.
type Runner interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type ClaudeConfig struct {
	model string
	quiet bool
}

type Option func(*ClaudeConfig)

func (cc *ClaudeConfig) WithModel(model string) *ClaudeConfig {
	cc.model = model
	return cc
}

func (cc *ClaudeConfig) WithQuiet(quiet bool) *ClaudeConfig {
	cc.quiet = quiet
	return cc
}

func New() *ClaudeConfig {
	return &ClaudeConfig{
		model: DefaultOptClaudeModel,
		quiet: DefaultOptQuiet,
	}
}

// Generate sends a prompt to the Claude API and returns the generated response.
func (cc *ClaudeConfig) Generate(ctx context.Context, prompt string) (string, error) {
	systemPrompt, err := promptLib.ReadFile("prompts/PARAM_TASK_RUNNER.md")

	cc.logInfo("Starting Gonzo")
	cc.logInfo("  Model: %s", cc.model)

	err = cc.ensureProgressFileExists()
	if err != nil {
		return "", fmt.Errorf("failed to ensure progress file exists: %w", err)
	}

	cmd := commandContext(
		ctx,
		ClaudeCodeCli,
		"--dangerously-skip-permissions",
		"--print",
		"--model",
		cc.model,
		"--system-prompt",
		string(systemPrompt),
		prompt)
	out, err := cmd.Output()

	cc.logInfo("Task completed!")
	return string(out), err
}

func (cc *ClaudeConfig) ensureProgressFileExists() error {
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

func (cc *ClaudeConfig) logInfo(format string, args ...interface{}) {
	if !cc.quiet {
		fmt.Printf(format+"\n", args...)
	}
}
