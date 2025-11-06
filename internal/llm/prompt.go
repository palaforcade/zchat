package llm

import (
	"fmt"
	"strings"

	"github.com/palaforcade/zchat/internal/context"
)

// buildSystemPrompt creates a comprehensive system prompt with context
func buildSystemPrompt(sysCtx *context.SystemContext) string {
	var sb strings.Builder

	sb.WriteString("You are a command-line expert assistant. Generate a single shell command that accomplishes the user's goal.\n\n")
	sb.WriteString("CRITICAL RULES:\n")
	sb.WriteString("- Output ONLY the command itself, nothing else\n")
	sb.WriteString("- No explanations, no markdown, no code blocks, no backticks\n")
	sb.WriteString("- The command will be executed directly in the shell\n")
	sb.WriteString("- Make sure the command is safe and correct\n\n")
	sb.WriteString("SYSTEM CONTEXT:\n")
	sb.WriteString(fmt.Sprintf("- Operating System: %s\n", sysCtx.OS))
	sb.WriteString(fmt.Sprintf("- Architecture: %s\n", sysCtx.Arch))
	sb.WriteString(fmt.Sprintf("- Shell: %s\n", sysCtx.Shell))
	sb.WriteString(fmt.Sprintf("- Current Directory: %s\n", sysCtx.WorkingDir))

	if len(sysCtx.Files) > 0 {
		sb.WriteString(fmt.Sprintf("- Available Files: %s\n", strings.Join(sysCtx.Files, ", ")))
	} else {
		sb.WriteString("- Available Files: (none visible)\n")
	}

	sb.WriteString("\nGenerate the appropriate command for the user's request.")

	return sb.String()
}

// parseCommandFromResponse cleans up the LLM response and extracts the command
func parseCommandFromResponse(response string) (string, error) {
	// Trim whitespace
	response = strings.TrimSpace(response)

	// Remove markdown code blocks if present
	if strings.HasPrefix(response, "```") {
		// Find the first newline after ```
		lines := strings.Split(response, "\n")
		if len(lines) > 2 {
			// Skip first line (```bash or similar) and last line (```)
			response = strings.Join(lines[1:len(lines)-1], "\n")
			response = strings.TrimSpace(response)
		}
	}

	// Remove any backticks
	response = strings.Trim(response, "`")
	response = strings.TrimSpace(response)

	if response == "" {
		return "", fmt.Errorf("received empty response from LLM")
	}

	return response, nil
}
