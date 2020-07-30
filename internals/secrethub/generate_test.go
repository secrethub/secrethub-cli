package secrethub

import (
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/clip/fakeclip"
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
	testData := []byte("random generated secret")
	testPath := "namespace/repo/secret"

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
					Ret: testData,
					Err: nil,
				},
				firstArg: testPath,
			},
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{Version: 1}, nil
			},
			path: "namespace/repo/secret",
			data: testData,
			err:  nil,
			out:  "A randomly generated secret has been written to namespace/repo/secret:1.\n",
		},
		"copy to clipboard": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: testData,
					Err: nil,
				},
				firstArg:        testPath,
				copyToClipboard: true,
				clipper:         fakeclip.New(),
			},
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{Version: 1}, nil
			},
			path: "namespace/repo/secret",
			data: testData,
			out: "A randomly generated secret has been written to namespace/repo/secret:1.\n" +
				"The generated value has been copied to the clipboard. It will be cleared after Less than a second.\n",
		},
		"length flag": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: testData,
					Err: nil,
				},
				firstArg:   testPath,
				lengthFlag: newIntValue(24),
			},
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{Version: 1}, nil
			},
			path: "namespace/repo/secret",
			data: testData,
			err:  nil,
			out:  "A randomly generated secret has been written to namespace/repo/secret:1.\n",
		},
		"length flag and arg": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: testData,
					Err: nil,
				},
				firstArg:   "rand",
				secondArg:  testPath,
				lengthFlag: newIntValue(24),
				lengthArg:  newIntValue(24),
			},
			err: ErrCannotUseLengthArgAndFlag,
		},
		"backwards compatibility rand": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: testData,
					Err: nil,
				},
				firstArg:  "rand",
				secondArg: testPath,
				lengthArg: newIntValue(23),
			},
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{Version: 1}, nil
			},
			path: "namespace/repo/secret",
			data: testData,
			err:  nil,
			out:  "A randomly generated secret has been written to namespace/repo/secret:1.\n",
		},
		"length arg 0": {
			cmd: GenerateSecretCommand{
				firstArg:  "rand",
				secondArg: testPath,
				lengthArg: newIntValue(0),
			},
			err: ErrInvalidRandLength,
			out: "",
		},
		"length arg negative": {
			cmd: GenerateSecretCommand{
				firstArg:  "rand",
				secondArg: testPath,
				lengthArg: newIntValue(-1),
			},
			err: ErrInvalidRandLength,
			out: "",
		},
		// The length arg is only for backwards compatibility of the `generate rand` command.
		"length arg without rand": {
			cmd: GenerateSecretCommand{
				firstArg:  testPath,
				lengthArg: newIntValue(24),
			},
			err: errors.New("unexpected 24"),
		},
		// The second arg should only be used to supply the path when the first arg is `rand` (backwards compatibility).
		"second arg without rand": {
			cmd: GenerateSecretCommand{
				firstArg:  testPath,
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
				firstArg: testPath,
			},
			err: testErr,
		},
		"client creation error": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: testData,
					Err: nil,
				},
				firstArg: testPath,
			},
			clientCreationErr: testErr,
			err:               testErr,
		},
		"client error": {
			cmd: GenerateSecretCommand{
				generator: randchargeneratorfakes.FakeRandomGenerator{
					Ret: testData,
					Err: nil,
				},
				firstArg: testPath,
			},
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return nil, testErr
			},
			path: "namespace/repo/secret",
			data: testData,
			err:  testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var argPath string
			var argData []byte

			// Setup
			testIO := fakeui.NewIO(t)
			tc.cmd.io = testIO

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

			// Act
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, argPath, tc.path)
			assert.Equal(t, argData, tc.data)
			assert.Equal(t, testIO.Out.String(), tc.out)
		})
	}
}
