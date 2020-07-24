package secrethub

import (
	"bytes"
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
	"gotest.tools/assert"
)

func TestRmCommand_Run(t *testing.T) {

	var (
		testErr                = errors.New("test")
		ErrCannotRemoveDir     = errMain.Code("cannot_remove_dir").Error("cannot remove directory. Use the -r flag to remove directories.")
		ErrCannotRemoveRootDir = errMain.Code("cannot_remove_root_dir").Errorf(
			"cannot remove root directory. Use the repo rm command to remove a repository",
		)
	)

	cases := map[string]struct {
		cmd          RmCommand
		in           string
		promptErr    error
		argPath      api.Path
		promptOut    string
		out          string
		err          error
		deleteErr    error
		newClientErr error
		getTreeErr   error
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
		"fail-path-does-not-exist-dir": {
			cmd: RmCommand{
				force:     true,
				recursive: true,
				path:      "namespace/repo/di",
			},
			argPath:    "namespace/repo/di",
			getTreeErr: testErr,
			err:        testErr,
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
			err:          testErr,
			newClientErr: testErr,
		},
		"fail-deletion-error-dir": {
			cmd: RmCommand{
				path: "namespace/repo/dir",
			},
			err:       testErr,
			deleteErr: testErr,
		},
		"prompt-error-dir": {
			cmd: RmCommand{
				path: "namespace/repo/dir",
			},
			promptErr: testErr,
			err:       testErr,
		},

		"success-force-secret-version": {
			cmd: RmCommand{
				path:  "text/text/text:latest",
				force: true,
			},
			argPath: "namespace/repo/dir",
			out:     "Removal complete! The secret version text/text/text:latest has been permanently removed.\n",
		},

		"success-non-force-secret-version": {
			cmd: RmCommand{
				force: false,
				path:  "text/text/text:latest",
			},
			argPath: "text/text/text:latest",
			promptOut: "[WARNING] This action cannot be undone. This will permanently remove the text/text/text:latest secret version. " +
				"Please type in the name of the secret and the version (<name>:<version>) to confirm: ",
			in:  "text/text/text:latest",
			out: "Removal complete! The secret version text/text/text:latest has been permanently removed.\n",
		},

		"fail-abort-secret-version": {
			cmd: RmCommand{
				path:  "text/text/text:latest",
				force: false,
			},
			argPath: "namespace/repo/dir",
			promptOut: "[WARNING] This action cannot be undone. This will permanently remove the text/text/text:latest secret version. " +
				"Please type in the name of the secret and the version (<name>:<version>) to confirm: ",
			in:  "text/text/text:oldversion",
			out: "Name does not match. Aborting.\n",
		},
		"fail-path-does-not-exist-secret-version": {
			cmd: RmCommand{
				force: true,
				path:  "text/text/text:latest",
			},
			argPath:   "text/text/text:latest",
			deleteErr: testErr,
			err:       testErr,
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
								return tc.deleteErr
							},
						},
					},
					DirService: &fakeclient.DirService{
						DeleteFunc: func(path string) error {
							argPath = path
							return tc.deleteErr
						},
						GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
							if path == "namespace/repo/dir" {
								return &api.Tree{
									ParentPath: "namespace/repo",
									RootDir: &api.Dir{
										Name:    "dir",
										SubDirs: []*api.Dir{{Name: "subdir"}},
										Secrets: []*api.Secret{{Name: "secret"}},
									},
								}, tc.err
							}
							return nil, tc.getTreeErr
						},
					},
				}, tc.newClientErr
			}

			err := tc.cmd.Run()

			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
			if tc.err != nil && len(argPath) > 0 {
				assert.Equal(t, argPath, tc.argPath.String())
			}
		})
	}
}
