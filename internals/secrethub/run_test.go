package secrethub

import (
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/validation"
	"github.com/secrethub/secrethub-cli/internals/tpl"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func TestNewEnv(t *testing.T) {
	cases := map[string]struct {
		tpl          map[string]string
		replacements map[string]string
		expected     map[string]string
		err          error
	}{
		"success": {
			tpl: map[string]string{
				"yml": "foo: bar\nbaz: ${path/to/secret}",
				"env": "foo=bar\nbaz=${path/to/secret}",
			},
			replacements: map[string]string{
				"path/to/secret": "foobar",
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "foobar",
			},
		},
		"= in value": {
			tpl: map[string]string{
				"yml": "foo: foo=bar\nbar: baz",
				"env": "foo=foo=bar\nbar=baz",
			},
			expected: map[string]string{
				"foo": "foo=bar",
				"bar": "baz",
			},
		},
		"double ==": {
			tpl: map[string]string{
				"yml": "foo: =foobar\nbar: baz",
				"env": "foo==foobar\nbar=baz",
			},
			expected: map[string]string{
				"foo": "=foobar",
				"bar": "baz",
			},
		},
		"inject not closed yml": {
			tpl: map[string]string{
				"yml": "foo: ${path/to/secret",
			},
			err: ErrTemplate(1, errors.New("template is not formatted as key=value pairs")),
		},
		"inject not closed env": {
			tpl: map[string]string{
				"env": "foo=${path/to/secret",
			},
			err: ErrTemplate(1, tpl.ErrTagNotClosed("}")),
		},
		"nested yml": {
			tpl: map[string]string{
				"yml": `ROOT:
	SUB:
		NAME1: val1`,
			},
			err: ErrTemplate(1, errors.New("template is not formatted as key=value pairs")),
		},
		"invalid key yml": {
			tpl: map[string]string{
				"yml": "FOO\000: bar",
			},
			err: ErrTemplate(1, errors.New("template is not formatted as key=value pairs")),
		},
		"invalid key env": {
			tpl: map[string]string{
				"env": "FOO\000=bar",
			},
			err: ErrTemplate(1, validation.ErrInvalidEnvarName("FOO\000")),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			for format, tpl := range tc.tpl {
				t.Run(format, func(t *testing.T) {
					env, err := NewEnv(tpl)
					if err != nil {
						assert.Equal(t, err, tc.err)
					} else {
						actual, err := env.Env(tc.replacements)
						assert.Equal(t, err, tc.err)

						assert.Equal(t, actual, tc.expected)
					}
				})
			}

		})
	}
}
