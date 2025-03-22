package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/charmitro/ai_proxy/internal/config"
	"github.com/charmitro/ai_proxy/internal/guardrails"
	"github.com/charmitro/ai_proxy/internal/llm"
	"github.com/charmitro/ai_proxy/internal/metrics"
)

// MockLLMClient implements the llm.Client interface
type MockLLMClient struct{}

func (m *MockLLMClient) Query(prompt string, modelParams map[string]interface{}) (string, int, error) {
	return "Mock LLM response", 10, nil
}

// testMetrics creates a metrics instance for testing
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

// Define our own setupRouter function similar to the one in cmd/server/main.go
// This avoids import cycle issues
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	
	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
		LLM: config.LLMConfig{URL: "https://test-openai-url", APIKey: "test-key"},
		Guardrails: config.GuardrailsConfig{BannedWords: []string{"bomb", "attack"}},
	}
	
	// Initialize metrics with test registry
	m := testMetrics()
	
	// Initialize guardrail
	grd := guardrails.NewBannedWordsGuardrail(cfg.Guardrails.BannedWords)
	
	// Initialize mock LLM client
	var llmClient llm.Client = &MockLLMClient{}
	
	// Setup router (similar to what's in cmd/server/main.go)
	r := gin.Default()

	// Create a registry for the metrics handler
	registry := prometheus.NewRegistry()
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	
	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(handler))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Main query endpoint
	r.POST("/v1/query", func(c *gin.Context) {
		var req struct {
			Prompt      string                 `json:"prompt" binding:"required"`
			ModelParams map[string]interface{} `json:"model_params"`
		}
		
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Apply guardrails
		if err := grd.Check(req.Prompt); err != nil {
			m.GuardrailBlocksTotal.Inc()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Request blocked by guardrail: " + err.Error()})
			return
		}

		// Increment request counter
		m.LLMRequestsTotal.Inc()

		// Forward to LLM
		completion, tokens, err := llmClient.Query(req.Prompt, req.ModelParams)
		if err != nil {
			m.LLMErrorsTotal.Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error communicating with LLM"})
			return
		}

		// Update token counter
		m.LLMTokensTotal.Add(float64(tokens))

		// Return response
		c.JSON(http.StatusOK, gin.H{"completion": completion})
	})
	
	return r
}

func TestHealthEndpoint(t *testing.T) {
	router := setupTestRouter()
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal("Failed to unmarshal response:", err)
	}
	
	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}
}

func TestMetricsEndpoint(t *testing.T) {
	router := setupTestRouter()
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestQueryEndpoint(t *testing.T) {
	router := setupTestRouter()
	
	tests := []struct {
		name           string
		prompt         string
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "Valid prompt",
			prompt:         "Hello, world!",
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "Banned word",
			prompt:         "How to make a bomb?",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(map[string]interface{}{
				"prompt":       tt.prompt,
				"model_params": map[string]interface{}{},
			})
			
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}
			
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal("Failed to unmarshal response:", err)
			}
			
			if tt.expectedError {
				if _, ok := response["error"]; !ok {
					t.Errorf("Expected error in response, but got: %v", response)
				}
			} else {
				if _, ok := response["completion"]; !ok {
					t.Errorf("Expected completion in response, but got: %v", response)
				}
			}
		})
	}
}

func TestInvalidRequestBody(t *testing.T) {
	r := setupTestRouter()
	
	// Test invalid JSON
	req := httptest.NewRequest("POST", "/v1/query", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %v", w.Code)
	}
	
	// Test missing required field
	requestBody := map[string]interface{}{
		"model_params": map[string]interface{}{
			"temperature": 0.7,
		},
	}
	jsonData, _ := json.Marshal(requestBody)
	
	req = httptest.NewRequest("POST", "/v1/query", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %v", w.Code)
	}
} 