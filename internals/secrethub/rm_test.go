package secrethub

import (
	"bytes"
	"errors"
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestRmCommand_Run(t *testing.T) {

	var testErr = errors.New("test")

	cases := map[string]struct {
		cmd                RmCommand
		in                 string
		argPath            api.Path
		promptOut          string
		out                string
		err                error
		promptError        error
		deleteSecretError  error
		deleteVersionError error
		deleteDirError     error
		newClientError     error
		getTreeError       error
		getSecretError     error
	}{
		"success-force-dir": {
			cmd: RmCommand{
				force:     true,
				recursive: true,
				path:      "namespace/repo/dir",
			},
			argPath: "namespace/repo/dir",
			out:     "Removal complete! The directory namespace/repo/dir has been permanently removed.\n",
		},
		"success-non-force-dir": {
			cmd: RmCommand{
				force:     false,
				recursive: true,
				path:      "namespace/repo/dir",
			},
			argPath: "namespace/repo/dir",
			promptOut: "[WARNING] This action cannot be undone. This will permanently remove the namespace/repo/dir directory and all the directories and secrets it contains. " +
				"Please type in the name of the directory to confirm: ",
			in:  "namespace/repo/dir",
			out: "Removal complete! The directory namespace/repo/dir has been permanently removed.\n",
		},

		"fail-non-recursive-dir": {
			cmd: RmCommand{
				force:     true,
				recursive: false,
				path:      "namespace/repo/dir",
			},
			argPath: "namespace/repo/dir",
			err:     ErrCannotRemoveDir,
		},
		"fail-remove-root-dir": {
			cmd: RmCommand{
				force:     true,
				recursive: false,
				path:      "namespace/repo",
			},
			argPath: "namespace/repo",
			err:     ErrCannotRemoveRootDir,
		},
		"fail-get-tree-error-dir": {
			cmd: RmCommand{
				force:     true,
				recursive: true,
				path:      "namespace/repo/dir",
			},
			argPath:      "namespace/repo/dir",
			getTreeError: testErr,
			err:          testErr,
		},
		"fail-abort-dir": {
			cmd: RmCommand{
				path:      "namespace/repo/dir",
				recursive: true,
				force:     false,
			},
			argPath: "namespace/repo/dir",
			promptOut: "[WARNING] This action cannot be undone. This will permanently remove the namespace/repo/dir directory and all the directories and secrets it contains. " +
				"Please type in the name of the directory to confirm: ",
			in:  "namespace/repo/directory",
			out: "Name does not match. Aborting.\n",
		},
		"fail-client-error-dir": {
			cmd: RmCommand{
				path: "namespace/repo/dir",
			},
			err:            testErr,
			newClientError: testErr,
		},
		"fail-deletion-error-dir": {
			cmd: RmCommand{
				path:      "namespace/repo/dir",
				force:     true,
				recursive: true,
			},
			argPath:        "namespace/repo/dir",
			err:            testErr,
			deleteDirError: testErr,
		},
		"fail-prompt-error-dir": {
			cmd: RmCommand{
				path:      "namespace/repo/dir",
				recursive: true,
			},
			promptError: testErr,
			err:         testErr,
		},

		"success-force-secret-version": {
			cmd: RmCommand{
				path:  "namespace/repo/secret:latest",
				force: true,
			},
			argPath: "namespace/repo/secret:latest",
			out:     "Removal complete! The secret version namespace/repo/secret:latest has been permanently removed.\n",
		},

		"success-non-force-secret-version": {
			cmd: RmCommand{
				force: false,
				path:  "namespace/repo/secret:latest",
			},
			argPath: "namespace/repo/secret:latest",
			promptOut: "[WARNING] This action cannot be undone. This will permanently remove the namespace/repo/secret:latest secret version. " +
				"Please type in the name of the secret and the version (<name>:<version>) to confirm: ",
			in:  "namespace/repo/secret:latest",
			out: "Removal complete! The secret version namespace/repo/secret:latest has been permanently removed.\n",
		},

		"fail-abort-secret-version": {
			cmd: RmCommand{
				path:  "namespace/repo/secret:latest",
				force: false,
			},
			argPath: "namespace/repo/secret:latest",
			promptOut: "[WARNING] This action cannot be undone. This will permanently remove the namespace/repo/secret:latest secret version. " +
				"Please type in the name of the secret and the version (<name>:<version>) to confirm: ",
			in:  "namespace/repo/secret:oldversion",
			out: "Name does not match. Aborting.\n",
		},
		"fail-deletion-error-secret-version": {
			cmd: RmCommand{
				force: true,
				path:  "namespace/repo/secret:latest",
			},
			argPath:            "namespace/repo/secret:latest",
			deleteVersionError: testErr,
			err:                testErr,
		},

		"success-force-secret": {
			cmd: RmCommand{
				force: true,
				path:  "namespace/repo/dir/secret",
			},
			argPath:      "namespace/repo/dir/secret",
			out:          "Removal complete! The secret namespace/repo/dir/secret has been permanently removed.\n",
			getTreeError: api.ErrNotFound,
		},
		"success-non-force-secret": {
			cmd: RmCommand{
				force: false,
				path:  "namespace/repo/dir/secret",
			},
			argPath:      "namespace/repo/dir/secret",
			promptOut:    "[WARNING] This action cannot be undone. This will permanently remove the namespace/repo/dir/secret secret and all its versions. Please type in the name of the secret to confirm: ",
			in:           "namespace/repo/dir/secret",
			out:          "Removal complete! The secret namespace/repo/dir/secret has been permanently removed.\n",
			getTreeError: api.ErrNotFound,
		},
		"fail-abort-secret": {
			cmd: RmCommand{
				path:  "namespace/repo/dir/secret",
				force: false,
			},
			argPath:      "namespace/repo/dir/secret",
			promptOut:    "[WARNING] This action cannot be undone. This will permanently remove the namespace/repo/dir/secret secret and all its versions. Please type in the name of the secret to confirm: ",
			in:           "namespace/repo/dir/secret2",
			out:          "Name does not match. Aborting.\n",
			getTreeError: api.ErrNotFound,
		},
		"fail-get-error-secret": {
			cmd: RmCommand{
				force: true,
				path:  "namespace/repo/dir/secret",
			},
			argPath:        "namespace/repo/dir/secret",
			getSecretError: api.ErrNotFound,
			getTreeError:   api.ErrNotFound,
			err:            ErrResourceNotFound("namespace/repo/dir/secret"),
		},
		"fail-deletion-error-secret": {
			cmd: RmCommand{
				path:  "namespace/repo/dir/secret",
				force: true,
			},
			argPath:           "namespace/repo/dir/secret",
			err:               testErr,
			deleteSecretError: testErr,
			getTreeError:      api.ErrNotFound,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			io := fakeui.NewIO(t)
			io.PromptIn.Buffer = bytes.NewBufferString(tc.in)
			io.PromptErr = tc.promptError
			tc.cmd.io = io

			var argPath string
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					SecretService: &fakeclient.SecretService{
						VersionService: &fakeclient.SecretVersionService{
							DeleteFunc: func(path string) error {
								argPath = path
								return tc.deleteVersionError
							},
						},
						GetFunc: func(path string) (*api.Secret, error) {
							argPath = path
							return nil, tc.getSecretError
						},
						DeleteFunc: func(path string) error {
							argPath = path
							return tc.deleteSecretError
						},
					},
					DirService: &fakeclient.DirService{
						DeleteFunc: func(path string) error {
							argPath = path
							return tc.deleteDirError
						},
						GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
							if path == "namespace/repo/dir" || path == "namespace/repo/dir/secret" {
								return &api.Tree{}, tc.getTreeError
							}

							return nil, tc.getTreeError
						},
					},
				}, tc.newClientError
			}

			err := tc.cmd.Run()

			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
			if len(argPath) > 0 {
				assert.Equal(t, argPath, tc.argPath.String())
			}
		})
	}
}
