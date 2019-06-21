package token

// Tokens
var (
	Dollar    = '$'
	LBracket  = '{'
	RBracket  = '}'
	Backslash = '\\'

	tokens = map[rune]struct{}{
		Dollar:    {},
		LBracket:  {},
		RBracket:  {},
		Backslash: {},
	}
)

// IsToken returns whether the given rune is a token.
func IsToken(ch rune) bool {
	_, isToken := tokens[ch]
	return isToken
}
