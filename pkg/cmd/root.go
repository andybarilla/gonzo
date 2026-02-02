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
	ModelClaudeHaiku:  {gonzo.CLAUDE_HAIKU},
	ModelClaudeSonnet: {gonzo.CLAUDE_SONNET},
	ModelClaudeOpus:   {gonzo.CLAUDE_OPUS},
}

var llmModel = ModelClaudeOpus
var maxIterations int
var quiet bool

// claudeGenerate is a variable that wraps gonzo.ClaudeGenerate for testing.
var claudeGenerate = gonzo.ClaudeGenerate

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gonzo",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
		fmt.Sprintf("Language model to use (options: %s, %s, %s)", gonzo.CLAUDE_HAIKU, gonzo.CLAUDE_SONNET, gonzo.CLAUDE_OPUS))

	rootCmd.PersistentFlags().IntVarP(
		&maxIterations,
		"max-iterations",
		"i",
		10,
		"Maximum number of iterations")

	rootCmd.PersistentFlags().BoolVarP(
		&quiet,
		"quiet", "q", false,
		"Disable output messages",
	)
}

func runClaudePrompt(cmd *cobra.Command, args []string) {
	var prompt string

	// Check if stdin is a pipe (has data)
	stdinStat, _ := os.Stdin.Stat()
	stdinIsPipe := (stdinStat.Mode() & os.ModeCharDevice) == 0

	if len(args) > 0 {
		prompt = strings.Join(args, " ")
	} else if stdinIsPipe {
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		prompt = strings.Join(lines, "\n")
	}

	if prompt == "" {
		_ = cmd.Help()
		return
	}

	response, err := claudeGenerate(cmd.Context(), llmModelNames[llmModel][0], prompt, quiet)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)
}
