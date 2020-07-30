package secrethub

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/clip/fakeclip"
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
		serviceErr      error
		expectedContent string
		out             string
		err             error
	}{
		"success read": {
			cmd: ReadCommand{
				path: "test/repo/secret",
			},
			secretVersion: api.SecretVersion{Data: testSecret},
			out:           string(testSecret) + "\n",
		},
		"success clipboard": {
			cmd: ReadCommand{
				path:                "test/repo/secret",
				clipper:             fakeclip.New(),
				useClipboard:        true,
				clearClipboardAfter: 5 * time.Minute,
			},
			secretVersion:   api.SecretVersion{Data: testSecret},
			expectedContent: string(testSecret),
			out:             "Copied test/repo/secret to clipboard. It will be cleared after 5 minutes.\n",
		},
		"success file": {
			cmd: ReadCommand{
				path:     "test/repo/secret",
				outFile:  "secret.txt",
				fileMode: filemode.New(os.ModePerm),
			},
			secretVersion:   api.SecretVersion{Data: testSecret},
			expectedContent: string(testSecret) + "\n",
			out:             "",
		},
		"fail file": {
			cmd: ReadCommand{
				path:     "test/repo/secret",
				outFile:  "/fail/read.txt",
				fileMode: filemode.New(os.ModeAppend),
			},
			secretVersion: api.SecretVersion{Data: testSecret},
			out:           "",
			err:           ErrCannotWrite("/fail/read.txt", "open /fail/read.txt: no such file or directory"),
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"read error": {
			serviceErr: testErr,
			err:        testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := fakeui.NewIO(t)
			tc.cmd.io = io

			if tc.newClientErr != nil {
				tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
					return nil, tc.newClientErr
				}
			} else {
				tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return &tc.secretVersion, tc.serviceErr
								},
							},
						},
					}, nil
				}
			}

			// Run
			err := tc.cmd.Run()
			content := ""
			if _, err := os.Stat(tc.cmd.outFile); err == nil {
				res, _ := ioutil.ReadFile(tc.cmd.outFile)
				content = string(res)
				os.Remove(tc.cmd.outFile)
			} else if tc.cmd.useClipboard {
				res, _ := tc.cmd.clipper.ReadAll()
				content = string(res)
			}

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
			assert.Equal(t, content, tc.expectedContent)
		})
	}
}
