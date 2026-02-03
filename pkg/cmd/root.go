package cmd

import (
	"bufio"
	"fmt"
	"gonzo/pkg/config"
	"gonzo/pkg/gonzo"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thediveo/enumflag/v2"
)

type LLMModel enumflag.Flag

const (
	ModelClaudeHaiku LLMModel = iota + 1
	ModelClaudeSonnet
	ModelClaudeOpus
)

var llmModelNames = map[LLMModel][]string{
	ModelClaudeHaiku:  {gonzo.ClaudeHaiku},
	ModelClaudeSonnet: {gonzo.ClaudeSonnet},
	ModelClaudeOpus:   {gonzo.ClaudeOpus},
}

var llmModel = ModelClaudeOpus
var maxIterations int
var quiet bool
var branch bool
var tests bool
var pr bool

// newRunner creates a new gonzo.Runner. Replaceable for testing.
var newRunner = func(model string, quiet bool, maxIter int, branch bool, tests bool, pr bool) gonzo.Runner {
	return gonzo.New().WithModel(model).WithQuiet(quiet).WithMaxIterations(maxIter).WithBranch(branch).WithTests(tests).WithPR(pr)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gonzo [flags] feature",
	Short: "Implementation of the Ralph Technique for LLMs",
	Long: `Gonzo is a CLI that encapsulates Claude Code.
It uses iterative prompting to refine responses from the model by running
multiple iterations.

The feature can be specified as:
  - A direct feature description: gonzo "add a login button"
  - A path to a file containing the feature: gonzo feature.txt
  - Via stdin: echo "add a login button" | gonzo

Configuration can be provided via:
  - Command-line flags (highest priority)
  - Environment variables (GONZO_ prefix, e.g., GONZO_MODEL, GONZO_MAX_ITERATIONS)
  - Config file (~/.gonzo.yaml, ~/.config/gonzo/gonzo.yaml, or ./gonzo.yaml)
  - Default values (lowest priority)`,
	Args:              cobra.ArbitraryArgs,
	PersistentPreRunE: initConfig,
	Run:               runClaudePrompt,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// initConfig initializes Viper configuration and binds flags.
// This is called as PersistentPreRunE to ensure config is loaded before the command runs.
func initConfig(cmd *cobra.Command, args []string) error {
	// Initialize Viper with defaults, config file, and env vars
	if err := config.Init(); err != nil {
		return err
	}

	// Bind Cobra flags to Viper
	if err := config.BindFlags(cmd); err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.PersistentFlags().VarP(
		enumflag.New(&llmModel, "model", llmModelNames, enumflag.EnumCaseInsensitive),
		"model", "m",
		fmt.Sprintf("Language model to use (options: %s, %s, %s)", gonzo.ClaudeHaiku, gonzo.ClaudeSonnet, gonzo.ClaudeOpus))

	rootCmd.PersistentFlags().IntVarP(
		&maxIterations,
		"max-iterations",
		"i",
		config.DefaultMaxIterations,
		"Maximum number of iterations")

	rootCmd.PersistentFlags().BoolVarP(
		&quiet,
		"quiet", "q", config.DefaultQuiet,
		"Disable output messages")

	rootCmd.PersistentFlags().BoolVarP(
		&branch,
		"branch", "b", config.DefaultBranch,
		"Create a new git branch for the changes")

	rootCmd.PersistentFlags().BoolVarP(
		&tests,
		"tests", "t", config.DefaultTests,
		"Implement tests as part of the quality checks")

	rootCmd.PersistentFlags().BoolVarP(
		&pr,
		"pr", "p", config.DefaultPR,
		"Create a pull request if one does not already exist for this branch")
}

func runClaudePrompt(cmd *cobra.Command, args []string) {
	var feature string

	// Check if stdin is a pipe (has data)
	stdinStat, _ := os.Stdin.Stat()
	stdinIsPipe := (stdinStat.Mode() & os.ModeCharDevice) == 0

	if len(args) > 0 {
		feature = strings.Join(args, " ")
		// Check if feature is a single argument that looks like a file path
		if len(args) == 1 {
			if content, err := readFeatureFromFile(args[0]); err == nil {
				feature = content
			}
		}
	} else if stdinIsPipe {
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		feature = strings.Join(lines, "\n")
	}

	if feature == "" {
		_ = cmd.Help()
		return
	}

	// Get config values from Viper (which already merged flag, env, and config file values)
	// For the model, check if the flag was explicitly set; otherwise use Viper's value
	modelValue := llmModelNames[llmModel][0]
	if !cmd.Flags().Changed(config.KeyModel) {
		// Flag wasn't explicitly set, check Viper (env var or config file)
		viperModel := viper.GetString(config.KeyModel)
		if viperModel != "" {
			modelValue = viperModel
		}
	}

	runner := newRunner(
		modelValue,
		viper.GetBool(config.KeyQuiet),
		viper.GetInt(config.KeyMaxIterations),
		viper.GetBool(config.KeyBranch),
		viper.GetBool(config.KeyTests),
		viper.GetBool(config.KeyPR),
	)

	response, err := runner.Generate(cmd.Context(), feature)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)
}

// readFeatureFromFile attempts to read feature content from a file.
// If the path exists and is a regular file, it returns the file contents.
// Otherwise, it returns an error indicating the argument should be treated as a feature string.
func readFeatureFromFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	// Only read regular files (not directories, etc.)
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}
