package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/charmitro/ai_proxy/internal/config"
	"github.com/charmitro/ai_proxy/internal/guardrails"
	"github.com/charmitro/ai_proxy/internal/metrics"
)

// MockLLMClient implements the llm.Client interface for testing
type MockLLMClient struct{}

func (m *MockLLMClient) Query(prompt string, modelParams map[string]interface{}) (string, int, error) {
	return "Mock response for tests", 10, nil
}

// testMetrics creates metrics with a test registry to avoid conflicts
func testMetrics() *metrics.Metrics {
	reg := prometheus.NewRegistry()
	
	m := &metrics.Metrics{
		LLMRequestsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_llm_requests_total",
			Help: "Total number of LLM requests processed",
		}),
		LLMErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_llm_errors_total",
			Help: "Total number of errors from LLM calls",
		}),
		LLMTokensTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_llm_tokens_total",
			Help: "Total number of tokens used in LLM calls",
		}),
		GuardrailBlocksTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_guardrail_blocks_total",
			Help: "Total number of requests blocked by guardrails",
		}),
	}
	
	reg.MustRegister(m.LLMRequestsTotal)
	reg.MustRegister(m.LLMErrorsTotal)
	reg.MustRegister(m.LLMTokensTotal)
	reg.MustRegister(m.GuardrailBlocksTotal)
	
	return m
}

func TestSetupRouter(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Server:    config.ServerConfig{Port: 8080},
		LLM:       config.LLMConfig{URL: "https://test-url", APIKey: "test-key"},
		Guardrails: config.GuardrailsConfig{BannedWords: []string{"test-banned-word"}},
	}

	// Setup test components
	m := testMetrics()
	grd := guardrails.NewBannedWordsGuardrail(cfg.Guardrails.BannedWords)
	llmClient := &MockLLMClient{}

	// Initialize router
	router := setupRouter(cfg, m, grd, llmClient)

	// Test health endpoint
	t.Run("Health endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response["status"])
		}
	})

	// Test metrics endpoint
	t.Run("Metrics endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
	
	// Test query endpoint
	t.Run("Query endpoint", func(t *testing.T) {
		// Valid query
		requestBody, _ := json.Marshal(map[string]interface{}{
			"prompt": "test prompt",
			"model_params": map[string]interface{}{
				"temperature": 0.7,
			},
		})
		
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response queryResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if response.Completion != "Mock response for tests" {
			t.Errorf("Expected 'Mock response for tests', got '%s'", response.Completion)
		}
	})
	
	// Test guardrail blocking
	t.Run("Guardrail blocking", func(t *testing.T) {
		requestBody, _ := json.Marshal(map[string]interface{}{
			"prompt": "test-banned-word should be blocked",
			"model_params": map[string]interface{}{},
		})
		
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
		
		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if _, exists := response["error"]; !exists {
			t.Error("Expected error in response, got none")
		}
	})
} 