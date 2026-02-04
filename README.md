# Gonzo

## Overview

Gonzo is a in implementation in [Go](https://go.dev) of the [Ralph Wiggum technique](https://ghuntley.com/ralph/) and is
inspired by [Snarktank's](https://github.com/snarktank) [Ralph](https://github.com/snarktank/ralph) 
script

## Installation

### Quick Install (Recommended)

Install the latest version using the install script:

```sh
# Using curl
curl -fsSL https://raw.githubusercontent.com/andybarilla/gonzo/main/install.sh | sh

# Using wget
wget -qO- https://raw.githubusercontent.com/andybarilla/gonzo/main/install.sh | sh
```

### Install Options

```sh
# Install to a specific directory
curl -fsSL https://raw.githubusercontent.com/andybarilla/gonzo/main/install.sh | sh -s -- -b /usr/local/bin

# Install a specific version
curl -fsSL https://raw.githubusercontent.com/andybarilla/gonzo/main/install.sh | sh -s -- -t v1.0.0

# Enable debug output
curl -fsSL https://raw.githubusercontent.com/andybarilla/gonzo/main/install.sh | sh -s -- -d
```

### Manual Installation

Download the appropriate binary for your platform from the [releases page](https://github.com/andybarilla/gonzo/releases).

## Usage

### Basic Usage

Run gonzo with a feature description:

```sh
# Specify a feature directly
gonzo "add a login button to the navbar"

# Read feature from a file
gonzo feature.txt

# Pipe feature from stdin
echo "add user authentication" | gonzo
cat feature-request.md | gonzo
```

### Command Line Options

```sh
gonzo [flags] <feature>

Flags:
  -m, --model <model>        Language model to use (default: claude-opus-4-5)
                             Options: claude-haiku-3-5, claude-sonnet-4, claude-opus-4-5
  -i, --max-iterations <n>   Maximum agentic iterations before stopping (default: 10)
  -q, --quiet                Disable output messages
  -b, --branch               Create a new git branch for changes (default: true)
  -t, --tests                Run tests as part of quality checks (default: true)
  -p, --pr                   Create a pull request if one doesn't exist (default: true)
  -h, --help                 Show help
  -v, --version              Show version
```

### Examples

```sh
# Use Claude Haiku for faster, cheaper responses
gonzo -m claude-haiku-3-5 "fix the typo in README"

# Run more iterations for complex features
gonzo -i 20 "implement user authentication with JWT"

# Skip branch creation and PR
gonzo --branch=false --pr=false "quick fix for bug"

# Skip tests for documentation-only changes
gonzo --tests=false "update the API documentation"

# Quiet mode for CI/CD pipelines
gonzo -q "add CI workflow"
```

## Configuration

Gonzo supports configuration through multiple sources (in order of priority):

1. **Command-line flags** (highest priority)
2. **Environment variables** (GONZO_ prefix)
3. **Configuration file** (gonzo.yaml)
4. **Default values** (lowest priority)

### Configuration File

Create a `gonzo.yaml` file in one of these locations:
- `./gonzo.yaml` (current directory, project-specific)
- `~/.config/gonzo/gonzo.yaml` (user config directory)
- `~/gonzo.yaml` (home directory)

Example configuration:

```yaml
# Language model to use
model: claude-opus-4-5

# Maximum number of agentic iterations
max-iterations: 10

# Whether to run tests during quality checks
tests: true

# Whether to create a pull request
pr: true
```

See [gonzo.sample.yaml](gonzo.sample.yaml) for a complete example.

### Environment Variables

All configuration options can be set via environment variables with the `GONZO_` prefix:

```sh
export GONZO_MODEL=claude-sonnet-4
export GONZO_MAX_ITERATIONS=15
export GONZO_QUIET=true
export GONZO_BRANCH=true
export GONZO_TESTS=true
export GONZO_PR=false

gonzo "add a new feature"
```

## Prerequisites

- **Git**: Must be installed and configured with `user.name` and `user.email`
- **Claude Code**: Gonzo wraps Claude Code CLI - ensure it's installed and authenticated
- **gh CLI** (optional): Required for automatic PR creation (`--pr` flag)

## How It Works

Gonzo implements the [Ralph Wiggum technique](https://ghuntley.com/ralph/) for autonomous coding:

1. **Creates a branch** for your changes (configurable)
2. **Reads the progress log** at `.gonzo/progress.txt`
3. **Implements the feature** using Claude Code
4. **Runs quality checks** (typecheck, lint, tests)
5. **Commits changes** with descriptive messages
6. **Creates a pull request** (configurable)
7. **Updates the progress log** with learnings

The agent runs iteratively until the task is complete or max iterations are reached.

## Work In Progress

This is a work in progress. Features may not be complete, and bugs may exist. Use at your
own risk.
