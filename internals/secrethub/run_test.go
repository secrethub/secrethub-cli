package secrethub

import (
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
	"testing"
)

func TestNewEnv(t *testing.T) {
	cases := map[string]struct {
		template string
		client   fakeclient.WithDataGetter
		expected map[string]string
		err      error
	}{
		"yml": {
			template: "foo: bar\nbaz: ${path/to/secret}",
			client: fakeclient.WithDataGetter{
				ReturnsVersion: &api.SecretVersion{
					Data: []byte("foobar"),
				},
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "foobar",
			},
		},
		"env": {
			template: "foo=bar\nbaz=${path/to/secret}",
			client: fakeclient.WithDataGetter{
				ReturnsVersion: &api.SecretVersion{
					Data: []byte("foobar"),
				},
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "foobar",
			},
		},
		"env = in value": {
			template: "foo=foo=bar",
			expected: map[string]string{
				"foo": "foo=bar",
			},
		},
		"env double ==": {
			template: "foo==foobar",
			expected: map[string]string{
				"foo": "=foobar",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client := fakeclient.Client{
				SecretService: &fakeclient.SecretService{
					VersionService: &fakeclient.SecretVersionService{
						WithDataGetter: tc.client,
					},
				},
			}

			env, err := NewEnv(tc.template)
			assert.OK(t, err)

			actual, err := env.Env(client)
			assert.OK(t, err)

			assert.Equal(t, actual, tc.expected)
		})
	}
}
