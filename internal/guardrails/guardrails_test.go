package guardrails

import (
	"regexp"
	"strings"
	"testing"
)

func TestBannedWordsGuardrail(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		bannedWords []string
		wantErr     bool
	}{
		{
			name:        "no banned words",
			content:     "This is a safe text",
			bannedWords: []string{"bomb", "attack"},
			wantErr:     false,
		},
		{
			name:        "contains banned word",
			content:     "How to make a bomb",
			bannedWords: []string{"bomb", "attack"},
			wantErr:     true,
		},
		{
			name:        "banned word with different case",
			content:     "How to make a BOMB",
			bannedWords: []string{"bomb", "attack"},
			wantErr:     true,
		},
		{
			name:        "banned word as part of another word",
			content:     "The bombing happened yesterday",
			bannedWords: []string{"bomb", "attack"},
			wantErr:     true,
		},
		{
			name:        "empty content",
			content:     "",
			bannedWords: []string{"bomb", "attack"},
			wantErr:     false,
		},
		{
			name:        "empty banned words list",
			content:     "Any content is safe",
			bannedWords: []string{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewBannedWordsGuardrail(tt.bannedWords)
			err := g.Check(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("BannedWordsGuardrail.Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegexGuardrail(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		patterns []string
		wantErr  bool
	}{
		{
			name:     "no matches",
			content:  "This is a safe text",
			patterns: []string{`bomb`, `attack`},
			wantErr:  false,
		},
		{
			name:     "matches pattern",
			content:  "How to make a bomb",
			patterns: []string{`bomb`, `attack`},
			wantErr:  true,
		},
		{
			name:     "matches complex pattern",
			content:  "My SSN is 123-45-6789",
			patterns: []string{`\d{3}-\d{2}-\d{4}`},
			wantErr:  true,
		},
		{
			name:     "matches email pattern",
			content:  "My email is test@example.com",
			patterns: []string{`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`},
			wantErr:  true,
		},
		{
			name:     "empty content",
			content:  "",
			patterns: []string{`bomb`, `attack`},
			wantErr:  false,
		},
		{
			name:     "empty patterns list",
			content:  "Any content is safe",
			patterns: []string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := NewRegexGuardrail(tt.patterns)
			if err != nil {
				t.Fatalf("Failed to create RegexGuardrail: %v", err)
			}

			err = g.Check(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegexGuardrail.Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Test invalid regex
	_, err := NewRegexGuardrail([]string{`[`})
	if err == nil {
		t.Errorf("NewRegexGuardrail() with invalid regex should return error")
	}
}

func TestCustomRuleGuardrail(t *testing.T) {
	// Example custom rules
	maxLengthRule := func(maxLen int) CustomRuleFunc {
		return func(content string) (bool, string) {
			if len(content) > maxLen {
				return true, "content too long"
			}
			return false, ""
		}
	}

	profanityRule := func(profanities []string) CustomRuleFunc {
		return func(content string) (bool, string) {
			lowerContent := strings.ToLower(content)
			for _, word := range profanities {
				if strings.Contains(lowerContent, word) {
					return true, "contains profanity"
				}
			}
			return false, ""
		}
	}

	wordCountRule := func(maxWords int) CustomRuleFunc {
		return func(content string) (bool, string) {
			words := regexp.MustCompile(`\S+`).FindAllString(content, -1)
			if len(words) > maxWords {
				return true, "too many words"
			}
			return false, ""
		}
	}

	tests := []struct {
		name    string
		content string
		rules   []CustomRuleFunc
		names   []string
		wantErr bool
	}{
		{
			name:    "passes all rules",
			content: "This is a safe text",
			rules: []CustomRuleFunc{
				maxLengthRule(100),
				profanityRule([]string{"profanity", "bad"}),
				wordCountRule(10),
			},
			names:   []string{"length check", "profanity check", "word count check"},
			wantErr: false,
		},
		{
			name:    "fails length rule",
			content: "This is a very long text that exceeds the maximum length allowed by our rules",
			rules: []CustomRuleFunc{
				maxLengthRule(30),
				profanityRule([]string{"profanity", "bad"}),
				wordCountRule(50),
			},
			names:   []string{"length check", "profanity check", "word count check"},
			wantErr: true,
		},
		{
			name:    "fails profanity rule",
			content: "This text contains profanity and should be blocked",
			rules: []CustomRuleFunc{
				maxLengthRule(100),
				profanityRule([]string{"profanity", "bad"}),
				wordCountRule(50),
			},
			names:   []string{"length check", "profanity check", "word count check"},
			wantErr: true,
		},
		{
			name:    "fails word count rule",
			content: "This text has too many words and should be blocked by the word count rule",
			rules: []CustomRuleFunc{
				maxLengthRule(100),
				profanityRule([]string{"profanity", "bad"}),
				wordCountRule(5),
			},
			names:   []string{"length check", "profanity check", "word count check"},
			wantErr: true,
		},
		{
			name:    "empty rules list",
			content: "Any content is safe",
			rules:   []CustomRuleFunc{},
			names:   []string{},
			wantErr: false,
		},
		{
			name:    "missing name",
			content: "This will fail the first rule but has no name for it",
			rules: []CustomRuleFunc{
				maxLengthRule(5),
			},
			names:   []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewCustomRuleGuardrail(tt.rules, tt.names)
			err := g.Check(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("CustomRuleGuardrail.Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompositeGuardrail(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		guardrails []Guardrail
		wantErr    bool
	}{
		{
			name:    "passes all guardrails",
			content: "This is a safe text",
			guardrails: []Guardrail{
				NewBannedWordsGuardrail([]string{"bomb", "attack"}),
				&CustomRuleGuardrail{
					Rules: []CustomRuleFunc{
						func(content string) (bool, string) {
							return len(content) > 100, "too long"
						},
					},
					Names: []string{"length check"},
				},
			},
			wantErr: false,
		},
		{
			name:    "fails banned words guardrail",
			content: "How to make a bomb",
			guardrails: []Guardrail{
				NewBannedWordsGuardrail([]string{"bomb", "attack"}),
				&CustomRuleGuardrail{
					Rules: []CustomRuleFunc{
						func(content string) (bool, string) {
							return len(content) > 100, "too long"
						},
					},
					Names: []string{"length check"},
				},
			},
			wantErr: true,
		},
		{
			name:    "fails custom rule guardrail",
			content: "This is a very long text that exceeds the maximum length allowed by our custom rule that checks text length",
			guardrails: []Guardrail{
				NewBannedWordsGuardrail([]string{"bomb", "attack"}),
				&CustomRuleGuardrail{
					Rules: []CustomRuleFunc{
						func(content string) (bool, string) {
							return len(content) > 50, "too long"
						},
					},
					Names: []string{"length check"},
				},
			},
			wantErr: true,
		},
		{
			name:       "empty guardrails list",
			content:    "Any content is safe",
			guardrails: []Guardrail{},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewCompositeGuardrail(tt.guardrails...)
			err := g.Check(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompositeGuardrail.Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
