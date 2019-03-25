package secrethub

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"strings"
	"testing"

	"github.com/keylockerbv/secrethub-cli/pkg/validation"
)

const nested = `ROOT:
  SUB:
    NAME1: val1
    NAME2: val2
ROOT2: val3	
`

const multiline = `NAME1: |
  foo
  bar
NAME2: baz
`

func TestParseEnvFile(t *testing.T) {
	cases := map[string]struct {
		in       string
		expected map[string]string
		errcheck func(t *testing.T, err error)
	}{
		"empty": {
			in:       "",
			expected: map[string]string{},
		},
		"simple": {
			in: "FOO: bar\nBAR: baz",
			expected: map[string]string{
				"FOO": "bar",
				"BAR": "baz",
			},
		},
		"nested": {
			in:       nested,
			expected: nil,
			errcheck: func(t *testing.T, err error) {
				t.Helper()
				if !strings.Contains(err.Error(), "yaml: unmarshal errors:") {
					t.Errorf("unexpected error: %v (actual) != yaml: unmarshal errors (expected)", err)
				}
			},
		},
		"multiline": {
			in: multiline,
			expected: map[string]string{
				"NAME1": "foo\nbar\n", // Literal values with | contain newlines.
				"NAME2": "baz",
			},
		},
		"empty_line": {
			in: "FOO: bar\n \nBAR: baz",
			expected: map[string]string{
				"FOO": "bar",
				"BAR": "baz",
			},
		},
		"duplicates": {
			in: "FOO: bar\nFOO: baz",
			expected: map[string]string{
				"FOO": "baz",
			},
		},
		"numbers": {
			in: "FOO: 123",
			expected: map[string]string{
				"FOO": "123",
			},
		},
		"invalid_name": {
			in: "FOO=: bar",
			errcheck: func(t *testing.T, err error) {
				assert.Equal(t, err, validation.ErrInvalidEnvarName("FOO="))
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			actual, err := parseEnvFile(tc.in)

			// Assert
			if tc.errcheck != nil {
				tc.errcheck(t, err)
			} else {
				assert.OK(t, err)
			}
			assert.Equal(t, actual, tc.expected)
		})
	}
}
