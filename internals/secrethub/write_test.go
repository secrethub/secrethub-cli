package secrethub

import (
	"bytes"
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/clip/fakeclip"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestWriteCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd               WriteCommand
		writeFunc         func(path string, data []byte) (*api.SecretVersion, error)
		in                string
		piped             bool
		promptIn          string
		promptErr         error
		passwordIn        string
		newClientError    error
		passwordErr       error
		readErr           error
		expectedPath      api.SecretPath
		expectedData      []byte
		expectedPromptOut string
		expectedOut       string
		expectedErr       error
	}{
		"path with version": {
			cmd: WriteCommand{
				path: "namespace/repo/secret:1",
			},
			expectedErr: errCannotWriteToVersion,
		},
		"empty secret piped": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			in:          "",
			piped:       true,
			expectedErr: errEmptySecret,
		},
		"write success piped": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			in:    "secret value",
			piped: true,
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{
					Version: 1,
				}, nil
			},
			expectedErr:  nil,
			expectedPath: "namespace/repo/secret",
			expectedData: []byte("secret value"),
			expectedOut:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
		},
		"client error": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			in:    "secret value",
			piped: true,
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return nil, secrethub.ErrEmptySecret
			},
			expectedErr:  secrethub.ErrEmptySecret,
			expectedPath: "namespace/repo/secret",
			expectedData: []byte("secret value"),
			expectedOut:  "Writing secret value...\n",
		},
		"write space no-trim": {
			cmd: WriteCommand{
				path:   "namespace/repo/secret",
				noTrim: true,
			},
			in:          " ",
			piped:       true,
			expectedErr: errEmptySecret,
		},
		"write secret prefixed with a space, trim": {
			cmd: WriteCommand{
				path:   "namespace/repo/secret",
				noTrim: false,
			},
			in:    " secret value",
			piped: true,
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{
					Version: 1,
				}, nil
			},
			expectedErr:  nil,
			expectedPath: "namespace/repo/secret",
			expectedData: []byte("secret value"),
			expectedOut:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
		},
		"write secret prefixed with a space, no-trim": {
			cmd: WriteCommand{
				path:   "namespace/repo/secret",
				noTrim: true,
			},
			in:    " secret value",
			piped: true,
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{
					Version: 1,
				}, nil
			},
			expectedErr:  nil,
			expectedPath: "namespace/repo/secret",
			expectedData: []byte(" secret value"),
			expectedOut:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
		},
		"ask secret success": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			passwordIn:        "asked secret value",
			expectedPromptOut: "Please type in the value of the secret, followed by an [ENTER]:\n",
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{
					Version: 1,
				}, nil
			},
			expectedErr:  nil,
			expectedPath: "namespace/repo/secret",
			expectedData: []byte("asked secret value"),
			expectedOut:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
		},
		"ask secret prompt error": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			promptErr:   testErr,
			expectedErr: testErr,
		},
		"ask secret read password error": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			expectedPromptOut: "Please type in the value of the secret, followed by an [ENTER]:",
			passwordErr:       testErr,
			expectedErr:       ui.ErrReadInput(testErr),
		},
		"piped read error": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			readErr:     testErr,
			piped:       true,
			expectedErr: ui.ErrReadInput(testErr),
		},
		"from clipboard": {
			cmd: WriteCommand{
				path:         "namespace/repo/secret",
				useClipboard: true,
				clipper:      fakeclip.NewWithValue([]byte("clipped secret value")),
			},
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{
					Version: 1,
				}, nil
			},
			expectedErr:  nil,
			expectedPath: "namespace/repo/secret",
			expectedData: []byte("clipped secret value"),
			expectedOut:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
		},
		"from clipboard error": {
			cmd: WriteCommand{
				path:         "namespace/repo/secret",
				useClipboard: true,
				clipper:      fakeclip.NewWithErr(clip.ErrCannotRead("read error"), nil),
			},
			expectedErr: clip.ErrCannotRead("read error"),
		},
		"clip and in-file": {
			cmd: WriteCommand{
				inFile:       "file",
				useClipboard: true,
			},
			expectedErr: errClipAndInFile,
		},
		"multiline and clip": {
			cmd: WriteCommand{
				multiline:    true,
				useClipboard: true,
			},
			expectedErr: errMultilineWithNonInteractiveFlag,
		},
		"cannot open file": {
			cmd: WriteCommand{
				inFile: "filename",
			},
			expectedErr: ErrReadFile("filename", errors.New("open filename: no such file or directory")),
		},
		"client creation error": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			in:             "secret value",
			piped:          true,
			expectedErr:    testErr,
			newClientError: testErr,
			expectedOut:    "Writing secret value...\n",
		},
		"empty multiline": {
			cmd: WriteCommand{
				path:      "namespace/repo/secret",
				multiline: true,
			},
			expectedPromptOut: "Please type in the value of the secret, followed by [CTRL-D]:\n\n",
			expectedErr:       errEmptySecret,
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
				}, tc.newClientError
			}

			io := fakeui.NewIO(t)
			io.In.ReadErr = tc.readErr
			io.PromptIn.Buffer = bytes.NewBufferString(tc.promptIn)
			io.PromptErr = tc.promptErr
			io.PasswordReader.Buffer = bytes.NewBufferString(tc.passwordIn)
			io.PasswordReader.ReadErr = tc.passwordErr
			io.In.Piped = tc.piped
			io.In.Buffer = bytes.NewBufferString(tc.in)

			tc.cmd.io = io

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.expectedErr)
			assert.Equal(t, argPath, tc.expectedPath)
			assert.Equal(t, argData, tc.expectedData)
			assert.Equal(t, io.PromptOut.String(), tc.expectedPromptOut)
			assert.Equal(t, io.Out.String(), tc.expectedOut)
		})
	}
}
