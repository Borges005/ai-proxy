package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	LLM        LLMConfig        `yaml:"llm"`
	Guardrails GuardrailsConfig `yaml:"guardrails"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Port int `yaml:"port"`
}

// LLMConfig contains LLM-related configuration
type LLMConfig struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

// GuardrailsConfig contains guardrails-related configuration
type GuardrailsConfig struct {
	BannedWords      []string     `yaml:"banned_words"`
	RegexPatterns    []string     `yaml:"regex_patterns"`
	CustomRules      []RuleConfig `yaml:"custom_rules"`
	MaxContentLength int          `yaml:"max_content_length"`
	MaxPromptLength  int          `yaml:"max_prompt_length"`
}

// RuleConfig defines a custom rule configuration
type RuleConfig struct {
	Name       string                 `yaml:"name"`
	Type       string                 `yaml:"type"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

// LoadConfig loads the configuration from a file or environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port: 8080, // Default port
		},
		LLM: LLMConfig{
			URL: "https://api.openai.com/v1/chat/completions",
		},
		Guardrails: GuardrailsConfig{
			BannedWords:      []string{},
			RegexPatterns:    []string{},
			CustomRules:      []RuleConfig{},
			MaxContentLength: 10000, // Default maximum content length
			MaxPromptLength:  4000,  // Default maximum prompt length
		},
	}

	// Try to load from config file first
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err = yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("error parsing config file: %v", err)
			}
			logrus.Infof("Loaded configuration from %s", configPath)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error reading config file: %v", err)
		}
	}

	// Override with environment variables if they exist
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	if url := os.Getenv("LLM_URL"); url != "" {
		config.LLM.URL = url
	}

	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
	}

	if bannedWordsStr := os.Getenv("BANNED_WORDS"); bannedWordsStr != "" {
		config.Guardrails.BannedWords = strings.Split(bannedWordsStr, ",")
	}

	// Validate config
	if config.LLM.APIKey == "" {
		logrus.Warn("LLM API key is not set, requests to LLM will fail")
	}

	return config, nil
}
