package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics contains all the metrics for the application
type Metrics struct {
	LLMRequestsTotal     prometheus.Counter
	LLMErrorsTotal       prometheus.Counter
	LLMTokensTotal       prometheus.Counter
	GuardrailBlocksTotal prometheus.Counter
}

// NewMetrics creates and registers all the metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		LLMRequestsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "llm_requests_total",
			Help: "Total number of LLM requests processed",
		}),
		LLMErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "llm_errors_total",
			Help: "Total number of errors from LLM calls",
		}),
		LLMTokensTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "llm_tokens_total",
			Help: "Total number of tokens used in LLM calls",
		}),
		GuardrailBlocksTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "guardrail_blocks_total",
			Help: "Total number of requests blocked by guardrails",
		}),
	}

	return m
}
