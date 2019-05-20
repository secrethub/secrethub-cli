// Package tpl provides a way to parse a template string and inject secret values into it.
package tpl

import (
	"strings"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// Errors
var (
	errTemplate             = errio.Namespace("template")
	ErrKeyNotFound          = errTemplate.Code("key_not_found").ErrorPref("no value supplied for key %s")
	ErrReplacementNotClosed = errTemplate.Code("replacement_not_closed").ErrorPref("missing closing '%s'")
)

const (
	// DefaultStartDelimiter defines the characters a template block starts with.
	DefaultStartDelimiter = "${"
	// DefaultEndDelimiter defines the characters a template block ends with.
	DefaultEndDelimiter = "}"
	// DefaultTrimChars defines the cutset of characters that will be trimmed from template blocks.
	DefaultTrimChars = " "
)

// Parser parses a raw string into a template.
type Parser interface {
	Parse(raw string) (Template, error)
}

type parser struct {
	startDelim string
	endDelim   string
	trimChars  string
}

// Template helps with injecting values into strings that contain the template syntax.
type Template interface {
	Inject(replacements map[string]string) (string, error)
	Keys() []string
}

type template struct {
	nodes []node
}

// NewParser creates a new template parser.
func NewParser() Parser {
	return parser{
		startDelim: DefaultStartDelimiter,
		endDelim:   DefaultEndDelimiter,
		trimChars:  DefaultTrimChars,
	}
}

// Inject inserts the given replacements at their corresponding places in
// the raw template and returns the injected template.
func (t template) Inject(replacements map[string]string) (string, error) {
	res := ""
	for _, n := range t.nodes {
		injected, err := n.inject(replacements)
		if err != nil {
			return "", err
		}
		res += injected
	}
	return res, nil
}

// Keys returns all keys the template contains.
func (t template) Keys() []string {
	set := map[string]bool{}
	for _, n := range t.nodes {
		s, ok := n.(secret)
		if ok {
			set[string(s)] = true
		}
	}

	res := make([]string, len(set))
	i := 0
	for s := range set {
		res[i] = s
		i++
	}
	return res
}

// node is a part of the template, either a plain text value or
// a path to a secret.
type node interface {
	inject(secrets map[string]string) (string, error)
}

type val string

func (v val) inject(map[string]string) (string, error) {
	return string(v), nil
}

type secret string

func (s secret) inject(replacements map[string]string) (string, error) {
	data, ok := replacements[string(s)]
	if !ok {
		return "", ErrKeyNotFound(s)
	}
	return data, nil
}

func (p parser) Parse(raw string) (Template, error) {
	nodes, err := p.parse(raw)
	if err != nil {
		return nil, err
	}
	return template{
		nodes: nodes,
	}, nil
}

// parse is a recursive helper function that parses a string to a list of SecretPaths contained between inject delimiters.
func (p parser) parse(raw string) ([]node, error) {
	parts := strings.SplitN(raw, p.startDelim, 2)
	if len(parts) == 1 {
		return []node{val(parts[0])}, nil
	}
	if len(parts[0]) > 0 {
		tail, err := p.parse(p.startDelim + parts[1])
		if err != nil {
			return nil, err
		}
		return append([]node{val(parts[0])}, tail...), nil
	}

	parts = strings.SplitN(parts[1], p.endDelim, 2)
	if len(parts) == 1 {
		return nil, ErrReplacementNotClosed(p.endDelim)
	}

	path := strings.Trim(parts[0], p.trimChars)

	err := api.ValidateSecretPath(path)
	if err != nil {
		return nil, err
	}

	tail, err := p.parse(parts[1])
	if err != nil {
		return nil, err
	}

	return append([]node{secret(path)}, tail...), nil
}
