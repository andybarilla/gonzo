package cmd

import (
	"bufio"
	"fmt"
	"gonzo/pkg/gonzo"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
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

// newRunner creates a new gonzo.Runner. Replaceable for testing.
var newRunner = func(model string, quiet bool, maxIter int, branch bool) gonzo.Runner {
	return gonzo.New().WithModel(model).WithQuiet(quiet).WithMaxIterations(maxIter).WithBranch(branch)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gonzo [flags] feature",
	Short: "Implementation of the Ralph Technique for LLMs",
	Long: `Gonzo is a CLI that encapsulates Claude Code.
It uses iterative prompting to refine responses from the model by running
multiple iterations.`,
	Args: cobra.ArbitraryArgs,
	Run:  runClaudePrompt,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
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
		10,
		"Maximum number of iterations")

	rootCmd.PersistentFlags().BoolVarP(
		&quiet,
		"quiet", "q", false,
		"Disable output messages")

	rootCmd.PersistentFlags().BoolVarP(
		&branch,
		"branch", "b", true,
		"Create a new git branch for the changes")
}

func runClaudePrompt(cmd *cobra.Command, args []string) {
	var feature string

	// Check if stdin is a pipe (has data)
	stdinStat, _ := os.Stdin.Stat()
	stdinIsPipe := (stdinStat.Mode() & os.ModeCharDevice) == 0

	if len(args) > 0 {
		feature = strings.Join(args, " ")
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

	runner := newRunner(llmModelNames[llmModel][0], quiet, maxIterations, branch)

	response, err := runner.Generate(cmd.Context(), feature)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)
}
