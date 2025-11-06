package llm

import (
	"strings"
	"testing"

	"github.com/palaforcade/zchat/internal/context"
)

func TestBuildSystemPrompt(t *testing.T) {
	sysCtx := &context.SystemContext{
		OS:         "darwin",
		Arch:       "arm64",
		Shell:      "/bin/zsh",
		WorkingDir: "/Users/test/project",
		Files:      []string{"main.go", "README.md", "config.yaml"},
	}

	prompt := buildSystemPrompt(sysCtx)

	// Check that all context is included
	if !strings.Contains(prompt, "darwin") {
		t.Error("Prompt should contain OS")
	}

	if !strings.Contains(prompt, "arm64") {
		t.Error("Prompt should contain architecture")
	}

	if !strings.Contains(prompt, "/bin/zsh") {
		t.Error("Prompt should contain shell")
	}

	if !strings.Contains(prompt, "/Users/test/project") {
		t.Error("Prompt should contain working directory")
	}

	if !strings.Contains(prompt, "main.go") {
		t.Error("Prompt should contain files")
	}

	// Check critical instructions
	if !strings.Contains(prompt, "Output ONLY the command") {
		t.Error("Prompt should emphasize command-only output")
	}

	if !strings.Contains(prompt, "No explanations") {
		t.Error("Prompt should forbid explanations")
	}
}

func TestBuildSystemPrompt_NoFiles(t *testing.T) {
	sysCtx := &context.SystemContext{
		OS:         "linux",
		Arch:       "amd64",
		Shell:      "/bin/bash",
		WorkingDir: "/tmp",
		Files:      []string{},
	}

	prompt := buildSystemPrompt(sysCtx)

	if !strings.Contains(prompt, "(none visible)") {
		t.Error("Prompt should indicate no visible files")
	}
}

func TestParseCommandFromResponse_Clean(t *testing.T) {
	response := "ls -la"
	command, err := parseCommandFromResponse(response)

	if err != nil {
		t.Fatalf("parseCommandFromResponse() failed: %v", err)
	}

	if command != "ls -la" {
		t.Errorf("Expected 'ls -la', got '%s'", command)
	}
}

func TestParseCommandFromResponse_WithWhitespace(t *testing.T) {
	response := "  \n  ls -la  \n  "
	command, err := parseCommandFromResponse(response)

	if err != nil {
		t.Fatalf("parseCommandFromResponse() failed: %v", err)
	}

	if command != "ls -la" {
		t.Errorf("Expected 'ls -la', got '%s'", command)
	}
}

func TestParseCommandFromResponse_MarkdownCodeBlock(t *testing.T) {
	response := "```bash\nls -la\n```"
	command, err := parseCommandFromResponse(response)

	if err != nil {
		t.Fatalf("parseCommandFromResponse() failed: %v", err)
	}

	if command != "ls -la" {
		t.Errorf("Expected 'ls -la', got '%s'", command)
	}
}

func TestParseCommandFromResponse_MarkdownCodeBlock_NoLanguage(t *testing.T) {
	response := "```\nls -la\n```"
	command, err := parseCommandFromResponse(response)

	if err != nil {
		t.Fatalf("parseCommandFromResponse() failed: %v", err)
	}

	if command != "ls -la" {
		t.Errorf("Expected 'ls -la', got '%s'", command)
	}
}

func TestParseCommandFromResponse_Backticks(t *testing.T) {
	response := "`ls -la`"
	command, err := parseCommandFromResponse(response)

	if err != nil {
		t.Fatalf("parseCommandFromResponse() failed: %v", err)
	}

	if command != "ls -la" {
		t.Errorf("Expected 'ls -la', got '%s'", command)
	}
}

func TestParseCommandFromResponse_MultiLine(t *testing.T) {
	response := "```\nfind . -name '*.go' | \\\n  xargs wc -l\n```"
	command, err := parseCommandFromResponse(response)

	if err != nil {
		t.Fatalf("parseCommandFromResponse() failed: %v", err)
	}

	expected := "find . -name '*.go' | \\\n  xargs wc -l"
	if command != expected {
		t.Errorf("Expected multiline command, got '%s'", command)
	}
}

func TestParseCommandFromResponse_Empty(t *testing.T) {
	response := ""
	_, err := parseCommandFromResponse(response)

	if err == nil {
		t.Error("Expected error for empty response")
	}
}

func TestParseCommandFromResponse_OnlyWhitespace(t *testing.T) {
	response := "   \n\n   "
	_, err := parseCommandFromResponse(response)

	if err == nil {
		t.Error("Expected error for whitespace-only response")
	}
}
