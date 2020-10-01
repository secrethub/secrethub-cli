package secrethub

import (
	"bytes"
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestRmCommand_Run(t *testing.T) {
	const warningText = "[WARNING] This action cannot be undone. This will permanently remove the"

	testErr := errors.New("test")

	cases := map[string]struct {
		cmd               RmCommand
		in                string
		argPath           api.Path
		deleteSecretErr   error
		deleteVersionErr  error
		deleteDirErr      error
		newClientErr      error
		getTreeErr        error
		getSecretErr      error
		promptErr         error
		expectedPromptOut string
		expectedOut       string
		expectedErr       error
	}{
		"success force dir": {
			cmd: RmCommand{
				force:     true,
				recursive: true,
				path:      "namespace/repo/dir",
			},
			argPath:     "namespace/repo/dir",
			expectedOut: "Removal complete! The directory namespace/repo/dir has been permanently removed.\n",
		},
		"success non force dir": {
			cmd: RmCommand{
				force:     false,
				recursive: true,
				path:      "namespace/repo/dir",
			},
			argPath: "namespace/repo/dir",
			expectedPromptOut: warningText + " namespace/repo/dir directory and all the directories and secrets it contains. " +
				"Please type in the name of the directory to confirm: ",
			in:          "namespace/repo/dir",
			expectedOut: "Removal complete! The directory namespace/repo/dir has been permanently removed.\n",
		},
		"fail non recursive dir": {
			cmd: RmCommand{
				force:     true,
				recursive: false,
				path:      "namespace/repo/dir",
			},
			argPath:     "namespace/repo/dir",
			expectedErr: ErrCannotRemoveDir,
		},
		"fail remove root dir": {
			cmd: RmCommand{
				force:     true,
				recursive: false,
				path:      "namespace/repo",
			},
			argPath:     "namespace/repo",
			expectedErr: ErrCannotRemoveRootDir,
		},
		"fail get tree error dir": {
			cmd: RmCommand{
				force:     true,
				recursive: true,
				path:      "namespace/repo/dir",
			},
			argPath:     "namespace/repo/dir",
			getTreeErr:  testErr,
			expectedErr: testErr,
		},
		"fail abort dir": {
			cmd: RmCommand{
				path:      "namespace/repo/dir",
				recursive: true,
				force:     false,
			},
			argPath: "namespace/repo/dir",
			expectedPromptOut: warningText + " namespace/repo/dir directory and all the directories and secrets it contains. " +
				"Please type in the name of the directory to confirm: ",
			in:          "namespace/repo/directory",
			expectedOut: "Name does not match. Aborting.\n",
		},
		"fail client error dir": {
			cmd: RmCommand{
				path: "namespace/repo/dir",
			},
			expectedErr:  testErr,
			newClientErr: testErr,
		},
		"fail deletion error dir": {
			cmd: RmCommand{
				path:      "namespace/repo/dir",
				force:     true,
				recursive: true,
			},
			argPath:      "namespace/repo/dir",
			expectedErr:  testErr,
			deleteDirErr: testErr,
		},
		"fail prompt error dir": {
			cmd: RmCommand{
				path:      "namespace/repo/dir",
				recursive: true,
			},
			promptErr:   testErr,
			expectedErr: testErr,
		},
		"success force secret version": {
			cmd: RmCommand{
				path:  "namespace/repo/secret:latest",
				force: true,
			},
			argPath:     "namespace/repo/secret:latest",
			expectedOut: "Removal complete! The secret version namespace/repo/secret:latest has been permanently removed.\n",
		},
		"success non force secret version": {
			cmd: RmCommand{
				force: false,
				path:  "namespace/repo/secret:latest",
			},
			argPath: "namespace/repo/secret:latest",
			expectedPromptOut: warningText + " namespace/repo/secret:latest secret version. " +
				"Please type in the name of the secret and the version (<name>:<version>) to confirm: ",
			in:          "namespace/repo/secret:latest",
			expectedOut: "Removal complete! The secret version namespace/repo/secret:latest has been permanently removed.\n",
		},
		"fail abort secret version": {
			cmd: RmCommand{
				path:  "namespace/repo/secret:latest",
				force: false,
			},
			argPath: "namespace/repo/secret:latest",
			expectedPromptOut: warningText + " namespace/repo/secret:latest secret version. " +
				"Please type in the name of the secret and the version (<name>:<version>) to confirm: ",
			in:          "namespace/repo/secret:oldversion",
			expectedOut: "Name does not match. Aborting.\n",
		},
		"fail deletion error secret version": {
			cmd: RmCommand{
				force: true,
				path:  "namespace/repo/secret:latest",
			},
			argPath:          "namespace/repo/secret:latest",
			deleteVersionErr: testErr,
			expectedErr:      testErr,
		},
		"success force secret": {
			cmd: RmCommand{
				force: true,
				path:  "namespace/repo/dir/secret",
			},
			argPath:     "namespace/repo/dir/secret",
			expectedOut: "Removal complete! The secret namespace/repo/dir/secret has been permanently removed.\n",
			getTreeErr:  api.ErrNotFound,
		},
		"success non force secret": {
			cmd: RmCommand{
				force: false,
				path:  "namespace/repo/dir/secret",
			},
			argPath:           "namespace/repo/dir/secret",
			expectedPromptOut: warningText + " namespace/repo/dir/secret secret and all its versions. Please type in the name of the secret to confirm: ",
			in:                "namespace/repo/dir/secret",
			expectedOut:       "Removal complete! The secret namespace/repo/dir/secret has been permanently removed.\n",
			getTreeErr:        api.ErrNotFound,
		},
		"fail abort secret": {
			cmd: RmCommand{
				path:  "namespace/repo/dir/secret",
				force: false,
			},
			argPath:           "namespace/repo/dir/secret",
			expectedPromptOut: warningText + " namespace/repo/dir/secret secret and all its versions. Please type in the name of the secret to confirm: ",
			in:                "namespace/repo/dir/secret2",
			expectedOut:       "Name does not match. Aborting.\n",
			getTreeErr:        api.ErrNotFound,
		},
		"fail get error secret": {
			cmd: RmCommand{
				force: true,
				path:  "namespace/repo/dir/secret",
			},
			argPath:      "namespace/repo/dir/secret",
			getSecretErr: api.ErrNotFound,
			getTreeErr:   api.ErrNotFound,
			expectedErr:  ErrResourceNotFound("namespace/repo/dir/secret"),
		},
		"fail deletion error secret": {
			cmd: RmCommand{
				path:  "namespace/repo/dir/secret",
				force: true,
			},
			argPath:         "namespace/repo/dir/secret",
			expectedErr:     testErr,
			deleteSecretErr: testErr,
			getTreeErr:      api.ErrNotFound,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			io := fakeui.NewIO(t)
			io.PromptIn.Buffer = bytes.NewBufferString(tc.in)
			io.PromptErr = tc.promptErr
			tc.cmd.io = io

			var argPath string
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					SecretService: &fakeclient.SecretService{
						VersionService: &fakeclient.SecretVersionService{
							DeleteFunc: func(path string) error {
								argPath = path
								return tc.deleteVersionErr
							},
						},
						GetFunc: func(path string) (*api.Secret, error) {
							argPath = path
							return nil, tc.getSecretErr
						},
						DeleteFunc: func(path string) error {
							argPath = path
							return tc.deleteSecretErr
						},
					},
					DirService: &fakeclient.DirService{
						DeleteFunc: func(path string) error {
							argPath = path
							return tc.deleteDirErr
						},
						GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
							if path == "namespace/repo/dir" || path == "namespace/repo/dir/secret" {
								return &api.Tree{}, tc.getTreeErr
							}

							return nil, tc.getTreeErr
						},
					},
				}, tc.newClientErr
			}

			err := tc.cmd.Run()

			assert.Equal(t, err, tc.expectedErr)
			assert.Equal(t, io.Out.String(), tc.expectedOut)
			assert.Equal(t, io.PromptOut.String(), tc.expectedPromptOut)
			if len(argPath) > 0 {
				assert.Equal(t, argPath, tc.argPath.String())
			}
		})
	}
}
