package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// testMetrics creates metrics with a test registry to avoid conflicts
func testMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	
	m := &Metrics{
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

func TestMetricsInitialization(t *testing.T) {
	m := testMetrics()
	
	// Verify counters start at zero
	if value := testutil.ToFloat64(m.LLMRequestsTotal); value != 0 {
		t.Errorf("LLMRequestsTotal should start at 0, got %f", value)
	}
	
	if value := testutil.ToFloat64(m.LLMErrorsTotal); value != 0 {
		t.Errorf("LLMErrorsTotal should start at 0, got %f", value)
	}
	
	if value := testutil.ToFloat64(m.LLMTokensTotal); value != 0 {
		t.Errorf("LLMTokensTotal should start at 0, got %f", value)
	}
	
	if value := testutil.ToFloat64(m.GuardrailBlocksTotal); value != 0 {
		t.Errorf("GuardrailBlocksTotal should start at 0, got %f", value)
	}
}

func TestMetricsIncrement(t *testing.T) {
	m := testMetrics()
	
	// Test incrementing counters
	m.LLMRequestsTotal.Inc()
	if value := testutil.ToFloat64(m.LLMRequestsTotal); value != 1 {
		t.Errorf("LLMRequestsTotal should be 1 after increment, got %f", value)
	}
	
	m.GuardrailBlocksTotal.Inc()
	m.GuardrailBlocksTotal.Inc()
	if value := testutil.ToFloat64(m.GuardrailBlocksTotal); value != 2 {
		t.Errorf("GuardrailBlocksTotal should be 2 after two increments, got %f", value)
	}
	
	// Test adding to counters
	m.LLMTokensTotal.Add(42.0)
	if value := testutil.ToFloat64(m.LLMTokensTotal); value != 42.0 {
		t.Errorf("LLMTokensTotal should be 42.0 after adding, got %f", value)
	}
} 