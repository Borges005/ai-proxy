package guardrails

import (
	"fmt"
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
