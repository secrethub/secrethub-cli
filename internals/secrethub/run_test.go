package secrethub

import (
	"testing"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestNewEnv(t *testing.T) {
	cases := map[string]struct {
		tpl      map[string]string
		client   fakeclient.WithDataGetter
		expected map[string]string
		err      error
	}{
		"success": {
			tpl: map[string]string{
				"yml": "foo: bar\nbaz: ${path/to/secret}",
				"env": "foo=bar\nbaz=${path/to/secret}",
			},
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
		"inject not closed": {
			tpl: map[string]string{
				"yml": "foo: ${path/to/secret",
				"env": "foo=${path/to/secret",
			},
			expected: map[string]string{
				"foo": "${path/to/secret",
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

			for format, tpl := range tc.tpl {
				t.Run(format, func(t *testing.T) {
					actual, err := NewEnv(tpl).Env(client)
					assert.Equal(t, err, tc.err)

					assert.Equal(t, actual, tc.expected)
				})
			}

		})
	}
}
