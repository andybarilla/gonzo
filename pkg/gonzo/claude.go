package gonzo

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
const DefaultMaxIterations = 10
const DefaultBranch = true
const DefaultCompletionSignal = "<promise>COMPLETE</promise>"

//go:embed prompts
var promptLib embed.FS

// Runner is the interface for generating responses from Claude.
type Runner interface {
	Generate(ctx context.Context, feature string) (string, error)
}

type ClaudeConfig struct {
	model            string
	quiet            bool
	maxIterations    int
	branch           bool
	completionSignal string
}

type Option func(*ClaudeConfig)

func New() *ClaudeConfig {
	return &ClaudeConfig{
		model:            DefaultOptClaudeModel,
		quiet:            DefaultOptQuiet,
		maxIterations:    DefaultMaxIterations,
		branch:           DefaultBranch,
		completionSignal: DefaultCompletionSignal,
	}
}

func (cc *ClaudeConfig) WithModel(model string) *ClaudeConfig {
	cc.model = model
	return cc
}

func (cc *ClaudeConfig) WithQuiet(quiet bool) *ClaudeConfig {
	cc.quiet = quiet
	return cc
}

func (cc *ClaudeConfig) WithMaxIterations(maxIterations int) *ClaudeConfig {
	cc.maxIterations = maxIterations
	return cc
}

// Generate sends a prompt to the Claude API and returns the generated response.
func (cc *ClaudeConfig) Generate(ctx context.Context, feature string) (string, error) {
	systemPrompt, err := promptLib.ReadFile("prompts/system_prompt.tmpl")

	cc.logInfo("Starting Gonzo")
	cc.logInfo("  Model: %s", cc.model)
	cc.logInfo("  Max Iterations: %d", cc.maxIterations)

	err = cc.ensureProgressFileExists()
	if err != nil {
		return "", fmt.Errorf("failed to ensure progress file exists: %w", err)
	}

	var out string

	for i := 1; i <= cc.maxIterations; i++ {
		cc.logInfo("===============================================================")
		cc.logInfo("  Iteration %d of %d", i, cc.maxIterations)
		cc.logInfo("===============================================================")

		var outBytes []byte

		outBytes, err = cc.callClaudeCLI(
			ctx,
			string(systemPrompt),
			feature)
		if err != nil {
			//noinspection GoErrorStringFormatInspection
			return "", fmt.Errorf("Claude CLI call failed at iteration %d: %w", i, err)
		}

		out = string(outBytes)
		if strings.Contains(out, "") {
			cc.logInfo("Task completed!")
			cc.logInfo("Completed at iteration %d of %d", i, cc.maxIterations)
			break
		}
	}

	if len(out) == 0 {
		cc.logInfo("Reached max iterations %d without completion signal", cc.maxIterations)
		return "", fmt.Errorf("reached max iterations %d without completion signal", cc.maxIterations)
	}
	return out, err
}

func (cc *ClaudeConfig) callClaudeCLI(ctx context.Context, systemPrompt string, prompt string) ([]byte, error) {
	cmd := commandContext(
		ctx,
		ClaudeCodeCli,
		"--dangerously-skip-permissions",
		"--print",
		"--model",
		cc.model,
		"--system-prompt",
		systemPrompt,
		prompt)
	return cmd.Output()
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
			Now    time.Time
			Branch bool
		}{
			Now:    time.Now(),
			Branch: cc.branch,
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
