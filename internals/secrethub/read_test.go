package secrethub

import (
	"os"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestReadCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")
	testSecret := []byte("Remember! Reality's an illusion, the universe is a hologram, buy gold! Bye! - Bill Cipher")

	cases := map[string]struct {
		cmd             ReadCommand
		newClientErr    error
		secretVersion   api.SecretVersion
		fileErr         error
		serviceErr      error
		expectedClip    []byte
		expectedFileOut []byte
		expectedOut     string
		expectedErr     error
	}{
		"success read": {
			cmd: ReadCommand{
				path: "test/repo/secret",
			},
			secretVersion: api.SecretVersion{Data: testSecret},
			expectedOut:   string(testSecret) + "\n",
		},
		"success clipboard": {
			cmd: ReadCommand{
				path:         "test/repo/secret",
				useClipboard: true,
			},
			secretVersion: api.SecretVersion{Data: testSecret},
			expectedClip:  testSecret,
			expectedOut:   "Copied test/repo/secret to clipboard. It will be cleared after 45 seconds.\n",
		},
		"success file": {
			cmd: ReadCommand{
				path:     "test/repo/secret",
				outFile:  "secret.txt",
				fileMode: filemode.New(os.ModePerm),
			},
			secretVersion:   api.SecretVersion{Data: testSecret},
			expectedFileOut: []byte(string(testSecret) + "\n"),
			expectedOut:     "",
		},
		"fail file": {
			cmd: ReadCommand{
				path:     "test/repo/secret",
				outFile:  "/fail/read.txt",
				fileMode: filemode.New(os.ModeAppend),
			},
			fileErr:       testErr,
			secretVersion: api.SecretVersion{Data: testSecret},
			expectedOut:   "",
			expectedErr:   ErrCannotWrite("/fail/read.txt", testErr.Error()),
		},
		"new client error": {
			cmd:           ReadCommand{},
			secretVersion: api.SecretVersion{Data: testSecret},
			newClientErr:  testErr,
			expectedErr:   testErr,
		},
		"read error": {
			cmd:           ReadCommand{},
			secretVersion: api.SecretVersion{Data: testSecret},
			serviceErr:    testErr,
			expectedErr:   testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			var fileOut []byte
			testIO := fakeui.NewIO(t)
			tc.cmd.io = testIO

			clipWriter := &FakeClipboardWriter{}
			tc.cmd.clipWriter = clipWriter

			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					SecretService: &fakeclient.SecretService{
						VersionService: &fakeclient.SecretVersionService{
							GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
								return &tc.secretVersion, tc.serviceErr
							},
						},
					},
				}, tc.newClientErr
			}
			tc.cmd.writeFileFunc = func(filename string, data []byte, perm os.FileMode) error {
				if tc.fileErr == nil {
					fileOut = data
				}
				return tc.fileErr
			}

			// Run
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.expectedErr)
			assert.Equal(t, testIO.Out.String(), tc.expectedOut)
			assert.Equal(t, clipWriter.Buffer.Bytes(), tc.expectedClip)
			assert.Equal(t, fileOut, tc.expectedFileOut)
		})
	}
}
