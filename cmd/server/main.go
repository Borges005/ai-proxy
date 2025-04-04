package main

import (
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strings"

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

// setupRouter initializes and returns the Gin router with all routes configured
func setupRouter(cfg *config.Config, m *metrics.Metrics, grd guardrails.Guardrail, llmClient llm.Client) *gin.Engine {
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

	return r
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

	// Initialize guardrails
	guardrailComponents := []guardrails.Guardrail{}

	// Add banned words guardrail if configured
	if len(cfg.Guardrails.BannedWords) > 0 {
		guardrailComponents = append(guardrailComponents,
			guardrails.NewBannedWordsGuardrail(cfg.Guardrails.BannedWords))
	}

	// Add regex patterns guardrail if configured
	if len(cfg.Guardrails.RegexPatterns) > 0 {
		regexGuardrail, err := guardrails.NewRegexGuardrail(cfg.Guardrails.RegexPatterns)
		if err != nil {
			logrus.Warnf("Failed to initialize regex guardrail: %v", err)
		} else {
			guardrailComponents = append(guardrailComponents, regexGuardrail)
		}
	}

	// Add custom length check guardrails if configured
	if cfg.Guardrails.MaxPromptLength > 0 {
		lengthCheckRule := func(content string) (bool, string) {
			if len(content) > cfg.Guardrails.MaxPromptLength {
				return true, fmt.Sprintf("prompt exceeds maximum length of %d characters", cfg.Guardrails.MaxPromptLength)
			}
			return false, ""
		}

		guardrailComponents = append(guardrailComponents,
			guardrails.NewCustomRuleGuardrail(
				[]guardrails.CustomRuleFunc{lengthCheckRule},
				[]string{"prompt length check"},
			),
		)
	}

	// Process custom rules if configured
	for _, ruleConfig := range cfg.Guardrails.CustomRules {
		switch ruleConfig.Type {
		case "word_count":
			// Handle different numeric types for max_words parameter
			var maxWords int
			maxWordsParam := ruleConfig.Parameters["max_words"]
			switch v := maxWordsParam.(type) {
			case float64:
				maxWords = int(v)
			case int:
				maxWords = v
			case int64:
				maxWords = int(v)
			case float32:
				maxWords = int(v)
			default:
				logrus.Warnf("Invalid max_words parameter type: %T", maxWordsParam)
				continue
			}

			wordCountRule := func(content string) (bool, string) {
				words := strings.Fields(content)
				if len(words) > maxWords {
					return true, fmt.Sprintf("text contains %d words, exceeding limit of %d", len(words), maxWords)
				}
				return false, ""
			}

			guardrailComponents = append(guardrailComponents,
				guardrails.NewCustomRuleGuardrail(
					[]guardrails.CustomRuleFunc{wordCountRule},
					[]string{ruleConfig.Name},
				),
			)
		case "contains_pattern":
			patternParam := ruleConfig.Parameters["pattern"]
			pattern, ok := patternParam.(string)
			if !ok {
				logrus.Warnf("Invalid pattern parameter type: %T", patternParam)
				continue
			}

			patternRule := func(content string) (bool, string) {
				matched, err := regexp.MatchString(pattern, content)
				if err != nil {
					logrus.Warnf("Error matching pattern: %v", err)
					return false, ""
				}
				if matched {
					return true, fmt.Sprintf("content matches forbidden pattern")
				}
				return false, ""
			}

			guardrailComponents = append(guardrailComponents,
				guardrails.NewCustomRuleGuardrail(
					[]guardrails.CustomRuleFunc{patternRule},
					[]string{ruleConfig.Name},
				),
			)
		default:
			logrus.Warnf("Unknown custom rule type: %s", ruleConfig.Type)
		}
	}

	// Create composite guardrail from all components
	var grd guardrails.Guardrail
	if len(guardrailComponents) > 0 {
		grd = guardrails.NewCompositeGuardrail(guardrailComponents...)
	} else {
		// Create a default pass-through guardrail if no components configured
		grd = &guardrails.CustomRuleGuardrail{
			Rules: []guardrails.CustomRuleFunc{func(content string) (bool, string) { return false, "" }},
			Names: []string{"default-passthrough"},
		}
	}

	// Initialize LLM client
	llmClient := llm.NewOpenAIClient(&cfg.LLM)

	// Set up Gin router
	r := setupRouter(cfg, m, grd, llmClient)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logrus.Infof("Starting server on %s", addr)
	if err := r.Run(addr); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
