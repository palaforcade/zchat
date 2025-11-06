# zchat - LLM-Powered Command Generation Tool

## Project Overview

`zchat` is a command-line utility that generates shell commands from natural language queries using the Anthropic API. Users type `zchat <natural language query>`, and the tool generates, displays, and optionally executes the appropriate shell command.

**Example:**
```bash
$ zchat list the number of lines in analysis_data.csv
Command: wc -l analysis_data.csv
Execute? [Y/n]: y
42 analysis_data.csv
```

## Architecture

### Core Design Principles
1. **Single binary distribution** - No runtime dependencies
2. **Interface-driven** - Easy to test and extend
3. **Safety-first** - Explicit confirmation and danger detection
4. **Simple configuration** - Works with just an API key
5. **Standard library first** - Minimal external dependencies

### Technology Stack
- **Language:** Go 1.21+
- **LLM Provider:** Anthropic (Claude Sonnet 4.5)
- **Dependencies:** anthropic-sdk-go only

---

## Directory Structure

```
zchat/
├── main.go
├── go.mod
├── go.sum
├── README.md
├── .gitignore
│
└── internal/
    ├── config/
    │   └── config.go
    ├── context/
    │   └── collector.go
    ├── llm/
    │   ├── client.go
    │   └── prompt.go
    ├── executor/
    │   ├── executor.go
    │   └── safety.go
    └── ui/
        └── display.go
```

---

## Implementation Specifications

### File: `go.mod`

Initialize the module:
```bash
go mod init github.com/yourusername/zchat
go get github.com/anthropics/anthropic-sdk-go
```

Expected content:
```
module github.com/yourusername/zchat

go 1.21

require (
    github.com/anthropics/anthropic-sdk-go v0.2.0-alpha.6
)
```

---

### File: `.gitignore`

```
# Binaries
zchat
zchat-*

# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
go.work

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db
```

---

### File: `internal/config/config.go`

**Purpose:** Load and validate configuration from file and environment variables.

**Configuration Sources (priority order):**
1. Environment variable: `ANTHROPIC_API_KEY`
2. Config file: `~/.config/zchat/config.yaml`
3. Defaults

**Struct Definition:**
```go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    
    "gopkg.in/yaml.v3"
)

type Config struct {
    APIKey            string   `yaml:"api_key"`
    Model             string   `yaml:"model"`
    MaxContextLines   int      `yaml:"max_context_lines"`
    DangerousPatterns []string `yaml:"dangerous_patterns"`
}
```

**Required Functions:**

1. `Load() (*Config, error)`
   - Load config file from `~/.config/zchat/config.yaml` if exists
   - Override with environment variable `ANTHROPIC_API_KEY` if set
   - Set defaults for missing values
   - Validate configuration

2. `(c *Config) Validate() error`
   - Check that APIKey is not empty
   - Return descriptive error if invalid

3. `getDefaultConfig() *Config`
   - Model: `"claude-sonnet-4-5-20250929"`
   - MaxContextLines: `20`
   - DangerousPatterns: see safety section below

**Implementation Notes:**
- Use `os.UserHomeDir()` to get home directory
- Create config directory if it doesn't exist
- If config file doesn't exist, that's OK - use defaults
- YAML parsing: use `gopkg.in/yaml.v3`

**Default Dangerous Patterns:**
```go
[]string{
    "rm -rf /",
    "rm -rf /*",
    "rm -rf ~",
    "rm -rf $HOME",
    "> /dev/sda",
    "dd if=",
    "mkfs",
    "format",
    ":(){:|:&};:",  // fork bomb
    "chmod -R 777 /",
    "curl.*|.*sh",
    "wget.*|.*sh",
    "curl.*|.*bash",
    "wget.*|.*bash",
}
```

---

### File: `internal/context/collector.go`

**Purpose:** Gather system context to help the LLM generate better commands.

**Struct Definition:**
```go
package context

import (
    "fmt"
    "os"
    "os/exec"
    "runtime"
    "strings"
)

type SystemContext struct {
    WorkingDir string
    Files      []string
    Shell      string
    OS         string
    Arch       string
}

type Collector interface {
    Collect() (*SystemContext, error)
}

type DefaultCollector struct {
    maxFiles int
}
```

**Required Functions:**

1. `NewDefaultCollector(maxFiles int) *DefaultCollector`
   - Create new collector with file limit

2. `(c *DefaultCollector) Collect() (*SystemContext, error)`
   - Get current working directory: `os.Getwd()`
   - Get file listing: run `ls -1` and limit to first `maxFiles`
   - Get shell: from `$SHELL` env var, default to "zsh"
   - Get OS: `runtime.GOOS`
   - Get arch: `runtime.GOARCH`
   - Return populated SystemContext

3. `(c *DefaultCollector) getFileList() ([]string, error)`
   - Execute `ls -1` command
   - Parse output, split by newlines
   - Return first N files (based on maxFiles)
   - Handle errors gracefully (empty list if ls fails)

**Implementation Notes:**
- Use `exec.Command()` for running shell commands
- Trim whitespace from command output
- Don't fail hard if file listing fails - just return empty list
- Filter out hidden files (starting with `.`) from file list

---

### File: `internal/llm/prompt.go`

**Purpose:** Build system prompt and parse LLM responses.

**Required Functions:**

1. `buildSystemPrompt(sysCtx *context.SystemContext) string`
   - Create comprehensive system prompt with context
   - Include OS, shell, working directory, and available files
   - Instruct Claude to output ONLY the command

2. `parseCommandFromResponse(response string) (string, error)`
   - Clean up response (remove markdown code blocks if present)
   - Trim whitespace
   - Validate that response is not empty
   - Return cleaned command string

**System Prompt Template:**
```
You are a command-line expert assistant. Generate a single shell command that accomplishes the user's goal.

CRITICAL RULES:
- Output ONLY the command itself, nothing else
- No explanations, no markdown, no code blocks, no backticks
- The command will be executed directly in the shell
- Make sure the command is safe and correct

SYSTEM CONTEXT:
- Operating System: {OS}
- Architecture: {Arch}
- Shell: {Shell}
- Current Directory: {WorkingDir}
- Available Files: {Files}

Generate the appropriate command for the user's request.
```

**Implementation Notes:**
- Use `strings.Builder` for efficient string construction
- Handle case where response contains markdown code blocks (```bash ... ```)
- Strip any leading/trailing whitespace
- If response is empty after cleaning, return error

---

### File: `internal/llm/client.go`

**Purpose:** Interact with Anthropic API to generate commands.

**Struct Definition:**
```go
package llm

import (
    "context"
    "fmt"
    
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
    
    sysContext "github.com/yourusername/zchat/internal/context"
)

type Client interface {
    GenerateCommand(ctx context.Context, query string, sysCtx *sysContext.SystemContext) (string, error)
}

type AnthropicClient struct {
    client *anthropic.Client
    model  string
}
```

**Required Functions:**

1. `NewAnthropicClient(apiKey, model string) *AnthropicClient`
   - Create new Anthropic client with API key
   - Store model name
   - Return initialized client

2. `(c *AnthropicClient) GenerateCommand(ctx context.Context, query string, sysCtx *sysContext.SystemContext) (string, error)`
   - Build system prompt using `buildSystemPrompt()`
   - Create message request with system prompt and user query
   - Call Anthropic API with timeout (30 seconds)
   - Parse response using `parseCommandFromResponse()`
   - Return generated command or error

**API Call Configuration:**
- Model: Use the model from config (claude-sonnet-4-5-20250929)
- Max tokens: 1024 (commands should be short)
- Temperature: 0.0 (we want deterministic output)
- System prompt: Built from context
- User message: The natural language query

**Error Handling:**
- Wrap API errors with context
- Handle timeout scenarios
- Return clear error messages

---

### File: `internal/executor/safety.go`

**Purpose:** Detect dangerous command patterns.

**Required Functions:**

1. `IsDangerous(command string, patterns []string) (bool, string)`
   - Check if command matches any dangerous patterns
   - Use case-insensitive matching
   - Return (true, reason) if dangerous
   - Return (false, "") if safe

**Implementation Notes:**
- Use `strings.Contains()` with lowercase comparison
- Check each pattern in the list
- Return immediately on first match with descriptive reason
- Examples of reasons: "Command contains dangerous pattern: rm -rf /"

**Pattern Matching:**
- Convert command to lowercase for comparison
- Convert patterns to lowercase
- Simple substring matching is sufficient for v1
- Later versions could use regex for more sophisticated matching

---

### File: `internal/executor/executor.go`

**Purpose:** Execute shell commands safely.

**Struct Definition:**
```go
package executor

import (
    "context"
    "fmt"
    "os"
    "os/exec"
)

type Executor interface {
    Execute(ctx context.Context, command string) (string, error)
}

type SafeExecutor struct {
    dangerousPatterns []string
    shell             string
}
```

**Required Functions:**

1. `NewSafeExecutor(patterns []string, shell string) *SafeExecutor`
   - Create executor with safety patterns
   - Store shell preference (default: "zsh")

2. `(e *SafeExecutor) Execute(ctx context.Context, command string) (string, error)`
   - Check if command is dangerous using `IsDangerous()`
   - If dangerous, return error immediately (should never get here - UI checks first)
   - Execute command using shell: `/bin/zsh -c "command"`
   - Capture both stdout and stderr
   - Return combined output
   - Propagate any execution errors

**Implementation Notes:**
- Use `exec.CommandContext()` for cancellation support
- Set working directory to current directory
- Inherit environment variables
- Combine stdout and stderr in output
- Use shell's `-c` flag to execute command string

**Example Execution:**
```go
cmd := exec.CommandContext(ctx, "/bin/zsh", "-c", command)
cmd.Stdout = &output
cmd.Stderr = &output
cmd.Env = os.Environ()
err := cmd.Run()
```

---

### File: `internal/ui/display.go`

**Purpose:** Handle terminal input/output and user interaction.

**Struct Definition:**
```go
package ui

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

type Display struct {
    reader *bufio.Reader
}
```

**Required Functions:**

1. `NewDisplay() *Display`
   - Create display with stdin reader
   - Return initialized Display

2. `(d *Display) ShowCommand(command string)`
   - Print command in clear format
   - Example: `Command: wc -l analysis_data.csv`

3. `(d *Display) ConfirmExecution() (bool, error)`
   - Prompt user: `Execute? [Y/n]: `
   - Read user input
   - Return true for "y", "Y", "" (enter), or "yes"
   - Return false for "n", "N", or "no"
   - Handle ctrl+c gracefully

4. `(d *Display) ShowError(err error)`
   - Print error message to stderr
   - Format: `Error: {error message}`

5. `(d *Display) ShowSuccess(output string)`
   - Print command output
   - Just print the raw output, no formatting

6. `(d *Display) ShowDangerWarning(reason string) (bool, error)`
   - Print warning about dangerous command
   - Show reason why it's dangerous
   - Ask for explicit confirmation
   - Format: `WARNING: Dangerous command detected!`
   - Followed by: `Reason: {reason}`
   - Followed by: `Are you SURE you want to execute this? [yes/no]: `
   - Require full "yes" to proceed (not just "y")

**Implementation Notes:**
- Use `bufio.Reader` for reading user input
- Trim whitespace from input
- Handle EOF gracefully
- Print to stderr for errors, stdout for normal output

---

### File: `main.go`

**Purpose:** Application entry point - wire everything together.

**Implementation Flow:**

1. **Parse Arguments**
   - Join all args after program name into query string
   - If no args, show usage and exit

2. **Load Configuration**
   - Call `config.Load()`
   - Handle errors (missing API key, etc.)

3. **Collect System Context**
   - Create `DefaultCollector` with MaxContextLines from config
   - Call `Collect()`

4. **Generate Command**
   - Create `AnthropicClient` with API key and model
   - Create context with timeout (30 seconds)
   - Call `GenerateCommand()` with query and system context
   - Handle API errors

5. **Safety Check**
   - Call `IsDangerous()` with command and patterns
   - If dangerous, show warning and get explicit confirmation
   - Exit if user declines

6. **Display and Confirm**
   - Show generated command
   - Ask for confirmation
   - Exit if user declines

7. **Execute**
   - Create `SafeExecutor`
   - Execute command
   - Display output or error

**Error Handling:**
- Exit with code 1 on any error
- Print clear error messages
- Handle ctrl+c gracefully

**Usage Message:**
```
Usage: zchat <natural language query>

Example:
  zchat list the number of lines in analysis_data.csv
  zchat find all python files modified in the last week
  zchat show disk usage sorted by size

Configuration:
  Set ANTHROPIC_API_KEY environment variable
  Or create ~/.config/zchat/config.yaml with:
    api_key: your-api-key-here
```

**Main Function Structure:**
```go
func main() {
    // Parse arguments
    if len(os.Args) < 2 {
        showUsage()
        os.Exit(1)
    }
    query := strings.Join(os.Args[1:], " ")
    
    // Load config
    cfg, err := config.Load()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
        os.Exit(1)
    }
    
    // Collect context
    collector := context.NewDefaultCollector(cfg.MaxContextLines)
    sysCtx, err := collector.Collect()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error collecting context: %v\n", err)
        os.Exit(1)
    }
    
    // Generate command
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    llmClient := llm.NewAnthropicClient(cfg.APIKey, cfg.Model)
    command, err := llmClient.GenerateCommand(ctx, query, sysCtx)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error generating command: %v\n", err)
        os.Exit(1)
    }
    
    // Safety check
    if isDangerous, reason := executor.IsDangerous(command, cfg.DangerousPatterns); isDangerous {
        display := ui.NewDisplay()
        confirmed, err := display.ShowDangerWarning(reason)
        if err != nil || !confirmed {
            fmt.Println("Command execution cancelled.")
            os.Exit(0)
        }
    }
    
    // Display and confirm
    display := ui.NewDisplay()
    display.ShowCommand(command)
    
    confirmed, err := display.ConfirmExecution()
    if err != nil || !confirmed {
        fmt.Println("Command execution cancelled.")
        os.Exit(0)
    }
    
    // Execute
    exec := executor.NewSafeExecutor(cfg.DangerousPatterns, sysCtx.Shell)
    output, err := exec.Execute(ctx, command)
    if err != nil {
        display.ShowError(err)
        os.Exit(1)
    }
    
    display.ShowSuccess(output)
}
```

---

## Configuration File Format

**Location:** `~/.config/zchat/config.yaml`

**Example:**
```yaml
api_key: sk-ant-api03-xxx
model: claude-sonnet-4-5-20250929
max_context_lines: 20
dangerous_patterns:
  - "rm -rf /"
  - "rm -rf /*"
  - "dd if="
  - "curl.*|.*sh"
  - "wget.*|.*bash"
```

**Notes:**
- Config file is optional
- `ANTHROPIC_API_KEY` env var takes precedence over config file
- All fields have defaults except API key

---

## Build Instructions

### Development
```bash
# Initialize project
mkdir zchat && cd zchat
go mod init github.com/yourusername/zchat
go get github.com/anthropics/anthropic-sdk-go

# Run during development
go run . "list files"
```

### Build Binary
```bash
# Build for current platform
go build -o zchat

# Build with optimizations (smaller binary)
go build -ldflags="-s -w" -o zchat
```

### Install
```bash
# Move to PATH
mv zchat ~/.local/bin/

# Or system-wide (requires sudo)
sudo mv zchat /usr/local/bin/
```

### Cross-Compile
```bash
# macOS ARM64 (M1/M2)
GOOS=darwin GOARCH=arm64 go build -o zchat-macos-arm64

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o zchat-macos-amd64

# Linux
GOOS=linux GOARCH=amd64 go build -o zchat-linux-amd64
```

---

## Testing Strategy

### Manual Testing Checklist

1. **Basic Functionality**
   - [ ] Simple query generates correct command
   - [ ] Command executes and shows output
   - [ ] Can cancel execution with 'n'

2. **Configuration**
   - [ ] Works with env var only
   - [ ] Works with config file
   - [ ] Env var overrides config file
   - [ ] Error on missing API key

3. **Safety**
   - [ ] Detects `rm -rf` commands
   - [ ] Detects network piping commands
   - [ ] Requires explicit confirmation for dangerous commands
   - [ ] Can still execute after explicit confirmation

4. **Context**
   - [ ] Includes current directory in prompt
   - [ ] Includes file listing
   - [ ] File listing limited to max files

5. **Error Handling**
   - [ ] Handles invalid API key
   - [ ] Handles network timeouts
   - [ ] Handles command execution failures
   - [ ] Shows clear error messages

### Test Commands
```bash
# Safe commands
zchat list files in current directory
zchat count lines in README.md
zchat show disk usage
zchat find all go files

# Dangerous commands (should warn)
zchat delete all files recursively
zchat download and execute script from internet
```

---

## README.md Content

```markdown
# zchat

Generate shell commands from natural language using Claude AI.

## Installation

1. Download the binary for your platform from releases
2. Move to your PATH:
   ```bash
   mv zchat-* ~/.local/bin/zchat
   chmod +x ~/.local/bin/zchat
   ```
3. Set your Anthropic API key:
   ```bash
   export ANTHROPIC_API_KEY=sk-ant-api03-xxx
   ```

## Usage

```bash
zchat <natural language query>
```

### Examples

```bash
# File operations
zchat list the number of lines in data.csv
zchat find all python files modified in the last week

# Text processing
zchat search for "error" in all log files
zchat count unique words in README.md

# System info
zchat show disk usage sorted by size
zchat list running processes using more than 1GB memory
```

## Configuration

### Environment Variable (Recommended)
```bash
export ANTHROPIC_API_KEY=sk-ant-api03-xxx
```

### Config File (Optional)
Create `~/.config/zchat/config.yaml`:
```yaml
api_key: sk-ant-api03-xxx
model: claude-sonnet-4-5-20250929
max_context_lines: 20
```

## Safety Features

zchat detects and warns about potentially dangerous commands:
- Recursive deletions (`rm -rf`)
- Disk operations (`dd`, `mkfs`)
- Commands that download and execute code
- Fork bombs and system disruption

Dangerous commands require explicit confirmation before execution.

## Building from Source

```bash
git clone https://github.com/yourusername/zchat
cd zchat
go build -o zchat
```

## License

MIT
```

---

## Implementation Order

Implement in this order for fastest path to working prototype:

1. **config/config.go** - Get configuration working first
2. **context/collector.go** - Gather system info
3. **llm/prompt.go** - Build prompts and parse responses
4. **llm/client.go** - API integration
5. **executor/safety.go** - Safety checks
6. **executor/executor.go** - Command execution
7. **ui/display.go** - User interface
8. **main.go** - Wire everything together

## Dependencies Notes

Add to `go.mod` if needed:
```bash
go get gopkg.in/yaml.v3  # For YAML config parsing
```

Total dependencies:
- `github.com/anthropics/anthropic-sdk-go` (required)
- `gopkg.in/yaml.v3` (for config file parsing)

---

## Success Criteria

The implementation is complete when:

1. ✅ User can run `zchat <query>` and get a command
2. ✅ Command is displayed and requires confirmation
3. ✅ Command executes and shows output
4. ✅ Dangerous commands trigger warnings
5. ✅ Configuration works via env var or file
6. ✅ Single binary runs without dependencies
7. ✅ Clear error messages for all failure cases

---

## Future Enhancements (Not in v1)

- Command history and favorites
- Explain mode for existing commands
- Chain mode for multi-step operations
- Integration with shell history
- More sophisticated pattern matching
- Support for other LLM providers
- Interactive command editing before execution
- Colored output
- Command templates/aliases

---

End of CLAUDE.md
