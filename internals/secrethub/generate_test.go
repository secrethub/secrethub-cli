package secrethub

import (
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	randchargeneratorfakes "github.com/secrethub/secrethub-go/pkg/randchar/fakes"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func newIntValue(v int) intValue {
	return intValue{v: &v}
}

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
				firstArg: "namespace/repo/secret",
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
		"length flag": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: []byte("random generated secret"),
					Err: nil,
				},
				firstArg:   "namespace/repo/secret",
				lengthFlag: newIntValue(24),
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
		"length flag and arg": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: []byte("random generated secret"),
					Err: nil,
				},
				firstArg:   "rand",
				secondArg:  "namespace/repo/secret",
				lengthFlag: newIntValue(24),
				lengthArg:  newIntValue(24),
			},
			err: ErrCannotUseLengthArgAndFlag,
		},
		"backwards compatibility rand": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: []byte("random generated secret"),
					Err: nil,
				},
				firstArg:  "rand",
				secondArg: "namespace/repo/secret",
				lengthArg: newIntValue(23),
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
		"length arg 0": {
			cmd: GenerateSecretCommand{
				firstArg:  "rand",
				secondArg: "namespace/repo/secret",
				lengthArg: newIntValue(0),
			},
			err: ErrInvalidRandLength,
			out: "",
		},
		"length arg negative": {
			cmd: GenerateSecretCommand{
				firstArg:  "rand",
				secondArg: "namespace/repo/secret",
				lengthArg: newIntValue(-1),
			},
			err: ErrInvalidRandLength,
			out: "",
		},
		// The length arg is only for backwards compatibility of the `generate rand` command.
		"length arg without rand": {
			cmd: GenerateSecretCommand{
				firstArg: "namespace/repo/secret",
				lengthArg:newIntValue(24),
			},
			err: errors.New("unexpected 24"),
		},
		// The second arg should only be used to supply the path when the first arg is `rand` (backwards compatibility).
		"second arg without rand": {
			cmd: GenerateSecretCommand{
				firstArg: "namespace/repo/secret",
				secondArg: "namespace/repo/secret2",
			},
			err: errors.New("unexpected namespace/repo/secret2"),
		},
		"generate error": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: nil,
					Err: testErr,
				},
				firstArg: "namespace/repo/secret",
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
				firstArg: "namespace/repo/secret",
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
				firstArg: "namespace/repo/secret",
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
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
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
