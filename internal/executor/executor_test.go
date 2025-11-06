package executor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewSafeExecutor(t *testing.T) {
	patterns := []string{"rm -rf"}
	shell := "/bin/bash"

	exec := NewSafeExecutor(patterns, shell)

	if exec.shell != shell {
		t.Errorf("Expected shell '%s', got '%s'", shell, exec.shell)
	}

	if len(exec.dangerousPatterns) != len(patterns) {
		t.Errorf("Expected %d patterns, got %d", len(patterns), len(exec.dangerousPatterns))
	}
}

func TestNewSafeExecutor_DefaultShell(t *testing.T) {
	exec := NewSafeExecutor([]string{}, "")

	if exec.shell != "/bin/zsh" {
		t.Errorf("Expected default shell '/bin/zsh', got '%s'", exec.shell)
	}
}

func TestExecute_Success(t *testing.T) {
	exec := NewSafeExecutor([]string{}, "/bin/zsh")
	ctx := context.Background()

	output, err := exec.Execute(ctx, "echo 'test'")

	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if !strings.Contains(output, "test") {
		t.Errorf("Expected output to contain 'test', got '%s'", output)
	}
}

func TestExecute_CommandWithPipe(t *testing.T) {
	exec := NewSafeExecutor([]string{}, "/bin/zsh")
	ctx := context.Background()

	output, err := exec.Execute(ctx, "echo 'hello' | wc -c")

	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// wc -c counts characters, "hello\n" is 6 characters
	if !strings.Contains(strings.TrimSpace(output), "6") {
		t.Errorf("Expected output to contain '6', got '%s'", output)
	}
}

func TestExecute_CommandFailure(t *testing.T) {
	exec := NewSafeExecutor([]string{}, "/bin/zsh")
	ctx := context.Background()

	_, err := exec.Execute(ctx, "nonexistentcommand12345")

	if err == nil {
		t.Error("Expected error for non-existent command")
	}
}

func TestExecute_DangerousCommand(t *testing.T) {
	patterns := []string{"rm -rf"}
	exec := NewSafeExecutor(patterns, "/bin/zsh")
	ctx := context.Background()

	_, err := exec.Execute(ctx, "rm -rf /tmp/test")

	if err == nil {
		t.Error("Expected error for dangerous command")
	}

	if !strings.Contains(err.Error(), "dangerous") {
		t.Errorf("Error should mention dangerous command, got: %v", err)
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	exec := NewSafeExecutor([]string{}, "/bin/zsh")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Command that takes longer than timeout
	_, err := exec.Execute(ctx, "sleep 10")

	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

func TestExecute_StderrCapture(t *testing.T) {
	exec := NewSafeExecutor([]string{}, "/bin/zsh")
	ctx := context.Background()

	// Command that writes to stderr
	output, _ := exec.Execute(ctx, "echo 'error message' >&2")

	if !strings.Contains(output, "error message") {
		t.Errorf("Expected stderr to be captured in output, got '%s'", output)
	}
}

func TestExecute_MultilineCommand(t *testing.T) {
	exec := NewSafeExecutor([]string{}, "/bin/zsh")
	ctx := context.Background()

	cmd := `echo "line1"
echo "line2"`
	output, err := exec.Execute(ctx, cmd)

	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if !strings.Contains(output, "line1") || !strings.Contains(output, "line2") {
		t.Errorf("Expected both lines in output, got '%s'", output)
	}
}
