package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/charmitro/ai_proxy/internal/config"
	"github.com/sirupsen/logrus"
)

// Client is an interface for LLM clients
type Client interface {
	Query(prompt string, modelParams map[string]interface{}) (string, int, error)
}

// OpenAIClient implements Client for OpenAI API
type OpenAIClient struct {
	config     *config.LLMConfig
	httpClient *http.Client
}

// OpenAIChatRequest represents the request structure for OpenAI chat completions API
type OpenAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	// Add other parameters as needed
}

// OpenAIChatMessage represents a message in the OpenAI chat API
type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIChatResponse represents the response from OpenAI chat completions API
type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewOpenAIClient creates a new OpenAI API client
func NewOpenAIClient(config *config.LLMConfig) *OpenAIClient {
	return &OpenAIClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Query sends a prompt to the OpenAI API and returns the response
func (c *OpenAIClient) Query(prompt string, modelParams map[string]interface{}) (string, int, error) {
	// Extract model parameters with defaults
	model := "gpt-3.5-turbo"
	temperature := 0.7
	maxTokens := 256

	if val, ok := modelParams["model"]; ok {
		if modelStr, ok := val.(string); ok {
			model = modelStr
		}
	}

	if val, ok := modelParams["temperature"]; ok {
		if tempFloat, ok := val.(float64); ok {
			temperature = tempFloat
		}
	}

	if val, ok := modelParams["max_tokens"]; ok {
		if tokensInt, ok := val.(float64); ok {
			maxTokens = int(tokensInt)
		}
	}

	// Prepare request
	reqBody := OpenAIChatRequest{
		Model: model,
		Messages: []OpenAIChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("error marshalling request: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", c.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("error sending request to LLM: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("error reading response body: %v", err)
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("LLM API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openAIResp OpenAIChatResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", 0, fmt.Errorf("error parsing response: %v", err)
	}

	// Extract completion text
	if len(openAIResp.Choices) == 0 {
		return "", 0, fmt.Errorf("no completion found in response")
	}

	completion := openAIResp.Choices[0].Message.Content
	totalTokens := openAIResp.Usage.TotalTokens

	logrus.Infof("LLM query successful, tokens used: %d", totalTokens)
	return completion, totalTokens, nil
}
