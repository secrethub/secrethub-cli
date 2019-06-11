package tpl_test

import (
	"testing"

	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"
	generictpl "github.com/secrethub/secrethub-cli/internals/tpl"
	"github.com/secrethub/secrethub-go/internals/assert"
)

func TestV2(t *testing.T) {
	cases := map[string]struct {
		raw     string
		vars    map[string]string
		secrets map[string]string

		expected         string
		parseErr         error
		injectVarsErr    error
		injectSecretsErr error
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
		"template var": {
			raw: "hello {{ ${app}/greeting }}",
			vars: map[string]string{
				"app": "company/helloworld",
			},
			secrets: map[string]string{
				"company/helloworld/greeting": "world",
			},
			expected: "hello world",
		},
		"missing var": {
			raw:  "hello {{ ${app}/greeting }}",
			vars: map[string]string{},
			secrets: map[string]string{
				"company/helloworld/greeting": "world",
			},
			injectVarsErr: generictpl.ErrKeyNotFound("app"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			parsed, err := tpl.NewV2Parser().Parse(tc.raw)
			assert.Equal(t, err, tc.parseErr)

			if err != nil {
				return
			}

			varsInjected, err := parsed.InjectVars(tc.vars)
			assert.Equal(t, err, tc.injectVarsErr)

			if err != nil {
				return
			}

			actual, err := varsInjected.InjectSecrets(tc.secrets)
			assert.Equal(t, err, tc.injectSecretsErr)
			assert.Equal(t, actual, tc.expected)
		})
	}
}
