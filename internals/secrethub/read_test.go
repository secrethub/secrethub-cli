package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
	"io/ioutil"
	"os"
	"testing"
)

func TestReadCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd            ReadCommand
		newClientErr   error
		versionService fakeclient.SecretVersionService
		out            string
		err            error
	}{
		"success read": {
			cmd: ReadCommand{
				path: "test/repo/secret",
			},
			versionService: fakeclient.SecretVersionService{
				GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
					return &api.SecretVersion{
						Data: []byte("testSecret"),
					}, nil
				},
			},
			out: "testSecret\n",
		},
		//"success clipboard": {
		//	cmd: ReadCommand{
		//		path:                "test/repo/secret",
		//		clipper:             clip.NewClipboard(),
		//		useClipboard:        true,
		//		clearClipboardAfter: 5 * time.Minute,
		//	},
		//	versionService: fakeclient.SecretVersionService{
		//		GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
		//			return &api.SecretVersion{
		//				Data: []byte("testSecretClipboard"),
		//			}, nil
		//		},
		//	},
		//	out: "Copied test/repo/secret to clipboard. It will be cleared after 5 minutes.\n",
		//},
		"success file": {
			cmd: ReadCommand{
				path:     "test/repo/secret",
				outFile:  "secret.txt",
				fileMode: filemode.New(os.ModePerm),
			},
			versionService: fakeclient.SecretVersionService{
				GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
					return &api.SecretVersion{
						Data: []byte("testSecretFile"),
					}, nil
				},
			},
			out: "testSecretFile\n",
		},
		"fail file": {
			cmd: ReadCommand{
				path:     "test/repo/secret",
				outFile:  "/fail/read.txt",
				fileMode: filemode.New(os.ModeAppend),
			},
			versionService: fakeclient.SecretVersionService{
				GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
					return &api.SecretVersion{
						Data: []byte("testSecretFile"),
					}, nil
				},
			},
			out: "",
			err: ErrCannotWrite("/fail/read.txt", "open /fail/read.txt: no such file or directory"),
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"read error": {
			versionService: fakeclient.SecretVersionService{
				GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
					return nil, testErr
				},
			},
			err: testErr,
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
							VersionService: &tc.versionService,
						},
					}, nil
				}
			}

			// Run
			err := tc.cmd.Run()
			if name == "success file" {
				res, _ := ioutil.ReadFile(tc.cmd.outFile)
				io.Out.WriteString(string(res))
				os.Remove(tc.cmd.outFile)
			}

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}
}
