package words

import (
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/kljensen/snowball/english"
)

func Norm(phrase string) []string {
	words := make(map[string]struct{})
	tokens := strings.FieldsFunc(phrase, func(r rune) bool {
		return !unicode.IsDigit(r) && !unicode.IsLetter(r)
	})
	for _, w := range tokens {
		w := strings.ToLower(w)
		if english.IsStopWord(w) {
			continue
		}
		words[english.Stem(w, false)] = struct{}{}
	}
	return slices.Collect(maps.Keys(words))
}
