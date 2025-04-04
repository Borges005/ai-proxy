package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test loading from file
	cfg, err := LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check server config
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
	}

	// Check LLM config
	if cfg.LLM.URL != "https://api.openai.com/v1/chat/completions" {
		t.Errorf("Expected OpenAI URL, got %s", cfg.LLM.URL)
	}

	if cfg.LLM.APIKey != "test-api-key" {
		t.Errorf("Expected test API key, got %s", cfg.LLM.APIKey)
	}

	// Check guardrails config
	if len(cfg.Guardrails.BannedWords) != 2 {
		t.Errorf("Expected 2 banned words, got %d", len(cfg.Guardrails.BannedWords))
	}

	if len(cfg.Guardrails.RegexPatterns) != 1 {
		t.Errorf("Expected 1 regex pattern, got %d", len(cfg.Guardrails.RegexPatterns))
	}

	if cfg.Guardrails.MaxContentLength != 8000 {
		t.Errorf("Expected max content length 8000, got %d", cfg.Guardrails.MaxContentLength)
	}

	if cfg.Guardrails.MaxPromptLength != 2000 {
		t.Errorf("Expected max prompt length 2000, got %d", cfg.Guardrails.MaxPromptLength)
	}

	if len(cfg.Guardrails.CustomRules) != 1 {
		t.Errorf("Expected 1 custom rule, got %d", len(cfg.Guardrails.CustomRules))
	}

	if cfg.Guardrails.CustomRules[0].Name != "Test Rule" {
		t.Errorf("Expected custom rule name 'Test Rule', got %s", cfg.Guardrails.CustomRules[0].Name)
	}

	// Test env var override
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("LLM_API_KEY", "env-api-key")
	os.Setenv("BANNED_WORDS", "evil,bad,harmful")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("LLM_API_KEY")
		os.Unsetenv("BANNED_WORDS")
	}()

	cfg, err = LoadConfig("testdata/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check overridden values
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090 from env, got %d", cfg.Server.Port)
	}

	if cfg.LLM.APIKey != "env-api-key" {
		t.Errorf("Expected env API key, got %s", cfg.LLM.APIKey)
	}

	if len(cfg.Guardrails.BannedWords) != 3 {
		t.Errorf("Expected 3 banned words from env, got %d", len(cfg.Guardrails.BannedWords))
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Test loading with non-existent file (should use defaults)
	cfg, err := LoadConfig("non-existent-file.yaml")
	if err != nil {
		t.Fatalf("Failed to load config with defaults: %v", err)
	}

	// Check default values
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.LLM.URL != "https://api.openai.com/v1/chat/completions" {
		t.Errorf("Expected default OpenAI URL, got %s", cfg.LLM.URL)
	}

	if len(cfg.Guardrails.BannedWords) != 0 {
		t.Errorf("Expected empty banned words list by default, got %d items", len(cfg.Guardrails.BannedWords))
	}

	if len(cfg.Guardrails.RegexPatterns) != 0 {
		t.Errorf("Expected empty regex patterns list by default, got %d items", len(cfg.Guardrails.RegexPatterns))
	}

	if cfg.Guardrails.MaxContentLength != 10000 {
		t.Errorf("Expected default max content length 10000, got %d", cfg.Guardrails.MaxContentLength)
	}

	if cfg.Guardrails.MaxPromptLength != 4000 {
		t.Errorf("Expected default max prompt length 4000, got %d", cfg.Guardrails.MaxPromptLength)
	}

	if len(cfg.Guardrails.CustomRules) != 0 {
		t.Errorf("Expected empty custom rules list by default, got %d items", len(cfg.Guardrails.CustomRules))
	}
}
