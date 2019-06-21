package tpl

import (
	"bytes"
	"io"
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
	vars         map[string]string
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
	res, ok := ctx.vars[v.key]
	if !ok {
		return "", ErrTemplateVarNotFound(v.key)
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

		switch p.current {
		case token.Dollar:
			switch p.next {
			case token.LBracket:
				variable, err := p.parseVar()
				if err != nil {
					return nil, err
				}

				res = append(res, variable)

				err = p.readRune()
				if err == io.EOF {
					return res, nil
				}
				if err != nil {
					return nil, err
				}

				continue
			default:
				// We don't allow dollars before letters and underscores now,
				// as we might want to use these for $var support (without brackets) later.
				if unicode.IsLetter(p.next) || p.next == '_' {
					return nil, ErrUnexpectedDollar(p.lineNo, p.columnNo)
				}

				res = append(res, character(p.current))
				continue
			}
		case token.LBracket:
			switch p.next {
			case token.LBracket:
				secret, err := p.parseSecret()
				if err != nil {
					return nil, err
				}

				res = append(res, secret)

				err = p.readRune()
				if err == io.EOF {
					return res, nil
				}
				if err != nil {
					return nil, err
				}

				continue
			default:
				res = append(res, character(p.current))
				continue
			}
		case token.Backslash:
			if token.IsToken(p.next) {
				res = append(res, character(p.next))

				err = p.readRune()
				if err == io.EOF {
					return res, nil
				}
				if err != nil {
					return nil, err
				}
			} else {
				res = append(res, character(p.current))
			}

			continue
		default:
			res = append(res, character(p.current))
			continue
		}
	}
}

// parseVar parses the contents of a template variable up to the closing delimiter.
// The next character should be the last character of the opening delimiter ('{')
// when parseVar is called.
//
// When parseVar returns, the next character in the buffer is the closing delimiter
// of the template variable ('}').
func (p *v2Parser) parseVar() (node, error) {
	var buffer bytes.Buffer

	err := p.readRune()
	if err == io.EOF {
		return nil, ErrVariableTagNotClosed(p.lineNo, p.columnNo+1)
	}
	if err != nil {
		return nil, err
	}

	err = p.skipWhiteSpace()
	if err == io.EOF {
		return nil, ErrVariableTagNotClosed(p.lineNo, p.columnNo+1)
	}
	if err != nil {
		return nil, err
	}

	for {
		if token.IsRBracket(p.next) {
			return variable{
				key: buffer.String(),
			}, nil
		}

		if p.isAllowedWhiteSpace(p.next) {
			err := p.skipWhiteSpace()
			if err == io.EOF {
				return nil, ErrVariableTagNotClosed(p.lineNo, p.columnNo+1)
			}
			if err != nil {
				return nil, err
			}

			if token.IsRBracket(p.next) {
				return variable{
					key: buffer.String(),
				}, nil
			}

			return nil, ErrIllegalVariableCharacter(p.lineNo, p.columnNo, p.current)
		}

		if p.isVariableRune(p.next) {
			buffer.WriteRune(p.next)

			err := p.readRune()
			if err == io.EOF {
				return nil, ErrVariableTagNotClosed(p.lineNo, p.columnNo+1)
			}
			if err != nil {
				return nil, err
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

	err := p.readRune()
	if err == io.EOF {
		return nil, ErrSecretTagNotClosed(p.lineNo, p.columnNo+1)
	}
	if err != nil {
		return nil, err
	}

	err = p.skipWhiteSpace()
	if err == io.EOF {
		return nil, ErrSecretTagNotClosed(p.lineNo, p.columnNo+1)
	}
	if err != nil {
		return nil, err
	}

	for {
		err = p.readRune()
		if err == io.EOF {
			return nil, ErrSecretTagNotClosed(p.lineNo, p.columnNo+1)
		}
		if err != nil {
			return nil, err
		}

		if token.IsDollar(p.current) {
			if token.IsLBracket(p.next) {
				variable, err := p.parseVar()
				if err != nil {
					return nil, err
				}

				path = append(path, variable)

				err = p.readRune()
				if err == io.EOF {
					return nil, ErrSecretTagNotClosed(p.lineNo, p.columnNo+1)
				}
				if err != nil {
					return nil, err
				}

				continue
			}
			return nil, ErrIllegalSecretCharacter(p.lineNo, p.columnNo, p.current)
		}
		if p.isAllowedWhiteSpace(p.current) {
			err := p.skipWhiteSpace()
			if err == io.EOF {
				return nil, ErrSecretTagNotClosed(p.lineNo, p.columnNo+1)
			}
			if err != nil {
				return nil, err
			}

			if !token.IsRBracket(p.next) {
				return nil, ErrIllegalSecretCharacter(p.lineNo, p.columnNo, p.current)
			}

			err = p.readRune()
			if err == io.EOF {
				return nil, ErrSecretTagNotClosed(p.lineNo, p.columnNo+1)
			}
			if err != nil {
				return nil, err
			}

			if !token.IsRBracket(p.next) {
				return nil, ErrIllegalSecretCharacter(p.lineNo, p.columnNo-1, ' ')
			}

			return secret{
				path: path,
			}, nil
		}

		if token.IsRBracket(p.current) {
			if token.IsRBracket(p.next) {
				return secret{
					path: path,
				}, nil
			}

			return nil, ErrIllegalSecretCharacter(p.lineNo, p.columnNo, p.current)
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
func (p *v2Parser) isSecretPathRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == '/' || r == ':'
}

// isVariableRune returns whether the given rune is allowed to be used in a template variable key.
func (p *v2Parser) isVariableRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// isAllowedWhiteSpace returns whether the given rune is allowed as extra whitespace
// just after the opening tag and just before the closing tag.
func (p *v2Parser) isAllowedWhiteSpace(r rune) bool {
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

// Evaluate renders a template. It replaces all variable- and secret tags in the template.
func (t templateV2) Evaluate(vars map[string]string, sr SecretReader) (string, error) {
	ctx := context{
		vars:         vars,
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
