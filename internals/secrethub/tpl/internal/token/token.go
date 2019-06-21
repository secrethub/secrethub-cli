package token

// Tokens
var (
	Dollar    = '$'
	LBracket  = '{'
	RBracket  = '}'
	Backslash = '\\'

	tokens = []rune{Dollar, LBracket, RBracket, Backslash}
)

// IsToken returns whether the given rune is a token.
func IsToken(ch rune) bool {
	for _, token := range tokens {
		if ch == token {
			return true
		}
	}
	return false
}
