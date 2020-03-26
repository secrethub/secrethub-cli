package secrethub

import (
	"bytes"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestRepoRmCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd           RepoRmCommand
		promptIn      string
		promptReadErr error
		newClientErr  error
		promptErr     error
		repoService   fakeclient.RepoService
		promptOut     string
		out           string
		err           error
	}{
		"success": {
			cmd: RepoRmCommand{
				path: "namespace/repo",
			},
			promptIn: "namespace/repo",
			repoService: fakeclient.RepoService{
				GetFunc: func(path string) (*api.Repo, error) {
					return &api.Repo{}, nil
				},
				DeleteFunc: func(path string) error {
					return nil
				},
			},
			promptOut: "[DANGER ZONE] This action cannot be undone. " +
				"This will permanently remove the namespace/repo repository, all its secrets and all associated service accounts. " +
				"Please type in the full path of the repository to confirm: ",
			out: "Removing repository...\n" +
				"Removal complete! The repository namespace/repo has been permanently removed.\n",
		},
		"abort": {
			cmd: RepoRmCommand{
				path: "namespace/repo",
			},
			promptIn: "namespace/typo",
			repoService: fakeclient.RepoService{
				GetFunc: func(path string) (*api.Repo, error) {
					return &api.Repo{}, nil
				},
			},
			promptOut: "[DANGER ZONE] This action cannot be undone. " +
				"This will permanently remove the namespace/repo repository, all its secrets and all associated service accounts. " +
				"Please type in the full path of the repository to confirm: ",
			out: "Name does not match. Aborting.\n",
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"get repo error": {
			repoService: fakeclient.RepoService{
				GetFunc: func(path string) (*api.Repo, error) {
					return nil, testErr
				},
			},
			err: testErr,
		},
		"delete error": {
			cmd: RepoRmCommand{
				path: "namespace/repo",
			},
			promptIn: "namespace/repo",
			repoService: fakeclient.RepoService{
				GetFunc: func(path string) (*api.Repo, error) {
					return &api.Repo{}, nil
				},
				DeleteFunc: func(path string) error {
					return testErr
				},
			},
			promptOut: "[DANGER ZONE] This action cannot be undone. " +
				"This will permanently remove the namespace/repo repository, all its secrets and all associated service accounts. " +
				"Please type in the full path of the repository to confirm: ",
			out: "Removing repository...\n",
			err: testErr,
		},
		"prompt error": {
			repoService: fakeclient.RepoService{
				GetFunc: func(path string) (*api.Repo, error) {
					return &api.Repo{}, nil
				},
			},
			promptErr: ui.ErrCannotAsk,
			err:       ui.ErrCannotAsk,
		},
		"prompt read error": {
			cmd: RepoRmCommand{
				path: "namespace/repo",
			},
			repoService: fakeclient.RepoService{
				GetFunc: func(path string) (*api.Repo, error) {
					return &api.Repo{}, nil
				},
			},
			promptReadErr: testErr,
			err:           ui.ErrReadInput(testErr),
			promptOut: "[DANGER ZONE] This action cannot be undone. " +
				"This will permanently remove the namespace/repo repository, all its secrets and all associated service accounts. " +
				"Please type in the full path of the repository to confirm: ",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			if tc.newClientErr != nil {
				tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
					return nil, tc.newClientErr
				}
			} else {
				tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						RepoService: &tc.repoService,
					}, nil
				}
			}

			io := ui.NewFakeIO()
			io.PromptIn.Buffer = bytes.NewBufferString(tc.promptIn)
			io.PromptIn.ReadErr = tc.promptReadErr
			io.PromptErr = tc.promptErr
			tc.cmd.io = io

			// Run
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
		})
	}
}
