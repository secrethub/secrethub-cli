package secrethub

import (
	"errors"
	"strings"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"

	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl/fakes"

	generictpl "github.com/secrethub/secrethub-cli/internals/tpl"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func elemEqual(t *testing.T, actual []envvar, expected []envvar) {
isExpected:
	for _, a := range actual {
		for _, e := range expected {
			if a == e {
				continue isExpected
			}
		}
		t.Errorf("%+v encountered but not expected", a)
	}

isEncountered:
	for _, e := range expected {
		for _, a := range actual {
			if a == e {
				continue isEncountered
			}
		}
		t.Errorf("%+v expected but not encountered", e)
	}
}

func TestParseDotEnv(t *testing.T) {
	cases := map[string]struct {
		raw      string
		expected []envvar
		err      error
	}{
		"success": {
			raw: "foo=bar\nbaz={{path/to/secret}}",
			expected: []envvar{
				{
					key:      "foo",
					value:    "bar",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 5,
				},
				{
					key:      "baz",
					value:    "{{path/to/secret}}",
					lineNo:   2,
					keyColNo: 1,
					valColNo: 5,
				},
			},
		},
		"success with spaces": {
			raw: "key = value",
			expected: []envvar{
				{
					key:      "key",
					value:    "value",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 7,
				},
			},
		},
		"success with multiple spaces after key": {
			raw: "key    = value",
			expected: []envvar{
				{
					key:      "key",
					value:    "value",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 10,
				},
			},
		},
		"success with multiple spaces before value": {
			raw: "key =  value",
			expected: []envvar{
				{
					key:      "key",
					value:    "value",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 8,
				},
			},
		},
		"success with leading space": {
			raw: " key = value",
			expected: []envvar{
				{
					key:      "key",
					value:    "value",
					lineNo:   1,
					keyColNo: 2,
					valColNo: 8,
				},
			},
		},
		"success with leading tab": {
			raw: "\tkey = value",
			expected: []envvar{
				{
					key:      "key",
					value:    "value",
					lineNo:   1,
					keyColNo: 2,
					valColNo: 8,
				},
			},
		},
		"success with trailing space": {
			raw: "key = value ",
			expected: []envvar{
				{
					key:      "key",
					value:    "value",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 7,
				},
			},
		},
		"success with tabs": {
			raw: "key\t=\tvalue",
			expected: []envvar{
				{
					key:      "key",
					value:    "value",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 7,
				},
			},
		},
		"success with single quotes": {
			raw: "key='value'",
			expected: []envvar{
				{
					key:      "key",
					value:    "value",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 6,
				},
			},
		},
		"success with double quotes": {
			raw: `key="value"`,
			expected: []envvar{
				{
					key:      "key",
					value:    "value",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 6,
				},
			},
		},
		"success with quotes and whitespace": {
			raw: "key = 'value'",
			expected: []envvar{
				{
					key:      "key",
					value:    "value",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 8,
				},
			},
		},
		"success comment": {
			raw: "# database\nDB_USER = user\nDB_PASS = pass",
			expected: []envvar{
				{
					key:      "DB_USER",
					value:    "user",
					lineNo:   2,
					keyColNo: 1,
					valColNo: 11,
				},
				{
					key:      "DB_PASS",
					value:    "pass",
					lineNo:   3,
					keyColNo: 1,
					valColNo: 11,
				},
			},
		},
		"success comment prefixed with spaces": {
			raw: "    # database\nDB_USER = user\nDB_PASS = pass",
			expected: []envvar{
				{
					key:      "DB_USER",
					value:    "user",
					lineNo:   2,
					keyColNo: 1,
					valColNo: 11,
				},
				{
					key:      "DB_PASS",
					value:    "pass",
					lineNo:   3,
					keyColNo: 1,
					valColNo: 11,
				},
			},
		},
		"success comment prefixed with a tab": {
			raw: "\t# database\nDB_USER = user\nDB_PASS = pass",
			expected: []envvar{
				{
					key:      "DB_USER",
					value:    "user",
					lineNo:   2,
					keyColNo: 1,
					valColNo: 11,
				},
				{
					key:      "DB_PASS",
					value:    "pass",
					lineNo:   3,
					keyColNo: 1,
					valColNo: 11,
				},
			},
		},
		"success empty lines": {
			raw: "foo=bar\n\nbar=baz",
			expected: []envvar{
				{
					key:      "foo",
					value:    "bar",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 5,
				},
				{
					key:      "bar",
					value:    "baz",
					lineNo:   3,
					keyColNo: 1,
					valColNo: 5,
				},
			},
		},
		"success line with only spaces": {
			raw: "foo=bar\n    \nbar = baz",
			expected: []envvar{
				{
					key:      "foo",
					value:    "bar",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 5,
				},
				{
					key:      "bar",
					value:    "baz",
					lineNo:   3,
					keyColNo: 1,
					valColNo: 7,
				},
			},
		},
		"= sign in value": {
			raw: "foo=foo=bar",
			expected: []envvar{
				{
					key:      "foo",
					value:    "foo=bar",
					lineNo:   1,
					keyColNo: 1,
					valColNo: 5,
				},
			},
		},
		"invalid": {
			raw: "foobar",
			err: ErrTemplate(1, errors.New("template is not formatted as key=value pairs")),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual, err := parseDotEnv(strings.NewReader(tc.raw))

			elemEqual(t, actual, tc.expected)
			assert.Equal(t, err, tc.err)
		})
	}
}

func TestParseYML(t *testing.T) {
	cases := map[string]struct {
		raw      string
		expected []envvar
		err      error
	}{
		"success": {
			raw: "foo: bar\nbaz: ${path/to/secret}",
			expected: []envvar{
				{
					key:    "foo",
					value:  "bar",
					lineNo: -1,
				},
				{
					key:    "baz",
					value:  "${path/to/secret}",
					lineNo: -1,
				},
			},
		},
		"= in value": {
			raw: "foo: foo=bar\nbar: baz",
			expected: []envvar{
				{
					key:    "foo",
					value:  "foo=bar",
					lineNo: -1,
				},
				{
					key:    "bar",
					value:  "baz",
					lineNo: -1,
				},
			},
		},
		"nested yml": {
			raw: "ROOT:\n\tSUB\n\t\tNAME: val1",
			err: errors.New("yaml: line 2: found character that cannot start any token"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual, err := parseYML(strings.NewReader(tc.raw))

			elemEqual(t, actual, tc.expected)
			assert.Equal(t, err, tc.err)
		})
	}
}

func TestNewEnv(t *testing.T) {
	cases := map[string]struct {
		raw          string
		replacements map[string]string
		templateVars map[string]string
		expected     map[string]string
		err          error
	}{
		"success": {
			raw: "foo=bar\nbaz={{path/to/secret}}",
			replacements: map[string]string{
				"path/to/secret": "val",
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "val",
			},
		},
		"success with vars": {
			raw: "foo=bar\nbaz={{${app}/db/pass}}",
			replacements: map[string]string{
				"company/application/db/pass": "secret",
			},
			templateVars: map[string]string{
				"app": "company/application",
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "secret",
			},
		},
		"success with var in key": {
			raw: "${var}=value",
			templateVars: map[string]string{
				"var": "key",
			},
			expected: map[string]string{
				"key": "value",
			},
		},
		"success yml": {
			raw: "foo: bar\nbaz: ${path/to/secret}",
			replacements: map[string]string{
				"path/to/secret": "val",
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "val",
			},
		},
		"secret not allowed in key": {
			raw: "{{ path/to/secret }}key=value",
			err: ErrSecretsNotAllowedInKey,
		},
		"yml template error": {
			raw: "foo: bar: baz",
			err: ErrTemplate(1, errors.New("template is not formatted as key=value pairs")),
		},
		"yml secret template error": {
			raw: "foo: ${path/to/secret",
			err: generictpl.ErrTagNotClosed("}"),
		},
		"secret template error": {
			raw: "foo={{path/to/secret",
			err: tpl.ErrSecretTagNotClosed(1, 21),
		},
		"secret template error second line": {
			raw: "foo=bar\nbar={{ error@secretpath }}",
			err: tpl.ErrIllegalSecretCharacter(2, 13, '@'),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			env, err := NewEnv(strings.NewReader(tc.raw), tc.templateVars)
			if err != nil {
				assert.Equal(t, err, tc.err)
			} else {
				actual, err := env.Env(map[string]string{}, fakes.FakeSecretReader{Secrets: tc.replacements})
				assert.Equal(t, err, tc.err)

				assert.Equal(t, actual, tc.expected)
			}
		})
	}
}

func TestRunCommand_Run(t *testing.T) {
	cases := map[string]struct {
		command RunCommand
		err     error
	}{
		"invalid template var: start with a number": {
			command: RunCommand{
				templateVars: map[string]string{
					"0foo": "value",
				},
				envar: map[string]string{},
			},
			err: ErrInvalidTemplateVar("0foo"),
		},
		"invalid template var: illegal character": {
			command: RunCommand{
				templateVars: map[string]string{
					"foo@bar": "value",
				},
				envar: map[string]string{},
			},
			err: ErrInvalidTemplateVar("foo@bar"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.command.Run()
			assert.Equal(t, err, tc.err)
		})
	}
}

func TestTrimQuotes(t *testing.T) {
	cases := map[string]struct {
		in       string
		expected string
	}{
		"unquoted": {
			in:       `foo`,
			expected: `foo`,
		},
		"single quoted": {
			in:       `'foo'`,
			expected: `foo`,
		},
		"double quoted": {
			in:       `"foo"`,
			expected: `foo`,
		},
		"maintain quotes inside unquoted value": {
			in:       `{"foo":"bar"}`,
			expected: `{"foo":"bar"}`,
		},
		"empty string": {
			in:       "",
			expected: "",
		},
		"single quoted empty string": {
			in:       `''`,
			expected: ``,
		},
		"double qouted empty string": {
			in:       `""`,
			expected: ``,
		},
		"single quote wrapped in single quote": {
			in:       `''foo''`,
			expected: `'foo'`,
		},
		"single quote wrapped in double quote": {
			in:       `"'foo'"`,
			expected: `'foo'`,
		},
		"double quote wrapped in double quote": {
			in:       `""foo""`,
			expected: `"foo"`,
		},
		"double quote wrapped in single quote": {
			in:       `'"foo"'`,
			expected: `"foo"`,
		},
		"single quote opened but not closed": {
			in:       `'foo`,
			expected: `'foo`,
		},
		"double quote opened but not closed": {
			in:       `"foo`,
			expected: `"foo`,
		},
		"single quote closed but not opened": {
			in:       `foo'`,
			expected: `foo'`,
		},
		"double quote closed but not opened": {
			in:       `foo"`,
			expected: `foo"`,
		},
		"single quoted with inner leading whitespace": {
			in:       `' foo'`,
			expected: ` foo`,
		},
		"double quoted with inner leading whitespace": {
			in:       `" foo"`,
			expected: ` foo`,
		},
		"single quoted with inner trailing whitespace": {
			in:       `'foo '`,
			expected: `foo `,
		},
		"double quoted with inner trailing whitespace": {
			in:       `"foo "`,
			expected: `foo `,
		},

		// Trimming OUTER whitespace is explicitly not the responsibility of this function.
		"single quoted with outer leading whitespace": {
			in:       ` 'foo'`,
			expected: ` 'foo'`,
		},
		"double quoted with outer leading whitespace": {
			in:       ` "foo"`,
			expected: ` "foo"`,
		},
		"single quoted with outer trailing whitespace": {
			in:       `'foo' `,
			expected: `'foo' `,
		},
		"double quoted with outer trailing whitespace": {
			in:       `"foo" `,
			expected: `"foo" `,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual, _ := trimQuotes(tc.in)

			assert.Equal(t, actual, tc.expected)
		})
	}
}
