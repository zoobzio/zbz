package zbz

import (
	"strings"
	"unicode"
)

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	words := strings.Fields(s) // splits by whitespace
	for i, word := range words {
		if len(word) > 0 {
			// capitalize first rune, add rest as-is
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, "")
}

// toLowerCase converts a string to lowercase
func toLowerCase(s string) string {
	return strings.ToLower(s)
}
