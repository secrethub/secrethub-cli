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

// IsDollar returns whether the given rune is the dollar token.
func IsDollar(ch rune) bool {
	return ch == Dollar
}

// IsLBracket returns whether the given rune is the left bracket token.
func IsLBracket(ch rune) bool {
	return ch == LBracket
}

// IsRBracket returns whether the given rune is the right bracket token.
func IsRBracket(ch rune) bool {
	return ch == RBracket
}

// IsBackslash returns whether the given rune is the backlash token.
func IsBackslash(ch rune) bool {
	return ch == Backslash
}

// IsToken returns whether the given rune is a token.
func IsToken(ch rune) bool {
	_, isToken := tokens[ch]
	return isToken
}
