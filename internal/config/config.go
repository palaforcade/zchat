package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider          string   `yaml:"provider"` // "anthropic" or "ollama"
	APIKey            string   `yaml:"api_key"`
	Model             string   `yaml:"model"`
	OllamaURL         string   `yaml:"ollama_url"`
	MaxContextLines   int      `yaml:"max_context_lines"`
	DangerousPatterns []string `yaml:"dangerous_patterns"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	cfg := getDefaultConfig()

	// Try to load config file
	configPath, err := getConfigPath()
	if err == nil {
		if data, err := os.ReadFile(configPath); err == nil {
			// Config file exists, parse it
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
		// If file doesn't exist, that's OK - we'll use defaults
	}

	// Environment variables take precedence
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	}
	if provider := os.Getenv("ZCHAT_PROVIDER"); provider != "" {
		cfg.Provider = provider
	}
	if ollamaURL := os.Getenv("OLLAMA_URL"); ollamaURL != "" {
		cfg.OllamaURL = ollamaURL
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate provider
	if c.Provider != "anthropic" && c.Provider != "ollama" {
		return fmt.Errorf("invalid provider: %s (must be 'anthropic' or 'ollama')", c.Provider)
	}

	// Provider-specific validation
	if c.Provider == "anthropic" {
		if c.APIKey == "" {
			return fmt.Errorf("API key is required for Anthropic. Set ANTHROPIC_API_KEY environment variable or add api_key to ~/.config/zchat/config.yaml")
		}
	}

	return nil
}

// getDefaultConfig returns a configuration with default values
func getDefaultConfig() *Config {
	return &Config{
		Provider:        "ollama", // Default to ollama for local testing
		Model:           "qwen2.5-coder:7b",
		OllamaURL:       "http://localhost:11434",
		MaxContextLines: 20,
		DangerousPatterns: []string{
			"rm -rf /",
			"rm -rf /*",
			"rm -rf *",    // Delete all in current dir
			"rm -rf ~",
			"rm -rf $HOME",
			"> /dev/sda",
			"dd if=",
			"mkfs",
			"format",
			"diskutil",    // macOS disk utility
			":(){:|:&};:", // fork bomb
			"chmod -R 777 /",
			"| sh",        // Piping to shell
			"| bash",      // Piping to bash
			"| zsh",       // Piping to zsh
		},
	}
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zchat", "config.yaml"), nil
}
