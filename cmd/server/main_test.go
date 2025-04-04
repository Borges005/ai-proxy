package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/charmitro/ai_proxy/internal/config"
	"github.com/charmitro/ai_proxy/internal/guardrails"
	"github.com/charmitro/ai_proxy/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
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

func TestCustomRuleTypeHandling(t *testing.T) {
	// Test handling of different numeric types in custom rule parameters and
	// validation of pattern parameter for contains_pattern rule

	// Part 1: Test word_count rule with different numeric types
	wordCountTestCases := []struct {
		name        string
		maxWords    interface{}
		content     string
		shouldBlock bool
	}{
		{
			name:        "int value",
			maxWords:    5,
			content:     "This has six words in it",
			shouldBlock: true,
		},
		{
			name:        "float64 value",
			maxWords:    5.0,
			content:     "This has six words in it",
			shouldBlock: true,
		},
		{
			name:        "int64 value",
			maxWords:    int64(5),
			content:     "This has six words in it",
			shouldBlock: true,
		},
		{
			name:        "float32 value",
			maxWords:    float32(5.0),
			content:     "This has six words in it",
			shouldBlock: true,
		},
		{
			name:        "content under limit",
			maxWords:    10,
			content:     "Only five words here",
			shouldBlock: false,
		},
	}

	for _, tc := range wordCountTestCases {
		t.Run("WordCount_"+tc.name, func(t *testing.T) {
			// Setup rule and parameters
			parameters := map[string]interface{}{"max_words": tc.maxWords}

			// Extract the maxWords with type switch
			var maxWords int
			switch v := parameters["max_words"].(type) {
			case float64:
				maxWords = int(v)
			case int:
				maxWords = v
			case int64:
				maxWords = int(v)
			case float32:
				maxWords = int(v)
			default:
				t.Fatalf("Unsupported type for maxWords: %T", parameters["max_words"])
			}

			// Create rule function
			wordCountRule := func(content string) (bool, string) {
				words := strings.Fields(content)
				if len(words) > maxWords {
					return true, "too many words"
				}
				return false, ""
			}

			// Create guardrail
			grd := guardrails.NewCustomRuleGuardrail(
				[]guardrails.CustomRuleFunc{wordCountRule},
				[]string{"test word count rule"},
			)

			// Run check
			err := grd.Check(tc.content)

			// Verify result
			if (err != nil) != tc.shouldBlock {
				t.Errorf("Expected shouldBlock=%v, got err=%v", tc.shouldBlock, err)
			}
		})
	}

	// Part 2: Test contains_pattern rule with different parameter types
	patternTestCases := []struct {
		name        string
		pattern     interface{}
		content     string
		shouldParse bool
		shouldBlock bool
	}{
		{
			name:        "valid string pattern",
			pattern:     "test|pattern",
			content:     "this contains test word",
			shouldParse: true,
			shouldBlock: true,
		},
		{
			name:        "valid regex pattern",
			pattern:     "\\d{3}-\\d{2}-\\d{4}",
			content:     "SSN: 123-45-6789",
			shouldParse: true,
			shouldBlock: true,
		},
		{
			name:        "no match in content",
			pattern:     "nomatch",
			content:     "this doesn't contain the word",
			shouldParse: true,
			shouldBlock: false,
		},
		{
			name:        "invalid pattern type",
			pattern:     123,
			content:     "any content",
			shouldParse: false,
			shouldBlock: false,
		},
	}

	for _, tc := range patternTestCases {
		t.Run("Pattern_"+tc.name, func(t *testing.T) {
			// Setup rule and parameters
			parameters := map[string]interface{}{"pattern": tc.pattern}

			// Extract and validate pattern parameter
			patternParam := parameters["pattern"]
			pattern, ok := patternParam.(string)

			// Check if pattern parsing worked as expected
			if ok != tc.shouldParse {
				t.Fatalf("Expected pattern parsing=%v, got=%v", tc.shouldParse, ok)
			}

			// Skip rest of test if pattern parsing failed
			if !ok {
				return
			}

			// Create pattern rule function
			patternRule := func(content string) (bool, string) {
				matched, err := regexp.MatchString(pattern, content)
				if err != nil {
					return false, ""
				}
				if matched {
					return true, "pattern matched"
				}
				return false, ""
			}

			// Create guardrail
			grd := guardrails.NewCustomRuleGuardrail(
				[]guardrails.CustomRuleFunc{patternRule},
				[]string{"test pattern rule"},
			)

			// Run check
			err := grd.Check(tc.content)

			// Verify result
			if (err != nil) != tc.shouldBlock {
				t.Errorf("Expected shouldBlock=%v, got err=%v", tc.shouldBlock, err)
			}
		})
	}
}

func TestSetupRouter(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Server:     config.ServerConfig{Port: 8080},
		LLM:        config.LLMConfig{URL: "https://test-url", APIKey: "test-key"},
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
			"prompt":       "test-banned-word should be blocked",
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
