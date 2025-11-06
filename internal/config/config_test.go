package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDefaultConfig(t *testing.T) {
	cfg := getDefaultConfig()

	if cfg.Provider != "ollama" {
		t.Errorf("Expected default provider 'ollama', got '%s'", cfg.Provider)
	}

	if cfg.Model != "qwen2.5-coder:7b" {
		t.Errorf("Expected default model 'qwen2.5-coder:7b', got '%s'", cfg.Model)
	}

	if cfg.OllamaURL != "http://localhost:11434" {
		t.Errorf("Expected default OllamaURL 'http://localhost:11434', got '%s'", cfg.OllamaURL)
	}

	if cfg.MaxContextLines != 20 {
		t.Errorf("Expected MaxContextLines 20, got %d", cfg.MaxContextLines)
	}

	if len(cfg.DangerousPatterns) == 0 {
		t.Error("Expected default dangerous patterns, got none")
	}
}

func TestValidate_OllamaProvider(t *testing.T) {
	cfg := &Config{
		Provider: "ollama",
		Model:    "qwen2.5-coder:7b",
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Ollama config should be valid without API key, got error: %v", err)
	}
}

func TestValidate_AnthropicProvider_NoAPIKey(t *testing.T) {
	cfg := &Config{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5-20250929",
		APIKey:   "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for Anthropic provider without API key")
	}
}

func TestValidate_AnthropicProvider_WithAPIKey(t *testing.T) {
	cfg := &Config{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5-20250929",
		APIKey:   "test-key",
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Anthropic config with API key should be valid, got error: %v", err)
	}
}

func TestValidate_InvalidProvider(t *testing.T) {
	cfg := &Config{
		Provider: "invalid",
		Model:    "test",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for invalid provider")
	}
}

func TestLoad_EnvVariableOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	os.Setenv("ZCHAT_PROVIDER", "anthropic")
	os.Setenv("OLLAMA_URL", "http://custom:1234")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ZCHAT_PROVIDER")
		os.Unsetenv("OLLAMA_URL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.APIKey != "test-api-key" {
		t.Errorf("Expected APIKey 'test-api-key', got '%s'", cfg.APIKey)
	}

	if cfg.Provider != "anthropic" {
		t.Errorf("Expected Provider 'anthropic', got '%s'", cfg.Provider)
	}

	if cfg.OllamaURL != "http://custom:1234" {
		t.Errorf("Expected OllamaURL 'http://custom:1234', got '%s'", cfg.OllamaURL)
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "zchat")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `provider: anthropic
api_key: file-api-key
model: test-model
ollama_url: http://file:5678
max_context_lines: 50
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Override HOME to use temp directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.APIKey != "file-api-key" {
		t.Errorf("Expected APIKey 'file-api-key', got '%s'", cfg.APIKey)
	}

	if cfg.Model != "test-model" {
		t.Errorf("Expected Model 'test-model', got '%s'", cfg.Model)
	}

	if cfg.MaxContextLines != 50 {
		t.Errorf("Expected MaxContextLines 50, got %d", cfg.MaxContextLines)
	}
}

func TestDangerousPatterns(t *testing.T) {
	cfg := getDefaultConfig()

	expectedPatterns := []string{
		"rm -rf /",
		"rm -rf /*",
		"dd if=",
		"mkfs",
		":(){:|:&};:",
		"| sh",
		"| bash",
	}

	for _, pattern := range expectedPatterns {
		found := false
		for _, p := range cfg.DangerousPatterns {
			if p == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected dangerous pattern '%s' not found in config", pattern)
		}
	}
}
