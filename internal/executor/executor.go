package executor

import (
	"bytes"
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

// NewSafeExecutor creates a new executor with safety patterns
func NewSafeExecutor(patterns []string, shell string) *SafeExecutor {
	if shell == "" {
		shell = "/bin/zsh"
	}

	return &SafeExecutor{
		dangerousPatterns: patterns,
		shell:             shell,
	}
}

// Execute executes a shell command safely
func (e *SafeExecutor) Execute(ctx context.Context, command string) (string, error) {
	// Safety check (should never get here as UI checks first, but double-checking)
	if isDangerous, reason := IsDangerous(command, e.dangerousPatterns); isDangerous {
		return "", fmt.Errorf("refused to execute dangerous command: %s", reason)
	}

	// Execute command using shell
	cmd := exec.CommandContext(ctx, e.shell, "-c", command)

	// Capture output
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	cmd.Env = os.Environ()

	// Run command
	err := cmd.Run()
	if err != nil {
		return output.String(), fmt.Errorf("command execution failed: %w", err)
	}

	return output.String(), nil
}
