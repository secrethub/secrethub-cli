package secrethub

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/clip/fakeclip"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestWriteCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	fakeAskSecretFunc := func(io ui.IO, question string) (s string, err error) {
		reader, writer, err := io.Prompts()
		if err != nil {
			return "", err
		}
		_, err = writer.Write([]byte(question + "\n"))
		if err != nil {
			return "", err
		}
		line, _, err := bufio.NewReader(reader).ReadLine()
		return string(line), err
	}

	cases := map[string]struct {
		cmd       WriteCommand
		writeFunc func(path string, data []byte) (*api.SecretVersion, error)
		in        string
		piped     bool
		promptIn  string
		promptOut string
		promptErr error
		readErr   error
		err       error
		path      api.SecretPath
		data      []byte
		out       string
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
				path:      "namespace/repo/secret",
				askSecret: fakeAskSecretFunc,
			},
			promptIn:  "asked secret value",
			promptOut: "Please type in the value of the secret, followed by an [ENTER]:\n",
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
		"ask secret error": {
			cmd: WriteCommand{
				path:      "namespace/repo/secret",
				askSecret: fakeAskSecretFunc,
			},
			promptErr: testErr,
			err:       testErr,
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
				}, nil
			}

			io := ui.NewFakeIO()
			io.StdIn.ReadErr = tc.readErr
			io.PromptIn.Buffer = bytes.NewBufferString(tc.promptIn)
			io.PromptErr = tc.promptErr
			io.StdIn.Piped = tc.piped
			io.StdIn.Buffer = bytes.NewBufferString(tc.in)

			tc.cmd.io = io

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, argPath, tc.path)
			assert.Equal(t, argData, tc.data)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
