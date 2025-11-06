package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/palaforcade/zchat/internal/config"
	contextPkg "github.com/palaforcade/zchat/internal/context"
	"github.com/palaforcade/zchat/internal/executor"
	"github.com/palaforcade/zchat/internal/llm"
	"github.com/palaforcade/zchat/internal/ui"
)

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
	collector := contextPkg.NewDefaultCollector(cfg.MaxContextLines)
	sysCtx, err := collector.Collect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting context: %v\n", err)
		os.Exit(1)
	}

	// Generate command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create LLM client based on provider
	var llmClient llm.Client
	switch cfg.Provider {
	case "anthropic":
		llmClient = llm.NewAnthropicClient(cfg.APIKey, cfg.Model)
	case "ollama":
		llmClient = llm.NewOllamaClient(cfg.OllamaURL, cfg.Model)
	default:
		fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", cfg.Provider)
		os.Exit(1)
	}

	command, err := llmClient.GenerateCommand(ctx, query, sysCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating command: %v\n", err)
		os.Exit(1)
	}

	// Display command
	display := ui.NewDisplay()
	display.ShowCommand(command)

	// Safety check
	if isDangerous, reason := executor.IsDangerous(command, cfg.DangerousPatterns); isDangerous {
		confirmed, err := display.ShowDangerWarning(reason)
		if err != nil || !confirmed {
			fmt.Println("Command execution cancelled.")
			os.Exit(0)
		}
	}

	// Confirm execution
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
		// Still show output if there is any (e.g., error messages from the command)
		if output != "" {
			fmt.Println(output)
		}
		os.Exit(1)
	}

	display.ShowSuccess(output)
}

func showUsage() {
	fmt.Println("Usage: zchat <natural language query>")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  zchat list the number of lines in analysis_data.csv")
	fmt.Println("  zchat find all python files modified in the last week")
	fmt.Println("  zchat show disk usage sorted by size")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Default provider: Ollama (local)")
	fmt.Println("  Default model: qwen2.5-coder:7b")
	fmt.Println()
	fmt.Println("  To use Anthropic instead:")
	fmt.Println("    Set ANTHROPIC_API_KEY and ZCHAT_PROVIDER=anthropic")
	fmt.Println()
	fmt.Println("  Or create ~/.config/zchat/config.yaml with:")
	fmt.Println("    provider: ollama  # or anthropic")
	fmt.Println("    model: qwen2.5-coder:7b")
	fmt.Println("    ollama_url: http://localhost:11434")
}
