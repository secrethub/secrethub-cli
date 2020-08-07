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
		cmd            WriteCommand
		writeFunc      func(path string, data []byte) (*api.SecretVersion, error)
		in             string
		piped          bool
		promptIn       string
		promptOut      string
		promptErr      error
		passwordIn     string
		newClientError error
		passwordErr    error
		readErr        error
		err            error
		path           api.SecretPath
		data           []byte
		out            string
	}{
		"path with version": {
			cmd: WriteCommand{
				path: "namespace/repo/secret:1",
			},
			err: errCannotWriteToVersion,
		},
		"empty secret piped": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			in:    "",
			piped: true,
			err:   errEmptySecret,
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
			err:  nil,
			path: "namespace/repo/secret",
			data: []byte("secret value"),
			out:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
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
			err:  secrethub.ErrEmptySecret,
			path: "namespace/repo/secret",
			data: []byte("secret value"),
			out:  "Writing secret value...\n",
		},
		"write space no-trim": {
			cmd: WriteCommand{
				path:   "namespace/repo/secret",
				noTrim: true,
			},
			in:    " ",
			piped: true,
			err:   errEmptySecret,
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
			err:  nil,
			path: "namespace/repo/secret",
			data: []byte("secret value"),
			out:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
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
			err:  nil,
			path: "namespace/repo/secret",
			data: []byte(" secret value"),
			out:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
		},
		"ask secret success": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			passwordIn: "asked secret value",
			promptOut:  "Please type in the value of the secret, followed by an [ENTER]:\n",
			writeFunc: func(path string, data []byte) (*api.SecretVersion, error) {
				return &api.SecretVersion{
					Version: 1,
				}, nil
			},
			err:  nil,
			path: "namespace/repo/secret",
			data: []byte("asked secret value"),
			out:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
		},
		"ask secret prompt error": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			promptErr: testErr,
			err:       testErr,
		},
		"ask secret read password error": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			promptOut:   "Please type in the value of the secret, followed by an [ENTER]:",
			passwordErr: testErr,
			err:         ui.ErrReadInput(testErr),
		},
		"piped read error": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			readErr: testErr,
			piped:   true,
			err:     ui.ErrReadInput(testErr),
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
			err:  nil,
			path: "namespace/repo/secret",
			data: []byte("clipped secret value"),
			out:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
		},
		"from clipboard error": {
			cmd: WriteCommand{
				path:         "namespace/repo/secret",
				useClipboard: true,
				clipper:      fakeclip.NewWithErr(clip.ErrCannotRead("read error"), nil),
			},
			err: clip.ErrCannotRead("read error"),
		},
		"clip and in-file": {
			cmd: WriteCommand{
				inFile:       "file",
				useClipboard: true,
			},
			err: errClipAndInFile,
		},
		"multiline and clip": {
			cmd: WriteCommand{
				multiline:    true,
				useClipboard: true,
			},
			err: errMultilineWithNonInteractiveFlag,
		},
		"cannot open file": {
			cmd: WriteCommand{
				inFile: "filename",
			},
			err: ErrReadFile("filename", errors.New("open filename: no such file or directory")),
		},
		"client creation error": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
			},
			in:             "secret value",
			piped:          true,
			err:            testErr,
			newClientError: testErr,
			out:            "Writing secret value...\n",
		},
		"empty multiline": {
			cmd: WriteCommand{
				path:      "namespace/repo/secret",
				multiline: true,
			},
			promptOut: "Please type in the value of the secret, followed by [CTRL-D]:\n\n",
			err:       errEmptySecret,
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
			assert.Equal(t, err, tc.err)
			assert.Equal(t, argPath, tc.path)
			assert.Equal(t, argData, tc.data)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}
}
