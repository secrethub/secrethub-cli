package secrethub

import (
	"errors"
	"github.com/secrethub/secrethub-cli/internals/cli"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

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
		writeFunc         func(path string, data []byte) (*api.SecretVersion, error)
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
				firstArg: cli.StringArgValue{Param: "namespace/repo/secret"},
			},
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{Version: 1}, nil
			},
			path: "namespace/repo/secret",
			data: []byte("random generated secret"),
			err:  nil,
			out:  "A randomly generated secret has been written to namespace/repo/secret:1.\n",
		},
		"length flag": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: []byte("random generated secret"),
					Err: nil,
				},
				firstArg:   cli.StringArgValue{Param: "namespace/repo/secret"},
				lengthFlag: newIntValue(24),
			},
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{Version: 1}, nil
			},
			path: "namespace/repo/secret",
			data: []byte("random generated secret"),
			err:  nil,
			out:  "A randomly generated secret has been written to namespace/repo/secret:1.\n",
		},
		"length flag and arg": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: []byte("random generated secret"),
					Err: nil,
				},
				firstArg:   cli.StringArgValue{Param: "rand"},
				secondArg:  cli.StringArgValue{Param: "namespace/repo/secret"},
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
				firstArg:  cli.StringArgValue{Param: "rand"},
				secondArg: cli.StringArgValue{Param: "namespace/repo/secret"},
				lengthArg: newIntValue(23),
			},
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{Version: 1}, nil
			},
			path: "namespace/repo/secret",
			data: []byte("random generated secret"),
			err:  nil,
			out:  "A randomly generated secret has been written to namespace/repo/secret:1.\n",
		},
		"length arg 0": {
			cmd: GenerateSecretCommand{
				firstArg:  cli.StringArgValue{Param: "rand"},
				secondArg: cli.StringArgValue{Param: "namespace/repo/secret"},
				lengthArg: newIntValue(0),
			},
			err: ErrInvalidRandLength,
			out: "",
		},
		"length arg negative": {
			cmd: GenerateSecretCommand{
				firstArg:  cli.StringArgValue{Param: "rand"},
				secondArg: cli.StringArgValue{Param: "namespace/repo/secret"},
				lengthArg: newIntValue(-1),
			},
			err: ErrInvalidRandLength,
			out: "",
		},
		// The length arg is only for backwards compatibility of the `generate rand` command.
		"length arg without rand": {
			cmd: GenerateSecretCommand{
				firstArg:  cli.StringArgValue{Param: "namespace/repo/secret"},
				lengthArg: newIntValue(24),
			},
			err: errors.New("unexpected 24"),
		},
		// The second arg should only be used to supply the path when the first arg is `rand` (backwards compatibility).
		"second arg without rand": {
			cmd: GenerateSecretCommand{
				firstArg:  cli.StringArgValue{Param: "namespace/repo/secret"},
				secondArg: cli.StringArgValue{Param: "namespace/repo/secret2"},
			},
			err: errors.New("unexpected namespace/repo/secret2"),
		},
		"generate error": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: nil,
					Err: testErr,
				},
				firstArg: cli.StringArgValue{Param: "namespace/repo/secret"},
			},
			err: testErr,
		},
		"client creation error": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: []byte("random generated secret"),
					Err: nil,
				},
				firstArg: cli.StringArgValue{Param: "namespace/repo/secret"},
			},
			clientCreationErr: testErr,
			err:               testErr,
		},
		"client error": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: []byte("random generated secret"),
					Err: nil,
				},
				firstArg: cli.StringArgValue{Param: "namespace/repo/secret"},
			},
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return nil, testErr
			},
			path: "namespace/repo/secret",
			data: []byte("random generated secret"),
			err:  testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var argPath string
			var argData []byte

			// Setup
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					SecretService: &fakeclient.SecretService{
						WriteFunc: func(path string, data []byte) (*api.SecretVersion, error) {
							argPath = path
							argData = data
							return tc.writeFunc(path, data)
						},
					},
				}, tc.clientCreationErr
			}

			io := fakeui.NewIO(t)
			tc.cmd.io = io

			// Act
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, argPath, tc.path)
			assert.Equal(t, argData, tc.data)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}
}
