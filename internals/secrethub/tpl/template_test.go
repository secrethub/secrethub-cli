package tpl

import (
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func TestIsV1Template(t *testing.T) {
	cases := map[string]struct {
		raw      string
		expected bool
	}{
		"v1 tag without spaces": {
			raw:      "${path/to/secret}",
			expected: true,
		},
		"v1 tag with spaces": {
			raw:      "${ path/to/secret }",
			expected: true,
		},
		"v1 tag with version": {
			raw:      "${ path/to/secret:1 }",
			expected: true,
		},
		"v1 tag with latest version": {
			raw:      "${ path/to/secret:latest }",
			expected: true,
		},
		"v1 tag with 1 dir": {
			raw:      "${ path/to/dir/secret }",
			expected: true,
		},
		"v2 variable": {
			raw:      "${var}foo",
			expected: false,
		},
		"v2 unescaped var": {
			raw:      "$var",
			expected: false,
		},
		"v1 tag with tabs": {
			raw:      "${\tpath/to/secret\t}",
			expected: true,
		},
		"v1 tag without secret path": { // This is invalid in both template syntaxes
			raw:      "${ namespace/repo }",
			expected: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual := IsV1Template([]byte(tc.raw))

			assert.Equal(t, actual, tc.expected)
		})
	}
}
