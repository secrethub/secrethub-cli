package secrethub

import (
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestRepoInitCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		path         api.RepoPath
		newClientErr error
		createFunc   func(path string) (*api.Repo, error)
		argPath      api.RepoPath
		out          string
		err          error
	}{
		"success": {
			path: api.RepoPath("namespace/repo"),
			createFunc: func(path string) (*api.Repo, error) {
				return &api.Repo{}, nil
			},
			argPath: api.RepoPath("namespace/repo"),
			out: "Creating repository...\n" +
				"Create complete! The repository namespace/repo is now ready to use.\n",
			err: nil,
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"client error": {
			createFunc: func(path string) (*api.Repo, error) {
				return nil, testErr
			},
			out: "Creating repository...\n",
			err: testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var argPath string

			// Setup
			cmd := RepoInitCommand{
				path: tc.path,
			}

			if tc.newClientErr != nil {
				cmd.newClient = func() (secrethub.ClientInterface, error) {
					return nil, tc.newClientErr
				}
			} else {
				cmd.newClient = func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						RepoService: &fakeclient.RepoService{
							CreateFunc: func(path string) (*api.Repo, error) {
								argPath = path
								return tc.createFunc(path)
							},
						},
					}, nil
				}
			}

			io := fakeui.NewIO(t)
			cmd.io = io

			// Run
			err := cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
			assert.Equal(t, argPath, tc.argPath)
		})
	}
}
