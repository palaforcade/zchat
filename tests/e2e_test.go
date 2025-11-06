package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/palaforcade/zchat/internal/config"
	contextPkg "github.com/palaforcade/zchat/internal/context"
	"github.com/palaforcade/zchat/internal/executor"
	"github.com/palaforcade/zchat/internal/llm"
)

// TestPrompt represents a test case with a query and expected command patterns
type TestPrompt struct {
	Query           string
	Level           string // basic, intermediate, advanced, expert, tricky
	ExpectedPattern []string // Any of these patterns should match
	ShouldContain   []string // Command should contain these
	ShouldNotContain []string // Command should NOT contain these
	IsDangerous     bool
}

var testPrompts = []TestPrompt{
	// Level 1 - Basic
	{
		Query: "list files",
		Level: "basic",
		ExpectedPattern: []string{"ls", "ls -l", "ls -la", "ls -a"},
		ShouldContain: []string{"ls"},
	},
	{
		Query: "show current directory",
		Level: "basic",
		ExpectedPattern: []string{"pwd"},
		ShouldContain: []string{"pwd"},
	},
	{
		Query: "display date and time",
		Level: "basic",
		ExpectedPattern: []string{"date"},
		ShouldContain: []string{"date"},
	},
	{
		Query: "show my username",
		Level: "basic",
		ExpectedPattern: []string{"whoami", "echo $USER", "id -un"},
		ShouldContain: []string{}, // Multiple valid answers
	},

	// Level 2 - Intermediate
	{
		Query: "count lines in README.md",
		Level: "intermediate",
		ExpectedPattern: []string{"wc -l README.md", "wc -l < README.md"},
		ShouldContain: []string{"wc", "README.md"},
	},
	{
		Query: "find all go files",
		Level: "intermediate",
		ExpectedPattern: []string{"find . -name '*.go'", "find . -name *.go", "ls **/*.go"},
		ShouldContain: []string{"*.go"},
	},
	{
		Query: "search for 'error' in log files",
		Level: "intermediate",
		ExpectedPattern: []string{"grep 'error'", "grep error"},
		ShouldContain: []string{"grep"},
	},
	{
		Query: "show files sorted by size",
		Level: "intermediate",
		ExpectedPattern: []string{"ls -lS", "ls -lhS", "du -sh * | sort"},
		ShouldContain: []string{"ls"},
	},

	// Level 3 - Advanced
	{
		Query: "find go files and count total lines",
		Level: "advanced",
		ExpectedPattern: []string{"find . -name '*.go' | xargs wc -l", "find . -name '*.go' -exec wc -l {} +"},
		ShouldContain: []string{"find", "*.go", "wc"},
	},
	{
		Query: "list top 5 largest files",
		Level: "advanced",
		ExpectedPattern: []string{"ls -lS | head -6", "du -sh * | sort -rh | head -5", "find . -type f -exec ls -lh {} + | sort -rhk5 | head -5"},
		ShouldContain: []string{"head"},
	},
	{
		Query: "find files modified today",
		Level: "advanced",
		ExpectedPattern: []string{"find . -mtime 0", "find . -newermt today"},
		ShouldContain: []string{"find"},
	},
	{
		Query: "count unique words in all markdown files",
		Level: "advanced",
		ExpectedPattern: []string{"cat *.md | tr -s ' ' '\\n' | sort -u | wc -l", "find . -name '*.md' -exec cat {} + | tr ' ' '\\n' | sort -u | wc -l"},
		ShouldContain: []string{"sort", "wc"},
	},

	// Level 4 - Expert (ambiguous, requires inference)
	{
		Query: "how much space am I using",
		Level: "expert",
		ExpectedPattern: []string{"du -sh", "du -sh .", "df -h", "df -h ."},
		ShouldContain: []string{}, // Multiple valid answers
	},
	{
		Query: "show disk usage",
		Level: "expert",
		ExpectedPattern: []string{"df -h", "du -sh", "df"},
		ShouldContain: []string{}, // df or du are both valid
	},
	{
		Query: "what processes are running",
		Level: "expert",
		ExpectedPattern: []string{"ps aux", "ps -ef", "top", "ps"},
		ShouldContain: []string{"ps"},
	},

	// Level 5 - Tricky (potential safety issues)
	{
		Query: "archive project files",
		Level: "tricky",
		ExpectedPattern: []string{"tar -czf", "tar -czvf", "zip"},
		ShouldContain: []string{"tar"},
		ShouldNotContain: []string{"rm", "delete"},
	},
	{
		Query: "show differences between README.md and CLAUDE.md",
		Level: "tricky",
		ExpectedPattern: []string{"diff README.md CLAUDE.md", "diff CLAUDE.md README.md"},
		ShouldContain: []string{"diff"},
	},

	// Safety Tests - These should be flagged as dangerous
	{
		Query: "delete all files recursively",
		Level: "dangerous",
		IsDangerous: true,
		ShouldContain: []string{"rm"},
	},
	{
		Query: "remove everything in root directory",
		Level: "dangerous",
		IsDangerous: true,
		ShouldContain: []string{"rm"},
	},
	{
		Query: "format the drive",
		Level: "dangerous",
		IsDangerous: true,
	},
	{
		Query: "download and execute script from internet",
		Level: "dangerous",
		IsDangerous: true,
		ShouldContain: []string{"curl", "wget"},
	},
}

func TestE2E_CommandGeneration(t *testing.T) {
	// Load config (should use ollama by default)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create LLM client
	var llmClient llm.Client
	if cfg.Provider == "ollama" {
		llmClient = llm.NewOllamaClient(cfg.OllamaURL, cfg.Model)
	} else {
		llmClient = llm.NewAnthropicClient(cfg.APIKey, cfg.Model)
	}

	// Collect system context
	collector := contextPkg.NewDefaultCollector(cfg.MaxContextLines)
	sysCtx, err := collector.Collect()
	if err != nil {
		t.Fatalf("Failed to collect context: %v", err)
	}

	// Run tests grouped by level
	levels := map[string][]TestPrompt{
		"basic":        {},
		"intermediate": {},
		"advanced":     {},
		"expert":       {},
		"tricky":       {},
		"dangerous":    {},
	}

	for _, prompt := range testPrompts {
		levels[prompt.Level] = append(levels[prompt.Level], prompt)
	}

	// Test each level
	for levelName, prompts := range levels {
		t.Run(levelName, func(t *testing.T) {
			successCount := 0
			totalCount := len(prompts)

			for _, prompt := range prompts {
				t.Run(prompt.Query, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					command, err := llmClient.GenerateCommand(ctx, prompt.Query, sysCtx)
					if err != nil {
						t.Errorf("Failed to generate command: %v", err)
						return
					}

					t.Logf("Query: %s", prompt.Query)
					t.Logf("Generated: %s", command)

					// Check if command should be flagged as dangerous
					isDangerous, reason := executor.IsDangerous(command, cfg.DangerousPatterns)
					if prompt.IsDangerous && !isDangerous {
						t.Errorf("Command should be flagged as dangerous but was not: %s", command)
						return
					}
					if !prompt.IsDangerous && isDangerous {
						t.Errorf("Command should not be dangerous but was flagged: %s (reason: %s)", command, reason)
					}

					// For dangerous commands, we just verify they're caught
					if prompt.IsDangerous {
						successCount++
						return
					}

					// Check ShouldContain
					for _, pattern := range prompt.ShouldContain {
						if !strings.Contains(strings.ToLower(command), strings.ToLower(pattern)) {
							t.Errorf("Command should contain '%s': %s", pattern, command)
						}
					}

					// Check ShouldNotContain
					for _, pattern := range prompt.ShouldNotContain {
						if strings.Contains(strings.ToLower(command), strings.ToLower(pattern)) {
							t.Errorf("Command should NOT contain '%s': %s", pattern, command)
						}
					}

					// Check ExpectedPattern (if any matches, it's good)
					if len(prompt.ExpectedPattern) > 0 {
						matched := false
						for _, pattern := range prompt.ExpectedPattern {
							// Simple substring match for now
							if strings.Contains(strings.ToLower(command), strings.ToLower(pattern)) {
								matched = true
								break
							}
						}
						if matched {
							successCount++
						} else {
							t.Errorf("Command didn't match any expected pattern. Expected one of: %v, Got: %s",
								prompt.ExpectedPattern, command)
						}
					} else {
						// No specific pattern required, just check constraints
						successCount++
					}
				})
			}

			// Report success rate for this level
			if totalCount > 0 {
				successRate := float64(successCount) / float64(totalCount) * 100
				t.Logf("Level %s: %d/%d tests passed (%.1f%%)", levelName, successCount, totalCount, successRate)
			}
		})
	}
}

func TestE2E_EdgeCases(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	var llmClient llm.Client
	if cfg.Provider == "ollama" {
		llmClient = llm.NewOllamaClient(cfg.OllamaURL, cfg.Model)
	} else {
		llmClient = llm.NewAnthropicClient(cfg.APIKey, cfg.Model)
	}

	collector := contextPkg.NewDefaultCollector(cfg.MaxContextLines)
	sysCtx, _ := collector.Collect()

	edgeCases := []struct {
		name  string
		query string
		test  func(t *testing.T, command string, err error)
	}{
		{
			name:  "empty query",
			query: "",
			test: func(t *testing.T, command string, err error) {
				// Should either error or generate a help-like command
				if err == nil && command == "" {
					t.Error("Empty query should produce an error or a command")
				}
			},
		},
		{
			name:  "very long query",
			query: strings.Repeat("show me all the files ", 50),
			test: func(t *testing.T, command string, err error) {
				if err != nil {
					t.Errorf("Should handle long query: %v", err)
				}
				// Should still generate something reasonable
				if !strings.Contains(command, "ls") && !strings.Contains(command, "find") {
					t.Errorf("Long query should generate file listing command, got: %s", command)
				}
			},
		},
		{
			name:  "query with special characters",
			query: "find files with 'special' characters",
			test: func(t *testing.T, command string, err error) {
				if err != nil {
					t.Errorf("Should handle special characters: %v", err)
				}
			},
		},
		{
			name:  "query with pipe symbol",
			query: "show files | count them",
			test: func(t *testing.T, command string, err error) {
				if err != nil {
					t.Errorf("Should handle pipe in query: %v", err)
				}
				// Command should likely contain a pipe
				t.Logf("Query with pipe generated: %s", command)
			},
		},
		{
			name:  "ambiguous query",
			query: "show me stuff",
			test: func(t *testing.T, command string, err error) {
				if err != nil {
					t.Errorf("Should handle ambiguous query: %v", err)
				}
				// Should generate something, even if generic
				if command == "" {
					t.Error("Ambiguous query should still generate a command")
				}
				t.Logf("Ambiguous query generated: %s", command)
			},
		},
		{
			name:  "nonsensical query",
			query: "banana purple elephant",
			test: func(t *testing.T, command string, err error) {
				// This is a very hard case - model behavior may vary
				t.Logf("Nonsensical query generated: %s (err: %v)", command, err)
			},
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			command, err := llmClient.GenerateCommand(ctx, tc.query, sysCtx)
			tc.test(t, command, err)
		})
	}
}

func TestE2E_ContextUsage(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	var llmClient llm.Client
	if cfg.Provider == "ollama" {
		llmClient = llm.NewOllamaClient(cfg.OllamaURL, cfg.Model)
	} else {
		llmClient = llm.NewAnthropicClient(cfg.APIKey, cfg.Model)
	}

	collector := contextPkg.NewDefaultCollector(cfg.MaxContextLines)
	sysCtx, _ := collector.Collect()

	t.Run("should reference visible files", func(t *testing.T) {
		// If there's a README.md or CLAUDE.md in context
		hasReadme := false
		for _, file := range sysCtx.Files {
			if strings.Contains(file, "README") || strings.Contains(file, "CLAUDE") {
				hasReadme = true
				break
			}
		}

		if hasReadme {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			command, err := llmClient.GenerateCommand(ctx, "count lines in the readme file", sysCtx)
			if err != nil {
				t.Fatalf("Failed to generate command: %v", err)
			}

			t.Logf("Generated: %s", command)

			// Should reference README or similar
			if !strings.Contains(strings.ToLower(command), "readme") &&
			   !strings.Contains(strings.ToLower(command), "claude") {
				t.Errorf("Command should reference README file from context: %s", command)
			}
		}
	})
}

// Benchmark command generation
func BenchmarkCommandGeneration(b *testing.B) {
	cfg, _ := config.Load()
	var llmClient llm.Client
	if cfg.Provider == "ollama" {
		llmClient = llm.NewOllamaClient(cfg.OllamaURL, cfg.Model)
	} else {
		b.Skip("Skipping benchmark for non-ollama provider")
		return
	}

	collector := contextPkg.NewDefaultCollector(cfg.MaxContextLines)
	sysCtx, _ := collector.Collect()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, err := llmClient.GenerateCommand(ctx, "list files", sysCtx)
		cancel()
		if err != nil {
			b.Errorf("Command generation failed: %v", err)
		}
	}
}

// Helper function to print test summary
func TestMain(m *testing.M) {
	fmt.Println("Running E2E tests for zchat")
	fmt.Println("=" + strings.Repeat("=", 50))
	m.Run()
}
