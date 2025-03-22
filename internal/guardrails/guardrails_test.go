package guardrails

import (
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
