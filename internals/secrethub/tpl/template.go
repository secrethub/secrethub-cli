package tpl

import (
	"regexp"

	"github.com/secrethub/secrethub-go/internals/errio"
)

// Errors
var (
	tplError = errio.Namespace("template")
)

// Parser parses a raw string to a template.
type Parser interface {
	Parse(raw string, column, line int) (Template, error)
}

// Template contains secret and variable references. It can be evaluated to resolve to a string.
type Template interface {
	// Evaluate renders a template. It replaces all variable- and secret tags in the template.
	// The supplied variables should have lowercase keys.
	Evaluate(varReader VariableReader, sr SecretReader) (string, error)
}

// NewParser returns a parser for the latest template syntax.
func NewParser() Parser {
	return NewV2Parser()
}

var v1SecretTag = regexp.MustCompile(`\${[\t ]*[_\-\.a-zA-Z0-9]+/[_\-\.a-zA-Z0-9]+(?:/[_\-\.a-zA-Z0-9]+)+(?::(?:[0-9]{1,9}|latest))?[\t ]*}`)

// IsV1Template returns whether v1 secret tags are used in the given raw bytes.
func IsV1Template(raw []byte) bool {
	return v1SecretTag.Match(raw)
}
