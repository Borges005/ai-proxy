# AI Proxy

A lightweight proxy for LLM interactions with basic guardrails, logging, and metrics.

## Features

- Single LLM provider support (OpenAI)
- Enhanced guardrails system:
  - Banned words filtering
  - Regex pattern matching
  - Content length limits
  - Custom filtering rules (word count, pattern detection, etc.)
- Logging of requests and responses
- Prometheus metrics
- Config-driven setup
- Docker deployment
- Monitoring with Prometheus and Grafana

## Configuration

The proxy can be configured using a YAML file or environment variables:

### Config File (`config/config.yaml`)

```yaml
server:
  port: 8080

llm:
  url: "https://api.openai.com/v1/chat/completions"
  api_key: "YOUR_OPENAI_API_KEY"

guardrails:
  # Basic banned words filtering
  banned_words:
    - "bomb"
    - "attack"
    - "weapon"
    - "terrorist"
  
  # Regex pattern filtering for sensitive information
  regex_patterns:
    - "\\b\\d{3}-\\d{2}-\\d{4}\\b"  # US Social Security Numbers
    - "\\b\\d{16}\\b"               # Credit card numbers (simplified)
  
  # Length limitations
  max_content_length: 10000
  max_prompt_length: 4000
  
  # Custom rules with configuration
  custom_rules:
    - name: "Max Words"
      type: "word_count"
      parameters:
        max_words: 1000
    
    - name: "PII Detection" 
      type: "contains_pattern"
      parameters:
        pattern: "(passport|ssn|social security|credit card)"
```

### Environment Variables

Configuration via environment variables is supported for basic settings:

- `SERVER_PORT`: Server port (default: 8080)
- `LLM_URL`: LLM API endpoint URL
- `LLM_API_KEY`: LLM API key
- `BANNED_WORDS`: Comma-separated list of banned words

For advanced guardrail configuration, it's recommended to use the config file approach.
Complex configurations like regex patterns and custom rules are better defined in the YAML config file.

## API

### Query Endpoint

**Request:**

```json
{
  "prompt": "Your prompt to the LLM",
  "model_params": {
    "model": "gpt-3.5-turbo",
    "temperature": 0.7,
    "max_tokens": 256
  }
}
```

**Response:**

```json
{
  "completion": "LLM response text..."
}
```

### Metrics Endpoint

```
GET /metrics
```

Returns Prometheus-formatted metrics including:
- `llm_requests_total`: Total number of LLM requests processed
- `llm_errors_total`: Total number of errors from LLM calls
- `llm_tokens_total`: Total number of tokens used in LLM calls
- `guardrail_blocks_total`: Total number of requests blocked by guardrails

### Health Check

```
GET /health
```

## Running Locally

```bash
# Build and run
go build -o ai-proxy ./cmd/server
./ai-proxy --config config/config.yaml
```

## Running with Docker

```bash
# Build Docker image
docker build -t ai-proxy:0.1 .

# Run with configuration in environment variables
docker run -p 8080:8080 \
  -e LLM_API_KEY=your_openai_api_key \
  ai-proxy:0.1

# Or mount a custom config file
docker run -p 8080:8080 \
  -v $(pwd)/config/config.yaml:/app/config/config.yaml \
  ai-proxy:0.1
```

## Monitoring Setup

The project includes a complete monitoring stack with Prometheus and Grafana.

### Running with Monitoring

```bash
# Start the entire stack (AI Proxy, Prometheus, and Grafana)
docker-compose up -d

# Access the services:
# - AI Proxy: http://localhost:8080
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (login with admin/admin)

# Stop the services
docker-compose down
```

## Example Usage

```bash
# Send a query
curl -X POST http://localhost:8080/v1/query \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Tell me a joke", "model_params": {"temperature": 0.7}}'

# Check metrics
curl http://localhost:8080/metrics
```

## Guardrail Types

The AI Proxy supports multiple types of guardrails that can be combined:

### 1. Banned Words Guardrail

This guardrail blocks requests containing specified banned words.

```yaml
banned_words:
  - "dangerous_word1"
  - "dangerous_word2"
```

### 2. Regex Pattern Guardrail

This guardrail uses regular expressions to block content matching specified patterns. Useful for detecting structured sensitive information like SSNs or credit cards.

```yaml
regex_patterns:
  - "\\b\\d{3}-\\d{2}-\\d{4}\\b"  # SSN pattern
```

### 3. Content Length Guardrails

Limits the maximum length of prompts and completions.

```yaml
max_content_length: 10000
max_prompt_length: 4000
```

### 4. Custom Rule Guardrails

Allows for configurable rules using predefined types. Each rule type supports specific parameters and type validation:

#### Word Count Rule

```yaml
custom_rules:
  - name: "Max Words"
    type: "word_count"
    parameters:
      max_words: 1000
```

#### Pattern Detection Rule

```yaml
custom_rules:
  - name: "PII Detection" 
    type: "contains_pattern"
    parameters:
      pattern: "(passport|ssn|social security|credit card)"  # Regular expression pattern
```

The pattern parameter must be a valid regular expression. Invalid regex patterns will be logged as warnings and skipped.

## Potential Enhancements

Some possible future enhancements could include:

1. **Additional LLM Support**  
   - Integration with other LLM providers (Anthropic Claude, Google Gemini, etc.)

2. **Authentication & Security**  
   - API key authentication
   - Basic rate limiting
   - Request validation

3. **Performance Improvements**  
   - Optional caching for common queries
   - Optimizations for high-traffic scenarios
   - Load balancing across multiple LLM providers
