package tpl

import "github.com/secrethub/secrethub-go/internals/errio"

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
	Evaluate(vars map[string]string, sr SecretReader) (string, error)
}

// NewParser returns a parser for the latest template syntax.
func NewParser() Parser {
	return NewV2Parser()
}
