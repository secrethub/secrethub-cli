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
					key:          "foo",
					value:        "bar",
					lineNumber:   1,
					columnNumber: 5,
				},
				{
					key:          "baz",
					value:        "{{path/to/secret}}",
					lineNumber:   2,
					columnNumber: 5,
				},
			},
		},
		"success with spaces": {
			raw: "key = value",
			expected: []envvar{
				{
					key:          "key",
					value:        "value",
					lineNumber:   1,
					columnNumber: 7,
				},
			},
		},
		"success with multiple spaces": {
			raw: "key    = value",
			expected: []envvar{
				{
					key:          "key",
					value:        "value",
					lineNumber:   1,
					columnNumber: 10,
				},
			},
		},
		"= sign in value": {
			raw: "foo=foo=bar",
			expected: []envvar{
				{
					key:          "foo",
					value:        "foo=bar",
					lineNumber:   1,
					columnNumber: 5,
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
					key:        "foo",
					value:      "bar",
					lineNumber: -1,
				},
				{
					key:        "baz",
					value:      "${path/to/secret}",
					lineNumber: -1,
				},
			},
		},
		"= in value": {
			raw: "foo: foo=bar\nbar: baz",
			expected: []envvar{
				{
					key:        "foo",
					value:      "foo=bar",
					lineNumber: -1,
				},
				{
					key:        "bar",
					value:      "baz",
					lineNumber: -1,
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
			err: tpl.ErrIllegalSecretCharacter('@', 2, 13),
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
