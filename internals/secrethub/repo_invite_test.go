package secrethub

import (
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestRepoInviteCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd               RepoInviteCommand
		newClientErr      error
		userService       fakeclient.UserService
		repoUserService   fakeclient.RepoUserService
		getArgUsername    string
		inviteArgUsername string
		inviteArgPath     api.RepoPath
		out               string
		err               error
	}{
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"get user error": {
			userService: fakeclient.UserService{
				Getter: fakeclient.UserGetter{
					Err: testErr,
				},
			},
			err: testErr,
		},
		"success force": {
			cmd: RepoInviteCommand{
				path:     "dev2/repo",
				username: "dev1",
				force:    true,
			},
			userService: fakeclient.UserService{
				Getter: fakeclient.UserGetter{
					ReturnsUser: &api.User{
						Username: "dev1",
					},
				},
			},
			repoUserService: fakeclient.RepoUserService{
				RepoInviter: fakeclient.RepoInviter{
					ReturnsRepoMember: &api.RepoMember{},
				},
			},
			getArgUsername:    "dev1",
			inviteArgUsername: "dev1",
			inviteArgPath:     "dev2/repo",
			out:               "Inviting user...\nInvite complete! The user dev1 is now a member of the dev2/repo repository.\n",
		},
		"invite error": {
			cmd: RepoInviteCommand{
				path:     "dev2/repo",
				username: "dev1",
				force:    true,
			},
			userService: fakeclient.UserService{
				Getter: fakeclient.UserGetter{
					ReturnsUser: &api.User{
						Username: "dev1",
					},
				},
			},
			repoUserService: fakeclient.RepoUserService{
				RepoInviter: fakeclient.RepoInviter{
					Err: testErr,
				},
			},
			getArgUsername:    "dev1",
			inviteArgUsername: "dev1",
			inviteArgPath:     "dev2/repo",
			out:               "Inviting user...\n",
			err:               testErr,
		},
		// TODO SHDEV-1029: Add cases for confirm and abort after extracting AskForConfirmation out of ui.IO.
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
						RepoService: &fakeclient.RepoService{
							UserService: &tc.repoUserService,
						},
						UserService: &tc.userService,
					}, nil
				}
			}

			io := ui.NewFakeIO()
			tc.cmd.io = io

			// Run
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
			assert.Equal(t, tc.userService.Getter.ArgUsername, tc.getArgUsername)
			assert.Equal(t, tc.repoUserService.RepoInviter.ArgUsername, tc.inviteArgUsername)
			assert.Equal(t, tc.repoUserService.RepoInviter.ArgPath, tc.inviteArgPath)
		})
	}
}
