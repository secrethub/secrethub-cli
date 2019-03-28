package secrethub

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestRepoInitCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		path         api.RepoPath
		newClientErr error
		service      fakeclient.RepoService
		argPath      api.RepoPath
		out          string
		err          error
	}{
		"success": {
			path: api.RepoPath("namespace/repo"),
			service: fakeclient.RepoService{
				Creater: fakeclient.RepoCreater{
					ReturnsRepo: &api.Repo{},
					Err:         nil,
				},
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
			service: fakeclient.RepoService{
				Creater: fakeclient.RepoCreater{
					Err: testErr,
				},
			},
			out: "Creating repository...\n",
			err: testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			cmd := RepoInitCommand{
				path: tc.path,
			}

			if tc.newClientErr != nil {
				cmd.newClient = func() (secrethub.Client, error) {
					return nil, tc.newClientErr
				}
			} else {
				cmd.newClient = func() (secrethub.Client, error) {
					return fakeclient.Client{
						RepoService: &tc.service,
					}, nil
				}
			}

			io := ui.NewFakeIO()
			cmd.io = io

			// Run
			err := cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
			assert.Equal(t, tc.service.Creater.Argpath, tc.argPath)
		})
	}
}
