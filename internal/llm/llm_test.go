package llm

import (
	"net/http"
	"net/http/httptest"
	"testing"
	
	"github.com/charmitro/ai_proxy/internal/config"
)

func TestOpenAIClient_Query(t *testing.T) {
	// Create a test server that returns a mock OpenAI response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers and body
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header with API key")
		}
		
		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "test-id",
			"object": "chat.completion",
			"created": 1677858242,
			"model": "gpt-3.5-turbo",
			"choices": [
				{
					"message": {
						"role": "assistant",
						"content": "Test response"
					},
					"finish_reason": "stop"
				}
			],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 20,
				"total_tokens": 30
			}
		}`))
	}))
	defer server.Close()
	
	// Create client with test server URL
	cfg := &config.LLMConfig{
		URL: server.URL,
		APIKey: "test-api-key",
	}
	client := NewOpenAIClient(cfg)
	
	// Test query
	resp, tokens, err := client.Query("Test prompt", map[string]interface{}{
		"temperature": 0.7,
	})
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp != "Test response" {
		t.Errorf("Expected 'Test response', got '%s'", resp)
	}
	if tokens != 30 {
		t.Errorf("Expected 30 tokens, got %d", tokens)
	}
} 