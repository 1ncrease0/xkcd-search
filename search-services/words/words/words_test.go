package words

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNorm_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc     string
		given    string
		expected []string
	}{
		{
			desc:     "empty",
			given:    "",
			expected: []string{},
		},
		{
			desc:     "simple",
			given:    "simple",
			expected: []string{"simpl"},
		},
		{
			desc:     "followers",
			given:    "I follow followers",
			expected: []string{"follow"},
		},
		{
			desc:     "punctuation",
			given:    "I shouted: 'give me your car!!!",
			expected: []string{"shout", "give", "car"},
		},
		{
			desc:     "stop words only",
			given:    "I and you or me or them, who will?",
			expected: []string{},
		},
		{
			desc:     "weird",
			given:    "Moscow!123'check-it'or   123, man,that,difficult:heck",
			expected: []string{"moscow", "check", "123", "man", "difficult", "heck"},
		},
		{
			desc:     "duplicates",
			given:    "connected connecting connection",
			expected: []string{"connect"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			got := Norm(tt.given)
			assert.ElementsMatch(t, tt.expected, got)
		})
	}
}
