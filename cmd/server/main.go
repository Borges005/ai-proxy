package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/charmitro/ai_proxy/internal/config"
	"github.com/charmitro/ai_proxy/internal/guardrails"
	"github.com/charmitro/ai_proxy/internal/llm"
	"github.com/charmitro/ai_proxy/internal/metrics"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type queryRequest struct {
	Prompt      string                 `json:"prompt" binding:"required"`
	ModelParams map[string]interface{} `json:"model_params"`
}

type queryResponse struct {
	Completion string `json:"completion"`
}

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config/config.yaml", "Path to config file")
	flag.Parse()

	// Initialize logger
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize metrics
	m := metrics.NewMetrics()

	// Initialize guardrail
	grd := guardrails.NewBannedWordsGuardrail(cfg.Guardrails.BannedWords)

	// Initialize LLM client
	llmClient := llm.NewOpenAIClient(&cfg.LLM)

	// Set up Gin router
	r := gin.Default()

	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Main query endpoint
	r.POST("/v1/query", func(c *gin.Context) {
		var req queryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logrus.Errorf("Invalid request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Apply guardrails
		if err := grd.Check(req.Prompt); err != nil {
			logrus.Warnf("Guardrail blocked request: %v", err)
			m.GuardrailBlocksTotal.Inc()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Request blocked by guardrail: " + err.Error()})
			return
		}

		// Increment request counter
		m.LLMRequestsTotal.Inc()

		// Forward to LLM
		completion, tokens, err := llmClient.Query(req.Prompt, req.ModelParams)
		if err != nil {
			logrus.Errorf("LLM error: %v", err)
			m.LLMErrorsTotal.Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error communicating with LLM"})
			return
		}

		// Update token counter
		m.LLMTokensTotal.Add(float64(tokens))

		// Return response
		c.JSON(http.StatusOK, queryResponse{
			Completion: completion,
		})
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logrus.Infof("Starting server on %s", addr)
	if err := r.Run(addr); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
