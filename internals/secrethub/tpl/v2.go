package tpl

import (
	"bytes"
	"io"
	"strings"
	"unicode"

	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl/internal/token"
)

// NewV2Parser returns a parser for the v2 template syntax.
//
// V2 templates can contain secret paths between brackets:
// {{ path/to/secret }}
//
// Within secret paths, variables can be used. Variables are
// given between `${` and `}`.
// For example:
// {{ ${app}/db/secret }}
// Variables cannot be used outside of secret paths.
//
// Spaces directly after opening delimiters (`{{` and `${`) and directly
// before closing delimiters (`}}`, `}`) are ignored. They are not
// included in the secret pahts and variable names.
func NewV2Parser() Parser {
	return parserV2{}
}

type context struct {
	varReader    VariableReader
	secretReader SecretReader
}

func (ctx context) secret(path string) (string, error) {
	return ctx.secretReader.ReadSecret(path)
}

type node interface {
	evaluate(ctx context) (string, error)
}

type secret struct {
	path []node
}

func (s secret) evaluate(ctx context) (string, error) {
	var buffer bytes.Buffer
	for _, p := range s.path {
		eval, err := p.evaluate(ctx)
		if err != nil {
			return "", err
		}

		buffer.WriteString(eval)
	}
	return ctx.secret(buffer.String())
}

type variable struct {
	key string
}

func (v variable) evaluate(ctx context) (string, error) {
	res, err := ctx.varReader.ReadVariable(v.key)
	if err != nil {
		return "", err
	}
	return res, nil
}

type character rune

func (c character) evaluate(ctx context) (string, error) {
	return string(c), nil
}

type templateV2 struct {
	nodes []node
}

type parserV2 struct{}

// Parse parses a secret template from a raw string.
//
// Syntax rules:
// - A secret template can contain references to secrets in secret tags. A
//   secret tag is enclosed in double brackets: `{{ path/to/secret }}`.
// - A secret template can contain references to variables in variable tags. A
//   variable tag is enclosed between ${ and }: `${ variable }`.
// - Extra spaces can be added just after the opening delimiter and just before the
//   closing delimiter of a tag: {{ path/to/secret }} has the same output as
//   {{path/to/secret}} has.
// - Secret tags can also contain variable tags: `{{ path/with/${var}/to/secret }}`
// - Variable tags cannot contain secret tags.
// - Secret tags cannot contain secret tags (they cannot be nested).
// - Variable tags cannot contain variable tags (they cannot be nested).
func (p parserV2) Parse(raw string, line, column int) (Template, error) {
	parser := newV2Parser(bytes.NewBufferString(raw), line, column)

	nodes, err := parser.parse()
	if err != nil {
		return nil, err
	}

	return templateV2{
		nodes: nodes,
	}, nil
}

func newV2Parser(buf *bytes.Buffer, line, column int) v2Parser {
	return v2Parser{
		buf:    buf,
		lineNo: line,
		// The column number indicates the index (starting at 1) of the current rune.
		// We subtract 2 of the given value. One because we have not read the current rune yet and
		// one more because we are reading the next rune in advance (which we don't want to count).
		columnNo: column - 2,
	}
}

type v2Parser struct {
	buf      *bytes.Buffer
	lineNo   int
	columnNo int

	current rune
	next    rune
}

// readRune reads the next rune from the raw template.
func (p *v2Parser) readRune() error {
	p.current = p.next
	if p.current == '\n' {
		p.lineNo++
		p.columnNo = 0
	} else {
		p.columnNo++
	}

	var err error
	p.next, _, err = p.buf.ReadRune()
	return err
}

func (p *v2Parser) parse() ([]node, error) {
	res := []node{}

	err := p.readRune()
	if err == io.EOF {
		return res, nil
	}
	if err != nil {
		return nil, err
	}

	for {
		err := p.readRune()
		if err == io.EOF {
			return append(res, character(p.current)), nil
		}
		if err != nil {
			return nil, err
		}

		n, err := p.parseRoot()
		if err == io.EOF {
			return append(res, n), nil
		}
		if err != nil {
			return nil, err
		}

		res = append(res, n)
	}
}

// parseRoot parses the contents of a template at root level, outside of any
// tag.
// The current character should be the character to parse. When parseRoot returns,
// the current character is the last processed character.
func (p *v2Parser) parseRoot() (node, error) {
	if p.current == token.Dollar && p.next == token.LBracket {
		variable, err := p.parseVar()
		if err != nil {
			return nil, err
		}
		return variable, p.readRune()
	}

	if p.current == token.Dollar && p.isVariableStartRune(p.next) {
		return p.parseVarWithoutBrackets()
	}

	if p.current == token.LBracket && p.next == token.LBracket {
		secret, err := p.parseSecret()
		if err != nil {
			return nil, err
		}
		return secret, p.readRune()
	}

	if p.current == token.Backslash && token.IsToken(p.next) {
		token := character(p.next)
		return token, p.readRune()
	}

	return character(p.current), nil
}

func (p *v2Parser) parseVarWithoutBrackets() (node, error) {
	var buffer bytes.Buffer

	for p.isVariableRune(p.next) {
		buffer.WriteRune(p.next)

		err := p.readRune()
		if err == io.EOF {
			return variable{
				key: strings.ToLower(buffer.String()),
			}, err
		}
		if err != nil {
			return nil, err
		}
	}

	return variable{
		key: strings.ToLower(buffer.String()),
	}, nil
}

// parseVar parses the contents of a template variable up to the closing delimiter.
// The next character should be the last character of the opening delimiter ('{')
// when parseVar is called.
//
// When parseVar returns, the next character in the buffer is the closing delimiter
// of the template variable ('}').
func (p *v2Parser) parseVar() (node, error) {
	var buffer bytes.Buffer

	checkError := func(err error) error {
		if err == io.EOF {
			return ErrVariableTagNotClosed(p.lineNo, p.columnNo+1)
		}
		return err
	}

	err := p.readRune()
	if err != nil {
		return nil, checkError(err)
	}

	err = p.skipWhiteSpace()
	if err != nil {
		return nil, checkError(err)
	}

	for {
		if p.next == token.RBracket {
			return variable{
				key: strings.ToLower(buffer.String()),
			}, nil
		}

		if p.isAllowedWhiteSpace(p.next) {
			err := p.skipWhiteSpace()
			if err != nil {
				return nil, checkError(err)
			}

			if p.next == token.RBracket {
				return variable{
					key: strings.ToLower(buffer.String()),
				}, nil
			}

			return nil, ErrUnexpectedCharacter(p.lineNo, p.columnNo+1, p.next, token.RBracket)
		}

		if p.isVariableRune(p.next) {
			buffer.WriteRune(p.next)

			err := p.readRune()
			if err != nil {
				return nil, checkError(err)
			}

			continue
		}

		return nil, ErrIllegalVariableCharacter(p.lineNo, p.columnNo+1, p.next)
	}
}

// parseSecret parses the contents of a secret tag up to the closing delimiter.
// The next character should be the last character of the opening delimiter ('{')
// when parseSecret is called.
//
// When parseSecret returns, the next character in the buffer is the last character
// of the closing delimiter of the secret tag ('}').
func (p *v2Parser) parseSecret() (node, error) {
	path := []node{}

	checkError := func(err error) error {
		if err == io.EOF {
			return ErrSecretTagNotClosed(p.lineNo, p.columnNo+1)
		}
		return err
	}

	err := p.readRune()
	if err != nil {
		return nil, checkError(err)
	}

	err = p.skipWhiteSpace()
	if err != nil {
		return nil, checkError(err)
	}

	for {
		err = p.readRune()
		if err != nil {
			return nil, checkError(err)
		}

		if p.current == token.Dollar {
			if p.next == token.LBracket {
				variable, err := p.parseVar()
				if err != nil {
					return nil, err
				}

				path = append(path, variable)

				err = p.readRune()
				if err != nil {
					return nil, checkError(err)
				}

				continue
			}
			if p.isVariableStartRune(p.next) {
				variable, err := p.parseVarWithoutBrackets()
				if err != nil {
					return nil, checkError(err)
				}
				path = append(path, variable)

				continue
			}

			return nil, ErrIllegalSecretCharacter(p.lineNo, p.columnNo, p.current)
		}

		if p.isAllowedWhiteSpace(p.current) {
			err := p.skipWhiteSpace()
			if err != nil {
				return nil, checkError(err)
			}

			if p.next != token.RBracket {
				return nil, ErrUnexpectedCharacter(p.lineNo, p.columnNo+1, p.next, token.RBracket)
			}

			err = p.readRune()
			if err != nil {
				return nil, checkError(err)
			}

			if p.next != token.RBracket {
				return nil, ErrUnexpectedCharacter(p.lineNo, p.columnNo+1, p.next, token.RBracket)
			}

			return secret{
				path: path,
			}, nil
		}

		if p.current == token.RBracket {
			if p.next == token.RBracket {
				return secret{
					path: path,
				}, nil
			}
			return nil, ErrUnexpectedCharacter(p.lineNo, p.columnNo+1, p.next, token.RBracket)
		}

		if p.isSecretPathRune(p.current) {
			path = append(path, character(p.current))
			continue
		}

		return nil, ErrIllegalSecretCharacter(p.lineNo, p.columnNo, p.current)
	}
}

// isSecretPathRune returns whether the given rune is allowed to be used in
// a secret path.
func (p v2Parser) isSecretPathRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == '/' || r == ':'
}

// isVariableRune returns whether the given rune is allowed to be used in a template variable key.
func (p v2Parser) isVariableRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// isVariableStartRune returns whether the given rune is allowed to be used at the start of a template variable key.
func (p v2Parser) isVariableStartRune(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

// isAllowedWhiteSpace returns whether the given rune is allowed as extra whitespace
// just after the opening tag and just before the closing tag.
func (p v2Parser) isAllowedWhiteSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// skipWhiteSpace reads new runes until the next rune is not a space or tab.
func (p *v2Parser) skipWhiteSpace() error {
	for p.isAllowedWhiteSpace(p.next) {
		err := p.readRune()
		if err != nil {
			return err
		}
	}
	return nil
}

// SecretReader fetches a secret by its path.
type SecretReader interface {
	ReadSecret(path string) (string, error)
}

// VariableReader fetches a template variable by its name.
type VariableReader interface {
	ReadVariable(name string) (string, error)
}

// Evaluate renders a template. It replaces all variable- and secret tags in the template.
// The supplied variables should have lowercase keys.
func (t templateV2) Evaluate(varReader VariableReader, sr SecretReader) (string, error) {
	ctx := context{
		varReader:    varReader,
		secretReader: sr,
	}

	var buffer bytes.Buffer
	for _, n := range t.nodes {
		eval, err := n.evaluate(ctx)
		if err != nil {
			return "", err
		}

		buffer.WriteString(eval)
	}

	return buffer.String(), nil
}
