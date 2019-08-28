package secrethub

import (
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func Test_roleNameFromRole(t *testing.T) {
	cases := map[string]struct {
		role     string
		expected string
	}{
		"role name": {
			role:     "my-role",
			expected: "my-role",
		},
		"role prefix": {
			role:     "role/my-role",
			expected: "my-role",
		},
		"arn": {
			role:     "arn:aws:iam::123456789012:role/my-role",
			expected: "my-role",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual := roleNameFromRole(tc.role)

			assert.Equal(t, actual, tc.expected)
		})
	}
}
