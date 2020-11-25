package secrethub

import (
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func Test_envkeysToPaths(t *testing.T) {
	cases := map[string]struct {
		envkeys  []string
		expected map[string]string
	}{
		"single key": {
			envkeys: []string{
				"STRIPE_API_KEY",
			},
			expected: map[string]string{
				"STRIPE_API_KEY": "stripe-api-key",
			},
		},
		"multiple different keys": {
			envkeys: []string{
				"STRIPE_API_KEY",
				"DB_USER",
			},
			expected: map[string]string{
				"STRIPE_API_KEY": "stripe-api-key",
				"DB_USER":        "db-user",
			},
		},
		"keys with common prefix": {
			envkeys: []string{
				"STRIPE_API_KEY",
				"DB_USER",
				"DB_PASSWORD",
			},
			expected: map[string]string{
				"STRIPE_API_KEY": "stripe-api-key",
				"DB_USER":        "db/user",
				"DB_PASSWORD":    "db/password",
			},
		},
		"prefix with multiple underscores": {
			envkeys: []string{
				"MY_APP_STRIPE_API_KEY",
				"MY_APP_DB_PASSWORD",
			},
			expected: map[string]string{
				"MY_APP_STRIPE_API_KEY": "my-app/stripe-api-key",
				"MY_APP_DB_PASSWORD":    "my-app/db-password",
			},
		},
		"two levels of directories": {
			envkeys: []string{
				"MY_APP_STRIPE_API_KEY",
				"MY_APP_DB_USER",
				"MY_APP_DB_PASSWORD",
			},
			expected: map[string]string{
				"MY_APP_STRIPE_API_KEY": "my-app/stripe-api-key",
				"MY_APP_DB_USER":        "my-app/db/user",
				"MY_APP_DB_PASSWORD":    "my-app/db/password",
			},
		},
		"key without underscores": {
			envkeys: []string{
				"ENVIRONMENT",
			},
			expected: map[string]string{
				"ENVIRONMENT": "environment",
			},
		},
		"one key equal to the start of another": {
			envkeys: []string{
				"STRIPE_API_KEY",
				"STRIPE_API",
			},
			expected: map[string]string{
				"STRIPE_API_KEY": "stripe-api-key",
				"STRIPE_API":     "stripe-api",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual := envkeysToPaths(tc.envkeys)

			assert.Equal(t, actual, tc.expected)
		})
	}
}
