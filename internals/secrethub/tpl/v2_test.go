package tpl

import (
	"bytes"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl/fakes"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func TestParserV2_parse(t *testing.T) {
	cases := map[string]struct {
		input    string
		expected []node
		err      error
	}{
		"empty input": {
			input:    "",
			expected: []node{},
		},
		"no vars, no secrets": {
			input: "hello world",
			expected: []node{
				character('h'),
				character('e'),
				character('l'),
				character('l'),
				character('o'),
				character(' '),
				character('w'),
				character('o'),
				character('r'),
				character('l'),
				character('d'),
			},
		},
		"start with var": {
			input: "${var} world",
			expected: []node{
				variable{
					key: "var",
				},
				character(' '),
				character('w'),
				character('o'),
				character('r'),
				character('l'),
				character('d'),
			},
		},
		"end with var": {
			input: "hello ${var}",
			expected: []node{
				character('h'),
				character('e'),
				character('l'),
				character('l'),
				character('o'),
				character(' '),
				variable{
					key: "var",
				},
			},
		},
		"var in middle": {
			input: "hello ${var} world",
			expected: []node{
				character('h'),
				character('e'),
				character('l'),
				character('l'),
				character('o'),
				character(' '),
				variable{
					key: "var",
				},
				character(' '),
				character('w'),
				character('o'),
				character('r'),
				character('l'),
				character('d'),
			},
		},
		"secret path": {
			input: "{{path/to/secret}}",
			expected: []node{
				secret{
					path: []node{
						character('p'),
						character('a'),
						character('t'),
						character('h'),
						character('/'),
						character('t'),
						character('o'),
						character('/'),
						character('s'),
						character('e'),
						character('c'),
						character('r'),
						character('e'),
						character('t'),
					},
				},
			},
		},
		"secret path in middle": {
			input: "hello {{path/to/secret}} secret",
			expected: []node{
				character('h'),
				character('e'),
				character('l'),
				character('l'),
				character('o'),
				character(' '),
				secret{
					path: []node{
						character('p'),
						character('a'),
						character('t'),
						character('h'),
						character('/'),
						character('t'),
						character('o'),
						character('/'),
						character('s'),
						character('e'),
						character('c'),
						character('r'),
						character('e'),
						character('t'),
					},
				},
				character(' '),
				character('s'),
				character('e'),
				character('c'),
				character('r'),
				character('e'),
				character('t'),
			},
		},
		"two secret tags": {
			input: "{{ a }}{{ b }}",
			expected: []node{
				secret{
					path: []node{
						character('a'),
					},
				},
				secret{
					path: []node{
						character('b'),
					},
				},
			},
		},
		"variable in secret path at start": {
			input: "{{${var}/secret}}",
			expected: []node{
				secret{
					path: []node{
						variable{
							key: "var",
						},
						character('/'),
						character('s'),
						character('e'),
						character('c'),
						character('r'),
						character('e'),
						character('t'),
					},
				},
			},
		},
		"variable in secret path at end": {
			input: "{{secret${var}}}",
			expected: []node{
				secret{
					path: []node{
						character('s'),
						character('e'),
						character('c'),
						character('r'),
						character('e'),
						character('t'),
						variable{
							key: "var",
						},
					},
				},
			},
		},
		"variable in secret path at end with space": {
			input: "{{ secret${var} }}",
			expected: []node{
				secret{
					path: []node{
						character('s'),
						character('e'),
						character('c'),
						character('r'),
						character('e'),
						character('t'),
						variable{
							key: "var",
						},
					},
				},
			},
		},
		"variable in secret path in middle": {
			input: "{{path/to/${var}/secret}}",
			expected: []node{
				secret{
					path: []node{
						character('p'),
						character('a'),
						character('t'),
						character('h'),
						character('/'),
						character('t'),
						character('o'),
						character('/'),
						variable{
							key: "var",
						},
						character('/'),
						character('s'),
						character('e'),
						character('c'),
						character('r'),
						character('e'),
						character('t'),
					},
				},
			},
		},
		"variable with spaces": {
			input: "${ var }",
			expected: []node{
				variable{
					key: "var",
				},
			},
		},
		"secret with spaces": {
			input: "{{ path/to/secret }}",
			expected: []node{
				secret{
					path: []node{
						character('p'),
						character('a'),
						character('t'),
						character('h'),
						character('/'),
						character('t'),
						character('o'),
						character('/'),
						character('s'),
						character('e'),
						character('c'),
						character('r'),
						character('e'),
						character('t'),
					},
				},
			},
		},
		"{ and } chars used": {
			input: `{"key": "value"}`,
			expected: []node{
				character('{'),
				character('"'),
				character('k'),
				character('e'),
				character('y'),
				character('"'),
				character(':'),
				character(' '),
				character('"'),
				character('v'),
				character('a'),
				character('l'),
				character('u'),
				character('e'),
				character('"'),
				character('}'),
			},
		},
		"}} used outside secret tag": {
			input: `{"a": {"b": "c"}}`,
			expected: []node{
				character('{'),
				character('"'),
				character('a'),
				character('"'),
				character(':'),
				character(' '),
				character('{'),
				character('"'),
				character('b'),
				character('"'),
				character(':'),
				character(' '),
				character('"'),
				character('c'),
				character('"'),
				character('}'),
				character('}'),
			},
		},
		"$ used": {
			input: `$12.50`,
			expected: []node{
				character('$'),
				character('1'),
				character('2'),
				character('.'),
				character('5'),
				character('0'),
			},
		},
		"escaped dollar": {
			input: `\$`,
			expected: []node{
				character('$'),
			},
		},
		"escaped dollar + bracket": {
			input: `\${var}`,
			expected: []node{
				character('$'),
				character('{'),
				character('v'),
				character('a'),
				character('r'),
				character('}'),
			},
		},
		"escaped double bracket": {
			input: `\{{ path }}`,
			expected: []node{
				character('{'),
				character('{'),
				character(' '),
				character('p'),
				character('a'),
				character('t'),
				character('h'),
				character(' '),
				character('}'),
				character('}'),
			},
		},
		"escaped backslash": {
			input: `\\`,
			expected: []node{
				character('\\'),
			},
		},
		"escaped opening bracket": {
			input: `\{`,
			expected: []node{
				character('{'),
			},
		},
		"escaped closing bracket": {
			input: `\}`,
			expected: []node{
				character('}'),
			},
		},
		"backslash followed by letter": {
			input: `\a`,
			expected: []node{
				character('\\'),
				character('a'),
			},
		},
		"$ followed by lowercase letter": {
			input: "$var",
			err:   ErrUnexpectedDollar(1, 1),
		},
		"$ followed by uppercase letter": {
			input: "$VAR",
			err:   ErrUnexpectedDollar(1, 1),
		},
		"$ followed by underscore": {
			input: "$_var",
			err:   ErrUnexpectedDollar(1, 1),
		},
		"illegal variable space": {
			input: "${ va r }",
			err:   ErrIllegalVariableCharacter(1, 6, ' '),
		},
		"illegal double variable space": {
			input: "${ va  r }",
			err:   ErrIllegalVariableCharacter(1, 6, ' '),
		},
		"illegal variable tab": {
			input: "${ va\tr }",
			err:   ErrIllegalVariableCharacter(1, 6, '\t'),
		},
		"illegal variable tab followed by space": {
			input: "${ va\t r }",
			err:   ErrIllegalVariableCharacter(1, 6, '\t'),
		},
		"illegal secret space": {
			input: "{{ secret with space }}",
			err:   ErrIllegalSecretCharacter(1, 10, ' '),
		},
		"illegal secret space followed by bracket": {
			input: "{{ secret }with space }}",
			err:   ErrIllegalSecretCharacter(1, 10, ' '),
		},
		"illegal secret tab": {
			input: "{{ secret\twith a tab }}",
			err:   ErrIllegalSecretCharacter(1, 10, '\t'),
		},
		"illegal secret tab followed by bracket": {
			input: "{{ secret\t}with tab }}",
			err:   ErrIllegalSecretCharacter(1, 10, '\t'),
		},
		"illegal double secret space": {
			input: "{{ secret  with two spaces }}",
			err:   ErrIllegalSecretCharacter(1, 10, ' '),
		},
		"illegal secret tab followed by space": {
			input: "{{ secret\t with tab and space }}",
			err:   ErrIllegalSecretCharacter(1, 10, '\t'),
		},
		"illegal variable character": {
			input: "${ var@var }",
			err:   ErrIllegalVariableCharacter(1, 7, '@'),
		},
		"illegal secret character": {
			input: "{{ a@b }}",
			err:   ErrIllegalSecretCharacter(1, 5, '@'),
		},
		"illegal { at start of secret tag": {
			input: "{{{ path/to/secret }}}",
			err:   ErrIllegalSecretCharacter(1, 3, '{'),
		},
		"illegal secret character $": {
			input: "{{ a$b }}",
			err:   ErrIllegalSecretCharacter(1, 5, '$'),
		},
		"illegal variable char in secret tag": {
			input: "{{ path/with/${var@b} }}",
			err:   ErrIllegalVariableCharacter(1, 19, '@'),
		},
		"error on new line": {
			input: "{{ path/to/secret }}\n{{ a%b }}",
			err:   ErrIllegalSecretCharacter(2, 5, '%'),
		},
		"secret tag not closed": {
			input: "{{ path",
			err:   ErrSecretTagNotClosed(1, 8),
		},
		"secret tag not closed after space at end": {
			input: "{{ path ",
			err:   ErrSecretTagNotClosed(1, 9),
		},
		"secret tag not closed after multiple space at end": {
			input: "{{ path  ",
			err:   ErrSecretTagNotClosed(1, 10),
		},
		"secret tag not closed after space at start": {
			input: "{{ ",
			err:   ErrSecretTagNotClosed(1, 4),
		},
		"secret tag not closed at start of tag": {
			input: "{{",
			err:   ErrSecretTagNotClosed(1, 3),
		},
		"secret tag not closed after var end": {
			input: "{{ foo/${var}",
			err:   ErrSecretTagNotClosed(1, 14),
		},
		"secret tag not closed after space after var": {
			input: "{{ foo/${var} ",
			err:   ErrSecretTagNotClosed(1, 15),
		},
		"secret tag not closed after first bracket": {
			input: "{{ foo/bar }",
			err:   ErrSecretTagNotClosed(1, 13),
		},
		"variable tag not closed": {
			input: "${ var",
			err:   ErrVariableTagNotClosed(1, 7),
		},
		"variable tag not closed after space at start": {
			input: "${ ",
			err:   ErrVariableTagNotClosed(1, 4),
		},
		"variable tag not closed after space at end": {
			input: "${ var ",
			err:   ErrVariableTagNotClosed(1, 8),
		},
		"variable tag not closed at start of tag": {
			input: "${",
			err:   ErrVariableTagNotClosed(1, 3),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			parser := newV2Parser(bytes.NewBufferString(tc.input), 1, 1)
			actual, err := parser.parse()

			assert.Equal(t, actual, tc.expected)
			assert.Equal(t, err, tc.err)
		})
	}
}

func TestV2(t *testing.T) {
	cases := map[string]struct {
		raw     string
		vars    map[string]string
		secrets map[string]string

		expected string
		parseErr error
		evalErr  error
	}{
		"no secrets": {
			raw:      "hello world",
			expected: "hello world",
		},
		"secret": {
			raw: "hello {{ secret }}",
			secrets: map[string]string{
				"secret": "world",
			},
			expected: "hello world",
		},
		"template var in secret": {
			raw: "hello {{ ${app}/greeting }}",
			vars: map[string]string{
				"app": "company/helloworld",
			},
			secrets: map[string]string{
				"company/helloworld/greeting": "world",
			},
			expected: "hello world",
		},
		"end with template var": {
			raw: "hello {{company/helloworld/${greeting}}}",
			vars: map[string]string{
				"greeting": "hello",
			},
			secrets: map[string]string{
				"company/helloworld/hello": "world",
			},
			expected: "hello world",
		},
		"missing var": {
			raw:  "hello {{ ${app}/greeting }}",
			vars: map[string]string{},
			secrets: map[string]string{
				"company/helloworld/greeting": "world",
			},
			evalErr: ErrTemplateVarNotFound("app"),
		},
		"missing var with spaces": {
			raw:  "hello {{ ${ app }/greeting }}",
			vars: map[string]string{},
			secrets: map[string]string{
				"company/helloworld/greeting": "world",
			},
			evalErr: ErrTemplateVarNotFound("app"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			parsed, err := NewV2Parser().Parse(tc.raw, 1, 1)
			assert.Equal(t, err, tc.parseErr)

			if err != nil {
				return
			}

			actual, err := parsed.Evaluate(tc.vars, fakes.FakeSecretReader{Secrets: tc.secrets})
			assert.Equal(t, err, tc.evalErr)
			assert.Equal(t, actual, tc.expected)
		})
	}
}
