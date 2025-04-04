package guardrails

import (
	"fmt"
	"regexp"
	"strings"
)

// Guardrail defines the interface for content checking
type Guardrail interface {
	Check(content string) error
}

// BannedWordsGuardrail checks if content contains banned words
type BannedWordsGuardrail struct {
	BannedWords []string
}

// NewBannedWordsGuardrail creates a new BannedWordsGuardrail
func NewBannedWordsGuardrail(bannedWords []string) *BannedWordsGuardrail {
	// Convert all banned words to lowercase for case-insensitive comparison
	lowerBannedWords := make([]string, len(bannedWords))
	for i, word := range bannedWords {
		lowerBannedWords[i] = strings.ToLower(word)
	}

	return &BannedWordsGuardrail{
		BannedWords: lowerBannedWords,
	}
}

// Check checks if the content contains any banned words
func (g *BannedWordsGuardrail) Check(content string) error {
	content = strings.ToLower(content)

	for _, word := range g.BannedWords {
		if strings.Contains(content, word) {
			return fmt.Errorf("content contains banned word: %s", word)
		}
	}

	return nil
}

// RegexGuardrail checks if content matches regex patterns
type RegexGuardrail struct {
	Patterns []*regexp.Regexp
	Names    []string
}

// NewRegexGuardrail creates a new RegexGuardrail
func NewRegexGuardrail(patterns []string) (*RegexGuardrail, error) {
	compiledPatterns := make([]*regexp.Regexp, 0, len(patterns))
	patternNames := make([]string, 0, len(patterns))

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern '%s': %v", pattern, err)
		}
		compiledPatterns = append(compiledPatterns, re)
		patternNames = append(patternNames, pattern)
	}

	return &RegexGuardrail{
		Patterns: compiledPatterns,
		Names:    patternNames,
	}, nil
}

// Check checks if the content matches any regex patterns
func (g *RegexGuardrail) Check(content string) error {
	for i, pattern := range g.Patterns {
		if pattern.MatchString(content) {
			return fmt.Errorf("content matches forbidden pattern: %s", g.Names[i])
		}
	}

	return nil
}

// CustomRuleFunc is a function that checks content against a custom rule
type CustomRuleFunc func(content string) (bool, string)

// CustomRuleGuardrail applies user-defined custom rules to content
type CustomRuleGuardrail struct {
	Rules []CustomRuleFunc
	Names []string
}

// NewCustomRuleGuardrail creates a new CustomRuleGuardrail
func NewCustomRuleGuardrail(rules []CustomRuleFunc, names []string) *CustomRuleGuardrail {
	return &CustomRuleGuardrail{
		Rules: rules,
		Names: names,
	}
}

// Check checks if the content violates any custom rules
func (g *CustomRuleGuardrail) Check(content string) error {
	for i, rule := range g.Rules {
		if matched, details := rule(content); matched {
			name := "custom rule"
			if i < len(g.Names) {
				name = g.Names[i]
			}

			if details != "" {
				return fmt.Errorf("content blocked by %s: %s", name, details)
			}
			return fmt.Errorf("content blocked by %s", name)
		}
	}

	return nil
}

// CompositeGuardrail applies multiple guardrails in sequence
type CompositeGuardrail struct {
	Guardrails []Guardrail
}

// NewCompositeGuardrail creates a new CompositeGuardrail
func NewCompositeGuardrail(guardrails ...Guardrail) *CompositeGuardrail {
	return &CompositeGuardrail{
		Guardrails: guardrails,
	}
}

// Check applies all guardrails in sequence
func (g *CompositeGuardrail) Check(content string) error {
	for _, guardrail := range g.Guardrails {
		if err := guardrail.Check(content); err != nil {
			return err
		}
	}

	return nil
}
