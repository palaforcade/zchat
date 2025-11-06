package executor

import (
	"fmt"
	"strings"
)

// IsDangerous checks if a command matches any dangerous patterns
func IsDangerous(command string, patterns []string) (bool, string) {
	commandLower := strings.ToLower(command)

	for _, pattern := range patterns {
		patternLower := strings.ToLower(pattern)

		if strings.Contains(commandLower, patternLower) {
			return true, fmt.Sprintf("Command contains dangerous pattern: %s", pattern)
		}
	}

	return false, ""
}
