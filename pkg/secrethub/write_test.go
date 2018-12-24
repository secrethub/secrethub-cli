package secrethub

import (
	"testing"

	"bytes"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/keylockerbv/secrethub-cli/pkg/clip"
	"github.com/keylockerbv/secrethub/testutil"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestWriteCommand_Run(t *testing.T) {
	testutil.Unit(t)

	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd       WriteCommand
		service   fakeclient.SecretService
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
			service: fakeclient.SecretService{
				Writer: fakeclient.Writer{
					ReturnsVersion: &api.SecretVersion{
						Version: 1,
					},
					Err: nil,
				},
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
			service: fakeclient.SecretService{
				Writer: fakeclient.Writer{
					Err: secrethub.ErrEmptySecret,
				},
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
			service: fakeclient.SecretService{
				Writer: fakeclient.Writer{
					ReturnsVersion: &api.SecretVersion{
						Version: 1,
					},
					Err: nil,
				},
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
			service: fakeclient.SecretService{
				Writer: fakeclient.Writer{
					ReturnsVersion: &api.SecretVersion{
						Version: 1,
					},
					Err: nil,
				},
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
			promptIn:  "asked secret value",
			promptOut: "Please type in the value of the secret, followed by an [ENTER]:\n",
			service: fakeclient.SecretService{
				Writer: fakeclient.Writer{
					ReturnsVersion: &api.SecretVersion{
						Version: 1,
					},
					Err: nil,
				},
			},
			err:  nil,
			path: "namespace/repo/secret",
			data: []byte("asked secret value"),
			out:  "Writing secret value...\nWrite complete! The given value has been written to namespace/repo/secret:1\n",
		},
		"ask secret error": {
			cmd: WriteCommand{
				path: "namespace/repo/secret",
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
				clipper:      testutil.NewTestClipboardWithValue([]byte("clipped secret value")),
			},
			service: fakeclient.SecretService{
				Writer: fakeclient.Writer{
					ReturnsVersion: &api.SecretVersion{
						Version: 1,
					},
					Err: nil,
				},
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
				clipper:      testutil.NewErrClipboard(clip.ErrCannotRead("read error"), nil),
			},
			err: clip.ErrCannotRead("read error"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					SecretService: &tc.service,
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
			testutil.Compare(t, err, tc.err)
			testutil.Compare(t, tc.service.Writer.ArgPath, tc.path)
			testutil.Compare(t, tc.service.Writer.ArgData, tc.data)
			testutil.Compare(t, io.PromptOut.String(), tc.promptOut)
			testutil.Compare(t, io.StdOut.String(), tc.out)
		})
	}
}
