package secrethub

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	randchargeneratorfakes "github.com/secrethub/secrethub-go/pkg/randchar/fakes"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestGenerateSecretCommand_run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd               GenerateSecretCommand
		service           fakeclient.SecretService
		clientCreationErr error
		path              api.SecretPath
		data              []byte
		err               error
		out               string
	}{
		"success": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: []byte("random generated secret"),
					Err: nil,
				},
				path:   "namespace/repo/secret",
				length: 23,
			},
			service: fakeclient.SecretService{
				Writer: fakeclient.Writer{
					ReturnsVersion: &api.SecretVersion{
						Version: 1,
					},
					Err: nil,
				},
			},
			path: "namespace/repo/secret",
			data: []byte("random generated secret"),
			err:  nil,
			out: "Generating secret value...\n" +
				"Writing secret value...\n" +
				"Write complete! A randomly generated secret has been written to namespace/repo/secret:1.\n",
		},
		"length 0": {
			cmd: GenerateSecretCommand{
				length: 0,
			},
			err: ErrInvalidRandLength,
			out: "",
		},
		"negative length": {
			cmd: GenerateSecretCommand{
				length: -1,
			},
			err: ErrInvalidRandLength,
			out: "",
		},
		"generate error": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: nil,
					Err: testErr,
				},
				length: 22,
			},
			err: testErr,
			out: "Generating secret value...\n",
		},
		"client creation error": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: []byte("random generated secret"),
					Err: nil,
				},
				length: 23,
			},
			clientCreationErr: testErr,
			err:               testErr,
			out:               "Generating secret value...\n",
		},
		"client error": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: []byte("random generated secret"),
					Err: nil,
				},
				path:   "namespace/repo/secret",
				length: 23,
			},
			service: fakeclient.SecretService{
				Writer: fakeclient.Writer{
					ReturnsVersion: nil,
					Err:            testErr,
				},
			},
			path: "namespace/repo/secret",
			data: []byte("random generated secret"),
			err:  testErr,
			out: "Generating secret value...\n" +
				"Writing secret value...\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					SecretService: &tc.service,
				}, tc.clientCreationErr
			}

			io := ui.NewFakeIO()
			tc.cmd.io = io

			// Act
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, tc.service.Writer.ArgPath, tc.path)
			assert.Equal(t, tc.service.Writer.ArgData, tc.data)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
